// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

// Public API.

var (
	ErrNotFound         = errors.New("not found")
	ErrRelationNotFound = errors.New("relation not found")
	ErrDuplicate        = errors.New("duplicate")
)

type (
	DB struct {
		master *pgxpool.Pool
		lb     *lb
	}
)

// Private API.

type (
	lb struct {
		replicas     []*pgxpool.Pool
		currentIndex uint64
	}
	config struct {
		WintrStorage struct {
			PrimaryURL  string   `yaml:"primaryURL" mapstructure:"primaryURL"`   //nolint:tagliatelle // Nope.
			ReplicaURLs []string `yaml:"replicaURLs" mapstructure:"replicaURLs"` //nolint:tagliatelle // Nope.
			RunDDL      bool     `yaml:"runDDL" mapstructure:"runDDL"`           //nolint:tagliatelle // Nope.
		} `yaml:"wintr/connectors/storage/v2" mapstructure:"wintr/connectors/storage/v2"` //nolint:tagliatelle // Nope.
	}
)
