// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"errors"
	"fmt"
	"log"
	"sync"

	messagebrokerfixture "github.com/ice-blockchain/wintr/connectors/message_broker/fixture"
	storagefixture "github.com/ice-blockchain/wintr/connectors/storage/fixture"
)

func TestSetup(pkg string) func() {
	wg := new(sync.WaitGroup)
	cleanUpStorage := storagefixture.SetupDB(pkg, wg)
	cleanUpMessageBroker := messagebrokerfixture.SetupMessageBroker(pkg, wg)
	wg.Wait()

	return func() {
		wg := new(sync.WaitGroup)
		dbError := storagefixture.CleanUpDB(cleanUpStorage, wg)
		mbError := messagebrokerfixture.CleanUp(cleanUpMessageBroker, wg)
		wg.Wait()

		if dbError != nil || mbError != nil {
			err := errors.New(fmt.Sprintf("%v fixture cleanup failed", pkg))
			log.Panic(err, "dbError", dbError, "mbError", mbError)
		}
	}
}
