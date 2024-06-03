// SPDX-License-Identifier: ice License 1.0

package sms

import (
	"context"
	stdlibtime "time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	twilioclient "github.com/twilio/twilio-go/client"
	twilioopenapi "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/sms/internal"
	"github.com/ice-blockchain/wintr/terror"
	"github.com/ice-blockchain/wintr/time"
)

func New(applicationYAMLKey string) Client {
	client, lb := internal.New(applicationYAMLKey)

	return &sms{
		client: client,
		lb:     lb,
	}
}

func (s *sms) VerifyPhoneNumber(ctx context.Context, number string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context error")
	}

	return errors.Wrap(retry(ctx, func() error {
		return s.verifyPhoneNumber(ctx, number)
	}), "failed to verify phone number via twilio")
}

func (s *sms) verifyPhoneNumber(ctx context.Context, number string) error { //nolint:revive // confusing-naming: It's an internal impl that can be retried.
	if ctx.Err() != nil {
		//nolint:wrapcheck // It's a proxy.
		return backoff.Permanent(ctx.Err())
	}
	lookupResponse, err := s.client.LookupsV1.FetchPhoneNumber(number, nil)
	if err != nil {
		//nolint:errorlint // errors.As(err,*twilioclient.TwilioRestError) doesn't seem to work.
		if tErr, ok := err.(*twilioclient.TwilioRestError); !ok || tErr.Code != 20404 || tErr.Status != 404 {
			return errors.Wrapf(err, "failed to validate and lookup phone number %v", number)
		}

		return backoff.Permanent(ErrInvalidPhoneNumber) //nolint:wrapcheck // No need to do that, we have everything we need.
	}
	if lookupResponse.PhoneNumber != nil && number != *lookupResponse.PhoneNumber {
		//nolint:wrapcheck // No need to do that, we have everything we need.
		return backoff.Permanent(terror.New(ErrInvalidPhoneNumberFormat, map[string]any{"phoneNumber": *lookupResponse.PhoneNumber}))
	}

	return nil
}

func (s *sms) Send(ctx context.Context, parcel *Parcel) error {
	return errors.Wrapf(retry(ctx, func() error {
		if ctx.Err() != nil {
			return backoff.Permanent(ctx.Err())
		}
		msg := new(twilioopenapi.CreateMessageParams).
			SetTo(parcel.ToNumber).
			SetFrom(s.lb.PhoneNumber()).
			SetBody(parcel.Message)
		if parcel.SendAt != nil {
			if parcel.SendAt.Sub(*time.Now().Time) < MinimumSchedulingDurationInAdvance {
				return backoff.Permanent(ErrSchedulingDateTooEarly)
			}
			msg = msg.
				SetMessagingServiceSid(s.lb.SchedulingMessageServiceSID()).
				SetScheduleType("fixed").
				SetSendAt(*parcel.SendAt.Time)
		}
		_, err := s.client.Api.CreateMessage(msg)

		//nolint:wrapcheck // It's wrapped outside.
		return err
	}), "failed to send sms message via twilio")
}

func retry(ctx context.Context, op func() error) error {
	//nolint:wrapcheck // No need, its just a proxy.
	return backoff.RetryNotify(
		op,
		backoff.WithContext(&backoff.ExponentialBackOff{
			InitialInterval:     100 * stdlibtime.Millisecond, //nolint:mnd,gomnd // .
			RandomizationFactor: 0.5,                          //nolint:mnd,gomnd // .
			Multiplier:          2.5,                          //nolint:mnd,gomnd // .
			MaxInterval:         stdlibtime.Second,
			MaxElapsedTime:      requestDeadline,
			Stop:                backoff.Stop,
			Clock:               backoff.SystemClock,
		}, ctx),
		func(e error, next stdlibtime.Duration) {
			log.Error(errors.Wrapf(e, "wintr/sms call failed. retrying in %v... ", next))
		})
}
