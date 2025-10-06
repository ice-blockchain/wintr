// SPDX-License-Identifier: ice License 1.0

package internal

import (
	stdlibtime "time"
)

// Public API.

type (
	PhoneNumbersRoundRobinLB struct {
		schedulingMessagingServiceSID string
		phoneNumbers                  []string
		currentIndex                  uint64
	}
)

// Private API.

const (
	requestDeadline = 25 * stdlibtime.Second
)

type (
	countryCode = string
	config      struct {
		WintrSMS struct {
			MessageServiceSIDs map[countryCode]string `yaml:"messageServiceSIDs" mapstructure:"messageServiceSIDs"` //nolint:tagliatelle // .
			Credentials        struct {
				User     string `yaml:"user"`
				Password string `yaml:"password"`
			} `yaml:"credentials" mapstructure:"credentials"`
		} `yaml:"wintr/sms" mapstructure:"wintr/sms"` //nolint:tagliatelle // Nope.
	}
)
