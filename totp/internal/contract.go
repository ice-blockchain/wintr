// SPDX-License-Identifier: ice License 1.0

package internal

import (
	stdlibtime "time"
)

type (
	Generator interface {
		Create(userSecret string) TOTP
	}
	TOTP interface {
		ProvisioningUri(accountName, issuerName string) string
		VerifyTime(code string, time stdlibtime.Time) bool
	}
)
