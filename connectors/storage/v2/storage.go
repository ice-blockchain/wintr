// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func MustConnect(ctx context.Context, ddl, applicationYAMLKey string) *DB {
	var cfg config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	var replicas []*pgxpool.Pool
	var master *pgxpool.Pool
	if cfg.WintrStorage.PrimaryURL != "" {
		master = mustConnectPool(ctx, cfg.WintrStorage.PrimaryURL)
	}
	for ix, url := range cfg.WintrStorage.ReplicaURLs {
		if ix == 0 {
			replicas = make([]*pgxpool.Pool, len(cfg.WintrStorage.ReplicaURLs)) //nolint:makezero // Not needed, we know the size.
		}
		replicas[ix] = mustConnectPool(ctx, url)
	}
	if master != nil && ddl != "" && cfg.WintrStorage.RunDDL {
		for _, statement := range strings.Split(ddl, "----") {
			_, err := master.Exec(ctx, statement)
			log.Panic(errors.Wrapf(err, "failed to run statement: %v", statement))
		}
	}

	return &DB{master: master, lb: &lb{replicas: replicas}}
}

func mustConnectPool(ctx context.Context, url string) (db *pgxpool.Pool) {
	poolConfig, err := pgxpool.ParseConfig(url)
	log.Panic(errors.Wrapf(err, "failed to parse pool config: %v", url)) //nolint:revive // Intended
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		var res int
		if qErr := conn.QueryRow(ctx, `SELECT 1`).Scan(&res); qErr != nil {
			return errors.Wrapf(qErr, "dummy select failed")
		}
		if res != 1 {
			return errors.New("db validation failed")
		}

		return nil
	}
	db, err = pgxpool.NewWithConfig(ctx, poolConfig)
	log.Panic(errors.Wrapf(err, "failed to start pool for config: %v", url))

	return db
}

func (db *DB) Primary() *pgxpool.Pool {
	return db.master
}

func (db *DB) Replica() *pgxpool.Pool {
	return db.lb.replicas[atomic.AddUint64(&db.lb.currentIndex, 1)%uint64(len(db.lb.replicas))]
}
