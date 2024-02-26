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
				w.WriteHeader(http.StatusBadRequest)

				return
			}
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				log.Error(errors.Wrapf(err, "getting stream failed"))
				w.WriteHeader(500)
				return
			}
			defer stream.Close()
			go wsHandler.Read(ctx, &wsAdapter{stream: stream})
			go wsHandler.Write(ctx, &wsAdapter{stream: stream})
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

func (w *wsAdapter) WriteMessage(_ int, data []byte) (err error) {
	_, err = w.stream.Write(data)
	return errors.Wrapf(err, "failed to write data to webtransport stream")
}

func (w *wsAdapter) Close() error {
	return w.stream.Close()
}

func (w *wsAdapter) ReadMessage() (messageType int, p []byte, err error) {
	_, err = w.stream.Read(p)
	return 1, p, errors.Wrapf(err, "failed to read data from webtransport stream")
}
