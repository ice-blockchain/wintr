// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	"errors"
	"time"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

// Public API.

type (
	// | This interface should be implemented at the service's integration tests setup.
	State interface {
		// Custom services (global variables) initialization.
		InitializeServices(context.Context, context.CancelFunc)
		// Custom services (global variables) closing.
		CloseServices() error
		// Custom processors initialization to handle topic's messages.
		Processors() map[string]messagebroker.Processor
		// Custom db and mb case check.
		TestAllWhenDBAndMBAreDown(context.Context) int
	}

	TestsRunner struct {
		State
		testMb messagebroker.Client
		appKey string
	}
)

// Private API.

var (
	errFixtureCleanUp = errors.New("fixture cleanup failed")
	errRecover        = errors.New("fixture recover is not empty")
)

const testsContextTimeout = 10 * time.Minute
