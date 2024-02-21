// SPDX-License-Identifier: ice License 1.0

package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"net/http"
)

var development bool

type (
	srv struct {
		server  *http.Server
		handler http.HandlerFunc
		cfg     *internal.Config
	}
	wsReadWriter struct {
		websocket *websocket.Conn
	}
)
