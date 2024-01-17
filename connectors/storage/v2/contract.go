// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Public API.

var (
	ErrNotFound         = newStorageError("not found")
	ErrRelationNotFound = newStorageError("relation not found")
	ErrRelationInUse    = newStorageError("relation in use")
	ErrDuplicate        = newStorageError("duplicate")
	ErrCheckFailed      = newStorageError("check failed")
	ErrTxAborted        = newStorageError("transaction aborted")
)

type (
	DB struct {
		master *pgxpool.Pool
		lb     *lb
	}
)

// Private API.

type (
	storageErr struct {
		Msg string
	}
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
