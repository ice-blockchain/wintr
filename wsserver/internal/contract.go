// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"context"
	"io"
	"net"
	stdlibtime "time"

	"github.com/quic-go/webtransport-go"
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
		Read(ctx context.Context, reader WS)
		Write(ctx context.Context, writer WSWriter)
		// We have to add something to update context / WS (another wrapper) on app side to handle random challenge string for NIP-42
		// and / or something else related to connection.
	}

	Config struct {
		WSServer struct {
			CertPath     string              `yaml:"certPath"`
			KeyPath      string              `yaml:"keyPath"`
			Port         uint16              `yaml:"port"`
			WriteTimeout stdlibtime.Duration `yaml:"writeTimeout"`
			ReadTimeout  stdlibtime.Duration `yaml:"readTimeout"`
		} `yaml:"wsServer"`
		Development bool `yaml:"development"`
	}

	WebtransportAdapter struct {
		stream       webtransport.Stream
		closeChannel chan struct{}
		writeTimeout stdlibtime.Duration
		readTimeout  stdlibtime.Duration
	}

	WebsocketAdapter struct {
		conn         net.Conn
		closeChannel chan struct{}
		writeTimeout stdlibtime.Duration
		readTimeout  stdlibtime.Duration
	}
)

type (
	customCancelContext struct {
		context.Context //nolint:containedctx // Custom implementation.
		ch              <-chan struct{}
	}
)
