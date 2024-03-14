// SPDX-License-Identifier: ice License 1.0

package http3

import (
	"context"
	"net/http"

	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	cws "github.com/ice-blockchain/wintr/wsserver/internal/connect-ws-upgrader"
)

//nolint:gochecknoglobals // We need single instance.
var (
	//nolint:gochecknoglobals // We need single instance.
	websocketupgrader = cws.ConnectUpgrader{}
)

func (s *srv) handleWebsocket(writer http.ResponseWriter, req *http.Request) (h3ws internal.WS, ctx context.Context, err error) {
	conn, _, _, err := websocketupgrader.Upgrade(req, writer)
	if err != nil {
		err = errors.Wrapf(err, "upgrading http3/websocket failed")
		log.Error(err)
		writer.WriteHeader(http.StatusBadRequest)

		return
	}
	wsocket, ctx := internal.NewWebSocketAdapter(req.Context(), conn, s.cfg.WSServer.ReadTimeout, s.cfg.WSServer.WriteTimeout)

	return wsocket, ctx, nil
}
