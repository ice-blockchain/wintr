// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"context"
	"regexp"
	"slices"
	"time"

	"github.com/pkg/errors"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/log"
)

func (s *testMessageStore) Process(ctx context.Context, msg *messagebroker.Message) error {
	if ctx.Err() != nil {
		log.Panic(errors.Wrap(ctx.Err(), "unexpected deadline while processing message"))
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	log.Debug("new record processed", "message.value", string(msg.Value), "message", msg)
	s.chronologicalMessageList = append(s.chronologicalMessageList, RawMessage{
		Key:   msg.Key,
		Value: string(msg.Value),
		Topic: msg.Topic,
	})

	return nil
}

func (s *testMessageStore) VerifyNoMessages(ctx context.Context, notExpected ...RawMessage) error {
	for ctx.Err() == nil {
		if !s.recordsNotFound(notExpected...) {
			return errors.Errorf("verifyNoMessages failed! not expected: %#v, but found: %#v", notExpected, s.chronologicalMessageList)
		}
	}

	return nil
}

func (s *testMessageStore) VerifyMessages(ctx context.Context, expected ...RawMessage) error {
	for ctx.Err() == nil && !s.recordsFound(expected...) { //nolint:revive // It's checking continuously until it finds them.
	}

	if !s.recordsFound(expected...) {
		return errors.Errorf("verifyMessages failed! expected %#v, actual %#v", expected, s.chronologicalMessageList)
	}

	return nil
}

func (s *testMessageStore) recordsNotFound(notExpected ...RawMessage) bool {
	s.mx.RLock()
	defer s.mx.RUnlock()

	if len(s.chronologicalMessageList) == 0 {
		time.Sleep(1 * time.Second)

		return true
	}

	specificNotExpected := s.findExpectedInGlobalMessageSource(notExpected)
	for i := range specificNotExpected {
		if specificNotExpected[i] != (RawMessage{}) {
			return false
		}
	}

	return true
}

func (s *testMessageStore) recordsFound(expected ...RawMessage) bool {
	s.mx.RLock()
	defer s.mx.RUnlock()

	if len(s.chronologicalMessageList) == 0 {
		return false
	}

	specificExpected := s.findExpectedInGlobalMessageSource(expected)

	return !slices.Contains(specificExpected, RawMessage{})
}

func (s *testMessageStore) findExpectedInGlobalMessageSource(expected []RawMessage) []RawMessage {
	actualFound := make([]RawMessage, len(expected), len(expected)) //nolint:gosimple,staticcheck // Prefer to set it explicitly.
	for i := range expected {
		for j := range s.chronologicalMessageList {
			if expected[i].Key == s.chronologicalMessageList[j].Key &&
				expected[i].Topic == s.chronologicalMessageList[j].Topic &&
				regexp.MustCompile(expected[i].Value).MatchString(s.chronologicalMessageList[j].Value) {
				actualFound[i] = s.chronologicalMessageList[j]

				break
			}
		}
	}

	return actualFound
}
