// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/pkg/errors"
)

// Public API.

var (
	ErrUserNotFound = errors.New("user not found")
	ErrConflict     = errors.New("change conflicts with another user")
)

type (
	Token struct {
		Claims map[string]any
		UserID string `json:"userId,omitempty"`
		Role   string `json:"role,omitempty"`
		Email  string `json:"email,omitempty"`
	}
	Client interface {
		VerifyToken(ctx context.Context, token string) (*Token, error)
		UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error
		UpdateEmail(ctx context.Context, userID, email string) error
		UpdatePhoneNumber(ctx context.Context, userID, phoneNumber string) error
		DeleteUser(ctx context.Context, userID string) error
	}
)

// Private API.

type (
	auth struct {
		client *firebaseAuth.Client
	}
)
