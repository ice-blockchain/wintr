// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"
	"fmt"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

func (a *authFirebase) VerifyToken(ctx context.Context, token string) (*Token, error) {
	return a.verifyFBToken(ctx, token)
}

func (a *authFirebase) verifyFBToken(ctx context.Context, token string) (*Token, error) {
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
		UserID:   firebaseToken.UID,
		Claims:   firebaseToken.Claims,
		Email:    email,
		Role:     role,
		provider: firebaseToken.Issuer,
	}, nil
}

func (a *authFirebase) UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error {
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

func (a *authFirebase) UpdateEmail(ctx context.Context, userID, email string) error {
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

func (a *authFirebase) UpdatePhoneNumber(ctx context.Context, userID, phoneNumber string) error {
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

func (a *authFirebase) DeleteUser(ctx context.Context, userID string) error {
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

func (a *authFirebase) GetUser(ctx context.Context, userID string) (*firebaseAuth.UserRecord, error) {
	user, err := a.client.GetUser(ctx, userID)

	return user, errors.Wrapf(err, "failed to get user from firebase")
}
