// SPDX-License-Identifier: ice License 1.0

package riverqueue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

type (
	Register = river.Workers
	JobArgs  = river.JobArgs

	Job[T JobArgs]            = river.Job[T]
	Worker[T JobArgs]         = river.Worker[T]
	WorkerDefaults[T JobArgs] = river.WorkerDefaults[T]

	Config struct {
		Credentials struct {
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"credentials" mapstructure:"credentials"`
		PrimaryURLs     []string      `yaml:"primaryURLs" mapstructure:"primaryURLs"`
		QueueName       string        `yaml:"queueName" mapstructure:"queueName"`
		ID              string        `yaml:"id,omitempty" mapstructure:"id"`
		MaxQueueWorkers int           `yaml:"maxQueueWorkers" mapstructure:"maxQueueWorkers"`
		JobMaxTimeout   time.Duration `yaml:"maxJobTimeout" mapstructure:"maxJobTimeout"`
	}
	Option func(*riverq)

	Client interface {
		Register() *Register
		Push(ctx context.Context, jobs ...JobArgs) error
		Stop(ctx context.Context) error
		Start(ctx context.Context) error
		HealthCheck(ctx context.Context) error
		Close(ctx context.Context) error
	}

	riverClient = river.Client[pgx.Tx]

	riverq struct {
		WorkerRegister *river.Workers
		DB             *databaseClient
		Cfg            *Config
		River          atomic.Pointer[riverClient]
		SwitchMu       sync.Mutex
	}
)

const (
	defaultJobTimeout = 10 * time.Minute
	defaultWorkers    = 95
)

var (
	ErrNotConnected     = errors.New("not connected to job queue")
	ErrInvalidArguments = errors.New("invalid arguments")
)

func newClient(ctx context.Context, applicationYAMLKey string, opts ...Option) (*riverq, error) {
	var client riverq

	if err := client.initOptions(applicationYAMLKey, opts...); err != nil {
		return nil, fmt.Errorf("failed to configure client: %w", err)
	}

	client.WorkerRegister = river.NewWorkers()

	log.Debug(fmt.Sprintf("initializing job queue client with queue name: %s", client.Cfg.QueueName))

	db, err := newDatabaseClient(ctx, client.Cfg.Credentials.User, client.Cfg.Credentials.Password, client.Cfg.PrimaryURLs...)
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	if err := client.initDatabaseClient(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to initialize database client: %w", err)
	}

	if err := client.initRiverClient(ctx, db.Get()); err != nil {
		return nil, fmt.Errorf("failed to initialize river client: %w", err)
	}

	return &client, nil
}

func WithConfig(cfg *Config) Option {
	return func(q *riverq) {
		q.Cfg = cfg
	}
}

func WithQueueName(name string) Option {
	return func(q *riverq) {
		if q.Cfg == nil {
			q.Cfg = &Config{}
		}
		q.Cfg.QueueName = name
	}
}

func WithClientID(id string) Option {
	return func(q *riverq) {
		if q.Cfg == nil {
			q.Cfg = &Config{}
		}
		q.Cfg.ID = id
	}
}

func MustNewClient(ctx context.Context, applicationYAMLKey string, opts ...Option) Client {
	client, err := newClient(ctx, applicationYAMLKey, opts...)
	if err != nil {
		log.Panic("failed to create rq client: " + err.Error())
	}
	return client
}

func (q *riverq) Close(ctx context.Context) error {
	return errors.Join(q.Stop(ctx), q.DB.Close())
}

func (q *riverq) Register() *Register {
	return q.WorkerRegister
}

func (q *riverq) initOptions(yamlKey string, opts ...Option) error {
	for _, opt := range opts {
		opt(q)
	}

	if q.Cfg == nil {
		var yamlConfig Config
		appcfg.MustLoadFromKey(yamlKey, &yamlConfig)
		q.Cfg = &yamlConfig
	}

	if q.Cfg.QueueName == "" && q.Cfg.ID == "" {
		return fmt.Errorf("either queue name or client ID must be provided: %w", ErrInvalidArguments)
	}

	if q.Cfg.MaxQueueWorkers <= 0 {
		q.Cfg.MaxQueueWorkers = defaultWorkers
	}
	if q.Cfg.JobMaxTimeout <= 0 {
		q.Cfg.JobMaxTimeout = defaultJobTimeout
	}

	if q.Cfg.QueueName == "" {
		q.Cfg.QueueName = formatQueueName(q.Cfg.ID)
	}

	return nil
}

func (q *riverq) initRiverClient(ctx context.Context, pool *pgxpool.Pool) error {
	log.Info(fmt.Sprintf("initializing river client: queue name: %s, client ID: %s", q.Cfg.QueueName, q.Cfg.ID))

	rClient, err := river.NewClient(
		riverpgxv5.New(pool),
		&river.Config{
			Queues: map[string]river.QueueConfig{
				q.Cfg.QueueName: {
					MaxWorkers: q.Cfg.MaxQueueWorkers,
				},
			},
			Workers:    q.WorkerRegister,
			JobTimeout: q.Cfg.JobMaxTimeout,
			ID:         q.Cfg.ID,
		},
	)
	if err != nil {
		return fmt.Errorf("cannot create river client: %w", err)
	}

	old := q.River.Swap(rClient)
	if old != nil {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		log.Warn("force stopping old river client")
		if err := old.StopAndCancel(ctx); err != nil {
			log.Error(err, "force stopping old river client returned error")
		}
	}
	return err
}

func (q *riverq) initDatabaseClient(ctx context.Context, db *databaseClient) (err error) {
	migrator, err := rivermigrate.New(riverpgxv5.New(db.Get()), &rivermigrate.Config{})
	if err != nil {
		return fmt.Errorf("cannot create river migrator: %w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("failed to migrate river: %w", err)
	}

	q.DB = db

	return nil
}

func (q *riverq) Stop(ctx context.Context) error {
	o := q.River.Load()
	if o == nil {
		return nil
	}
	return o.Stop(context.WithoutCancel(ctx))
}

func (q *riverq) Start(ctx context.Context) error {
	o := q.River.Load()
	if o == nil {
		return ErrNotConnected
	}
	return o.Start(ctx)
}

func (q *riverq) trySwitchMaster(ctx context.Context, reason error) error {
	q.SwitchMu.Lock()
	defer q.SwitchMu.Unlock()

	err := q.DB.switchMaster(ctx, reason)
	if err != nil {
		return fmt.Errorf("failed to switch master: %w", err)
	}

	err = q.initDatabaseClient(ctx, q.DB)
	if err != nil {
		return fmt.Errorf("failed to reinitialize database client after master switch: %w", err)
	}

	err = q.initRiverClient(ctx, q.DB.Get())
	if err != nil {
		return fmt.Errorf("failed to reinitialize river client after master switch: %w", err)
	}

	if err := q.Start(ctx); err != nil {
		return fmt.Errorf("failed to restart river client after master switch: %w", err)
	}

	return nil
}

func (q *riverq) Push(ctx context.Context, jobs ...JobArgs) error {
	for attempt := 1; ctx.Err() == nil; attempt++ {
		err := q.push(ctx, jobs...)
		if !shouldSwitchMaster(err) {
			return err
		}

		select {
		case <-time.After(time.Millisecond * 500):
			log.Debug(fmt.Sprintf("attempting to switch master and retry push, attempt %d", attempt))

		case <-ctx.Done():
			return ctx.Err()
		}

		if q.HealthCheck(ctx) == nil {
			log.Debug(fmt.Sprintf("health check passed, connection was already restored, attempt %d", attempt))
			continue
		}

		err = q.trySwitchMaster(ctx, err)
		if err != nil {
			log.Error(err, "master switching error, attempt %d", attempt)
			continue
		}
	}
	return ctx.Err()
}

func (q *riverq) push(ctx context.Context, jobs ...JobArgs) error {
	if len(jobs) == 0 {
		return nil
	}

	o := q.River.Load()
	if o == nil {
		return ErrNotConnected
	}

	opts := &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true},
		Queue:      q.Cfg.QueueName,
	}

	if len(jobs) == 1 {
		res, err := o.Insert(ctx, jobs[0], opts)
		if err == nil {
			log.Debug(fmt.Sprintf("pushed job to queue: job_id=%d, kind=%s", res.Job.ID, res.Job.Kind))
		}
		return err
	}

	var params []river.InsertManyParams
	for i := range jobs {
		params = append(params, river.InsertManyParams{
			Args:       jobs[i],
			InsertOpts: opts,
		})
	}

	res, err := o.InsertMany(ctx, params)
	if err != nil {
		return err
	}

	for _, r := range res {
		log.Debug(fmt.Sprintf("pushed job to queue: job_id=%d, kind=%s", r.Job.ID, r.Job.Kind))
	}

	return nil
}

func (q *riverq) HealthCheck(ctx context.Context) error {
	o := q.River.Load()
	if o == nil {
		return ErrNotConnected
	}
	return q.DB.Ping(ctx)
}

func formatQueueName(name string) string {
	const prefix = "rq_"
	var sb strings.Builder
	var runes = map[rune]rune{
		':': '_',
		'/': '_',
		'.': '_',
		'_': '_', // To avoid consecutive underscores at the edges.
	}

	sb.Grow(len(name) + len(prefix))
	sb.WriteString(prefix)
	var lastReplaced bool
	for i, r := range name {
		replacement, ok := runes[r]
		if ok {
			if !lastReplaced && i != len(name)-1 && i != 0 {
				sb.WriteRune(replacement)
			}
			lastReplaced = true
		} else {
			sb.WriteRune(r)
			lastReplaced = false
		}
	}
	return sb.String()
}

func RegisterWorker[T JobArgs](register *Register, worker Worker[T]) {
	log.Info(fmt.Sprintf("registering worker: %T", worker))
	river.AddWorker(register, worker)
}
