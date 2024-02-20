// SPDX-License-Identifier: ice License 1.0

package http3webtransport

import (
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"github.com/quic-go/webtransport-go"
)

var development bool

type (
	srv struct {
		server *webtransport.Server
		cfg    internal.Config
	}
)
