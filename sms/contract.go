package sms

import (
	"github.com/twilio/twilio-go"
)

// Public API.

type (
	Client = *twilio.RestClient
)

// Private API.

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	config struct {
		TwilioCredentials struct {
			SID   string `yaml:"SID"`
			Token string `yaml:"token"`
		} `yaml:"twilioCredentials"`
		FromPhoneNumber string `yaml:"fromPhoneNumber"`
	}
)
