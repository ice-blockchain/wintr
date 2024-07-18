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
	IceIDClaim                  = internal.IceIDClaim
	FirebaseIDClaim             = internal.FirebaseIDClaim
	ProviderIce                 = internal.ProviderIce
	ProviderFirebase            = internal.ProviderFirebase
	RegisteredWithProviderClaim = internal.RegisteredWithProviderClaim
)

var (
	ErrUserNotFound = firebaseauth.ErrUserNotFound
	ErrConflict     = firebaseauth.ErrConflict
	ErrForbidden    = firebaseauth.ErrForbidden

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
		UpdateEmail(ctx context.Context, userID, email string) error
		GenerateTokens(now *time.Time, userID, deviceUniqueID, email string, hashCode, seq int64, role string, extras ...map[string]any) (accessToken, refreshToken string, err error) //nolint:lll // .
		GenerateMetadata(now *time.Time, userID string, md map[string]any) (string, error)
		ModifyTokenWithMetadata(token *Token, metadataStr string) (*Token, error)
		GetUserUIDByEmail(ctx context.Context, email string) (string, error)
	}
)

// Private API.

type (
	auth struct {
		ice iceauth.Client
		fb  firebaseauth.Client
	}
)
