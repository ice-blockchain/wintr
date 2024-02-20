package internal

import (
	"context"
	"io"
)

type (
	Server interface {
		ListenAndServeTLS(certFile, keyFile string) error
		Shutdown(ctx context.Context) error
	}
	HandlerFunc func(stream io.ReadWriteCloser)

	Config struct {
		WSServer struct {
			CertPath string `yaml:"certPath"`
			KeyPath  string `yaml:"keyPath"`
			Port     uint16 `yaml:"port"`
		} `yaml:"wsServer"`
	}
)
