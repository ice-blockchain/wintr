// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"io"

	"github.com/redis/go-redis/v9"
)

// Public API.

type (
	DB interface {
		redis.Cmdable
		io.Closer
		Ping(ctx context.Context) *redis.StatusCmd
	}
)

// Private API.

type (
	config struct {
		WintrStorage struct {
			Credentials struct {
				User     string `yaml:"user"`
				Password string `yaml:"password"`
			} `yaml:"credentials" mapstructure:"credentials"`
			URL                string `yaml:"url" mapstructure:"url"`
			ConnectionsPerCore int    `yaml:"connectionsPerCore" mapstructure:"connectionsPerCore"`
		} `yaml:"wintr/connectors/storage/v3" mapstructure:"wintr/connectors/storage/v3"` //nolint:tagliatelle // Nope.
	}
)
