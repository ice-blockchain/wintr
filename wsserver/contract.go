// SPDX-License-Identifier: ice License 1.0

package wsserver

import (
	"context"
	"os"

	"github.com/ice-blockchain/wintr/wsserver/internal"
)

type (
	Server interface {
		// ListenAndServe starts everything and blocks indefinitely.
		ListenAndServe(ctx context.Context, cancel context.CancelFunc)
	}

	Service interface {
		internal.WSHandler
		Close(ctx context.Context) error
	}
	WSReader = internal.WSReader
	WSWriter = internal.WSWriter
)

type (
	srv struct {
		h3server internal.Server
		wsServer internal.Server
		cfg      *internal.Config
		quit     chan<- os.Signal
		service  Service
	}
)
