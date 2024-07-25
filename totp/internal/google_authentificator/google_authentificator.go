// SPDX-License-Identifier: ice License 1.0

package googleauthentificator

import (
	"encoding/base32"

	"github.com/xlzd/gotp"

	"github.com/ice-blockchain/wintr/totp/internal"
)

func New() internal.Generator {
	return &googleGenerator{}
}

func (*googleGenerator) Create(secret string) internal.TOTP {
	encodedSecret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(secret))
	totp := gotp.NewTOTP(encodedSecret, digitsInCode, rotationDuration, nil)

	return totp
}
