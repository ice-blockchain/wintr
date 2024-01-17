// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"strings"
	stdlibtime "time"

	"github.com/cenkalti/backoff/v4"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/terror"
)

type (
	Querier interface {
		pgxscan.Querier
	}
	Execer interface {
		Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	}
	QueryExecer interface {
		Querier
		Execer
	}
)

func DoInTransaction(ctx context.Context, db *DB, fn func(conn QueryExecer) error) error {
	txOptions := pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite, DeferrableMode: pgx.NotDeferrable}
	_, err := retry[any](ctx, func() (any, error) {
		//nolint:wrapcheck // We have nothing relevant to wrap.
		if err := pgx.BeginTxFunc(ctx, db.primary(), txOptions, func(tx pgx.Tx) error { return fn(tx) }); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return nil, backoff.Permanent(err) //nolint:wrapcheck // Not needed.
		}
	})
	if errors.Is(err, ErrSerializationFailure) {
		stdlibtime.Sleep(10 * stdlibtime.Millisecond)

		return DoInTransaction(ctx, db, fn)
	}

	return err
}

func Get[T any](ctx context.Context, db Querier, sql string, args ...any) (*T, error) {
	return retry[*T](ctx, func() (*T, error) {
		if resp, err := get[T](ctx, db, sql, args...); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err) //nolint:wrapcheck // Not needed.
		}
	})
}

func get[T any](ctx context.Context, db Querier, sql string, args ...any) (*T, error) { //nolint:revive // Nope.
	if pool, ok := db.(*DB); ok {
		db = pool.replica() //nolint:revive // Not an issue here.
	}
	resp := new(T)
	if err := pgxscan.Get(ctx, db, resp, sql, args...); err != nil {
		return nil, parseDBError(err)
	}

	return resp, nil
}

func Select[T any](ctx context.Context, db Querier, sql string, args ...any) ([]*T, error) {
	return retry[[]*T](ctx, func() ([]*T, error) {
		if resp, err := selectInternal[T](ctx, db, sql, args...); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err) //nolint:wrapcheck // Not needed.
		}
	})
}

func selectInternal[T any](ctx context.Context, db Querier, sql string, args ...any) ([]*T, error) {
	if pool, ok := db.(*DB); ok {
		db = pool.replica() //nolint:revive // Not an issue here.
	}
	var resp []*T
	if err := pgxscan.Select(ctx, db, &resp, sql, args...); err != nil {
		return nil, parseDBError(err)
	}

	return resp, nil
}

func Exec(ctx context.Context, db Execer, sql string, args ...any) (uint64, error) {
	return retry[uint64](ctx, func() (uint64, error) {
		if resp, err := exec(ctx, db, sql, args...); err != nil && IsUnexpected(err) {
			return 0, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err) //nolint:wrapcheck // Not needed.
		}
	})
}

func exec(ctx context.Context, db Execer, sql string, args ...any) (uint64, error) { //nolint:revive // Nope.
	if pool, ok := db.(*DB); ok {
		db = pool.primary() //nolint:revive // Not an issue here.
	}
	resp, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, parseDBError(err)
	}

	return uint64(resp.RowsAffected()), nil
}

func ExecOne[T any](ctx context.Context, db Querier, sql string, args ...any) (*T, error) {
	return retry[*T](ctx, func() (*T, error) {
		if resp, err := execOne[T](ctx, db, sql, args...); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err) //nolint:wrapcheck // Not needed.
		}
	})
}

func execOne[T any](ctx context.Context, db Querier, sql string, args ...any) (*T, error) { //nolint:revive // Nope.
	if pool, ok := db.(*DB); ok {
		db = pool.primary() //nolint:revive // Not an issue here.
	}
	resp := new(T)
	if err := pgxscan.Get(ctx, db, resp, sql, args...); err != nil {
		return nil, parseDBError(err)
	}

	return resp, nil
}

func ExecMany[T any](ctx context.Context, db Querier, sql string, args ...any) ([]*T, error) {
	return retry[[]*T](ctx, func() ([]*T, error) {
		if resp, err := execMany[T](ctx, db, sql, args...); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err) //nolint:wrapcheck // Not needed.
		}
	})
}

func execMany[T any](ctx context.Context, db Querier, sql string, args ...any) ([]*T, error) { //nolint:revive // Nope.
	if pool, ok := db.(*DB); ok {
		db = pool.primary() //nolint:revive // Not an issue here.
	}
	var resp []*T
	if err := pgxscan.Select(ctx, db, &resp, sql, args...); err != nil {
		return nil, parseDBError(err)
	}

	return resp, nil
}

func IsErr(err, target error, column ...string) bool {
	if !errors.Is(err, target) {
		return false
	}
	if tErr := terror.As(err); tErr != nil {
		if len(column) == 1 && column[0] != "" { //nolint:revive // Wrong.
			if val, found := tErr.Data["column"]; found {
				return val == column[0]
			}
		}
	}

	return true
}

func IsUnexpected(err error) bool {
	return !IsErr(err, ErrDuplicate) &&
		!IsErr(err, ErrRelationNotFound) &&
		!IsErr(err, ErrNotFound) &&
		!IsErr(err, ErrCheckFailed) &&
		!IsErr(err, ErrRelationInUse) &&
		!IsErr(err, ErrSerializationFailure) &&
		!IsErr(err, ErrTxAborted)
}

func parseDBError(err error) error { //nolint:funlen // .
	var dbErr *pgconn.PgError
	if errors.As(err, &dbErr) { //nolint:nestif // .
		if dbErr.SQLState() == "23505" {
			if strings.HasSuffix(dbErr.ConstraintName, "_pkey") {
				return terror.New(ErrDuplicate, map[string]any{"column": "pk"})
			} else { //nolint:revive // Uglier to write otherwise.
				column := strings.ReplaceAll(dbErr.ConstraintName, dbErr.TableName, "")
				column = strings.ReplaceAll(column, "_key", "")
				column = strings.ReplaceAll(column, "_", "")

				return terror.New(ErrDuplicate, map[string]any{"column": column})
			}
		}
		if dbErr.SQLState() == "23503" {
			column := strings.ReplaceAll(dbErr.ConstraintName, dbErr.TableName, "")
			column = strings.ReplaceAll(column, "_fkey", "")
			column = strings.ReplaceAll(column, "_", "")
			if strings.Contains(dbErr.Detail, "is still referenced from table") {
				return terror.New(ErrRelationInUse, map[string]any{"column": column})
			}

			return terror.New(ErrRelationNotFound, map[string]any{"column": column})
		}
		if dbErr.SQLState() == "23514" {
			column := strings.ReplaceAll(dbErr.ConstraintName, dbErr.TableName, "")
			column = strings.ReplaceAll(column, "_check", "")
			column = strings.ReplaceAll(column, "_", "")

			return terror.New(ErrCheckFailed, map[string]any{"column": column})
		}
		if dbErr.SQLState() == "40001" {
			return terror.New(ErrSerializationFailure, map[string]any{})
		}
		if dbErr.SQLState() == "25P02" {
			return terror.New(ErrTxAborted, map[string]any{})
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	return err
}
