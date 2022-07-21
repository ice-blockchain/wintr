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

func (s *testMessageStore) VerifyMessages(ctx context.Context, expected ...RawMessage) error {
	for ctx.Err() == nil && !s.recordsFound(expected...) { //nolint:revive // It's checking continuously until it finds them.
	}

	if !s.recordsFound(expected...) {
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
		if e.Topic != a.Topic || e.Key != a.Key || !regexp.MustCompile(e.Value).MatchString(a.Value) {
			return false
		}
	}

	return true
}

func (s *testMessageStore) findExpectedInGlobalMessageSource(expected []RawMessage) []RawMessage {
	var actualFound []RawMessage
	for _, e := range expected {
		for _, a := range s.chronologicalMessageList {
			if e.Key == a.Key && e.Topic == a.Topic && regexp.MustCompile(e.Value).MatchString(a.Value) {
				actualFound = append(actualFound, a)

				break
			}
		}
	}

	return actualFound
}
