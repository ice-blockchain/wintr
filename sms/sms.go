// SPDX-License-Identifier: BUSL-1.1

package sms

import (
	"context"
	"sync"
	stdlibtime "time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(applicationYamlKey string) Client {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	c := &sms{}
	c.client = twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.Credentials.Login,
		Password: cfg.Credentials.Password,
	})

	return c
}

func (s *sms) Send(ctx context.Context, parcel Parcel) error {
	return errors.Wrapf(retry(ctx, func() error {
		if ctx.Err() != nil {
			//nolint:wrapcheck // It's a proxy.
			return backoff.Permanent(ctx.Err())
		}

		_, err := s.client.Api.CreateMessage(new(openapi.CreateMessageParams).
			SetTo(parcel.ToNumber).
			SetFrom(cfg.FromPhoneNumber).
			SetBody(parcel.Message))

		//nolint:wrapcheck // It's wrapped outside.
		return err
	}), "failed to send sms message via twilio")
}

func (s *sms) SendMulti(ctx context.Context, parcels []Parcel) error {
	var wg sync.WaitGroup
	ch := make(chan error, len(parcels))

	for _, a := range parcels {
		wg.Add(1)
		copyA := a

		go func() {
			defer wg.Done()
			ch <- s.Send(ctx, copyA)
		}()
	}

	wg.Wait()
	close(ch)

	var m *multierror.Error
	for e := range ch {
		m = multierror.Append(m, e)
	}

	return errors.Wrapf(m.ErrorOrNil(), "error during sending multiple messages")
}

func retry(ctx context.Context, op func() error) error {
	//nolint:wrapcheck // No need, its just a proxy.
	return backoff.RetryNotify(
		op,
		//nolint:gomnd // Because those are static configs.
		backoff.WithContext(&backoff.ExponentialBackOff{
			InitialInterval:     100 * stdlibtime.Millisecond,
			RandomizationFactor: 0.5,
			Multiplier:          2.5,
			MaxInterval:         stdlibtime.Second,
			MaxElapsedTime:      25 * stdlibtime.Second,
			Stop:                backoff.Stop,
			Clock:               backoff.SystemClock,
		}, ctx),
		func(e error, next stdlibtime.Duration) {
			log.Error(errors.Wrapf(e, "call failed. retrying in %v... ", next))
		})
}
