package internal

import (
	"context"
	"io"
)

type (
	Server interface {
		ListenAndServeTLS(ctx context.Context, certFile, keyFile string) error
		Shutdown(ctx context.Context) error
	}
	// TODO custom interface with methods instead if io package
	//WriteMessage(messageType int, data []byte) error
	//Close
	//ReadMessage() (messageType int, p []byte, err error)
	//Close
	WSHandler interface {
		// TODO: read / write instead
		// call go read / go write in handler to have 2 routines
		HandleWS(ctx context.Context, stream io.ReadWriteCloser)
	}

	Config struct {
		WSServer struct {
			CertPath string `yaml:"certPath"`
			KeyPath  string `yaml:"keyPath"`
			Port     uint16 `yaml:"port"`
		} `yaml:"wsServer"`
		Development bool `yaml:"development"`
	}
)
