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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/terror"
)

const (
	maxConsecutiveListenerErrors  = 10
	notificationChannelBufferSize = 1000
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
			fb, _ := db.fallbackPrimary()
			err = parseDBError(pgx.BeginTxFunc(ctx, fb, txOptions, func(tx pgx.Tx) error { return fn(tx) }))
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
				var idx uint64
				primary, idx = pool.fallbackPrimary()
				log.Error(errors.Wrapf(prevErr, "[wintr/storage/v2]call failed. retrying in on fallback master %v", idx))
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
				var idx uint64
				primary, idx = pool.fallbackPrimary()
				log.Error(errors.Wrapf(prevErr, "[wintr/storage/v2]call failed. retrying in on fallback master %v", idx))
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
				var idx uint64
				primary, idx = pool.fallbackPrimary()
				log.Error(errors.Wrapf(prevErr, "[wintr/storage/v2]call failed. retrying in on fallback master %v", idx))
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

	return errors.As(err, &pgConnErr) || errors.As(err, &netOpErr)
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

func (db *DB) Listen(ctx context.Context, channel string) (*Listener, error) {
	conn, err := db.primary().Acquire(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to acquire connection for LISTEN")
	}

	listenerCtx, cancel := context.WithCancel(ctx)
	wg, wgCtx := errgroup.WithContext(listenerCtx)

	listener := &Listener{
		db:         db,
		conn:       conn,
		channel:    channel,
		done:       make(chan struct{}),
		notifCh:    make(chan *Notification, notificationChannelBufferSize),
		wg:         wg,
		cancelFunc: cancel,
	}

	err = executeListenCommand(ctx, conn, channel)
	if err != nil {
		cancel()
		conn.Release()

		return nil, errors.Wrapf(err, "failed to execute LISTEN command for channel %s", channel)
	}
	listener.wg.Go(func() error {
		return listener.receiveNotifications(wgCtx)
	})

	return listener, nil
}

func (l *Listener) receiveNotifications(ctx context.Context) error {
	defer close(l.notifCh)
	defer func() {
		l.connMx.Lock()
		if l.conn != nil {
			l.conn.Release()
			l.conn = nil
		}
		l.connMx.Unlock()
	}()

	bo := &backoff.ExponentialBackOff{
		InitialInterval:     100 * stdlibtime.Millisecond,
		RandomizationFactor: 0.5,
		Multiplier:          2.5,
		MaxInterval:         stdlibtime.Second,
		MaxElapsedTime:      0,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	bo.Reset()
	consecutiveErrors := 0

retryLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-l.done:
			return nil
		default:
			l.connMx.RLock()
			conn := l.conn
			l.connMx.RUnlock()

			if conn == nil {
				connected, shouldStop := l.connect(ctx, &consecutiveErrors, bo)
				if shouldStop {
					return ctx.Err()
				}
				if !connected {
					consecutiveErrors++
					if consecutiveErrors >= maxConsecutiveListenerErrors {
						var err error
						if l.lastErr != nil {
							err = errors.Wrapf(l.lastErr, "max consecutive errors (%d) reached during connection attempts for channel %s, closing listener", maxConsecutiveListenerErrors, l.channel)
						} else {
							err = errors.Errorf("max consecutive errors (%d) reached during connection attempts for channel %s, closing listener", maxConsecutiveListenerErrors, l.channel)
						}
						log.Error(err)

						return err
					}
				}
				continue retryLoop
			}

			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				l.setLastError(err)
				consecutiveErrors++
				if consecutiveErrors >= maxConsecutiveListenerErrors {
					wrappedErr := errors.Wrapf(err, "max consecutive errors (%d) reached for channel %s, closing listener", maxConsecutiveListenerErrors, l.channel)
					log.Error(wrappedErr)

					return wrappedErr
				}
				if l.isConnectionError(err) {
					log.Error(errors.Wrapf(err, "connection error on channel %s (attempt %d/%d), reconnecting", l.channel, consecutiveErrors, maxConsecutiveListenerErrors))
					l.releaseCurrentConnection()
					connected, shouldStop := l.connect(ctx, &consecutiveErrors, bo)
					if shouldStop {
						return ctx.Err()
					}
					if !connected {
						log.Error(errors.Errorf("failed to reconnect on channel %s, will retry", l.channel))
					}

					continue retryLoop
				}
				nextBackoff := bo.NextBackOff()
				log.Error(errors.Wrapf(err, "error waiting for notification on channel %s (attempt %d/%d), retrying in %v", l.channel, consecutiveErrors, maxConsecutiveListenerErrors, nextBackoff))
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-l.done:
					return nil
				case <-stdlibtime.After(nextBackoff):
				}

				continue retryLoop
			}
			if consecutiveErrors > 0 {
				consecutiveErrors = 0
				bo.Reset()
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-l.done:
				return nil
			case l.notifCh <- &Notification{
				Channel: notification.Channel,
				Payload: notification.Payload,
				PID:     notification.PID,
			}:
			default:
				log.Warn("notification channel full, dropping notification", "channel", l.channel, "payload", notification.Payload)
			}
		}
	}
}

func (l *Listener) connect(ctx context.Context, consecutiveErrors *int, bo *backoff.ExponentialBackOff) (connected bool, shouldStop bool) {
	nextBackoff := bo.NextBackOff()
	log.Error(errors.Errorf("connecting to channel %s (attempt %d/%d), waiting %v", l.channel, *consecutiveErrors, maxConsecutiveListenerErrors, nextBackoff))

	select {
	case <-ctx.Done():
		return false, true
	case <-l.done:
		return false, true
	case <-stdlibtime.After(nextBackoff):
	}
	conn, err := l.db.primary().Acquire(ctx)
	if err != nil {
		l.setLastError(err)
		log.Error(errors.Wrapf(err, "failed to acquire connection for channel %s", l.channel))

		return false, false
	}

	err = executeListenCommand(ctx, conn, l.channel)
	if err != nil {
		conn.Release()
		l.setLastError(err)
		log.Error(errors.Wrapf(err, "failed to execute LISTEN command on channel %s", l.channel))

		return false, false
	}

	l.connMx.Lock()
	l.conn = conn
	l.connMx.Unlock()

	*consecutiveErrors = 0
	bo.Reset()

	log.Info("successfully connected to channel " + l.channel)

	return true, false
}

func (l *Listener) releaseCurrentConnection() {
	l.connMx.Lock()
	defer l.connMx.Unlock()
	if l.conn != nil {
		l.conn.Release()
		l.conn = nil
	}
}

func executeListenCommand(ctx context.Context, conn *pgxpool.Conn, channel string) error {
	_, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{channel}.Sanitize())

	return err
}

func (l *Listener) isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "08003", // connection_does_not_exist
			"08006", // connection_failure
			"57P01", // admin_shutdown (pg_terminate_backend)
			"57P02", // crash_shutdown
			"57P03": // cannot_connect_now
			return true
		}
	}

	return false
}

func (l *Listener) Channel() <-chan *Notification {
	return l.notifCh
}

func (l *Listener) BackendPID() uint32 {
	l.connMx.RLock()
	defer l.connMx.RUnlock()
	if l.conn == nil {
		return 0
	}

	return l.conn.Conn().PgConn().PID()
}

func (l *Listener) Close() error {
	var err error
	l.closeOnce.Do(func() {
		l.cancelFunc()
		close(l.done)
		err = l.wg.Wait()
	})

	return err
}

func (l *Listener) Err() error {
	l.errMx.RLock()
	defer l.errMx.RUnlock()

	return l.lastErr
}

func (l *Listener) setLastError(err error) {
	l.errMx.Lock()
	defer l.errMx.Unlock()

	l.lastErr = err
}
