// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

// Public API.

var (
	ErrNotFound             = errors.New("not found")
	ErrRelationNotFound     = errors.New("relation not found")
	ErrRelationInUse        = errors.New("relation in use")
	ErrDuplicate            = errors.New("duplicate")
	ErrCheckFailed          = errors.New("check failed")
	ErrSerializationFailure = errors.New("serialization failure")
	ErrTxAborted            = errors.New("transaction aborted")
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
			Credentials struct {
				User     string `yaml:"user"`
				Password string `yaml:"password"`
			} `yaml:"credentials" mapstructure:"credentials"`
			Timeout     string   `yaml:"timeout" mapstructure:"timeout"`         //nolint:tagliatelle // Nope.
			PrimaryURL  string   `yaml:"primaryURL" mapstructure:"primaryURL"`   //nolint:tagliatelle // Nope.
			ReplicaURLs []string `yaml:"replicaURLs" mapstructure:"replicaURLs"` //nolint:tagliatelle // Nope.
			RunDDL      bool     `yaml:"runDDL" mapstructure:"runDDL"`           //nolint:tagliatelle // Nope.
		} `yaml:"wintr/connectors/storage/v2" mapstructure:"wintr/connectors/storage/v2"` //nolint:tagliatelle // Nope.
	}
)
