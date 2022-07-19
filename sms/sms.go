// SPDX-License-Identifier: BUSL-1.1

package sms

import (
	"context"
	"os"
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

func New(applicationYAMLKey string) Client {
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	if cfg.Credentials.Login == "" {
		cfg.Credentials.Login = os.Getenv("SMS_CLIENT_LOGIN")
	}
	if cfg.Credentials.Password == "" {
		cfg.Credentials.Password = os.Getenv("SMS_CLIENT_PASSWORD")
	}

	return &sms{
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: cfg.Credentials.Login,
			Password: cfg.Credentials.Password,
		}),
	}
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
	wg.Add(len(parcels))
	ch := make(chan error, len(parcels))

	for ix := range parcels {
		go func(i int) {
			defer wg.Done()
			ch <- s.Send(ctx, parcels[i])
		}(ix)
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
