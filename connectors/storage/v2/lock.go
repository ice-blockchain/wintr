// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"

	"github.com/pkg/errors"
	"github.com/zeebo/xxh3"
)

func NewMutex(db *DB, lockID string) Mutex {
	lockIDHash := int64(xxh3.HashString(lockID))
	l := &advisoryLockMutex{conn: nil, db: db, id: lockIDHash}

	return l
}

func (l *advisoryLockMutex) Lock(ctx context.Context) error {
	isLockAquired := false
	conn, err := l.db.primary().Acquire(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to acquire connection to DB")
	}
	l.conn = conn
	if err = l.conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1);", l.id).Scan(&isLockAquired); err != nil {
		return errors.Wrapf(err, "failed to pg_try_advisory_lock for advisoryLockMutex %v", l.id)
	}
	if !isLockAquired {
		return ErrMutexNotLocked
	}
	l.db.registerLock(l.conn, l)

	return nil
}

func (l *advisoryLockMutex) Unlock(ctx context.Context) error {
	_, err := l.conn.Exec(ctx, "SELECT pg_advisory_unlock($1);", l.id)
	if err != nil {
		return errors.Wrapf(err, "failed to pg_advisory_unlock for advisoryLockMutex %v", l.id)
	}
	l.conn.Release()

	return nil
}

func (l *advisoryLockMutex) EnsureLocked(ctx context.Context) error {
	if l.conn == nil {
		return ErrMutexNotLocked
	}
	if l.conn.Conn().IsClosed() {
		return l.Lock(ctx)
	}
	if l.conn.Ping(ctx) != nil {
		return l.Lock(ctx)
	}

	return nil
}
