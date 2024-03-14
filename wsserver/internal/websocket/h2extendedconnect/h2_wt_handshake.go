// SPDX-License-Identifier: ice License 1.0

package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	stdlibtime "time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/quicvarint"
	"github.com/quic-go/webtransport-go"

	"github.com/ice-blockchain/wintr/log"
)

const (
	wtCapsuleResetStream              = 0x190B4D39
	wtCapsuleStopSending              = 0x190B4D3A
	wtCapsuleStream                   = 0x190B4D3B
	wtCapsuleStreamFin                = 0x190B4D3C
	wtCapsuleMaxData                  = 0x190B4D3D
	wtCapsuleMaxStreamData            = 0x190B4D3E
	wtCapsuleMaxStreams               = 0x190B4D3F
	wtCapsuleMaxStreamsUni            = 0x190B4D40
	wtCapsuleCloseWebTransportSession = 0x2843
	wtCapsuleDrainWebTransportSession = 0x78ae
)

type (
	Session interface {
		AcceptStream(ctx context.Context) webtransport.Stream
	}
	WebTransportUpgrader interface {
		UpgradeWebTransport() (Session, error)
	}
	webtransportStream struct {
		rw               *http2responseWriter
		streamReceivedCh chan io.Reader
		readFinished     chan struct{}
		reader           io.Reader
		streamReceived   bool
		finReceived      bool
	}
	wtMaxData struct {
		MaxData uint64
	}
	wtMaxStreams struct {
		MaxStreams uint64
	}
	wtMaxStreamData struct {
		StreamID uint64
		MaxData  uint64
	}
	wtStream struct {
		ws         *webtransportStream
		StreamID   uint32
		StreamData []byte
	}
)

func (rw *http2responseWriter) UpgradeWebTransport() (Session, error) {
	if !(rw.rws.req.Method == http.MethodConnect && rw.rws.req.Proto == "webtransport") {
		rw.WriteHeader(400)
		return nil, errors.New("invalid protocol")
	}
	rw.Header().Add(headerCapsuleProtocol, strconv.FormatBool(true))

	rw.WriteHeader(http.StatusOK)
	wts := &webtransportStream{
		rw:               rw,
		streamReceivedCh: make(chan io.Reader, 1),
		readFinished:     make(chan struct{}, 1),
	}
	rw.rws.conn.webtransportSessions.Store(rw.rws.stream.id, wts)

	return rw, nil
}

func (rw *http2responseWriter) AcceptStream(ctx context.Context) webtransport.Stream {
	var stream *webtransportStream
	if s, ok := rw.rws.conn.webtransportSessions.Load(rw.rws.stream.id); ok {
		stream = s.(*webtransportStream)
	}
	go stream.handleWebTransportStream()
	return stream
}

func (s *webtransportStream) handleWebTransportStream() {
	for {
		if s.finReceived || s.rw.rws.handlerDone || s.rw.rws.stream.state == http2stateClosed {
			return
		}
		if s.streamReceived {
			select {
			case <-s.readFinished:
			}
		}
		cType, data, err := http3.ParseCapsule(quicvarint.NewReader(s.rw.rws.req.Body))
		cData := bufio.NewReader(data)
		if err != nil {
			if !errors.Is(err, http2errClientDisconnected) {
				log.Error(errors.Wrap(err, "failed to parse capsule error (http2/wt/rfc9297)"))
			}
			break
		}
		if cType > 0 {
			log.Debug(fmt.Sprintf("http2/wt: Received capsule 0x%v", strconv.FormatUint(uint64(cType), 16)))
			switch cType {
			case wtCapsuleMaxStreamData:
				md := new(wtMaxStreamData)
				if err = md.Deserialize(cData); err == nil {
					s.rw.rws.stream.sc.sendWindowUpdate(s.rw.rws.stream, int(md.MaxData))
					s.rw.Flush()
				}
			case wtCapsuleMaxData:
				md := new(wtMaxData)
				if err = md.Deserialize(cData); err == nil {
					s.rw.rws.stream.sc.sendWindowUpdate(s.rw.rws.stream, int(md.MaxData))
					s.rw.Flush()
				}
			case wtCapsuleMaxStreams:
				ms := new(wtMaxStreams)
				if err = ms.Deserialize(cData); err == nil {
					s.rw.rws.stream.sc.advMaxStreams = uint32(ms.MaxStreams)
				}
			case wtCapsuleStreamFin:
			case wtCapsuleStream:
				s.finReceived = cType == wtCapsuleStreamFin
				str := &wtStream{ws: s}
				err = str.Deserialize(cData)
			case wtCapsuleResetStream:
				s.rw.rws.stream.endStream()
			case wtCapsuleStopSending:
				s.rw.handlerDone()
			case wtCapsuleDrainWebTransportSession:
				s.rw.rws.conn.startGracefulShutdown()
			case wtCapsuleCloseWebTransportSession:
				s.rw.handlerDone()
			default:
				_, err = io.ReadAll(cData)
			}

			if err != nil {
				log.Error(errors.Wrap(err, "failed to process capsule (http2/wt/rfc9297)"))
			}
		}
	}
}

func (s *wtStream) Deserialize(dataReader quicvarint.Reader) (err error) {
	var sID uint64
	sID, err = quicvarint.Read(dataReader)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse WT_STREAM/StreamID")
		return err
	}
	s.StreamID = uint32(sID)
	s.ws.streamReceivedCh <- dataReader
	return errors.Wrapf(err, "failed to copy content from WT_STREAM")
}
func (s *wtStream) Serialize() []byte {
	b := make([]byte, 0, 4+len(s.StreamData))
	b = quicvarint.Append(b, uint64(s.StreamID))
	b = append(b, s.StreamData...)

	return b
}

func (s *webtransportStream) Write(p []byte) (n int, err error) {
	err = s.rw.WriteCapsule(wtCapsuleStream, &wtStream{StreamData: p, StreamID: s.rw.rws.stream.id})

	return len(p), err
}

func (s *webtransportStream) Close() error {
	s.rw.handlerDone()

	return nil
}

func (s *webtransportStream) StreamID() quic.StreamID {
	return quic.StreamID(s.rw.rws.stream.id)
}

func (s *webtransportStream) CancelWrite(code webtransport.StreamErrorCode) {

}

func (s *webtransportStream) SetWriteDeadline(t stdlibtime.Time) error {
	return s.rw.SetWriteDeadline(t)
}

func (s *webtransportStream) Read(p []byte) (n int, err error) {
	if s.finReceived || s.rw.rws.handlerDone || s.rw.rws.stream.state == http2stateClosed {
		return 0, io.ErrUnexpectedEOF
	}
	var r io.Reader
	if s.reader == nil {
		select {
		case r = <-s.streamReceivedCh:
			s.reader = r
		default:
			return 0, io.EOF
		}
	}
	if r != s.reader {
		s.reader = nil
		return 0, io.EOF
	}
	n, err = s.reader.Read(p)
	if errors.Is(err, io.EOF) {
		s.reader = nil
		s.readFinished <- struct{}{}
	}
	return n, err
}

func (s *webtransportStream) CancelRead(code webtransport.StreamErrorCode) {
}

func (s *webtransportStream) SetReadDeadline(t stdlibtime.Time) error {
	return s.rw.SetReadDeadline(t)
}

func (s *webtransportStream) SetDeadline(t stdlibtime.Time) error {
	return multierror.Append(
		s.rw.SetReadDeadline(t),
		s.rw.SetWriteDeadline(t),
	).ErrorOrNil()
}

func (md *wtMaxData) Deserialize(dataReader quicvarint.Reader) (err error) {
	if md.MaxData, err = quicvarint.Read(dataReader); err != nil {
		err = errors.Wrapf(err, "failed to parse WT_MAX_DATA/MaxData")
	}
	return err
}
func (ms *wtMaxStreams) Deserialize(dataReader quicvarint.Reader) (err error) {
	if ms.MaxStreams, err = quicvarint.Read(dataReader); err != nil {
		err = errors.Wrapf(err, "failed to parse WT_MAX_STREAMS/Maximum Streams")
	}
	return err
}
func (md *wtMaxStreamData) Deserialize(dataReader quicvarint.Reader) (err error) {
	if md.StreamID, err = quicvarint.Read(dataReader); err != nil {
		err = errors.Wrapf(err, "failed to parse WT_MAX_STREAM_DATA/StreamID")
		return err
	}
	if md.MaxData, err = quicvarint.Read(dataReader); err != nil {
		return errors.Wrapf(err, "failed to parse WT_MAX_STREAM_DATA/MaxData")
	}
	return err
}
