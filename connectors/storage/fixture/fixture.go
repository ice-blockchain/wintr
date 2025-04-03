// SPDX-License-Identifier: ice License 1.0

package storagefixture

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/connectors/fixture"
	"github.com/ice-blockchain/wintr/connectors/storage"
	"github.com/ice-blockchain/wintr/log"
)

func NewTestConnector(applicationYAMLKey string, order int) TestConnector {
	var c cfg
	applicationYAMLTestKey := fmt.Sprintf("%v_test", applicationYAMLKey)
	config.MustLoadFromKey(applicationYAMLTestKey, &c)

	tc := &testConnector{
		cfg:                &c,
		applicationYAMLKey: applicationYAMLTestKey,
		order:              order,
	}
	tc.delegate = fixture.NewConnector("db", dockerComposeYAMLTemplate, "ready to accept requests", order, tc.findDBPort, createLuaScriptFile)

	return tc
}

func (tc *testConnector) Order() int {
	return tc.order
}

func (tc *testConnector) Setup(ctx context.Context) fixture.ContextErrClose {
	cleanUp := tc.delegate.Setup(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(cleanUp(ctx), "failed to cleanup storage connector due to premature panic"))
			log.Panic(e)
		}
	}()
	if tc.cfg.DB.SchemaPath != "" {
		tc.Connector = storage.MustConnect(ctx, func() {}, tc.schema(), tc.applicationYAMLKey)
	}

	return func(cctx context.Context) error {
		var errs []error
		if tc.Connector != nil {
			errs = append(errs, errors.Wrapf(tc.Connector.Close(), "failed closing the test storage client for %v", tc.applicationYAMLKey)) //nolint:staticcheck // .
		}
		errs = append(errs, errors.Wrapf(cleanUp(cctx), "failed to cleanup storage connector for %v", tc.applicationYAMLKey))

		return errors.Wrapf(multierror.Append(nil, errs...).ErrorOrNil(), "failed to cleanup storage test connector")
	}
}

func (tc *testConnector) schema() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Panic(errors.Wrap(err, "could not get working dir"))
	}
	for {
		if _, err = os.Stat(fmt.Sprintf("%v/go.mod", wd)); err != nil && errors.Is(err, os.ErrNotExist) {
			wd = path.Join(wd, "..")

			continue
		}

		break
	}
	fullSchemaPath := fmt.Sprintf("%v%c%v", wd, os.PathSeparator, tc.cfg.DB.SchemaPath)
	r, err := os.ReadFile(fullSchemaPath)
	log.Panic(errors.Wrapf(err, "failed to read %v", fullSchemaPath))

	return string(r)
}

func (tc *testConnector) findDBPort() (int, bool, error) {
	if len(tc.cfg.DB.URLs) == 0 {
		return 0, false, errors.Errorf("invalid/missing application.yaml for `%v`", tc.applicationYAMLKey)
	}
	port, err := strconv.Atoi(strings.Split(tc.cfg.DB.URLs[0], ":")[1])
	if err != nil {
		return 0, false, errors.Wrapf(err, "could not find a valid db port for `%v`", tc.applicationYAMLKey)
	}

	return port, false, nil
}

func createLuaScriptFile(dbPort int, tmpFolder string) error {
	startupScriptName := fmt.Sprintf("%s/%s", tmpFolder, scriptName)
	dbStartupScript := fmt.Sprintf(dbStartupScriptTemplate, dbPort)

	return errors.Wrap(os.WriteFile(startupScriptName, []byte(dbStartupScript), fileMode), "failed to create tmp db script")
}
