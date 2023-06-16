// SPDX-License-Identifier: ice License 1.0

package iceauth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/time"
)

//nolint:revive // .
func (a *auth) GenerateTokens(
	now *time.Time,
	userID, deviceUniqueID, email string,
	hashCode,
	seq int64,
	claims map[string]any,
) (refreshToken, accessToken string, err error) {
	refreshToken, err = a.generateRefreshToken(now, userID, deviceUniqueID, email, seq)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to generate jwt refreshToken for userID:%v", userID)
	}
	accessToken, err = a.generateAccessToken(now, seq, hashCode, userID, deviceUniqueID, email, claims)

	return refreshToken, accessToken, errors.Wrapf(err, "failed to generate jwt accessToken for userID:%v", userID)
}

func (a *auth) generateRefreshToken(now *time.Time, userID, deviceUniqueID, email string, seq int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Token{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    JwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(a.cfg.WintrAuthIce.RefreshExpirationTime)),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Email:          email,
		Seq:            seq,
		DeviceUniqueID: deviceUniqueID,
	})
	refreshToken, err := a.signToken(token)

	return refreshToken, errors.Wrapf(err, "failed to generate refresh token for userID:%v, email:%v, deviceUniqueId:%v", userID, email, deviceUniqueID)
}

//nolint:funlen,revive // Fields.
func (a *auth) generateAccessToken(
	now *time.Time, refreshTokenSeq, hashCode int64,
	userID, deviceUniqueID, email string,
	claims map[string]any,
) (string, error) {
	var customClaims *map[string]any
	role := defaultRole
	if clRole, ok := claims["role"]; ok {
		if roleS, isStr := clRole.(string); isStr {
			role = roleS
			delete(claims, "role")
		}
	}
	if len(claims) > 0 {
		customClaims = &claims
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Token{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    JwtIssuer,
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
		Custom:         customClaims,
	})
	tokenStr, err := a.signToken(token)

	return tokenStr, errors.Wrapf(err, "failed to generate access token for userID:%v, email:%v, deviceUniqueId:%v", userID, email, deviceUniqueID)
}
