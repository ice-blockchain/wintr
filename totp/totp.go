// SPDX-License-Identifier: ice License 1.0

package totp

import (
	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/time"
	googleauthentificator "github.com/ice-blockchain/wintr/totp/internal/google_authentificator"
)

func New(applicationYamlKey string) TOTP {
	google := googleauthentificator.New()
	var cfg config
	appcfg.MustLoadFromKey(applicationYamlKey, &cfg)

	return &totp{generator: google, cfg: &cfg}
}

func (t *totp) GenerateURI(userSecret, account string) string {
	code := t.generator.Create(userSecret)

	return code.ProvisioningUri(account, t.cfg.WintrTOTP.Issuer)
}

func (t *totp) Verify(now *time.Time, userSecret, totpCode string) bool {
	code := t.generator.Create(userSecret)

	return code.VerifyTime(totpCode, *now.Time)
}
