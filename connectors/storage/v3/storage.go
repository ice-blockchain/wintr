// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	stdlibtime "time"

	"github.com/goccy/go-reflect"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

//nolint:gomnd // Configs.
func MustConnect(ctx context.Context, applicationYAMLKey string) DB { //nolint:funlen // .
	var cfg config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if cfg.WintrStorage.ConnectionsPerCore == 0 {
		cfg.WintrStorage.ConnectionsPerCore = 10
	}

	opts, err := redis.ParseURL(cfg.WintrStorage.URL)
	log.Panic(err) //nolint:revive // That's intended.
	if opts.Username == "" {
		opts.Username = cfg.WintrStorage.Credentials.User
	}
	if opts.Password == "" {
		opts.Password = cfg.WintrStorage.Credentials.Password
	}
	opts.ClientName = applicationYAMLKey

	opts.MaxRetries = 25
	opts.MinRetryBackoff = 10 * stdlibtime.Millisecond
	opts.MaxRetryBackoff = 1 * stdlibtime.Second
	opts.DialTimeout = 30 * stdlibtime.Second
	opts.ReadTimeout = 30 * stdlibtime.Second
	opts.WriteTimeout = 30 * stdlibtime.Second
	opts.ConnMaxIdleTime = 60 * stdlibtime.Second
	opts.ContextTimeoutEnabled = true
	opts.PoolFIFO = true
	opts.PoolSize = cfg.WintrStorage.ConnectionsPerCore * runtime.GOMAXPROCS(-1)
	opts.MinIdleConns = 1
	opts.MaxIdleConns = 1
	client := redis.NewClient(opts)
	result, err := client.Ping(ctx).Result()
	log.Panic(err)
	if result != "PONG" {
		log.Panic(errors.Errorf("unexpected ping response: %v", result))
	}

	return client
}

func Set(ctx context.Context, db DB, values ...interface{ Key() string }) error {
	if len(values) == 0 {
		_, err := db.HSet(ctx, values[0].Key(), values[0]).Result()

		return err //nolint:wrapcheck // Not needed.
	}
	cmds, err := db.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for _, value := range values {
			if err := pipeliner.HSet(ctx, value.Key(), value).Err(); err != nil {
				return err //nolint:wrapcheck // Not needed.
			}
		}

		return nil
	})
	if err != nil {
		return err //nolint:wrapcheck // Not needed.
	}
	errs := make([]error, 0, len(cmds))
	for _, cmd := range cmds {
		errs = append(errs, cmd.Err())
	}

	return multierror.Append(nil, errs...).ErrorOrNil() //nolint:wrapcheck // Not needed.
}

func AtomicSet(ctx context.Context, db DB, values ...interface{ Key() string }) error {
	if len(values) == 0 {
		_, err := db.HSet(ctx, values[0].Key(), values[0]).Result()

		return err //nolint:wrapcheck // Not needed.
	}
	cmds, err := db.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for _, value := range values {
			if err := pipeliner.HSet(ctx, value.Key(), value).Err(); err != nil {
				return err //nolint:wrapcheck // Not needed.
			}
		}

		return nil
	})
	if err != nil {
		return err //nolint:wrapcheck // Not needed.
	}
	errs := make([]error, 0, len(cmds))
	for _, cmd := range cmds {
		errs = append(errs, cmd.Err())
	}

	return multierror.Append(nil, errs...).ErrorOrNil() //nolint:wrapcheck // Not needed.
}

func Get[T any](ctx context.Context, db DB, keys ...string) ([]*T, error) { //nolint:funlen,gocognit,gocyclo,revive,cyclop,varnamelen // .
	if len(keys) == 1 { //nolint:nestif // Not that bad.
		sliceResult := db.HMGet(ctx, keys[0], ProcessRedisFieldTags[T]()...)
		var resp any = new(T)
		if err := sliceResult.Scan(resp); err != nil {
			return nil, err //nolint:wrapcheck // Not needed.
		}
		anyNonNil := false
		for _, val := range sliceResult.Val() {
			if val != nil {
				anyNonNil = true

				break
			}
		}
		if anyNonNil {
			if intf, ok := resp.(interface{ SetKey(string) }); ok {
				intf.SetKey(sliceResult.Args()[1].(string)) //nolint:forcetypeassert // We know for sure.
			}

			return append(make([]*T, 0, 1), resp.(*T)), nil //nolint:forcetypeassert // We know for sure.
		}

		return nil, nil
	}
	redisFieldTags := ProcessRedisFieldTags[T]()
	if cmdResults, err := db.Pipelined(ctx, func(pipeliner redis.Pipeliner) error { //nolint:nestif // .
		for _, key := range keys {
			if err := pipeliner.HMGet(ctx, key, redisFieldTags...).Err(); err != nil {
				return err //nolint:wrapcheck // Not needed.
			}
		}

		return nil
	}); err != nil {
		return nil, err //nolint:wrapcheck // Not needed.
	} else { //nolint:revive // Nope.
		results := make([]*T, 0, len(cmdResults))
		for _, cmdResult := range cmdResults {
			sliceResult := cmdResult.(*redis.SliceCmd) //nolint:errcheck,forcetypeassert // Scan checks it.
			var resp any = new(T)
			if sErr := sliceResult.Scan(resp); sErr != nil {
				return nil, sErr //nolint:wrapcheck // We don't need to, no relevant extra info here.
			}
			anyNonNil := false
			for _, val := range sliceResult.Val() {
				if val != nil {
					anyNonNil = true

					break
				}
			}
			if anyNonNil {
				if intf, ok := resp.(interface{ SetKey(string) }); ok {
					intf.SetKey(sliceResult.Args()[1].(string)) //nolint:forcetypeassert // We know for sure.
				}
				results = append(results, resp.(*T)) //nolint:forcetypeassert // We know for sure.
			}
		}

		return results, nil
	}
}

func Bind[TT any](ctx context.Context, db DB, keys, fields []string, results *[]*TT) error { //nolint:funlen,gocognit,revive // .
	if cmdResults, err := db.Pipelined(ctx, func(pipeliner redis.Pipeliner) error { //nolint:nestif // .
		for _, key := range keys {
			if err := pipeliner.HMGet(ctx, key, fields...).Err(); err != nil {
				return err //nolint:wrapcheck // Not needed.
			}
		}

		return nil
	}); err != nil {
		return err //nolint:wrapcheck // Not needed.
	} else { //nolint:revive // Nope.
		res := *results
		for _, cmdResult := range cmdResults {
			sliceResult := cmdResult.(*redis.SliceCmd) //nolint:errcheck,forcetypeassert // Scan checks it.
			var resp any = new(TT)
			if sErr := sliceResult.Scan(resp); sErr != nil {
				return sErr //nolint:wrapcheck // We don't need to, no relevant extra info here.
			}
			anyNonNil := false
			for _, val := range sliceResult.Val() {
				if val != nil {
					anyNonNil = true

					break
				}
			}
			if anyNonNil {
				if intf, ok := resp.(interface{ SetKey(string) }); ok {
					intf.SetKey(sliceResult.Args()[1].(string)) //nolint:forcetypeassert // We know for sure.
				}
				res = append(res, resp.(*TT)) //nolint:forcetypeassert // We know for sure.
			}
		}
		*results = res

		return nil
	}
}

// .
var (
	//nolint:gochecknoglobals // Singleton.
	typeCache = new(sync.Map)
)

func ProcessRedisFieldTags[TT any]() []string {
	fieldNames, found := typeCache.Load(*new(TT))
	if !found {
		val := new(TT)
		fieldNames, _ = typeCache.LoadOrStore(*val, collectFields(reflect.TypeOf(val).Elem()))
	}
	fields := fieldNames.([]string) //nolint:forcetypeassert,errcheck // We know for sure.
	if len(fields) == 0 {
		log.Panic(fmt.Sprintf("%#v has no redis tags", new(TT)))
	}

	return fields
}

func collectFields(elem reflect.Type) (fields []string) {
	if elem.Kind() != reflect.Struct {
		return nil
	}
	for i := 0; i < elem.NumField(); i++ {
		if field := elem.Field(i); field.Anonymous {
			embeddedElem := field.Type
			if embeddedElem.Kind() == reflect.Ptr {
				embeddedElem = embeddedElem.Elem()
			}
			fields = append(fields, collectFields(embeddedElem)...)
		} else if redisTag := field.Tag.Get("redis"); redisTag != "" && redisTag != "-" {
			fields = append(fields, redisTag)
		}
	}

	return fields
}
