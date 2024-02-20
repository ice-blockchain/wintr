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

func New(cfg *internal.Config, handler internal.HandlerFunc) internal.Server {
	s := &srv{}
	wtserver := &webtransport.Server{
		H3: http3.Server{
			Addr:    fmt.Sprintf(":%v", cfg.WSServer.Port),
			Handler: s.handleWebSocket(handler),
			QuicConfig: &quic.Config{
				Tracer: qlog.DefaultTracer,
			},
		},
	}
	s.server = wtserver
	return s
}

func (s *srv) ListenAndServeTLS(certFile, keyFile string) error {
	return s.server.ListenAndServeTLS(certFile, keyFile)
}
func (s *srv) handleWebSocket(handlerFunc internal.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		conn, err := s.server.Upgrade(w, r)
		if err != nil {
			log.Error(errors.Wrapf(err, "upgrading failed"))
			w.WriteHeader(500)
			return
		}
		stream, err := conn.OpenStream()
		if err != nil {
			log.Error(errors.Wrapf(err, "getting stream failed"))
			w.WriteHeader(500)
			return
		}
		defer stream.Close()
		handlerFunc(stream)
	}
}
func (s *srv) Shutdown(ctx context.Context) error {
	return s.server.Close()
}
