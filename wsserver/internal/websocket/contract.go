// SPDX-License-Identifier: ice License 1.0

package websocket

import (
	"net"
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
	wsConnection struct {
		conn         net.Conn
		closeChannel chan struct{}
		writeTimeout stdlibtime.Duration
		readTimeout  stdlibtime.Duration
	}
)

const (
	websocketProtocol = "websocket"
)
