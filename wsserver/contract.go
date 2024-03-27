// SPDX-License-Identifier: ice License 1.0

package wsserver

import (
	"context"
	"os"

	"github.com/ice-blockchain/wintr/server"
	"github.com/ice-blockchain/wintr/wsserver/internal"
)

type (
	Router = server.Router
	Server interface {
		// ListenAndServe starts everything and blocks indefinitely.
		ListenAndServe(ctx context.Context, cancel context.CancelFunc)
	}

	Service interface {
		internal.WSHandler
		Init(ctx context.Context, cancel context.CancelFunc)
		Close(ctx context.Context) error
		RegisterRoutes(r *Router)
	}
	WSReader = internal.WSReader
	WSWriter = internal.WSWriter
	WS       = internal.WS
)

type (
	srv struct {
		h3server internal.Server
		wsServer internal.Server
		router   *Router
		cfg      *internal.Config
		quit     chan<- os.Signal
		service  Service
	}
)
