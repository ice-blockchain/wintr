// SPDX-License-Identifier: ice License 1.0

package totp

import (
	"encoding/base32"

	"github.com/xlzd/gotp"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/time"
)

func New(applicationYamlKey string) TOTP {
	gotpGen := newGotp()
	var cfg config
	appcfg.MustLoadFromKey(applicationYamlKey, &cfg)

	return &totp{generator: gotpGen, cfg: &cfg}
}

func (t *totp) GenerateURI(userSecret, account string) string {
	code := t.generator.CreateCode(userSecret)

	return code.ProvisioningUri(account, t.cfg.WintrTOTP.Issuer)
}

func (t *totp) Verify(now *time.Time, userSecret, totpCode string) bool {
	code := t.generator.CreateCode(userSecret)

	return code.VerifyTime(totpCode, *now.Time)
}

func newGotp() codeGenerator {
	return &gotpGenerator{}
}

func (*gotpGenerator) CreateCode(secret string) totpCode {
	encodedSecret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(secret))
	code := gotp.NewTOTP(encodedSecret, digitsInCode, rotationDuration, nil)

	return code
}
