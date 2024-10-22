// SPDX-License-Identifier: ice License 1.0

package iceauth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	"github.com/ice-blockchain/wintr/time"
)

//nolint:revive // .
func (a *auth) GenerateTokens(
	now *time.Time,
	userID, deviceUniqueID, email string,
	hashCode,
	seq int64,
	role string,
	extra map[string]any,
) (refreshToken, accessToken string, err error) {
	refreshToken, err = a.generateRefreshToken(now, userID, deviceUniqueID, email, seq, extra)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to generate jwt refreshToken for userID:%v", userID)
	}
	accessToken, err = a.generateAccessToken(now, seq, hashCode, userID, deviceUniqueID, email, role, extra)

	return refreshToken, accessToken, errors.Wrapf(err, "failed to generate jwt accessToken for userID:%v", userID)
}

//nolint:revive // .
func (a *auth) generateRefreshToken(now *time.Time, userID, deviceUniqueID, email string, seq int64, extra map[string]any) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Token{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    internal.RefreshJwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(a.cfg.WintrAuthIce.RefreshExpirationTime)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Email:          email,
		Seq:            seq,
		DeviceUniqueID: deviceUniqueID,
		Tenant:         a.cfg.WintrAuthIce.Tenant,
		Claims:         extra,
	})
	refreshToken, err := a.signToken(token)

	return refreshToken, errors.Wrapf(err, "failed to generate refresh token for userID:%v, email:%v, deviceUniqueId:%v", userID, email, deviceUniqueID)
}

//nolint:revive // Fields.
func (a *auth) generateAccessToken(
	now *time.Time, refreshTokenSeq, hashCode int64,
	userID, deviceUniqueID, email string,
	role string,
	extra map[string]any,
) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Token{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    internal.AccessJwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(a.cfg.WintrAuthIce.AccessExpirationTime)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Role:           role,
		Email:          email,
		DeviceUniqueID: deviceUniqueID,
		HashCode:       hashCode,
		Seq:            refreshTokenSeq,
		Tenant:         a.cfg.WintrAuthIce.Tenant,
		Claims:         extra,
	})
	tokenStr, err := a.signToken(token)

	return tokenStr, errors.Wrapf(err, "failed to generate access token for userID:%v, email:%v, deviceUniqueId:%v", userID, email, deviceUniqueID)
}

func (a *auth) GenerateMetadata(now *time.Time, tokenID string, metadata map[string]any) (string, error) {
	metadata["sub"] = tokenID
	metadata["iss"] = internal.MetadataIssuer
	metadata["iat"] = jwt.NewNumericDate(*now.Time)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(metadata))
	tokenStr, err := a.signToken(token)

	return tokenStr, errors.Wrapf(err, "failed to generate metadata token for payload tokenID:%v, metadata:%#v", tokenID, metadata)
}
