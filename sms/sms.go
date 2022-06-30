package sms

import (
	"context"
	stdlibtime "time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func InitClient(applicationYamlKey string) Client {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	return twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.TwilioCredentials.SID,
		Password: cfg.TwilioCredentials.Token,
	})
}

func Send(ctx context.Context, client Client, toNumber, message string) error {
	return errors.Wrapf(retry(ctx, func() error {
		if ctx.Err() != nil {
			//nolint:wrapcheck // It's a proxy.
			return backoff.Permanent(ctx.Err())
		}

		_, err := client.Api.CreateMessage(new(openapi.CreateMessageParams).
			SetTo(toNumber).
			SetFrom(cfg.FromPhoneNumber).
			SetBody(message))

		//nolint:wrapcheck // It's wrapped outside.
		return err
	}), "failed to send sms message via twilio")
}

func SendAsync(ctx context.Context, client Client, toNumber, message string) chan error {
	r := make(chan error)
	go func() {
		r <- Send(ctx, client, toNumber, message)
	}()

	return r
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
