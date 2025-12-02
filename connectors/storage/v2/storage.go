// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"errors"
	"fmt"
	neturl "net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

//nolint:gochecknoinits // GlobalDB is single instance, we initialize it here.
func init() {
	var cfg config
	appcfg.MustLoadFromKey(globalDBYamlKey, &cfg)
	if cfg.WintrStorage.PrimaryURL != "" || len(cfg.WintrStorage.ReplicaURLs) > 0 {
		globalDB = MustConnectWithCfg(context.Background(), &cfg.WintrStorage, nil)
	}
}

func MustConnect(ctx context.Context, applicationYAMLKey string, ddl DDL) *DB {
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if globalDB != nil && !cfg.WintrStorage.IgnoreGlobal {
		if globalDB.master != nil {
			mustRunDDL(ctx, globalDB.master, ddl)
		}

		return globalDB
	}

	return MustConnectWithCfg(ctx, &cfg.WintrStorage, ddl)
}

func MustConnectWithCfg(ctx context.Context, cfg *Cfg, ddl DDL) *DB {
	var replicas, fallbacks []*pgxpool.Pool
	var master *pgxpool.Pool

	if cfg.PrimaryURL != "" {
		master = mustConnectPool(ctx, cfg.Timeout, cfg.Credentials.User, cfg.Credentials.Password, cfg.PrimaryURL, cfg.SkipSettingsVerification)
	}
	if len(cfg.ReplicaURLs) > 0 {
		replicas = make([]*pgxpool.Pool, 0, len(cfg.ReplicaURLs))
		for _, url := range cfg.ReplicaURLs {
			conn := mustConnectPool(ctx, cfg.Timeout, cfg.Credentials.User, cfg.Credentials.Password, url, cfg.SkipSettingsVerification)
			replicas = append(replicas, conn)
		}
	}
	if len(cfg.PrimaryFallbackURLs) > 0 {
		fallbacks = make([]*pgxpool.Pool, 0, len(cfg.PrimaryFallbackURLs))
		for _, url := range cfg.PrimaryFallbackURLs {
			conn := mustConnectPool(ctx, cfg.Timeout, cfg.Credentials.User, cfg.Credentials.Password, url, cfg.SkipSettingsVerification)
			fallbacks = append(fallbacks, conn)
		}
	}
	if master != nil && ddl != nil && cfg.RunDDL {
		mustRunDDL(ctx, master, ddl)
	}
	log.Info(fmt.Sprintf("db connected: has master = %v, replicas = %v, fallbacks = %v", master != nil, len(replicas), len(fallbacks)))
	return &DB{
		master:          master,
		fallbackMasters: &lb{replicas: fallbacks},
		lb:              &lb{replicas: replicas},
		acquiredLocks:   make(map[int64]*pgxpool.Conn),
	}
}

func mustRunDDL(ctx context.Context, master *pgxpool.Pool, ddl DDL) {
	err := ddl.run(ctx, master)
	if !ignorableDDLError(err) {
		log.Panic(fmt.Errorf("failed to execute DDL: %w", maskError(err)))
	}
}

func ignorableDDLError(err error) bool {
	if err == nil {
		return true
	}
	var dbErr *pgconn.PgError
	if errors.As(err, &dbErr) {
		if dbErr.SQLState() == "25006" {
			return true
		}
	}

	return false
}

//nolint:mnd,gomnd,revive // Configuration.
func mustConnectPool(ctx context.Context, timeout, user, pass, url string, skipSettingsVerification bool) (db *pgxpool.Pool) {
	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Panic(fmt.Errorf("failed to parse pool config %v: %w", maskSensitive(url), maskError(err)))
	}

	if user != "" {
		poolConfig.ConnConfig.User = user
	}
	if pass != "" {
		poolConfig.ConnConfig.Password = pass
	}
	poolConfig.ConnConfig.StatementCacheCapacity = 1024
	poolConfig.ConnConfig.DescriptionCacheCapacity = 1024
	poolConfig.ConnConfig.Config.ConnectTimeout = 30 * time.Second //nolint:staticcheck // .
	if !strings.Contains(strings.ToLower(url), "pool_max_conn_idle_time") {
		poolConfig.MaxConnIdleTime = time.Minute
	}
	log.Info(fmt.Sprintf("poolConfig.MaxConnIdleTime=%v", poolConfig.MaxConnIdleTime))
	if !strings.Contains(strings.ToLower(url), "pool_health_check_period") {
		poolConfig.HealthCheckPeriod = 30 * time.Second
	}
	log.Info(fmt.Sprintf("poolConfig.HealthCheckPeriod=%v", poolConfig.HealthCheckPeriod))
	poolConfig.MaxConnLifetimeJitter = 10 * time.Minute
	poolConfig.MaxConnLifetime = 24 * time.Hour
	if !skipSettingsVerification {
		poolConfig.AfterConnect = func(cctx context.Context, conn *pgx.Conn) error { return doAfterConnect(cctx, timeout, conn) }
	}
	poolConfig.MinConns = 1

	db, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Panic(fmt.Errorf("failed to start pool for config: %v: %w", maskSensitive(url), maskError(err)))
	}
	return db
}

func doAfterConnect(ctx context.Context, timeout string, conn *pgx.Conn) error { //nolint:funlen // .
	actualTimeout := "30s"
	if timeout != "" {
		actualTimeout = timeout
	}
	log.Info(fmt.Sprintf("wintr/connectors/storage/v2:timeout:%v", timeout))
	customConnectionParameters := map[string]string{
		"statement_timeout":                   actualTimeout,
		"idle_in_transaction_session_timeout": actualTimeout,
		"lock_timeout":                        actualTimeout,
		// "tcp_user_timeout":                 actualTimeout,.
		"enable_partitionwise_join":      "on",
		"enable_partitionwise_aggregate": "on",
	}
	values := make([]string, 0, len(customConnectionParameters))
	for name, setting := range customConnectionParameters {
		values = append(values, fmt.Sprintf("'%v'", name))
		if _, qErr := conn.Exec(ctx, fmt.Sprintf(`SET %v = '%v'`, name, setting)); qErr != nil {
			return qErr //nolint:wrapcheck // Not needed.
		}
	}
	sql := fmt.Sprintf(`SELECT name, setting
						FROM pg_settings
						WHERE name IN (%v)`, strings.Join(values, ","))
	rows, qErr := conn.Query(ctx, sql)
	if qErr != nil {
		return fmt.Errorf("db validation query failed: %w", qErr)
	}
	var res []*struct{ Name, Setting string }
	if qErr = pgxscan.ScanAll(&res, rows); qErr != nil {
		return errors.New("scanning validation select rows failed")
	}
	actual := make(map[string]string, len(res))
	for _, row := range res {
		actual[row.Name] = strings.ReplaceAll(row.Setting, "0000", "0s")
	}
	if !reflect.DeepEqual(actual, customConnectionParameters) {
		return fmt.Errorf("db validation failed, expected:%#v, actual:%#v", customConnectionParameters, actual)
	}

	return nil
}

func (db *DB) registerLock(conn *pgxpool.Conn, lock *advisoryLockMutex) {
	db.locksMx.Lock()
	defer db.locksMx.Unlock()
	db.acquiredLocks[lock.id] = conn
}

func (db *DB) Close() error {
	db.locksMx.Lock()
	for lockID, conn := range db.acquiredLocks {
		conn.Release()
		delete(db.acquiredLocks, lockID)
	}
	db.locksMx.Unlock()
	if db.master != nil {
		db.master.Close()
	}
	if len(db.lb.replicas) != 0 {
		for _, replica := range db.lb.replicas {
			replica.Close()
		}
	}
	db.closedMx.Lock()
	defer db.closedMx.Unlock()
	db.closed = true

	return nil
}

func PingWithoutWriteCheck() PingOption {
	return func(opts *pingOptions) {
		opts.NoWriteCheck = true
	}
}

func (db *DB) Ping(ctx context.Context, opts ...PingOption) error {
	var wg sync.WaitGroup
	var options pingOptions

	for _, opt := range opts {
		opt(&options)
	}

	const masterChecks = 2
	errChan := make(chan error, len(db.lb.replicas)+masterChecks)
	if db.master != nil {
		wg.Go(func() {
			err := db.master.Ping(ctx)
			if err != nil {
				err = fmt.Errorf("ping failed for master: %w", err)
			}
			errChan <- err
		})
		wg.Go(func() {
			var err error
			if !options.NoWriteCheck {
				err = CheckWrite(ctx, db.master)
			}
			if err != nil {
				err = fmt.Errorf("write check failed for master: %w", err)
			}
			errChan <- err
		})
	}
	if len(db.lb.replicas) != 0 {
		for ix := range db.lb.replicas {
			wg.Go(func() {
				err := db.lb.replicas[ix].Ping(ctx)
				if err != nil {
					err = fmt.Errorf("ping failed for replica[%v]: %w", ix, err)
				}
				errChan <- err
			})
		}
	}
	wg.Wait()
	close(errChan)
	errs := make([]error, 0, len(db.lb.replicas)+masterChecks)
	for err := range errChan {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func CheckWrite(ctx context.Context, db Querier) error {
	res, err := ExecOne[struct {
		ReadOnly string `db:"transaction_read_only"`
	}](ctx, db, "show transaction_read_only;")
	if err != nil {
		return fmt.Errorf("failed to check write access: %w", err)
	}
	if res.ReadOnly == "on" {
		return ErrReadOnly
	}

	return nil
}

func (db *DB) primary() *pgxpool.Pool {
	return db.master
}

func (db *DB) fallbackPrimary() (*pgxpool.Pool, uint64) {
	idx := atomic.AddUint64(&db.fallbackMasters.currentIndex, 1) % uint64(len(db.fallbackMasters.replicas))

	return db.fallbackMasters.replicas[idx], idx
}

func (db *DB) replica() *pgxpool.Pool {
	return db.lb.replicas[atomic.AddUint64(&db.lb.currentIndex, 1)%uint64(len(db.lb.replicas))]
}

func (*DB) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	panic("should not be used because its implemented just for type matching")
}

func (*DB) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	panic("should not be used because its implemented just for type matching")
}

func retry[T any](ctx context.Context, op func(prevError error) (T, error)) (tt T, err error) {
	err = backoff.RetryNotify(
		func() error {
			tt, err = op(err)

			return err
		},
		//nolint:mnd,gomnd // Because those are static configs.
		backoff.WithContext(&backoff.ExponentialBackOff{
			InitialInterval:     100 * time.Millisecond,
			RandomizationFactor: 0.5,
			Multiplier:          2.5,
			MaxInterval:         time.Second,
			MaxElapsedTime:      25 * time.Second,
			Stop:                backoff.Stop,
			Clock:               backoff.SystemClock,
		}, ctx),
		func(e error, next time.Duration) {
			log.Error(fmt.Errorf("[wintr/storage/v2] call failed: %v. retrying in %v", maskError(e), next))
		})

	return tt, err
}

func maskError(err error) error {
	if err == nil {
		return nil
	}

	return errors.New(maskSensitive(err.Error()))
}

func maskSensitive(input string) string {
	if input == "" {
		return input
	}
	out := input
	if u, perr := neturl.Parse(out); perr == nil && u.Scheme != "" {
		u.User = nil
		if u.Path != "" {
			u.Path = "/***"
		}
		out = u.String()
	}
	out = regexp.MustCompile(`(?i)(user=)[^\s\)]+`).ReplaceAllString(out, `${1}***`)
	out = regexp.MustCompile(`(?i)(database=)[^\s\)]+`).ReplaceAllString(out, `${1}***`)
	out = regexp.MustCompile(`(?i)(db=)[^\s\)]+`).ReplaceAllString(out, `${1}***`)
	out = regexp.MustCompile(`(?i)(dbname=)[^\s\)]+`).ReplaceAllString(out, `${1}***`)
	out = regexp.MustCompile(`(?i)(password=)[^\s\)]+`).ReplaceAllString(out, `${1}***`)
	out = regexp.MustCompile(`(?i)(for user\s+")([^"]+)(")`).ReplaceAllString(out, `${1}***${3}`)

	return out
}
