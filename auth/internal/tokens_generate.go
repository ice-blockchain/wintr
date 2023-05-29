// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/time"
)

func generateRefreshToken(cr TokenCreator, now *time.Time, userID, email string, seq int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Token{
		RegisteredClaims: &jwt.RegisteredClaims{
			Issuer:    JwtIssuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(cr.RefreshDuration())),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Email: email,
		Seq:   seq,
	})
	refreshToken, err := cr.SignedString(token)

	return refreshToken, errors.Wrapf(err, "failed to generate refresh token for userID:%v, email:%v", userID, email)
}

//nolint:funlen,revive // Fields.
func generateAccessToken(
	cr TokenCreator,
	now *time.Time, refreshTokenSeq, hashCode int64,
	userID, email string,
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
			ExpiresAt: jwt.NewNumericDate(now.Add(cr.AccessDuration())),
			NotBefore: jwt.NewNumericDate(*now.Time),
			IssuedAt:  jwt.NewNumericDate(*now.Time),
		},
		Role:     role,
		Email:    email,
		HashCode: hashCode,
		Seq:      refreshTokenSeq,
		Custom:   customClaims,
	})
	tokenStr, err := cr.SignedString(token)

	return tokenStr, errors.Wrapf(err, "failed to generate access token for userID:%v and email:%v", userID, email)
}

//nolint:revive // .
func GenerateTokens(
	secret TokenCreator,
	now *time.Time,
	userID, email string,
	hashCode,
	seq int64,
	claims map[string]any,
) (refreshToken, accessToken string, err error) {
	refreshToken, err = generateRefreshToken(secret, now, userID, email, seq)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to generate jwt refreshToken for userID:%v", userID)
	}
	accessToken, err = generateAccessToken(secret, now, seq, hashCode, userID, email, claims)

	return refreshToken, accessToken, errors.Wrapf(err, "failed to generate accessToken for userID:%v", userID)
}
