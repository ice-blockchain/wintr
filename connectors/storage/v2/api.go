// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"net"
	"strings"
	stdlibtime "time"

	"github.com/cenkalti/backoff/v4"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/hashicorp/go-multierror"
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
	_, err := retry[any](ctx, func(_ error) (any, error) {
		if err := parseDBError(pgx.BeginTxFunc(ctx, db.primary(), txOptions, func(tx pgx.Tx) error { return fn(tx) })); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return nil, backoff.Permanent(err)
		}
	})
	if err != nil && (errors.Is(err, ErrSerializationFailure) || errors.Is(err, ErrTxAborted)) {
		stdlibtime.Sleep(10 * stdlibtime.Millisecond) //nolint:mnd,gomnd // Not a magic number.

		return DoInTransaction(ctx, db, fn)
	}
	if db.fallbackMasters != nil && len(db.fallbackMasters.replicas) > 0 && needRetryOnFallbackMaster(err) {
		idx := 0
		var txErr *multierror.Error
		for idx < len(db.fallbackMasters.replicas) && (needRetryOnFallbackMaster(err) || IsUnexpected(err)) {
			err = parseDBError(pgx.BeginTxFunc(ctx, db.fallbackPrimary(), txOptions, func(tx pgx.Tx) error { return fn(tx) }))
			txErr = multierror.Append(txErr, err)
			idx++
		}

		return errors.Wrap(txErr.ErrorOrNil(), "failed to execute tx on fallbacks")
	}

	return err
}

func Get[T any](ctx context.Context, db Querier, sql string, args ...any) (*T, error) {
	return retry[*T](ctx, func(_ error) (*T, error) {
		if resp, err := get[T](ctx, db, sql, args...); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err)
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
	return retry[[]*T](ctx, func(_ error) ([]*T, error) {
		if resp, err := selectInternal[T](ctx, db, sql, args...); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err)
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
	return retry[uint64](ctx, func(prevErr error) (uint64, error) {
		primary := db
		if pool, ok := db.(*DB); ok {
			if pool.fallbackMasters != nil && len(pool.fallbackMasters.replicas) > 0 && needRetryOnFallbackMaster(prevErr) {
				primary = pool.fallbackPrimary()
			} else {
				primary = pool.primary()
			}
		}
		if resp, err := exec(ctx, primary, sql, args...); err != nil && IsUnexpected(err) {
			return 0, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err)
		}
	})
}

func exec(ctx context.Context, db Execer, sql string, args ...any) (uint64, error) { //nolint:revive // Nope.
	resp, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, parseDBError(err)
	}

	return uint64(resp.RowsAffected()), nil //nolint:gosec // .
}

//nolint:varnamelen // .
func ExecOne[T any](ctx context.Context, db Querier, sql string, args ...any) (*T, error) {
	return retry[*T](ctx, func(prevErr error) (*T, error) {
		primary := db
		if pool, ok := db.(*DB); ok {
			if pool.fallbackMasters != nil && len(pool.fallbackMasters.replicas) > 0 && needRetryOnFallbackMaster(prevErr) {
				primary = pool.fallbackPrimary()
			} else {
				primary = pool.primary()
			}
		}
		if resp, err := execOne[T](ctx, primary, sql, args...); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err)
		}
	})
}

func execOne[T any](ctx context.Context, db Querier, sql string, args ...any) (*T, error) { //nolint:revive // Nope.
	resp := new(T)
	if err := pgxscan.Get(ctx, db, resp, sql, args...); err != nil {
		return nil, parseDBError(err)
	}

	return resp, nil
}

//nolint:varnamelen // .
func ExecMany[T any](ctx context.Context, db Querier, sql string, args ...any) ([]*T, error) {
	return retry[[]*T](ctx, func(prevErr error) ([]*T, error) {
		primary := db
		if pool, ok := db.(*DB); ok {
			if pool.fallbackMasters != nil && len(pool.fallbackMasters.replicas) > 0 && needRetryOnFallbackMaster(prevErr) {
				primary = pool.fallbackPrimary()
			} else {
				primary = pool.primary()
			}
		}
		if resp, err := execMany[T](ctx, primary, sql, args...); err != nil && IsUnexpected(err) {
			return nil, err
		} else { //nolint:revive // Nope.
			return resp, backoff.Permanent(err)
		}
	})
}

func execMany[T any](ctx context.Context, db Querier, sql string, args ...any) ([]*T, error) { //nolint:revive // Nope.
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
		if len(column) == 1 && column[0] != "" {
			if val, found := tErr.Data["column"]; found {
				return val == column[0]
			}
		}
	}

	return true
}

func IsUnexpected(err error) bool {
	var pgConnErr *pgconn.PgError
	var netOpErr *net.OpError

	return errors.As(err, &pgConnErr) || errors.As(err, &netOpErr) || errors.Is(err, ErrReadOnly)
}

func needRetryOnFallbackMaster(err error) bool {
	return err != nil && errors.Is(err, ErrReadOnly)
}

func parseDBError(err error) error { //nolint:funlen,gocognit,revive // .
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
			return ErrSerializationFailure
		}
		if dbErr.SQLState() == "25P02" {
			return ErrTxAborted
		}
		if dbErr.SQLState() == "23P01" {
			return ErrExclusionViolation
		}
		if dbErr.SQLState() == "25006" {
			return ErrReadOnly
		}

		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	return err
}
