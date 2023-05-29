// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	appCfg "github.com/ice-blockchain/wintr/config"
)

func New(ctx context.Context, applicationYAMLKey string) Client {
	var cfg config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	if cfg.WintrAuth.JWTSecret == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrAuth.JWTSecret = os.Getenv(fmt.Sprintf("%s_JWT_SECRET", module))
		if cfg.WintrAuth.JWTSecret == "" {
			cfg.WintrAuth.JWTSecret = os.Getenv("JWT_SECRET")
		}
	}

	return &auth{
		fb:  &authFirebase{client: internal.New(ctx, applicationYAMLKey)},
		ice: &authIce{cfg: cfg},
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
	u, err := a.fb.(*authFirebase).GetUser(ctx, userID)
	isFirebaseUser := err == nil && u != nil
	if isFirebaseUser {
		return errors.Wrapf(a.fb.UpdateCustomClaims(ctx, userID, customClaims), "failed to update phone number using firebase")
	}

	return errors.Wrapf(a.ice.UpdateCustomClaims(ctx, userID, customClaims), "failed to update custom claims using ice")
}

func (a *auth) UpdateEmail(ctx context.Context, userID, email string) error {
	u, err := a.fb.(*authFirebase).GetUser(ctx, userID)
	isFirebaseUser := err == nil && u != nil
	if isFirebaseUser {
		return errors.Wrapf(a.fb.UpdateEmail(ctx, userID, email), "failed to update email using firebase")
	}

	return errors.Wrapf(a.ice.UpdateEmail(ctx, userID, email), "failed to update email using ice")
}

func (a *auth) UpdatePhoneNumber(ctx context.Context, userID, phoneNumber string) error {
	u, err := a.fb.(*authFirebase).GetUser(ctx, userID)
	isFirebaseUser := err == nil && u != nil
	if isFirebaseUser {
		return errors.Wrapf(a.fb.UpdatePhoneNumber(ctx, userID, phoneNumber), "failed to update phone number using firebase")
	}

	return errors.Wrapf(a.ice.UpdatePhoneNumber(ctx, userID, phoneNumber), "failed to update phone number using ice")
}

func (a *auth) DeleteUser(ctx context.Context, userID string) error {
	u, err := a.fb.(*authFirebase).GetUser(ctx, userID)
	isFirebaseUser := err == nil && u != nil
	if isFirebaseUser {
		return errors.Wrapf(a.fb.DeleteUser(ctx, userID), "failed to delete user using firebase")
	}

	return errors.Wrapf(a.ice.DeleteUser(ctx, userID), "failed to delete user using ice")
}

func VerifyJWTCommonFields(jwtToken, secret string, res jwt.Claims) error {
	if _, err := jwt.ParseWithClaims(jwtToken, res, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if iss, err := token.Claims.GetIssuer(); err != nil || iss != jwtIssuer {
			return nil, errors.Wrapf(ErrInvalidToken, "invalid issuer:%v", iss)
		}

		return []byte(secret), nil
	}); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return errors.Wrapf(ErrExpiredToken, "expired or not valid yet token:%v", jwtToken)
		}

		return errors.Wrapf(err, "invalid token:%v", jwtToken)
	}

	return nil
}

func (tok *Token) IsICEToken() bool {
	return tok.provider == jwtIssuer
}
