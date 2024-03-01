package websocket

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	h2ec "github.com/ice-blockchain/wintr/wsserver/internal/websocket/h2extendedconnect"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"sync"
	stdlibtime "time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  internal.ReadBufferSize,
	WriteBufferPool: &sync.Pool{},
}

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
	return s.server.ListenAndServeTLS(certFile, keyFile)
}
func (s *srv) handleWebSocket(wsHandler internal.WSHandler, handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Error(errors.Wrapf(err, "upgrading failed"))
				w.WriteHeader(http.StatusBadRequest)

				return
			}
			go func() {
				ws := initWSConnection(conn, s.cfg)
				defer func() {
					log.Error(ws.Close(), "failed to close websocket conn")
				}()
				log.Info("ws esbablished")
				ctx := internal.NewCustomCancelContext(r.Context(), ws.closeChannel)
				go wsHandler.Write(ctx, ws)
				go s.ping(ctx, conn)
				wsHandler.Read(ctx, ws)
			}()

			return
		} else if handler != nil {
			handler.ServeHTTP(w, r)
		}
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s *srv) Shutdown(ctx context.Context) error {
	return s.server.Close()
}

func (s *srv) ping(ctx context.Context, conn *websocket.Conn) {
	ticker := stdlibtime.NewTicker(stdlibtime.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(s.cfg.WSServer.WriteTimeout)); err != nil {
				log.Error(errors.Wrapf(err, "failed to send ping message"))
			}
		case <-ctx.Done():
			return
		}
	}
}

func initWSConnection(conn *websocket.Conn, cfg *internal.Config) *wsConnection {
	return &wsConnection{conn: conn, writeTimeout: cfg.WSServer.WriteTimeout, readTimeout: cfg.WSServer.ReadTimeout, closeChannel: make(chan struct{}, 1)}
}

func (w *wsConnection) WriteMessage(messageType int, data []byte) error {
	var err *multierror.Error
	if w.writeTimeout > 0 {
		err = multierror.Append(nil, w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)))
	}
	return multierror.Append(err,
		w.conn.WriteMessage(messageType, data),
	).ErrorOrNil()

}

func (w *wsConnection) ReadMessage() (messageType int, p []byte, err error) {
	if w.readTimeout > 0 {
		_ = w.conn.SetReadDeadline(time.Now().Add(w.readTimeout))
	}
	typ, msgBytes, err := w.conn.ReadMessage() //nolint:wrapcheck // Proxy.
	if err != nil {
		return typ, msgBytes, err
	}
	if typ == websocket.PingMessage {
		err = w.conn.WriteMessage(websocket.PongMessage, nil)
		if err == nil {
			return w.ReadMessage()
		} else {
			return typ, msgBytes, err
		}
	}
	return typ, msgBytes, err
}

func (w *wsConnection) Close() error {
	close(w.closeChannel)
	err := w.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(w.writeTimeout))
	if err != nil && err != websocket.ErrCloseSent {
		return w.conn.Close()
	}
	return nil
}
