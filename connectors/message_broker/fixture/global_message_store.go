// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/log"
)

func (s *testMessageStore) Process(ctx context.Context, m *messagebroker.Message) error {
	if ctx.Err() != nil {
		log.Panic(errors.Wrap(ctx.Err(), "unexpected deadline while processing message"))
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	log.Debug("new record processed", "message.value", string(m.Value), "message", m)
	s.chronologicalMessageList = append(s.chronologicalMessageList, RawMessage{
		Key:   m.Key,
		Value: string(m.Value),
		Topic: m.Topic,
	})

	return nil
}

func (s *testMessageStore) VerifyMessages(ctx context.Context, expected ...RawMessage) error {
	for ctx.Err() == nil && !s.recordsFound(expected...) {
	}

	if !s.recordsFound(expected...) {
		//nolint:revive // This errors package is better.
		return errors.New(fmt.Sprintf("verifyMessages failed! expected %#v, actual %#v", expected, s.chronologicalMessageList))
	}

	return nil
}

func (s *testMessageStore) recordsFound(expected ...RawMessage) bool {
	s.mx.RLock()
	defer s.mx.RUnlock()

	if len(s.chronologicalMessageList) == 0 {
		return false
	}

	for i, a := range s.findExpectedInGlobalMessageSource(expected) {
		e := expected[i]
		if e.Key != a.Key || !regexp.MustCompile(e.Value).MatchString(a.Value) || e.Topic != a.Topic {
			return false
		}
	}

	return true
}

func (s *testMessageStore) findExpectedInGlobalMessageSource(expected []RawMessage) []RawMessage {
	var actualFound []RawMessage
	for _, e := range expected {
		for _, a := range s.chronologicalMessageList {
			if e.Key == a.Key && regexp.MustCompile(e.Value).MatchString(a.Value) && e.Topic == a.Topic {
				actualFound = append(actualFound, a)

				break
			}
		}
	}

	return actualFound
}
