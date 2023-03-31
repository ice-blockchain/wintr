// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"strings"

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

	return pgx.BeginTxFunc(ctx, db.primary(), txOptions, func(tx pgx.Tx) error { return fn(tx) }) //nolint:wrapcheck // We have nothing relevant to wrap.
}

func Get[T any](ctx context.Context, db Querier, sql string, args ...any) (*T, error) {
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
	if pool, ok := db.(*DB); ok {
		db = pool.replica() //nolint:revive // Not an issue here.
	}
	var resp []*T
	if err := pgxscan.Select(ctx, db, &resp, sql, args...); err != nil {
		return nil, parseDBError(err)
	}

	return resp, nil
}

func Exec(ctx context.Context, db Execer, sql string, args ...any) (affectedRows uint64, err error) {
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

func parseDBError(err error) error {
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

			return terror.New(ErrRelationNotFound, map[string]any{"column": column})
		}

		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	return err
}
