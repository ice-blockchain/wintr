// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"

	"github.com/ice-blockchain/wintr/auth/internal"
	firebaseauth "github.com/ice-blockchain/wintr/auth/internal/firebase"
	iceauth "github.com/ice-blockchain/wintr/auth/internal/ice"
	"github.com/ice-blockchain/wintr/time"
)

// Public API.

const (
	JwtIssuer = iceauth.JwtIssuer
)

var (
	ErrUserNotFound = firebaseauth.ErrUserNotFound
	ErrConflict     = firebaseauth.ErrConflict

	ErrInvalidToken   = iceauth.ErrInvalidToken
	ErrExpiredToken   = iceauth.ErrExpiredToken
	ErrWrongTypeToken = iceauth.ErrWrongTypeToken
)

type (
	Token    = internal.Token
	IceToken = iceauth.Token
	Client   interface {
		VerifyToken(ctx context.Context, token string) (*Token, error)
		ParseToken(token string) (*IceToken, error)
		UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error
		DeleteUser(ctx context.Context, userID string) error
		GenerateTokens(now *time.Time, userID, deviceUniqueID, email string, hashCode, seq int64, claims map[string]any) (string, string, error)
	}
)

// Private API.

type (
	auth struct {
		ice iceauth.Client
		fb  firebaseauth.Client
	}
)
