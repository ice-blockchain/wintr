// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"bufio"
	"context"
	"github.com/quic-go/quic-go"
	"io"
	"strings"
	stdlibtime "time"

	"github.com/pkg/errors"
	"github.com/quic-go/webtransport-go"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

func NewWebTransportAdapter(ctx context.Context, stream webtransport.Stream, readTimeout, writeTimeout stdlibtime.Duration) (WSWithWriter, context.Context) {
	wt := &WebtransportAdapter{
		stream:       stream,
		reader:       bufio.NewReaderSize(stream, 1024),
		closeChannel: make(chan struct{}, 1),
		out:          make(chan []byte),
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}

	return wt, NewCustomCancelContext(ctx, wt.closeChannel)
}

func (w *WebtransportAdapter) WriteMessage(_ int, data []byte) (err error) {
	w.wrErrMx.Lock()
	var sErr *quic.StreamError
	if errors.As(w.wrErr, &sErr) && sErr.ErrorCode == sessionCloseErrorCode {
		w.wrErrMx.Unlock()
		return w.Close()
	}
	w.wrErrMx.Unlock()
	w.out <- data

	return nil
}

func (w *WebtransportAdapter) writeMessageToStream(data []byte) error {
	if w.writeTimeout > 0 {
		_ = w.stream.SetWriteDeadline(time.Now().Add(w.writeTimeout)) //nolint:errcheck // .
	}
	data = append(data, 0x00)
	select {
	case <-w.closeChannel:
		return nil
	default:
		w.closeMx.Lock()
		if w.closed {
			w.closeMx.Unlock()

			return nil
		}
		w.closeMx.Unlock()
		_, err := w.stream.Write(data)
		w.wrErrMx.Lock()
		w.wrErr = err
		w.wrErrMx.Unlock()

		return errors.Wrapf(err, "failed to write data to webtransport stream")
	}
}

func (w *WebtransportAdapter) Write(ctx context.Context) {
	for msg := range w.out {
		var sErr *quic.StreamError
		if ctx.Err() != nil || (errors.As(w.wrErr, &sErr) && strings.Contains(sErr.Error(), "stream error 386759528")) {
			break
		}
		log.Error(w.writeMessageToStream(msg), "failed to send message to webtransport")
	}
}
func (w *WebsocketAdapter) Closed() bool {
	w.closeMx.Lock()
	defer w.closeMx.Unlock()
	return w.closed
}

func (w *WebtransportAdapter) Closed() bool {
	w.closeMx.Lock()
	closed := w.closed
	w.closeMx.Unlock()

	return closed
}

func (w *WebtransportAdapter) Close() error {
	w.closeMx.Lock()
	if w.closed {
		w.closeMx.Unlock()

		return nil
	}
	w.closed = true
	close(w.closeChannel)
	close(w.out)
	w.closeMx.Unlock()

	return errors.Wrap(w.stream.Close(), "failed to close http3/webtransport stream")
}

func (w *WebtransportAdapter) ReadMessage() (messageType int, readValue []byte, err error) {
	if w.readTimeout > 0 {
		_ = w.stream.SetReadDeadline(time.Now().Add(w.readTimeout)) //nolint:errcheck // .
	}
	readValue, err = w.reader.ReadBytes(0x00)
	if len(readValue) > 0 {
		readValue = readValue[0 : len(readValue)-1]
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}

	return 1, readValue, errors.Wrapf(err, "failed to read data from webtransport stream")
}
