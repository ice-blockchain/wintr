package websocket

import (
	"context"
	"fmt"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"net"
	"net/http"
)

func New(cfg *internal.Config, wshandler internal.WsHandlerFunc, handler http.Handler) internal.Server {
	s := &srv{cfg: cfg}
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.WSServer.Port),
		Handler: s.handleWebSocket(wshandler, handler),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	return s
}

func (s *srv) ListenAndServeTLS(certFile, keyFile string) error {
	return s.server.ListenAndServeTLS(certFile, keyFile)
}
func (s *srv) handleWebSocket(wsHandlerFunc internal.WsHandlerFunc, handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (s *srv) Shutdown(ctx context.Context) error {
	return s.server.Close()
}
