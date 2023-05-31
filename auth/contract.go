// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
)

// Public API.

const (
	JwtIssuer = internal.JwtIssuer
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrConflict     = errors.New("change conflicts with another user")

	ErrInvalidToken = internal.ErrInvalidToken
	ErrExpiredToken = errors.New("expired token")

	ErrWrongTypeToken = errors.New("wrong type token")
	//nolint:gochecknoglobals // Stores configuration.
	Secret internal.Secret
)

type (
	IceToken      = internal.Token
	TokenVerifier = internal.TokenVerifier

	Token struct {
		Claims   map[string]any
		UserID   string `json:"userId,omitempty"`
		Role     string `json:"role,omitempty"`
		Email    string `json:"email,omitempty"`
		provider string
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
	authFirebase struct {
		client *firebaseAuth.Client
	}
	authIce struct {
		secret internal.Secret
	}

	auth struct {
		ice *authIce
		fb  *authFirebase
	}
)
