// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"encoding"
	"fmt"
	"runtime"
	"strings"
	"sync"
	stdlibtime "time"

	"github.com/goccy/go-reflect"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

//nolint:gomnd,gocognit // Configs.
func MustConnect(ctx context.Context, applicationYAMLKey string, overriddenPoolSize ...int) DB { //nolint:funlen // .
	var cfg config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if cfg.WintrStorage.ConnectionsPerCore == 0 {
		cfg.WintrStorage.ConnectionsPerCore = 10
	}
	if cfg.WintrStorage.URL != "" && len(cfg.WintrStorage.URLs) == 0 {
		cfg.WintrStorage.URLs = append(make([]string, 0, 1), cfg.WintrStorage.URL)
	}
	if len(cfg.WintrStorage.URLs) == 0 {
		log.Panic(errors.New("at least one url is required"))
	}
	clients := make([]*redis.Client, 0, len(cfg.WintrStorage.URLs))
	for _, url := range cfg.WintrStorage.URLs {
		opts, err := redis.ParseURL(url)
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
		if len(overriddenPoolSize) == 1 {
			opts.PoolSize = overriddenPoolSize[0]
		}
		opts.PoolSize /= len(cfg.WintrStorage.URLs)
		if opts.PoolSize == 0 {
			opts.PoolSize = 1
		}
		opts.MinIdleConns = 1
		opts.MaxIdleConns = 1
		client := redis.NewClient(opts)
		result, err := client.Ping(ctx).Result()
		log.Panic(err)
		if result != "PONG" {
			log.Panic(errors.Errorf("unexpected ping response: %v", result))
		}
		clients = append(clients, client)
	}

	return &lb{instances: clients, urls: cfg.WintrStorage.URLs}
}

func Set(ctx context.Context, db DB, values ...interface{ Key() string }) error {
	if len(values) == 1 {
		value := values[0]
		if value == nil {
			return nil
		}
		_, err := db.HSet(ctx, value.Key(), SerializeValue(value)...).Result()

		return err //nolint:wrapcheck // Not needed.
	}
	cmds, err := db.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for _, value := range values {
			if value == nil {
				continue
			}
			if err := pipeliner.HSet(ctx, value.Key(), SerializeValue(value)...).Err(); err != nil {
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
		sliceResult := db.HMGet(ctx, keys[0], processRedisFieldTags[T]()...)
		var resp any = new(T)
		if err := DeserializeValue(resp, sliceResult.Scan); err != nil {
			return nil, err
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
	redisFieldTags := processRedisFieldTags[T]()
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
			if sErr := DeserializeValue(resp, sliceResult.Scan); sErr != nil {
				return nil, sErr
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

func Bind[TT any](ctx context.Context, db DB, keys []string, results *[]*TT) error { //nolint:funlen,gocognit,revive // .
	fields := processRedisFieldTags[TT]()
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
			if sErr := DeserializeValue(resp, sliceResult.Scan); sErr != nil {
				return sErr
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

func processRedisFieldTags[TT any]() []string {
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
			redisTag, _, _ = strings.Cut(redisTag, ",")
			fields = append(fields, redisTag)
		}
	}

	return fields
}

func DeserializeValue(value any, scan func(any) error) error { //nolint:gocognit,revive // .
	if err := scan(value); err != nil {
		return err
	}
	typ, val := reflect.TypeOf(value).Elem(), reflect.ValueOf(value).Elem()
	for ix := 0; ix < typ.NumField(); ix++ {
		typeField := typ.Field(ix)
		if !typeField.Anonymous {
			continue
		}
		valueField := val.Field(ix)
		if valueField.Kind() == reflect.Ptr && valueField.IsNil() && valueField.CanSet() {
			valueField.Set(reflect.New(typeField.Type.Elem()))
		}
		if valueField.Kind() == reflect.Struct && valueField.CanAddr() {
			valueField = valueField.Addr()
		}
		if valueField.Kind() == reflect.Ptr && valueField.CanInterface() {
			if err := DeserializeValue(valueField.Interface(), scan); err != nil {
				return err
			}
		}
	}

	return nil
}

func SerializeValue(value any) []any {
	reflVal := reflect.ValueOf(value)
	if reflVal.Type().Kind() == reflect.Ptr {
		if reflVal.IsNil() {
			log.Panic(fmt.Sprintf("`%#v` is nil", value))
		}
		reflVal = reflVal.Elem()
	}
	if reflVal.Type().Kind() != reflect.Struct {
		log.Panic(fmt.Sprintf("`%#v` is not a struct or a pointer to a struct", value))
	}

	return serializeStructFields(reflVal)
}

func serializeStructFields(value reflect.Value) (resp []any) { //nolint:funlen,gocognit,revive,cyclop // .
	typ := value.Type()
	for ix := 0; ix < typ.NumField(); ix++ {
		typeField := typ.Field(ix)
		if typeField.Anonymous {
			field := value.Field(ix)
			if typeField.Type.Kind() == reflect.Ptr {
				if field.IsNil() {
					continue
				}
				field = field.Elem()
			}
			if field.Type().Kind() == reflect.Struct {
				resp = append(resp, serializeStructFields(field)...)
			}

			continue
		}
		tag := typeField.Tag.Get("redis")
		if tag == "" || tag == "-" {
			continue
		}
		name, opt, _ := strings.Cut(tag, ",")
		if name == "" {
			continue
		}

		field := value.Field(ix)

		if omitEmpty(opt) && isEmptyValue(field) {
			continue
		}

		if field.CanInterface() {
			switch typedVal := field.Interface().(type) {
			case encoding.BinaryMarshaler:
				data, err := typedVal.MarshalBinary()
				log.Panic(err)
				resp = append(resp, name, string(data))
			case stdlibtime.Duration:
				resp = append(resp, name, fmt.Sprint(typedVal.Nanoseconds()))
			case string:
				resp = append(resp, name, typedVal)
			default:
				resp = append(resp, name, fmt.Sprint(typedVal))
			}
		}
	}

	return resp
}

func omitEmpty(opt string) bool {
	for opt != "" {
		var name string
		name, opt, _ = strings.Cut(opt, ",") //nolint:revive // Not a problem here.
		if name == "omitempty" {
			return true
		}
	}

	return false
}

func isEmptyValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	case reflect.Struct:
		return value.IsZero()
	case reflect.Invalid, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return false
	default:
		return value.IsZero()
	}
}
