// SPDX-License-Identifier: ice License 1.0

package websocket

import (
	"github.com/ice-blockchain/wintr/wsserver/internal"
	h2ec "github.com/ice-blockchain/wintr/wsserver/internal/websocket/h2extendedconnect"
	"net"
	"net/http"
	stdlibtime "time"
)

type (
	srv struct {
		server  *h2ec.Server
		handler http.HandlerFunc
		cfg     *internal.Config
	}
	wsConnection struct {
		conn         net.Conn
		writeTimeout stdlibtime.Duration
		readTimeout  stdlibtime.Duration
		closeChannel chan struct{}
	}
)
