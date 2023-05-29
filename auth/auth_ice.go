// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

func (a *authIce) VerifyToken(_ context.Context, token string) (*Token, error) {
	return a.verifyIceToken(token)
}

func (*authIce) UpdateCustomClaims(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

func (a *authIce) verifyIceToken(token string) (*Token, error) {
	var iceToken IceToken
	err := VerifyJWTCommonFields(token, a.cfg.WintrAuth.JWTSecret, &iceToken)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid email token:%v", token)
	}
	if iceToken.Role == "" {
		return nil, errors.Wrapf(ErrWrongTypeToken, "access to endpoint with refresh token")
	}

	return &Token{
		Claims: map[string]any{
			"email":    iceToken.Email,
			"role":     iceToken.Role,
			"seq":      iceToken.Seq,
			"hashCode": iceToken.HashCode,
		},
		UserID:   iceToken.Subject,
		Email:    iceToken.Email,
		Role:     iceToken.Role,
		provider: jwtIssuer,
	}, nil
}

func detectIceToken(jwtToken string) error {
	parser := jwt.NewParser()
	var claims IceToken
	token, _, err := parser.ParseUnverified(jwtToken, &claims)
	if err != nil {
		return errors.Wrapf(err, "parse unverified error for token:%v", jwtToken)
	}
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Name {
		return errors.Errorf("unexpected signing method:%v", token.Header["alg"])
	}
	if iss, iErr := token.Claims.GetIssuer(); iErr != nil || iss != jwtIssuer {
		return errors.Wrapf(ErrInvalidToken, "invalid issuer:%v", iss)
	}

	return nil
}

func (*authIce) UpdateEmail(_ context.Context, _, _ string) error {
	return nil
}

func (*authIce) UpdatePhoneNumber(_ context.Context, _, _ string) error {
	return nil
}

func (*authIce) DeleteUser(_ context.Context, _ string) error {
	return nil
}
