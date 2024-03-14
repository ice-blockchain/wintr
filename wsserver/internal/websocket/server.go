package websocket

import (
	"context"
	"fmt"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	h2ec "github.com/ice-blockchain/wintr/wsserver/internal/websocket/h2extendedconnect"
	"github.com/pkg/errors"
	"net"
	"net/http"
)

func New(cfg *internal.Config, wshandler internal.WSHandler, handler http.Handler) internal.Server {
	s := &srv{cfg: cfg}
	s.handler = s.handle(wshandler, handler)

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
func (s *srv) handle(wsHandler internal.WSHandler, handler http.Handler) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		var wsocket internal.WS
		var ctx context.Context
		var err error
		if req.Header.Get("Upgrade") == websocketProtocol || (req.Method == http.MethodConnect && req.Proto == websocketProtocol) {
			wsocket, ctx, err = s.handleWebsocket(writer, req)
		} else if req.Method == http.MethodConnect && req.Proto == webtransportProtocol {
			wsocket, ctx, err = s.handleWebTransport(writer, req)
		}
		if err != nil {
			log.Error(errors.Wrapf(err, "upgrading failed (http2 / %v)", req.Proto))
			writer.WriteHeader(http.StatusBadRequest)

			return
		}
		if wsocket != nil {
			go func() {
				defer func() {
					log.Error(wsocket.Close(), "failed to close websocket conn")
				}()
				go wsHandler.Write(ctx, wsocket)
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
