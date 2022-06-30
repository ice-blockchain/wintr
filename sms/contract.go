// SPDX-License-Identifier: BUSL-1.1

package sms

import (
	"context"

	"github.com/twilio/twilio-go"
)

// Public API.

type (
	Parcel struct {
		ToNumber string
		Message  string
	}

	Client interface {
		Send(ctx context.Context, parcel Parcel) error
		SendMulti(ctx context.Context, parcels []Parcel) error
	}
)

// Private API.

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	sms struct {
		client *twilio.RestClient
	}

	config struct {
		Credentials struct {
			Login    string `yaml:"login"`
			Password string `yaml:"password"`
		} `yaml:"credentials"`
		FromPhoneNumber string `yaml:"fromPhoneNumber"`
	}
)
