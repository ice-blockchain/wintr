// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"context"
	"io"
	"time"
)

type (
	Server interface {
		ListenAndServeTLS(ctx context.Context, certFile, keyFile string) error
		Shutdown(ctx context.Context) error
	}
	WSReader interface {
		ReadMessage() (messageType int, p []byte, err error)
		io.Closer
	}
	WSWriter interface {
		WriteMessage(messageType int, data []byte) error
		io.Closer
	}
	WS interface {
		WSWriter
		WSReader
	}
	WSHandler interface {
		Read(ctx context.Context, reader WSReader)
		Write(ctx context.Context, writer WSWriter)
	}

	Config struct {
		WSServer struct {
			CertPath     string        `yaml:"certPath"`
			KeyPath      string        `yaml:"keyPath"`
			Port         uint16        `yaml:"port"`
			WriteTimeout time.Duration `yaml:"writeTimeout"`
			ReadTimeout  time.Duration `yaml:"readTimeout"`
		} `yaml:"wsServer"`
		Development bool `yaml:"development"`
	}
)

type (
	customCancelContext struct {
		context.Context //nolint:containedctx // Custom implementation.
		ch              <-chan struct{}
	}
)
