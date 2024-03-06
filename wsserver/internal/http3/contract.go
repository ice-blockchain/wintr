// SPDX-License-Identifier: ice License 1.0

package http3

import (
	"net"
	"net/http"
	stdlibtime "time"

	"github.com/quic-go/webtransport-go"

	"github.com/ice-blockchain/wintr/wsserver/internal"
)

type (
	srv struct {
		server  *webtransport.Server
		handler http.HandlerFunc
		cfg     *internal.Config
	}
	webtransportAdapter struct {
		stream       webtransport.Stream
		closeChannel chan struct{}
		writeTimeout stdlibtime.Duration
		readTimeout  stdlibtime.Duration
	}
	websocketAdapter struct {
		conn         net.Conn
		closeChannel chan struct{}
		writeTimeout stdlibtime.Duration
		readTimeout  stdlibtime.Duration
	}
)

const (
	acceptStreamTimeout = 30 * stdlibtime.Second
)
