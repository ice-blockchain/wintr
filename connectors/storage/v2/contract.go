// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"sync"

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
	ErrExclusionViolation   = errors.New("exclusion violation")
	ErrMutexNotLocked       = errors.New("not locked")
)

type (
	DB struct {
		master        *pgxpool.Pool
		lb            *lb
		acquiredLocks map[int64]*pgxpool.Conn
		locksMx       sync.Mutex
		closed        bool
		closedMx      sync.Mutex
	}
	Mutex interface {
		Lock(ctx context.Context) error
		Unlock(ctx context.Context) error
		EnsureLocked(ctx context.Context) error
	}
)

// Private API.
var (
	//nolint:gochecknoglobals // .
	globalDB *DB
)

const (
	globalDBYamlKey = "global"
)

type (
	lb struct {
		replicas     []*pgxpool.Pool
		currentIndex uint64
	}
	config struct {
		WintrStorage storageCfg `yaml:"wintr/connectors/storage/v2" mapstructure:"wintr/connectors/storage/v2"` //nolint:tagliatelle // Nope.
	}
	storageCfg struct {
		Credentials struct {
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"credentials" mapstructure:"credentials"`
		Timeout     string   `yaml:"timeout" mapstructure:"timeout"`
		PrimaryURL  string   `yaml:"primaryURL" mapstructure:"primaryURL"`   //nolint:tagliatelle // Nope.
		ReplicaURLs []string `yaml:"replicaURLs" mapstructure:"replicaURLs"` //nolint:tagliatelle // Nope.
		RunDDL      bool     `yaml:"runDDL" mapstructure:"runDDL"`           //nolint:tagliatelle // Nope.
	}
	advisoryLockMutex struct {
		conn *pgxpool.Conn
		db   *DB
		id   int64
	}
)
