// SPDX-License-Identifier: ice License 1.0

package messagebroker

import (
	"context"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/ice-blockchain/wintr/log"
)

func (mb *messageBroker) SendMessage(ctx context.Context, msg *Message, responder chan<- error) {
	headers := make([]kgo.RecordHeader, 0, len(msg.Headers))
	if msg.Headers != nil {
		for k, v := range msg.Headers {
			headers = append(headers, kgo.RecordHeader{Key: k, Value: []byte(v)})
		}
	}
	record := &kgo.Record{
		Key:     []byte(msg.Key),
		Value:   msg.Value,
		Headers: headers,
		Topic:   msg.Topic,
	}
	mb.client.Produce(ctx, record, func(record *kgo.Record, err error) {
		if err != nil {
			log.Error(errors.Wrap(err, "failed to produce record"), "record.value", string(msg.Value), "record", msg)
		} else {
			log.Debug("record produced", "record.value", string(record.Value), "record", record)
		}
		if responder != nil {
			responder <- err
		}
	})
}
