// SPDX-License-Identifier: ice License 1.0

package totp

import (
	stdlibtime "time"

	"github.com/ice-blockchain/wintr/time"
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
		generator codeGenerator
		cfg       *config
	}
	config struct {
		WintrTOTP struct {
			Issuer string `yaml:"issuer" mapstructure:"issuer"`
		} `yaml:"wintr/totp" mapstructure:"wintr/totp"` //nolint:tagliatelle // .
	}
	codeGenerator interface {
		CreateCode(userSecret string) totpCode
	}
	totpCode interface {
		ProvisioningUri(accountName, issuerName string) string
		VerifyTime(code string, t stdlibtime.Time) bool
	}
	gotpGenerator struct{}
)

const (
	digitsInCode     = 6
	rotationDuration = 30
)
