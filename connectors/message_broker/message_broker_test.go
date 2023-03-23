// SPDX-License-Identifier: ice License 1.0

package messagebroker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/connectors/fixture"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	mbfixture "github.com/ice-blockchain/wintr/connectors/message_broker/fixture"
)

const (
	testApplicationYAMLKey                             = "self"
	testApplicationYAMLKeyWithOneGoroutinePerPartition = "self_one_goroutine"
	testTopic                                          = "testing-topic"
	testMessagesCount                                  = 150
	testDeadline                                       = 60 * time.Second
)

var (
	mbConnectorConcurrent   mbfixture.TestConnector         //nolint:gochecknoglobals // Testing env.
	mbConnectorOneGoroutine mbfixture.TestConnector         //nolint:gochecknoglobals // Testing env.
	counter                 *mbfixture.CallCounterProcessor //nolint:gochecknoglobals // Testing env.
)

func TestMain(m *testing.M) {
	counter = &mbfixture.CallCounterProcessor{}
	mbConnectorConcurrent = mbfixture.NewTestConnector(testApplicationYAMLKey, 0, counter)
	mbConnectorOneGoroutine = mbfixture.NewTestConnector(testApplicationYAMLKeyWithOneGoroutinePerPartition, 0, counter)
	fixture.NewTestRunner(testApplicationYAMLKey, nil, mbConnectorConcurrent, mbConnectorOneGoroutine).RunTests(m)
}

func produceTestData(ctx context.Context, tb testing.TB, mb messagebroker.Client, errBuilder mbfixture.ErrorBuilder) []mbfixture.RawMessage {
	tb.Helper()
	sentMessages := make([]mbfixture.RawMessage, testMessagesCount) //nolint:makezero // We're have specific amount of messages
	counter.Reset()
	counter.RetErrs = new(sync.Map)
	for ind := 0; ind < testMessagesCount; ind++ {
		msg := &messagebroker.Message{
			Key:   uuid.NewString(),
			Topic: testTopic,
			Value: []byte(uuid.NewString()),
		}
		err := errBuilder(msg)
		if err != nil {
			counter.RetErrs.Store(msg.Key, err)
		}
		responder := make(chan error)
		mb.SendMessage(ctx, msg, responder)
		require.NoError(tb, <-responder)
		sentMessages[ind] = mbfixture.RawMessage{
			Key:   msg.Key,
			Value: string(msg.Value),
			Topic: msg.Topic,
		}
	}

	return sentMessages
}

//nolint:paralleltest // Cuz of integration tests.
func TestConcurrentConsumerProcessorOk(t *testing.T) {
	t.Run("TestConcurrentConsumer_ProcessorOk", func(t *testing.T) {
		ConcurrentConsumerProcessorOkWithMB(t, mbConnectorConcurrent)
	})
	t.Run("TestConcurrentConsumer_ProcessorOk_OneRoutine", func(t *testing.T) {
		ConcurrentConsumerProcessorOkWithMB(t, mbConnectorOneGoroutine)
	})
}

func ConcurrentConsumerProcessorOkWithMB(t *testing.T, mbConnector mbfixture.TestConnector) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testDeadline)
	defer cancel()
	sentMsgs := produceTestData(ctx, t, mbConnector, func(message *messagebroker.Message) error { return nil })
	require.NoError(t, mbConnector.VerifyMessages(ctx, sentMsgs...))
	assert.Equal(t, uint64(len(sentMsgs)), counter.ProcessCallCounts)
	assert.Equal(t, counter.ProcessCallCounts, counter.SuccessProcessCallCounts)
}

//nolint:paralleltest // Cuz of integration tests.
func TestConcurrentConsumerProcessorNonCriticalError(t *testing.T) {
	t.Run("TestConcurrentConsumer_ProcessorNonCriticalError", func(t *testing.T) {
		ConcurrentConsumerProcessorNonCriticalErrorWithMB(t, mbConnectorConcurrent)
	})
	counter.Reset()
	t.Run("TestConcurrentConsumer_ProcessorNonCriticalError_OneRoutine", func(t *testing.T) {
		ConcurrentConsumerProcessorNonCriticalErrorWithMB(t, mbConnectorOneGoroutine)
	})
}

func ConcurrentConsumerProcessorNonCriticalErrorWithMB(t *testing.T, mbConnector mbfixture.TestConnector) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testDeadline)
	defer cancel()
	sentMsgs := produceTestData(ctx, t, mbConnector, func(msg *messagebroker.Message) error {
		return errors.Errorf("non critical error %v", msg.Key)
	})
	require.NoError(t, mbConnector.VerifyMessages(ctx, sentMsgs...))
	assert.Equal(t, uint64(len(sentMsgs)), counter.ProcessCallCounts)
	assert.NotEqual(t, counter.ProcessCallCounts, counter.SuccessProcessCallCounts)
}

//nolint:paralleltest // Cuz of integration tests.
func TestConcurrentConsumerProcessorUnrecoverableErrorWithMB(t *testing.T) {
	t.Run("TestConcurrentConsumer_ProcessorUnrecoverableError", func(t *testing.T) {
		ConcurrentConsumerProcessorUnrecoverableErrorWithMB(t, mbConnectorConcurrent)
	})
	counter.Reset()
	t.Run("TestConcurrentConsumer_ProcessorUnrecoverableError_OneRoutine", func(t *testing.T) {
		ConcurrentConsumerProcessorUnrecoverableErrorWithMB(t, mbConnectorOneGoroutine)
	})
}

func ConcurrentConsumerProcessorUnrecoverableErrorWithMB(t *testing.T, mbConnector mbfixture.TestConnector) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testDeadline)
	defer cancel()
	var errMessage mbfixture.RawMessage
	msgIndex := 0
	sentMsgs := produceTestData(ctx, t, mbConnector, func(message *messagebroker.Message) error {
		var err error
		if msgIndex == 1 {
			errMessage = mbfixture.RawMessage{
				Key:   message.Key,
				Value: string(message.Value),
				Topic: message.Topic,
			}
			err = messagebroker.ErrUnrecoverable
		}
		msgIndex++

		return err
	})
	require.NoError(t, mbConnector.VerifyMessages(ctx, sentMsgs[0]))
	windowedCtx, windowedCancel := context.WithTimeout(ctx, 10*time.Second)
	defer windowedCancel()
	require.NoError(t, mbConnector.VerifyNoMessages(windowedCtx, errMessage))
	go func() {
		<-time.Tick(10 * time.Second)
		counter.RetErrs.Store(errMessage.Key, nil) // Error resolved.
	}()
	require.NoError(t, mbConnector.VerifyMessages(ctx, sentMsgs...))
	assert.Greater(t, counter.ProcessCallCounts, uint64(len(sentMsgs)))
	assert.NotEqual(t, counter.ProcessCallCounts, counter.SuccessProcessCallCounts)
}

//nolint:paralleltest // Cuz of integration tests.
func TestConcurrentConsumerShutdownCommit(t *testing.T) {
	t.Run("TestConcurrentConsumer_ShutdownCommit", func(t *testing.T) {
		ConcurrentConsumerShutdownCommitWithMB(t, mbConnectorConcurrent, false)
	})
	counter.Reset()
	t.Run("TestConcurrentConsumer_ShutdownCommit_OneRoutine", func(t *testing.T) {
		ConcurrentConsumerShutdownCommitWithMB(t, mbConnectorOneGoroutine, true)
	})
}

//nolint:funlen,revive // A lo of checks
func ConcurrentConsumerShutdownCommitWithMB(t *testing.T, mbConnector mbfixture.TestConnector, oneGoroutine bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testDeadline)
	defer cancel()
	var errMessage mbfixture.RawMessage
	msgIndex := 0
	sentMsgs := produceTestData(ctx, t, mbConnector, func(message *messagebroker.Message) error {
		defer func() { msgIndex++ }()
		if msgIndex == 1 {
			errMessage = mbfixture.RawMessage{Key: message.Key, Value: string(message.Value), Topic: message.Topic}

			return messagebroker.ErrUnrecoverable
		}

		return nil
	})
	require.NoError(t, mbConnector.VerifyMessages(ctx, sentMsgs[0]))
	windowedCtx, windowedCancel := context.WithTimeout(ctx, 10*time.Second)
	defer windowedCancel()
	require.NoError(t, mbConnector.VerifyNoMessages(windowedCtx, errMessage))
	lastMsgBeforeShutdown := counter.LastReceivedMessage
	assert.Equal(t, lastMsgBeforeShutdown, errMessage)
	require.NoError(t, mbConnector.Close())
	mbConnector.ReconnectConsumer(ctx)
	recheckAfterReconnectCtx, recheckAfterReconnectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer recheckAfterReconnectCancel()
	require.NoError(t, mbConnector.VerifyNoMessages(recheckAfterReconnectCtx, errMessage))
	if oneGoroutine {
		assert.Equal(t, lastMsgBeforeShutdown, counter.LastReceivedMessage)
	}
	go func() { <-time.Tick(5 * time.Second); counter.RetErrs.Store(errMessage.Key, nil) }()
	require.NoError(t, mbConnector.VerifyMessages(ctx, sentMsgs...))
	assert.Equal(t, uint64(len(sentMsgs)), counter.SuccessProcessCallCounts)
}
