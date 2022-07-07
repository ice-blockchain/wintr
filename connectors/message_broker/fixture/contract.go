// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	_ "embed"
	"sync"

	connectorsfixture "github.com/ice-blockchain/wintr/connectors/fixture"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

// Public API.

type (
	RawMessage struct {
		Key, Value, Topic string
	}
	TestConnector interface {
		connectorsfixture.TestConnector

		messagebroker.Client

		VerifyMessages(context.Context, ...RawMessage) error
	}
)

// Private API.

//go:embed .testdata/docker-compose.yaml
var dockerComposeYAMLTemplate string

type (
	testConnector struct {
		delegate connectorsfixture.TestConnector
		messagebroker.Client
		*testMessageStore
		cfg                *messagebroker.Config
		applicationYAMLKey string
		order              int
	}
	testMessageStore struct {
		mx                       *sync.RWMutex
		chronologicalMessageList []RawMessage
	}
)
