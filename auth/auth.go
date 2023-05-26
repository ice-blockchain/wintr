// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/imdario/mergo"
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
		client: internal.New(ctx, applicationYAMLKey),
	}
}

func (a *auth) VerifyToken(ctx context.Context, token string) (*Token, error) {
	if err := detectCustomToken(token); err != nil { //nolint:nestif // .
		firebaseToken, vErr := a.client.VerifyIDToken(ctx, token)
		if vErr != nil {
			return nil, errors.Wrap(vErr, "error verifying firebase token")
		}
		var email, role string
		if len(firebaseToken.Claims) > 0 {
			if emailInterface, found := firebaseToken.Claims["email"]; found {
				email, _ = emailInterface.(string) //nolint:errcheck // Not needed.
			}
			if roleInterface, found := firebaseToken.Claims["role"]; found {
				role, _ = roleInterface.(string) //nolint:errcheck // Not needed.
			}
		}

		return &Token{
			UserID: firebaseToken.UID,
			Claims: firebaseToken.Claims,
			Email:  email,
			Role:   role,
		}, nil
	}
	authToken, cErr := verifyCustomToken(token)
	if cErr != nil {
		return nil, errors.Wrapf(cErr, "can't verify custom token:%v", token)
	}

	return authToken, nil
}

func (a *auth) UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	user, err := a.client.GetUser(ctx, userID)
	if err != nil {
		if strings.HasSuffix(err.Error(), fmt.Sprintf("no user exists with the uid: %q", userID)) {
			return ErrUserNotFound
		}

		return errors.Wrapf(err, "failed to get current user for userID:`%v`", userID)
	}
	if err = mergo.Merge(&customClaims, user.CustomClaims, mergo.WithOverride, mergo.WithTypeCheck); err != nil {
		return errors.Wrapf(err, "failed to merge %#v and %#v", customClaims, user.CustomClaims)
	}
	if err = a.client.SetCustomUserClaims(ctx, userID, customClaims); err != nil {
		if strings.HasSuffix(err.Error(), "no user record found for the given identifier") {
			return ErrUserNotFound
		}

		return errors.Wrapf(err, "failed to update custom claims to `%#v`, for userID:`%v`", customClaims, userID)
	}

	return nil
}

func (a *auth) UpdateEmail(ctx context.Context, userID, email string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if _, err := a.client.UpdateUser(ctx, userID, new(firebaseAuth.UserToUpdate).Email(email).EmailVerified(true)); err != nil {
		if strings.HasSuffix(err.Error(), "user with the provided email already exists") {
			return ErrConflict
		}
		if strings.HasSuffix(err.Error(), "no user record found for the given identifier") {
			return ErrUserNotFound
		}

		return errors.Wrapf(err, "failed to update email to `%v`, for userID:`%v`", email, userID)
	}

	return nil
}

func (a *auth) UpdatePhoneNumber(ctx context.Context, userID, phoneNumber string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if _, err := a.client.UpdateUser(ctx, userID, new(firebaseAuth.UserToUpdate).PhoneNumber(phoneNumber)); err != nil {
		if strings.HasSuffix(err.Error(), "user with the provided phone number already exists") {
			return ErrConflict
		}
		if strings.HasSuffix(err.Error(), "no user record found for the given identifier") {
			return ErrUserNotFound
		}

		return errors.Wrapf(err, "failed to update phoneNumber to `%v`, for userID:`%v`", phoneNumber, userID)
	}

	return nil
}

func (a *auth) DeleteUser(ctx context.Context, userID string) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if err := a.client.DeleteUser(ctx, userID); err != nil {
		if err.Error() == "no user record found for the given identifier" {
			return nil
		}

		return errors.Wrapf(err, "failed to delete user by ID:`%v`", userID)
	}

	return nil
}

func verifyCustomToken(jwtToken string) (*Token, error) {
	var token CustomToken
	err := VerifyJWTCommonFields(jwtToken, cfg.WintrAuth.JWTSecret, &token)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid email token:%v", jwtToken)
	}

	return &Token{
		Claims: map[string]any{
			"email": token.Email,
			"role":  token.Role,
			"seq":   token.Seq,
		},
		UserID: token.Subject,
		Email:  token.Email,
		Role:   token.Role,
	}, nil
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

func detectCustomToken(jwtToken string) error {
	parser := jwt.NewParser()
	var claims CustomToken
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
