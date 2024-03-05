package websocket

import (
	"context"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hashicorp/go-multierror"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	h2ec "github.com/ice-blockchain/wintr/wsserver/internal/websocket/h2extendedconnect"
	"github.com/ice-blockchain/wintr/wsserver/internal/websocket/h2upgrader"
	"github.com/pkg/errors"
	"net"
	"net/http"
	stdlibtime "time"
)

//	var upgrader = websocket.Upgrader{
//		ReadBufferSize:  internal.ReadBufferSize,
//		WriteBufferPool: &sync.Pool{},
//	}
var h2Upgrader = &h2upgrader.H2Upgrader{}

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
		var conn net.Conn = nil
		var err error
		if r.Header.Get("Upgrade") == "websocket" {
			conn, _, _, err = ws.DefaultHTTPUpgrader.Upgrade(r, w)
		} else if r.Method == http.MethodConnect && r.Proto == "websocket" {
			conn, _, _, err = h2Upgrader.Upgrade(r, w)
		}
		if err != nil {
			log.Error(errors.Wrapf(err, "upgrading failed"))
			w.WriteHeader(http.StatusBadRequest)

			return
		}
		if conn != nil {
			go func() {
				wsocket := initWSConnection(conn, s.cfg)
				defer func() {
					log.Error(wsocket.Close(), "failed to close websocket conn")
				}()
				log.Info("ws esbablished")
				ctx := internal.NewCustomCancelContext(r.Context(), wsocket.closeChannel)
				go wsHandler.Write(ctx, wsocket)
				go s.ping(ctx, conn)
				wsHandler.Read(ctx, wsocket)
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

func (s *srv) ping(ctx context.Context, conn net.Conn) {
	ticker := stdlibtime.NewTicker(stdlibtime.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := wsutil.WriteServerMessage(conn, ws.OpPing, nil); err != nil {
				log.Error(errors.Wrapf(err, "failed to send ping message"))
			}
		case <-ctx.Done():
			return
		}
	}
}

func initWSConnection(conn net.Conn, cfg *internal.Config) *wsConnection {
	return &wsConnection{conn: conn, writeTimeout: cfg.WSServer.WriteTimeout, readTimeout: cfg.WSServer.ReadTimeout, closeChannel: make(chan struct{}, 1)}
}

func (w *wsConnection) WriteMessage(messageType int, data []byte) error {
	var err error
	if w.writeTimeout > 0 {
		err = multierror.Append(nil, w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)))
	}
	err = multierror.Append(err,
		wsutil.WriteServerMessage(w.conn, ws.OpCode(messageType), data),
	).ErrorOrNil()

	if flusher, ok := w.conn.(http.Flusher); err == nil && ok {
		flusher.Flush()
	}

	return err
}

func (w *wsConnection) ReadMessage() (messageType int, p []byte, err error) {
	if w.readTimeout > 0 {
		_ = w.conn.SetReadDeadline(time.Now().Add(w.readTimeout))
	}
	msgBytes, typ, err := wsutil.ReadClientData(w.conn)
	if err != nil {
		return int(typ), msgBytes, err
	}
	if typ == ws.OpPing {
		err = wsutil.WriteServerMessage(w.conn, ws.OpPong, nil)
		if err == nil {
			return w.ReadMessage()
		}
		return int(typ), msgBytes, err
	}

	return int(typ), msgBytes, err
}

func (w *wsConnection) Close() error {
	close(w.closeChannel)

	return multierror.Append(
		wsutil.WriteServerMessage(w.conn, ws.OpClose, ws.NewCloseFrameBody(ws.StatusNormalClosure, "")),
		w.conn.Close(),
	).ErrorOrNil()
}
