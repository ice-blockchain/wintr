// SPDX-License-Identifier: ice License 1.0

package http2

import (
	"context"
	"net"
	"net/http"
	stdlibtime "time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	cws "github.com/ice-blockchain/wintr/wsserver/internal/connect-ws-upgrader"
)

//nolint:gochecknoglobals,grouper // We need single instance to avoid spending extra mem
var h2Upgrader = &cws.ConnectUpgrader{}

func (s *srv) handleWebsocket(writer http.ResponseWriter, req *http.Request) (h2ws internal.WSWithWriter, ctx context.Context, err error) {
	var conn net.Conn
	if req.Header.Get("Upgrade") == websocketProtocol {
		conn, _, _, err = ws.DefaultHTTPUpgrader.Upgrade(req, writer)
	} else if req.Method == http.MethodConnect && req.Proto == websocketProtocol {
		conn, _, _, err = h2Upgrader.Upgrade(req, writer)
	}
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to upgrade to websocket over http1/2: %v, upgrade: %v", req.Proto, req.Header.Get("Upgrade"))
	}
	wsocket, ctx := internal.NewWebSocketAdapter(req.Context(), conn, s.cfg.WSServer.ReadTimeout, s.cfg.WSServer.WriteTimeout)
	go s.ping(ctx, conn)

	return wsocket, ctx, nil
}

func (s *srv) ping(ctx context.Context, conn net.Conn) {
	ticker := stdlibtime.NewTicker(stdlibtime.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := multierror.Append(
				conn.SetWriteDeadline(time.Now().Add(s.cfg.WSServer.WriteTimeout)),
				wsutil.WriteServerMessage(conn, ws.OpPing, nil),
			).ErrorOrNil(); err != nil {
				log.Error(errors.Wrapf(err, "failed to send ping message"))
			}
		case <-ctx.Done():
			return
		}
	}
}
