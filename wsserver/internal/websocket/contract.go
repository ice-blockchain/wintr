// SPDX-License-Identifier: ice License 1.0

package websocket

import (
	"net/http"
	stdlibtime "time"

	"github.com/ice-blockchain/wintr/wsserver/internal"
	h2ec "github.com/ice-blockchain/wintr/wsserver/internal/websocket/h2extendedconnect"
)

type (
	srv struct {
		server  *h2ec.Server
		handler http.HandlerFunc
		cfg     *internal.Config
	}
)

const (
	websocketProtocol    = "websocket"
	webtransportProtocol = "webtransport"
	acceptStreamTimeout  = 30 * stdlibtime.Second
)
