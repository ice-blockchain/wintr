// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"

	"github.com/pkg/errors"
	"github.com/zeebo/xxh3"
)

func NewLock(ctx context.Context, db *DB, lockID string) (Lock, error) {
	conn, err := db.primary().Acquire(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to acquire connection to DB")
	}
	lockIDHash := int64(xxh3.HashString(lockID))
	l := &lock{conn: conn, id: lockIDHash}

	return l, nil
}

func (l *lock) Obtain(ctx context.Context) (bool, error) {
	isLockAquired := false
	err := l.conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1);", l.id).Scan(&isLockAquired)
	if err != nil {
		return false, errors.Wrapf(err, "failed to pg_try_advisory_lock for lock %v", l.id)
	}

	return isLockAquired, nil
}

func (l *lock) Unlock(ctx context.Context) error {
	_, err := l.conn.Exec(ctx, "SELECT pg_advisory_unlock($1);", l.id)
	if err != nil {
		return errors.Wrapf(err, "failed to pg_advisory_unlock for lock %v", l.id)
	}
	l.conn.Release()

	return nil
}
