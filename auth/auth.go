// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"
	"fmt"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
)

func New(ctx context.Context, applicationYAMLKey string) Client {
	return &auth{
		client: internal.New(ctx, applicationYAMLKey),
	}
}

func (a *auth) VerifyToken(ctx context.Context, token string) (*Token, error) {
	authToken, err := a.client.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, errors.Wrap(err, "error verifying ID token")
	}
	var email, role string
	if len(authToken.Claims) > 0 {
		if emailInterface, found := authToken.Claims["email"]; found {
			email, _ = emailInterface.(string) //nolint:errcheck // Not needed.
		}
		if roleInterface, found := authToken.Claims["role"]; found {
			role, _ = roleInterface.(string) //nolint:errcheck // Not needed.
		}
	}

	return &Token{
		UserID: authToken.UID,
		Claims: authToken.Claims,
		Email:  email,
		Role:   role,
	}, nil
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
