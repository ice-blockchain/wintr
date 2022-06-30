// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	"fmt"
	"sync"
	"testing"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	messagebrokerfixture "github.com/ice-blockchain/wintr/connectors/message_broker/fixture"
	storagefixture "github.com/ice-blockchain/wintr/connectors/storage/fixture"
	"github.com/ice-blockchain/wintr/log"
)

func TestSetup(pkg string) func() {
	cleanUpStorage, cleanUpMessageBroker := setupDBAndMessageBroker(pkg)

	return func() {
		dbError, mbError := cleanUp(cleanUpStorage, cleanUpMessageBroker)
		if dbError != nil || mbError != nil {
			log.Panic(fixtureCleanUpError(pkg), "dbError", dbError, "mbError", mbError)
		}
	}
}

func fixtureCleanUpError(pkg string) error {
	return fmt.Errorf("%s %w", pkg, errFixtureCleanUp)
}

func setupDBAndMessageBroker(pkg string) (func(), func()) {
	wg := new(sync.WaitGroup)
	var cleanUpStorage func()
	var cleanUpMessageBroker func()
	wg.Add(1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		cleanUpStorage = storagefixture.TestSetup(pkg)
	}()
	go func() {
		defer wg.Done()
		cleanUpMessageBroker = messagebrokerfixture.TestSetup(pkg)
	}()
	wg.Wait()

	return cleanUpStorage, cleanUpMessageBroker
}

func cleanUp(cleanUpStorage, cleanUpMessageBroker func()) (error, error) {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	wg.Add(1)
	var dbError error
	var mbError error
	go func() {
		defer wg.Done()
		if err := recover(); err != nil {
			dbError = err.(error)
		}
		cleanUpStorage()
	}()
	go func() {
		defer wg.Done()
		if err := recover(); err != nil {
			mbError = err.(error)
		}
		cleanUpMessageBroker()
	}()
	wg.Wait()

	return dbError, mbError
}

func NewTestsRunner(state State, appKey string) *TestsRunner {
	return &TestsRunner{State: state, appKey: appKey}
}

func (t *TestsRunner) RunTests(m *testing.M) (code int) {
	if testing.Short() {
		return m.Run()
	}
	cleanUP := TestSetup(t.appKey)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errRecover, e)
			if code == 0 {
				code = 1
			}
		}
	}()
	defer cleanUP()

	ctx, cancel := context.WithTimeout(context.Background(), testsContextTimeout)
	t.InitializeServices(ctx, cancel)
	t.testMb = connectAndStartConsumingFromTestMessageBroker(ctx, t.Processors(), t.appKey)
	defer func() {
		if cCode := t.cleanUp(ctx); code == 0 {
			code = cCode
		}
	}()
	defer cancel()

	return m.Run()
}

func (t *TestsRunner) cleanUp(ctx context.Context) int {
	if cErr := t.CloseServices(); cErr != nil {
		log.Error(cErr)

		return 1
	}
	if cErr := t.testMb.Close(); cErr != nil {
		log.Error(cErr)

		return 1
	}
	errCode := t.TestAllWhenDBAndMBAreDown(ctx)
	if errCode != 0 {
		log.Warn("main_test.testAllWhenDatabaseIsDown failed.")
	}

	return errCode
}

func connectAndStartConsumingFromTestMessageBroker(ctx context.Context, p map[string]messagebroker.Processor, applicationYamlKey string) messagebroker.Client {
	return messagebroker.MustConnectAndStartConsuming(ctx, func() {}, applicationYamlKey, p)
}
