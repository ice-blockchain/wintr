// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/ice-blockchain/go-tarantool-client"
	tntmulti "github.com/ice-blockchain/go-tarantool-client/multi"
	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/terror"
)

func MustConnect(ctx context.Context, cancel context.CancelFunc, ddl, applicationYAMLKey string) (db tarantool.Connector) {
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	var err error
	schemaInitCtx, schemaInitCancel := ctx, cancel
	if !cfg.DB.ReadOnly && !cfg.DB.SkipSchemaCreation {
		schemaInitCtx, schemaInitCancel = context.WithTimeout(ctx, dbSchemaInitDeadline)
	}
	if db, err = connectDB(schemaInitCtx, schemaInitCancel); err != nil {
		log.Panic(err)
	}
	if err = initDBSchema(db, ddl); err != nil {
		if cErr := db.Close(); cErr != nil {
			log.Error(errors.Wrap(cErr, "failed to closed db connector due to initDBSchema failure"))
		}
		log.Panic(err)
	}
	if cfg.DB.ReadOnly || cfg.DB.SkipSchemaCreation {
		return db
	}
	// The reason we close it and then reconnect it is because, sadly, schema loading happens after connection is established.
	// If you change the schema at runtime, the connector will not refresh the changes, so we are forced to reconnect to fetch the updated schema.
	if err = db.Close(); err != nil {
		log.Panic(err)
	}
	if db, err = connectDB(ctx, cancel); err != nil {
		log.Panic(err)
	}

	return db
}

func connectDB(ctx context.Context, cancel context.CancelFunc) (db tarantool.Connector, err error) {
	auth := tntmulti.BasicAuth{
		User: cfg.DB.User,
		Pass: cfg.DB.Password,
	}

	log.Info("connecting to DB...", "URLs", cfg.DB.URLs, "readOnly", cfg.DB.ReadOnly)
	if db, err = tntmulti.ConnectWithWritableAwareDefaults(ctx, cancel, !cfg.DB.ReadOnly, auth, cfg.DB.URLs...); err != nil {
		return nil, errors.Wrapf(err, "could not connect to tarantool instances: %v", cfg.DB.URLs)
	}

	return db, errors.Wrap(err, "failed to connect db")
}

func initDBSchema(db tarantool.Connector, ddl string) error {
	if !cfg.DB.ReadOnly && !cfg.DB.SkipSchemaCreation {
		log.Info("initializing DB schema...")
		if resp, err := db.Eval(fmt.Sprintf("%v\nenable_sync_on_all_user_spaces()", ddl), []any{}); err != nil || resp.Code != tarantool.OkCode {
			return errors.Wrap(err, "DDL eval failed")
		}
		log.Info("checking DB schema...")
		if err := checkDBSchema(db, ddl); err != nil {
			return errors.Wrap(err, "DB schema check failed")
		}
	}

	return nil
}

func checkDBSchema(db tarantool.Connector, ddl string) error {
	spaces, err := getAllUserSpaces(db)
	if err != nil {
		return errors.Wrap(err, "failed to getAllUserSpaces")
	}
	for spaceName, value := range spaces {
		log.Info(fmt.Sprintf("found space `%v`, metadata `%v`", spaceName, value))
	}

	expectedSpaces := detectSpaces(ddl)
	missingSpaces := make([]string, 0, len(expectedSpaces))
	for _, space := range expectedSpaces {
		if spaces[space] == nil {
			missingSpaces = append(missingSpaces, space)
		}
	}
	if len(missingSpaces) != 0 {
		return errors.Wrapf(ErrDDLInvalid, "spaces/tables %v are missing", strings.Join(missingSpaces, ","))
	}

	return nil
}

func detectSpaces(ddl string) (expectedSpaces []string) {
	const marker = "______________________"
	markedDDL := strings.ReplaceAll(strings.ToUpper(ddl), `BOX.EXECUTE([[CREATE TABLE IF NOT EXISTS `, marker)
	markedDDL = strings.ReplaceAll(markedDDL, `BOX.EXECUTE([[CREATE TABLE `, marker)
	expectedSpace := ""
	for expectedSpace == "" {
		leftIndex := strings.Index(markedDDL, marker)
		if leftIndex < 0 {
			break
		}
		rightIndex := strings.Index(markedDDL[leftIndex:], " ")
		expectedSpace = markedDDL[leftIndex+len(marker) : leftIndex+rightIndex]
		if expectedSpace == "" {
			break
		}
		if !strings.HasSuffix(expectedSpace, "]]") {
			expectedSpaces = append(expectedSpaces, expectedSpace)
		} else {
			expectedSpaces = append(expectedSpaces, detectRangedSpaces(expectedSpace, markedDDL[:leftIndex+len(marker)])...)
		}
		expectedSpace = ""
		markedDDL = markedDDL[leftIndex+rightIndex:]
	}

	return expectedSpaces
}

func detectRangedSpaces(expectedSpace, markedDDL string) []string {
	const iterationMarker = "FOR "
	const rangeMarker = "=0,"
	innerMarkedDDL := markedDDL
	ix := strings.LastIndex(innerMarkedDDL, iterationMarker)
	if ix < 0 {
		return nil
	}
	innerMarkedDDL = innerMarkedDDL[ix+len(iterationMarker):]
	ix = strings.Index(innerMarkedDDL, rangeMarker)
	if ix < 0 {
		return nil
	}
	rightRangeValue := innerMarkedDDL[ix+len(rangeMarker) : ix+len(rangeMarker)+strings.Index(innerMarkedDDL[ix+len(rangeMarker):], " ")]
	rightRange, err := strconv.Atoi(rightRangeValue)
	if err != nil {
		return nil
	}
	expectedSpaces := make([]string, 0, rightRange+1)
	for i := 0; i <= rightRange; i++ {
		expectedSpaces = append(expectedSpaces, strings.Replace(expectedSpace, "]]", strconv.Itoa(i), 1))
	}

	return expectedSpaces
}

func getAllUserSpaces(db tarantool.Connector) (map[string]any, error) {
	var spacesR []map[string]any
	getAllSpacesFuncName := getAllUserSpacesFunctionName
	if cfg.DB.ReadOnly {
		getAllSpacesFuncName = "{{non-writable}}" + getAllUserSpacesFunctionName
	}
	if err := db.Call17Typed(getAllSpacesFuncName, []any{}, &spacesR); err != nil || len(spacesR) != 1 {
		return nil, errors.Wrapf(err, "calling %s failed", getAllSpacesFuncName)
	}

	return spacesR[0], nil
}

func CheckSQLDMLErr(resp *tarantool.Response, err error) error {
	affectedRows, rErr := CheckSQLDMLResponse(resp, err)
	if rErr == nil && affectedRows == 0 {
		return ErrNotFound
	}

	return rErr
}

func CheckSQLDMLResponse(resp *tarantool.Response, err error) (affectedRows int64, rErr error) {
	if err != nil {
		if tErr := parseTarantoolDMLErr(err); tErr != nil {
			return 0, tErr
		}

		return 0, errors.Wrap(err, "SQL DML failed")
	}

	if len(resp.Data) == 0 {
		return 0, errors.New("unexpected SQL DML response: empty data")
	}

	count, ok := resp.Data[0].(int64)
	if !ok {
		unsignedCount, okUnsigned := resp.Data[0].(uint64)
		if !okUnsigned {
			return 0, errors.Errorf("unexpected SQL DML response: %[1]v %[1]T", resp.Data[0])
		}
		count = int64(unsignedCount) //nolint:gosec // .
	}

	return count, nil
}

func CheckNoSQLDMLErr(err error) error {
	if tErr := parseTarantoolDMLErr(err); tErr != nil {
		return tErr
	}

	return errors.Wrap(err, "noSQL DML failed")
}

func parseTarantoolDMLErr(err error) error {
	dbErr := new(tarantool.Error)
	if ok := errors.As(err, dbErr); ok {
		switch dbErr.Code { //nolint:revive // .
		case tarantool.ER_TUPLE_FOUND: //nolint:nosnakecase // External library.
			return terror.New(ErrDuplicate, map[string]any{
				IndexName: strings.Split(strings.Replace(dbErr.Msg, `Duplicate key exists in unique index "`, "", 1), `"`)[0],
			})
		case tarantool.ER_TUPLE_NOT_FOUND: //nolint:nosnakecase // External library.
			return ErrNotFound
		case tarantool.ER_SQL_EXECUTE: //nolint:nosnakecase // External library.
			// Here we guess as no other reliable info is available.
			if strings.Contains(dbErr.Msg, "FOREIGN KEY constraint failed") {
				return ErrRelationNotFound
			}
			// Hack/hotfix to go around a tarantool issue.
			if strings.Contains(dbErr.Msg, "NOT NULL constraint failed") {
				return errors.Wrap(ErrRetryOnInvalidForeignKey, "DML operation failed")
			}
		}
	}

	return nil
}
