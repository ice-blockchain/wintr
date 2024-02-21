// SPDX-License-Identifier: ice License 1.0

package websocket

import (
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"net/http"
)

var development bool

type (
	srv struct {
		server *http.Server
		cfg    *internal.Config
	}
)
