// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"io/fs"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
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
	ErrReadOnly             = errors.New("read only")
)

type (
	DDL interface {
		run(context.Context, *pgxpool.Pool) error
	}
	DB struct {
		master          *pgxpool.Pool
		fallbackMasters *lb
		lb              *lb
		acquiredLocks   map[int64]*pgxpool.Conn
		locksMx         sync.Mutex
		closed          bool
		closedMx        sync.Mutex
	}
	Mutex interface {
		Lock(ctx context.Context) error
		Unlock(ctx context.Context) error
		EnsureLocked(ctx context.Context) error
	}
	PingOption func(*pingOptions)

	Listener struct {
		db         *DB
		conn       *pgxpool.Conn
		channel    string
		done       chan struct{}
		notifCh    chan *Notification
		wg         *errgroup.Group
		lastErr    error
		connMx     sync.RWMutex
		errMx      sync.RWMutex
		cancelFunc context.CancelFunc
		closeOnce  sync.Once
	}
	Notification struct {
		Channel string
		Payload string
		PID     uint32
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
		WintrStorage Cfg `yaml:"wintr/connectors/storage/v2" mapstructure:"wintr/connectors/storage/v2"` //nolint:tagliatelle // Nope.
	}
	Cfg struct {
		Credentials struct {
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"credentials" mapstructure:"credentials"`
		Timeout                  string   `yaml:"timeout" mapstructure:"timeout"`
		PrimaryURL               string   `yaml:"primaryURL" mapstructure:"primaryURL"`                   //nolint:tagliatelle // Nope.
		PrimaryFallbackURLs      []string `yaml:"primaryFallbackURLs" mapstructure:"primaryFallbackURLs"` //nolint:tagliatelle // Nope.
		ReplicaURLs              []string `yaml:"replicaURLs" mapstructure:"replicaURLs"`                 //nolint:tagliatelle // Nope.
		RunDDL                   bool     `yaml:"runDDL" mapstructure:"runDDL"`                           //nolint:tagliatelle // Nope.
		SkipSettingsVerification bool     `yaml:"skipSettingsVerification" mapstructure:"skipSettingsVerification"`
		IgnoreGlobal             bool     `yaml:"ignoreGlobal" mapstructure:"ignoreGlobal"`
	}
	advisoryLockMutex struct {
		conn *pgxpool.Conn
		db   *DB
		id   int64
	}
	stringDDL struct {
		Data string
	}
	filesystemDDL struct {
		FS          fs.FS
		SchemeTable string
	}
	pingOptions struct {
		NoWriteCheck bool
	}
)
