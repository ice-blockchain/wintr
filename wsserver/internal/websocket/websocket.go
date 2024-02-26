package websocket

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferPool: &sync.Pool{},
}

func New(cfg *internal.Config, wshandler internal.WSHandler, handler http.Handler) internal.Server {
	s := &srv{cfg: cfg}
	s.handler = s.handleWebSocket(wshandler, handler)
	return s
}

func (s *srv) ListenAndServeTLS(ctx context.Context, certFile, keyFile string) error {
	s.server = &http.Server{
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
				p := initWSConnection(conn, s.cfg)
				defer p.Close()
				ctx := newCtx(r.Context(), p.closeChannel)
				go wsHandler.Write(ctx, p)
				wsHandler.Read(ctx, p)
			}()

		} else if handler != nil {
			handler.ServeHTTP(w, r)
		}
	}
}

func (s *srv) Shutdown(ctx context.Context) error {
	return s.server.Close()
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

func (r *wsConnection) ReadMessage() (messageType int, p []byte, err error) {
	if r.readTimeout > 0 {
		_ = r.conn.SetReadDeadline(time.Now().Add(r.readTimeout))
	}
	return r.conn.ReadMessage() //nolint:wrapcheck // Proxy.
}

func (w *wsConnection) Close() error {
	w.closeChannel <- struct{}{}
	err := w.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(w.writeTimeout))
	if err != nil && err != websocket.ErrCloseSent {
		return w.conn.Close()
	}
	return nil
}

func newCtx(ctx context.Context, ch <-chan struct{}) context.Context {
	return customCancelContext{Context: ctx, ch: ch}
}
func (c customCancelContext) Done() <-chan struct{} {
	return c.ch
}

func (c customCancelContext) Err() error {
	select {
	case <-c.ch:
		return context.Canceled
	default:
		return nil
	}
}
