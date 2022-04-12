// SPDX-License-Identifier: BUSL-1.1

package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/framey-io/go-tarantool"
	tntMulti "github.com/framey-io/go-tarantool/multi"
	"github.com/pkg/errors"

	appCfg "github.com/ICE-Blockchain/wintr/config"
	"github.com/ICE-Blockchain/wintr/log"
)

func MustConnect(ctx context.Context, cancel context.CancelFunc, ddl, applicationYamlKey string) (db tarantool.Connector) {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)
	var err error

	schemaInitCtx, schemaInitCancel := context.WithTimeout(ctx, dbSchemaInitDeadline)
	if db, err = connectDB(schemaInitCtx, schemaInitCancel); err != nil {
		log.Panic(err)
	}
	if err = initDBSchema(db, ddl); err != nil {
		if cErr := db.Close(); cErr != nil {
			log.Error(errors.Wrap(err, "failed to closed db connector due to initDBSchema failure"))
		}
		log.Panic(err)
	}
	// The reason we close it and then reconnect it is because, sadly, schema loading happens after connection is established.
	// If you change the schema at runtime, the connector will not refresh the changes, so we are forced to reconnect to fetch the updated schema.
	if err = db.Close(); err != nil {
		log.Panic(err)
	}
	if db, err = connectDB(ctx, cancel); err != nil {
		log.Panic(err)
	}

	return
}

func connectDB(ctx context.Context, cancel context.CancelFunc) (db tarantool.Connector, err error) {
	auth := tntMulti.BasicAuth{
		User: cfg.DB.User,
		Pass: cfg.DB.Password,
	}

	log.Info("connecting to DB...", "URLs", cfg.DB.URLs)
	if db, err = tntMulti.ConnectWithDefaults(ctx, cancel, auth, cfg.DB.URLs...); err != nil {
		return nil, errors.Wrapf(err, "could not connect to tarantool instances: %v", cfg.DB.URLs)
	}

	return
}

func initDBSchema(db tarantool.Connector, ddl string) error {
	log.Info("initializing DB schema...")
	if resp, err := db.Eval(ddl, []interface{}{}); err != nil || resp.Code != tarantool.OkCode {
		return errors.Wrap(err, "DDL eval failed")
	}

	log.Info("checking DB schema...")
	if err := checkDBSchema(db); err != nil {
		return errors.Wrap(err, "DB schema check failed")
	}

	return nil
}

func checkDBSchema(db tarantool.Connector) error {
	var spacesR []map[string]interface{}
	if err := db.Call17Typed(getAllUserSpacesFunctionName, []interface{}{}, &spacesR); err != nil || len(spacesR) != 1 {
		return errors.Wrapf(err, "calling %s failed", getAllUserSpacesFunctionName)
	}

	spaces := spacesR[0]
	for spaceName, value := range spaces {
		log.Info(fmt.Sprintf("found space `%v`, metadata `%v`", spaceName, value))
	}

	if len(cfg.DB.Spaces) == 0 {
		return ErrNoSpacesConfigured
	}
	missingSpaces := make([]string, 0, len(cfg.DB.Spaces))
	for _, space := range cfg.DB.Spaces {
		if spaces[space] == nil {
			missingSpaces = append(missingSpaces, space)
		}
	}
	if len(missingSpaces) != 0 {
		return errors.Wrapf(ErrDDLInvalid, "spaces/tables %v are missing", strings.Join(missingSpaces, ","))
	}

	return nil
}

func CheckSQLDMLErr(resp *tarantool.Response, err error) error {
	if err != nil {
		if tErr := parseTarantoolSQLDMLErr(err); tErr != nil {
			return tErr
		}

		return errors.Wrap(err, "SQL DML failed")
	}

	if len(resp.Data) == 0 {
		return errors.New("unexpected SQL DML response: empty data")
	}

	count, ok := resp.Data[0].(int64)
	if !ok {
		unsignedCount, okUnsigned := resp.Data[0].(uint64)
		if !okUnsigned {
			return errors.Errorf("unexpected SQL DML response: %[1]v %[1]T", resp.Data[0])
		}
		count = int64(unsignedCount)
	}
	if count == 0 {
		return ErrNotFound
	}

	return nil
}

func parseTarantoolSQLDMLErr(err error) error {
	e := new(tarantool.Error)
	if ok := errors.As(err, e); ok {
		switch e.Code {
		case tarantool.ER_TUPLE_FOUND:
			return ErrDuplicate
		case tarantool.ER_TUPLE_NOT_FOUND:
			return ErrNotFound
		case tarantool.ER_SQL_EXECUTE:
			// Here we guess as no other reliable info is available.
			if strings.Contains(e.Msg, "FOREIGN KEY constraint failed") {
				return ErrRelationNotFound
			}
			// Hack/hotfix to go around a tarantool issue.
			if strings.Contains(e.Msg, "NOT NULL constraint failed") {
				return errors.Wrap(ErrRetryOnInvalidForeignKey, "DML operation failed")
			}
		}
	}

	return nil
}
