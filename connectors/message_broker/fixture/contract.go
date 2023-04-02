// SPDX-License-Identifier: ice License 1.0

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

		MessageVerifier
		ReconnectConsumer(ctx context.Context)
	}
	MessageVerifier interface {
		VerifyMessages(context.Context, ...RawMessage) error
		VerifyNoMessages(context.Context, ...RawMessage) error
	}

	CallCounterProcessor struct {
		RetErrs                  *sync.Map // Map[string]error.
		LastReceivedMessage      RawMessage
		ProcessCallCounts        uint64
		SuccessProcessCallCounts uint64
	}
	ErrorBuilder func(*messagebroker.Message) error
)

// Private API.

// .
var (
	//go:embed .testdata/docker-compose.yaml
	dockerComposeYAMLTemplate string
)

type (
	testConnector struct {
		delegate connectorsfixture.TestConnector
		messagebroker.Client
		*testMessageStore
		cfg                *messagebroker.Config
		applicationYAMLKey string
		customProcessors   []messagebroker.Processor
		order              int
	}
	testMessageStore struct {
		mx                       *sync.RWMutex
		chronologicalMessageList []RawMessage
	}
	proxyProcessor struct {
		messageStore *testMessageStore
		processors   []messagebroker.Processor
	}
)
