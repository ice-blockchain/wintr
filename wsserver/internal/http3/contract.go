// SPDX-License-Identifier: ice License 1.0

package http3

import (
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
)

const (
	acceptStreamTimeout = 60 * stdlibtime.Second
)
