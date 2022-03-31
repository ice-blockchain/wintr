// SPDX-License-Identifier: BUSL-1.1

package messagebroker

import (
	"context"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/ICE-Blockchain/wintr/log"
)

func (mb *messageBroker) SendMessage(ctx context.Context, m *Message, responder chan<- error) {
	headers := make([]kgo.RecordHeader, 0, len(m.Headers))
	if m.Headers != nil {
		for k, v := range m.Headers {
			headers = append(headers, kgo.RecordHeader{Key: k, Value: []byte(v)})
		}
	}
	r := &kgo.Record{
		Key:     []byte(m.Key),
		Value:   m.Value,
		Headers: headers,
		Topic:   m.Topic,
	}
	mb.client.Produce(ctx, r, func(record *kgo.Record, err error) {
		if err != nil {
			log.Error(errors.Wrap(err, "failed to produce record"), "record.value", string(m.Value), "record", m)
		} else {
			log.Debug("record produced", "record.value", string(record.Value), "record", record)
		}
		if responder != nil {
			responder <- err
		}
	})
}
