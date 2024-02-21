// SPDX-License-Identifier: ice License 1.0

package http3webtransport

import (
	"context"
	"fmt"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/qlog"
	"github.com/quic-go/webtransport-go"
	"net/http"
)

func New(cfg *internal.Config, wshandler internal.WSHandler, handler http.Handler) internal.Server {
	s := &srv{cfg: cfg}
	s.handler = s.handleWebTransport(wshandler, handler)
	return s
}

func (s *srv) ListenAndServeTLS(ctx context.Context, certFile, keyFile string) error {
	wtserver := &webtransport.Server{
		H3: http3.Server{
			Addr:    fmt.Sprintf(":%v", s.cfg.WSServer.Port),
			Port:    int(s.cfg.WSServer.Port),
			Handler: s.handler,
			QuicConfig: &quic.Config{
				Tracer: qlog.DefaultTracer,
			},
		},
	}
	if s.cfg.Development {
		noCors := func(r *http.Request) bool {
			return true
		}
		wtserver.CheckOrigin = noCors
	}
	s.server = wtserver
	return s.server.ListenAndServeTLS(certFile, keyFile)
}
func (s *srv) handleWebTransport(wsHandler internal.WSHandler, handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method == http.MethodConnect {
			conn, err := s.server.Upgrade(w, r)
			if err != nil {
				log.Error(errors.Wrapf(err, "upgrading failed"))
				w.WriteHeader(500)
				return
			}
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				log.Error(errors.Wrapf(err, "getting stream failed"))
				w.WriteHeader(500)
				return
			}
			defer stream.Close()
			wsHandler.HandleWS(ctx, stream)
		} else {
			if handler != nil {
				handler.ServeHTTP(w, r)
			}
		}
	}
}

func (s *srv) Shutdown(ctx context.Context) error {
	return s.server.Close()
}
