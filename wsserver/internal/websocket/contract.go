// SPDX-License-Identifier: ice License 1.0

package websocket

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"net/http"
	stdlibtime "time"
)

type (
	srv struct {
		server  *http.Server
		handler http.HandlerFunc
		cfg     *internal.Config
	}
	wsConnection struct {
		conn         *websocket.Conn
		writeTimeout stdlibtime.Duration
		readTimeout  stdlibtime.Duration
		closeChannel chan struct{}
	}
	customCancelContext struct {
		context.Context
		ch <-chan struct{}
	}
)
