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
	"io"
	"net/http"
	"time"
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
			acceptCtx, acceptCancel := context.WithTimeout(r.Context(), acceptStreamTimeout)
			stream, err := conn.AcceptStream(acceptCtx)
			if err != nil {
				acceptCancel()
				log.Error(errors.Wrapf(err, "getting stream failed"))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			acceptCancel()
			ws := &wsAdapter{
				stream:       stream,
				conn:         conn,
				closeChannel: make(chan struct{}, 1),
				readTimeout:  s.cfg.WSServer.ReadTimeout,
				writeTimeout: s.cfg.WSServer.WriteTimeout,
			}
			ctx = internal.NewCustomCancelContext(conn.Context(), ws.closeChannel)
			defer func() {
				log.Error(ws.Close(), "failed to close webtransport conn")
			}()
			go wsHandler.Write(ctx, ws)
			wsHandler.Read(ctx, ws)
			return
		} else if r.Header.Get("Upgrade") == "websocket" {
			fmt.Println("h3 ws")
			w.WriteHeader(http.StatusOK)
			return
		} else {
			fmt.Println("Upgrade " + r.Header.Get("Upgrade"))
			if handler != nil {
				handler.ServeHTTP(w, r)
			}
		}
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s *srv) Shutdown(ctx context.Context) error {
	return s.server.Close()
}

func (w *wsAdapter) WriteMessage(_ int, data []byte) (err error) {
	if w.writeTimeout > 0 {
		_ = w.stream.SetWriteDeadline(time.Now().Add(w.writeTimeout))
	}
	_, err = w.stream.Write(data)
	return errors.Wrapf(err, "failed to write data to webtransport stream")
}

func (w *wsAdapter) Close() error {
	close(w.closeChannel)
	return w.stream.Close()
}

func (w *wsAdapter) ReadMessage() (messageType int, p []byte, err error) {
	if w.readTimeout > 0 {
		_ = w.stream.SetReadDeadline(time.Now().Add(w.readTimeout))
	}
	p, err = io.ReadAll(w.stream)
	return 1, p, errors.Wrapf(err, "failed to read data from webtransport stream")
}
