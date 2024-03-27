// SPDX-License-Identifier: ice License 1.0

package http3

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/qlog"
	"github.com/quic-go/webtransport-go"
	"net/http"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/wsserver/internal"
)

func New(cfg *internal.Config, wshandler internal.WSHandler, handler http.Handler) internal.Server {
	s := &srv{cfg: cfg}
	s.handler = s.handle(wshandler, handler)

	return s
}

func (s *srv) ListenAndServeTLS(_ context.Context, certFile, keyFile string) error {
	wtserver := &webtransport.Server{
		H3: http3.Server{
			Addr:    fmt.Sprintf(":%v", s.cfg.WSServer.Port),
			Port:    int(s.cfg.WSServer.Port),
			Handler: s.handler,
			QuicConfig: &quic.Config{
				Tracer:                qlog.DefaultTracer,
				HandshakeIdleTimeout:  acceptStreamTimeout,
				MaxIdleTimeout:        maxIdleTimeout,
				MaxIncomingStreams:    maxStreamsCount,
				MaxIncomingUniStreams: maxStreamsCount,
			},
		},
	}
	if s.cfg.Development {
		noCors := func(_ *http.Request) bool {
			return true
		}
		wtserver.CheckOrigin = noCors
	}
	s.server = wtserver

	return errors.Wrap(s.server.ListenAndServeTLS(certFile, keyFile), "failed to start http3/udp server")
}

//nolint:revive // .
func (s *srv) handle(wsHandler internal.WSHandler, handler http.Handler) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		var ws internal.WSWithWriter
		var err error
		var ctx context.Context
		if req.Method == http.MethodConnect && req.Proto == "webtransport" {
			ws, ctx, err = s.handleWebTransport(writer, req)
		} else if req.Method == http.MethodConnect && req.Proto == "websocket" {
			ws, ctx, err = s.handleWebsocket(writer, req)
		}
		if err != nil {
			log.Error(errors.Wrapf(err, "http3: upgrading failed for %v", req.Proto))
			writer.WriteHeader(http.StatusBadRequest)

			return
		}
		if ws != nil {
			go func() {
				defer func() {
					log.Error(ws.Close(), "failed to close http3 stream")
				}()
				go ws.Write(ctx)        //nolint:contextcheck // It is new context.
				wsHandler.Read(ctx, ws) //nolint:contextcheck // It is new context.
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
	return errors.Wrap(s.server.Close(), "failed to close http3 server")
}
