// SPDX-License-Identifier: ice License 1.0

package totp

import (
	"testing"
	stdlibtime "time"

	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/time"
)

func TestTOTP(t *testing.T) {
	t.Parallel()
	totp := New("self")
	secret := "bogusSecret" //nolint:gosec // Bogus.
	uri := totp.GenerateURI(secret, "bogusAccount")
	require.Equal(t, "otpauth://totp/ice.io:bogusAccount?issuer=ice.io&secret=MJXWO5LTKNSWG4TFOQ", uri)
	now := time.New(stdlibtime.Date(2024, 7, 25, 8, 15, 56, 0, stdlibtime.UTC))
	validCode := "799503"
	validCodeAfter30s := "395417"
	require.True(t, totp.Verify(now, secret, validCode))
	require.Equal(t, totp.GenerateCode(time.New(now.Add(3*stdlibtime.Second)), secret), validCode)
	require.Equal(t, totp.GenerateCode(time.New(now.Add(3*stdlibtime.Second)), secret), validCode)
	require.False(t, totp.Verify(now, secret, "697025"))
	require.False(t, totp.Verify(time.New(now.Add(31*stdlibtime.Second)), secret, validCode))
	require.True(t, totp.Verify(time.New(now.Add(31*stdlibtime.Second)), secret, validCodeAfter30s))
	require.Equal(t, totp.GenerateCode(time.New(now.Add(31*stdlibtime.Second)), secret), validCodeAfter30s)
	require.False(t, totp.Verify(now, "wrongSecret", validCode))
	require.False(t, totp.Verify(time.New(now.Add(31*stdlibtime.Second)), "wrongSecret", validCode))
	require.False(t, totp.Verify(time.Now(), secret, ""))
}
