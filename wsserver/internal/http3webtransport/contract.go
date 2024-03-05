// SPDX-License-Identifier: ice License 1.0

package http3webtransport

import (
	"net/http"
	stdlibtime "time"

	"github.com/quic-go/webtransport-go"

	"github.com/ice-blockchain/wintr/wsserver/internal"
)

var development bool

type (
	srv struct {
		server  *webtransport.Server
		handler http.HandlerFunc
		cfg     *internal.Config
	}
	wsAdapter struct {
		conn         *webtransport.Session
		stream       webtransport.Stream
		writeTimeout stdlibtime.Duration
		readTimeout  stdlibtime.Duration
		closeChannel chan struct{}
	}
)

const acceptStreamTimeout = 30 * stdlibtime.Second
