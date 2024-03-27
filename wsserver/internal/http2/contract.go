// SPDX-License-Identifier: ice License 1.0

package http2

import (
	"net/http"
	stdlibtime "time"

	h2ec "github.com/ice-blockchain/go/src/net/http"

	"github.com/ice-blockchain/wintr/wsserver/internal"
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
