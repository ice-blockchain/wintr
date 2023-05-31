// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
)

func New(ctx context.Context, applicationYAMLKey string) Client {
	Secret = internal.NewICEAuthSecret(ctx, applicationYAMLKey)

	return &auth{
		fb:  &authFirebase{client: internal.NewFirebase(ctx, applicationYAMLKey)},
		ice: &authIce{secret: Secret},
	}
}

func (a *auth) VerifyToken(ctx context.Context, token string) (*Token, error) {
	var authToken *Token
	if err := detectIceToken(token); err != nil {
		authToken, err = a.fb.VerifyToken(ctx, token)

		return authToken, errors.Wrapf(err, "can't verify fb token:%v", token)
	}
	authToken, err := a.ice.VerifyToken(ctx, token)

	return authToken, errors.Wrapf(err, "can't verify ice token:%v", token)
}

func (a *auth) UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error {
	u, err := a.fb.GetUser(ctx, userID)
	isFirebaseUser := err == nil && u != nil
	if isFirebaseUser {
		return errors.Wrapf(a.fb.UpdateCustomClaims(ctx, userID, customClaims), "failed to update custom claims for user:%v using firebase auth", userID)
	}

	return errors.Wrapf(a.ice.UpdateCustomClaims(ctx, userID, customClaims), "failed to update custom claims for user:%v using ice auth", userID)
}

func (a *auth) UpdateEmail(ctx context.Context, userID, email string) error {
	u, err := a.fb.GetUser(ctx, userID)
	isFirebaseUser := err == nil && u != nil
	if isFirebaseUser {
		return errors.Wrapf(a.fb.UpdateEmail(ctx, userID, email), "failed to update email for user:%v using firebase auth", userID)
	}

	return errors.Wrapf(a.ice.UpdateEmail(ctx, userID, email), "failed to update email for user:%v using ice auth", userID)
}

func (a *auth) UpdatePhoneNumber(ctx context.Context, userID, phoneNumber string) error {
	u, err := a.fb.GetUser(ctx, userID)
	isFirebaseUser := err == nil && u != nil
	if isFirebaseUser {
		return errors.Wrapf(a.fb.UpdatePhoneNumber(ctx, userID, phoneNumber), "failed to update phone number for user:%v using firebase auth", userID)
	}

	return errors.Wrapf(a.ice.UpdatePhoneNumber(ctx, userID, phoneNumber), "failed to update phone number for user:%v using ice auth", userID)
}

func (a *auth) DeleteUser(ctx context.Context, userID string) error {
	u, err := a.fb.GetUser(ctx, userID)
	isFirebaseUser := err == nil && u != nil
	if isFirebaseUser {
		return errors.Wrapf(a.fb.DeleteUser(ctx, userID), "failed to delete user:%v using firebase auth", userID)
	}

	return errors.Wrapf(a.ice.DeleteUser(ctx, userID), "failed to delete user:%v using ice auth", userID)
}

func VerifyJWTCommonFields(jwtToken string, verifier TokenVerifier, res jwt.Claims) error {
	if _, err := jwt.ParseWithClaims(jwtToken, res, verifier.Verify()); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return errors.Wrapf(ErrExpiredToken, "expired or not valid yet token:%v", jwtToken)
		}

		return errors.Wrapf(err, "invalid token:%v", jwtToken)
	}

	return nil
}

func (tok *Token) IsICEToken() bool {
	return tok.provider == JwtIssuer
}
