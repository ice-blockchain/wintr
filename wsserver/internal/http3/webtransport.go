// SPDX-License-Identifier: ice License 1.0

package http3

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/wsserver/internal"
)

func (s *srv) handleWebTransport(writer http.ResponseWriter, req *http.Request) (ws internal.WS, ctx context.Context, err error) {
	conn, err := s.server.Upgrade(writer, req)
	if err != nil {
		err = errors.Wrapf(err, "upgrading http3/webtransport failed")
		log.Error(err)
		writer.WriteHeader(http.StatusBadRequest)

		return nil, nil, err
	}
	acceptCtx, acceptCancel := context.WithTimeout(req.Context(), acceptStreamTimeout)
	stream, err := conn.AcceptStream(acceptCtx)
	if err != nil {
		acceptCancel()
		err = errors.Wrapf(err, "getting http3/webtransport stream failed")
		log.Error(err)
		writer.WriteHeader(http.StatusBadRequest)

		return nil, nil, err
	}
	acceptCancel()
	wt := &webtransportAdapter{
		stream:       stream,
		closeChannel: make(chan struct{}, 1),
		readTimeout:  s.cfg.WSServer.ReadTimeout,
		writeTimeout: s.cfg.WSServer.WriteTimeout,
	}
	ctx = internal.NewCustomCancelContext(conn.Context(), wt.closeChannel)

	return wt, ctx, nil
}

func (w *webtransportAdapter) WriteMessage(_ int, data []byte) (err error) {
	if w.writeTimeout > 0 {
		_ = w.stream.SetWriteDeadline(time.Now().Add(w.writeTimeout)) //nolint:errcheck // .
	}
	_, err = w.stream.Write(data)

	return errors.Wrapf(err, "failed to write data to webtransport stream")
}

func (w *webtransportAdapter) Close() error {
	close(w.closeChannel)

	return errors.Wrap(w.stream.Close(), "failed to close http3/webtransport stream")
}

func (w *webtransportAdapter) ReadMessage() (messageType int, readValue []byte, err error) {
	if w.readTimeout > 0 {
		_ = w.stream.SetReadDeadline(time.Now().Add(w.readTimeout)) //nolint:errcheck // .
	}
	readValue, err = io.ReadAll(w.stream)

	return 1, readValue, errors.Wrapf(err, "failed to read data from webtransport stream")
}
