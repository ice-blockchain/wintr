// SPDX-License-Identifier: ice License 1.0

package sms

import (
	"context"
	stdlibtime "time"

	"github.com/pkg/errors"
	"github.com/twilio/twilio-go"

	"github.com/ice-blockchain/wintr/sms/internal"
	"github.com/ice-blockchain/wintr/time"
)

// Public API.

const (
	MinimumSchedulingDurationInAdvance = 16 * stdlibtime.Minute
)

var (
	ErrInvalidPhoneNumber       = errors.New("phone number invalid")
	ErrInvalidPhoneNumberFormat = errors.New("phone number has invalid format")
	ErrSchedulingDateTooEarly   = errors.New("scheduled time is too early")
	ErrUnsupportedCountry       = errors.New("unsupported country")
)

type (
	Parcel struct {
		SendAt   *time.Time
		ToNumber string
		Message  string
	}

	Client interface {
		VerifyPhoneNumber(ctx context.Context, number string) error
		Send(ctx context.Context, p *Parcel) error
	}
)

// Private API.

const (
	requestDeadline = 25 * stdlibtime.Second
)

type (
	sms struct {
		client           *twilio.RestClient
		sendersByCountry map[string]*internal.MessagingService
	}
)
