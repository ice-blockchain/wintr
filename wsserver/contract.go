// SPDX-License-Identifier: ice License 1.0

package wsserver

import (
	"context"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"io"
	"os"
)

type (
	Server interface {
		// ListenAndServe starts everything and blocks indefinitely.
		ListenAndServe(ctx context.Context, cancel context.CancelFunc)
	}

	Service interface {
		HandleWS(stream io.ReadWriteCloser)
		Close(ctx context.Context) error
	}
)

type (
	srv struct {
		server  internal.Server
		cfg     *internal.Config
		quit    chan<- os.Signal
		service Service
	}
)
