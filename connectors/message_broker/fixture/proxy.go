// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"context"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

func (p *proxyProcessor) Process(ctx context.Context, msg *messagebroker.Message) error {
	var errs []error
	for i, proc := range p.processors {
		if err := proc.Process(ctx, msg); err != nil {
			if errors.Is(err, messagebroker.ErrUnrecoverable) {
				return errors.Wrap(err, "error occurred while processing test message")
			}
			errs = append(errs, errors.Wrapf(err, "failed to execute %v processor %T", i, proc))
		}
	}

	return multierror.Append( //nolint:wrapcheck // .
		errors.Wrap(p.messageStore.Process(ctx, msg),
			"failed to save message to testMessageStore"),
		errs...).ErrorOrNil()
}

func (p *proxyProcessor) VerifyNoMessages(ctx context.Context, msgs ...RawMessage) error {
	return p.messageStore.VerifyNoMessages(ctx, msgs...)
}

func (p *proxyProcessor) VerifyMessages(ctx context.Context, msgs ...RawMessage) error {
	return p.messageStore.VerifyMessages(ctx, msgs...)
}

func (c *CallCounterProcessor) Reset() {
	atomic.StoreUint64(&c.ProcessCallCounts, 0)
	atomic.StoreUint64(&c.SuccessProcessCallCounts, 0)
}

func (c *CallCounterProcessor) Process(_ context.Context, msg *messagebroker.Message) error {
	atomic.AddUint64(&c.ProcessCallCounts, 1)
	err, found := c.RetErrs.Load(msg.Key)
	c.LastReceivedMessage = RawMessage{
		Key:   msg.Key,
		Value: string(msg.Value),
		Topic: msg.Topic,
	}
	if found && err != nil {
		return err.(error) //nolint:forcetypeassert // We're sure.
	}
	atomic.AddUint64(&c.SuccessProcessCallCounts, 1)

	return nil
}
