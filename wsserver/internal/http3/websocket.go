// SPDX-License-Identifier: ice License 1.0

package http3

import (
	"context"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	cws "github.com/ice-blockchain/wintr/wsserver/internal/connect-ws-upgrader"
)

//nolint:gochecknoglobals // We need single instance.
var (
	//nolint:gochecknoglobals // We need single instance.
	websocketupgrader = cws.ConnectUpgrader{}
)

func (s *srv) handleWebsocket(writer http.ResponseWriter, req *http.Request) (h3ws internal.WS, ctx context.Context, err error) {
	conn, _, _, err := websocketupgrader.Upgrade(req, writer)
	if err != nil {
		err = errors.Wrapf(err, "upgrading http3/websocket failed")
		log.Error(err)
		writer.WriteHeader(http.StatusBadRequest)

		return
	}
	wsocket := &websocketAdapter{
		conn:         conn,
		closeChannel: make(chan struct{}, 1),
		readTimeout:  s.cfg.WSServer.ReadTimeout,
		writeTimeout: s.cfg.WSServer.WriteTimeout,
	}
	ctx = internal.NewCustomCancelContext(req.Context(), wsocket.closeChannel)

	return wsocket, ctx, nil
}

func (w *websocketAdapter) WriteMessage(messageType int, data []byte) error {
	var err error
	if w.writeTimeout > 0 {
		err = multierror.Append(nil, w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)))
	}
	err = multierror.Append(err,
		wsutil.WriteServerMessage(w.conn, ws.OpCode(messageType), data),
	).ErrorOrNil()

	if flusher, ok := w.conn.(http.Flusher); err == nil && ok {
		flusher.Flush()
	}

	return errors.Wrapf(err, "failed to write data to websocket")
}

func (w *websocketAdapter) ReadMessage() (messageType int, p []byte, err error) {
	if w.readTimeout > 0 {
		_ = w.conn.SetReadDeadline(time.Now().Add(w.readTimeout)) //nolint:errcheck // It is not crucial if we ignore it here.
	}
	msgBytes, typ, err := wsutil.ReadClientData(w.conn)
	if err != nil {
		return int(typ), msgBytes, err
	}
	if typ == ws.OpPing {
		err = wsutil.WriteServerMessage(w.conn, ws.OpPong, nil)
		if err == nil {
			return w.ReadMessage()
		}

		return int(typ), msgBytes, err
	}

	return int(typ), msgBytes, err
}

func (w *websocketAdapter) Close() error {
	close(w.closeChannel)

	return multierror.Append( //nolint:wrapcheck // .
		wsutil.WriteServerMessage(w.conn, ws.OpClose, ws.NewCloseFrameBody(ws.StatusNormalClosure, "")),
		w.conn.Close(),
	).ErrorOrNil()
}
