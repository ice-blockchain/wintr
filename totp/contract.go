// SPDX-License-Identifier: ice License 1.0

package totp

import (
	"github.com/ice-blockchain/wintr/time"
	"github.com/ice-blockchain/wintr/totp/internal"
)

type (
	TOTP interface {
		Generator
		Verifier
	}
	Generator interface {
		GenerateURI(userSecret, account string) string
	}
	Verifier interface {
		Verify(now *time.Time, userSecret, totpCode string) bool
	}
)

type (
	totp struct {
		generator internal.Generator
		cfg       *config
	}
	config struct {
		WintrTOTP struct {
			Issuer string `yaml:"issuer" mapstructure:"issuer"`
		} `yaml:"wintr/totp" mapstructure:"wintr/totp"` //nolint:tagliatelle // .
	}
)
