// SPDX-License-Identifier: ice License 1.0

package websocket

import (
	"context"
	"fmt"
	"net"
	"net/http"
	stdlibtime "time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	h2ec "github.com/ice-blockchain/wintr/wsserver/internal/websocket/h2extendedconnect"
	"github.com/ice-blockchain/wintr/wsserver/internal/websocket/h2upgrader"
)

//nolint:gochecknoglobals,grouper // We need single instance to avoid spending extra mem
var h2Upgrader = &h2upgrader.H2Upgrader{}

func New(cfg *internal.Config, wshandler internal.WSHandler, handler http.Handler) internal.Server {
	s := &srv{cfg: cfg}
	s.handler = s.handleWebSocket(wshandler, handler)

	return s
}

func (s *srv) ListenAndServeTLS(ctx context.Context, certFile, keyFile string) error {
	s.server = &h2ec.Server{
		Addr:    fmt.Sprintf(":%v", s.cfg.WSServer.Port),
		Handler: s.handler,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	return s.server.ListenAndServeTLS(certFile, keyFile) //nolint:contextcheck,wrapcheck // .
}

//nolint:funlen,revive // .
func (s *srv) handleWebSocket(wsHandler internal.WSHandler, handler http.Handler) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		var conn net.Conn
		var err error
		if req.Header.Get("Upgrade") == websocketProtocol {
			conn, _, _, err = ws.DefaultHTTPUpgrader.Upgrade(req, writer)
		} else if req.Method == http.MethodConnect && req.Proto == websocketProtocol {
			conn, _, _, err = h2Upgrader.Upgrade(req, writer)
		}
		if err != nil {
			log.Error(errors.Wrapf(err, "upgrading failed"))
			writer.WriteHeader(http.StatusBadRequest)

			return
		}
		if conn != nil {
			go func() {
				wsocket := initWSConnection(conn, s.cfg)
				defer func() {
					log.Error(wsocket.Close(), "failed to close websocket conn")
				}()
				ctx := internal.NewCustomCancelContext(req.Context(), wsocket.closeChannel)
				go wsHandler.Write(ctx, wsocket)
				go s.ping(ctx, conn)
				wsHandler.Read(ctx, wsocket)
			}()

			return
		} else if handler != nil {
			handler.ServeHTTP(writer, req)

			return
		}
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *srv) Shutdown(_ context.Context) error {
	return errors.Wrapf(s.server.Close(), "failed to close server")
}

func (*srv) ping(ctx context.Context, conn net.Conn) {
	ticker := stdlibtime.NewTicker(stdlibtime.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := wsutil.WriteServerMessage(conn, ws.OpPing, nil); err != nil {
				log.Error(errors.Wrapf(err, "failed to send ping message"))
			}
		case <-ctx.Done():
			return
		}
	}
}

func initWSConnection(conn net.Conn, cfg *internal.Config) *wsConnection {
	return &wsConnection{conn: conn, writeTimeout: cfg.WSServer.WriteTimeout, readTimeout: cfg.WSServer.ReadTimeout, closeChannel: make(chan struct{}, 1)}
}

func (w *wsConnection) WriteMessage(messageType int, data []byte) error {
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

func (w *wsConnection) ReadMessage() (messageType int, p []byte, err error) {
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

func (w *wsConnection) Close() error {
	close(w.closeChannel)

	return multierror.Append( //nolint:wrapcheck // .
		wsutil.WriteServerMessage(w.conn, ws.OpClose, ws.NewCloseFrameBody(ws.StatusNormalClosure, "")),
		w.conn.Close(),
	).ErrorOrNil()
}
