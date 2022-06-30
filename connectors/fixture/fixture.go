// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	messagebrokerfixture "github.com/ice-blockchain/wintr/connectors/message_broker/fixture"
	storagefixture "github.com/ice-blockchain/wintr/connectors/storage/fixture"
	"github.com/ice-blockchain/wintr/log"
)

func TestSetup(pkg string) func() {
	cleanUpStorage, cleanUpMessageBroker := setupDBAndMessageBroker(pkg)

	return func() {
		dbError, mbError := cleanUp(cleanUpStorage, cleanUpMessageBroker)
		if dbError != nil || mbError != nil {
			err := errors.New(fmt.Sprintf("%v fixture cleanup failed", pkg))
			log.Panic(err, "dbError", dbError, "mbError", mbError)
		}
	}
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
