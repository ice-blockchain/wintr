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
	config struct {
		WintrSMS struct {
			Credentials struct {
				User     string `yaml:"user"`
				Password string `yaml:"password"`
			} `yaml:"credentials" mapstructure:"credentials"`
		} `yaml:"wintr/sms" mapstructure:"wintr/sms"` //nolint:tagliatelle // Nope.
	}
)
