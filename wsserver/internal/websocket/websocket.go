package websocket

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"github.com/pkg/errors"
	"io"
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
		ctx := r.Context()
		if r.Header.Get("Upgrade") == "websocket" {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Error(errors.Wrapf(err, "upgrading failed"))
				w.WriteHeader(500)
				return
			}
			defer conn.Close()
			wsHandler.HandleWS(ctx, &wsReadWriter{websocket: conn})
		} else if handler != nil {
			handler.ServeHTTP(w, r)
		}
	}
}

func (s *srv) Shutdown(ctx context.Context) error {
	return s.server.Close()
}
func (w *wsReadWriter) Read(p []byte) (n int, err error) {
	_, b, err := w.websocket.ReadMessage()

	n = copy(p, b)

	return n, io.EOF
}

func (w *wsReadWriter) Write(p []byte) (n int, err error) {
	err = w.websocket.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

func (w *wsReadWriter) Close() error {
	return w.websocket.Close()
}
