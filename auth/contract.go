// SPDX-License-Identifier: ice License 1.0

package auth

import (
	"context"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

// Public API.

var (
	ErrUserNotFound = errors.New("user not found")
	ErrConflict     = errors.New("change conflicts with another user")

	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
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
		VerifyFBToken(ctx context.Context, token string) (*Token, error)
		VerifyIceToken(ctx context.Context, token string) (*Token, error)
		UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error
		UpdateEmail(ctx context.Context, userID, email string) error
		UpdatePhoneNumber(ctx context.Context, userID, phoneNumber string) error
		DeleteUser(ctx context.Context, userID string) error
	}

	IceToken struct {
		*jwt.RegisteredClaims
		Custom   *map[string]any `json:"custom,omitempty"`
		Role     string          `json:"role" example:"1"`
		Email    string          `json:"email" example:"jdoe@example.com"`
		HashCode int64           `json:"hashCode,omitempty" example:"12356789"`
		Seq      int64           `json:"seq" example:"1"`
	}
)

// Private API.

const (
	jwtIssuer = "ice.io"
)

// .
var (
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	cfg config
)

type (
	auth struct {
		client *firebaseAuth.Client
	}

	config struct {
		WintrAuth struct {
			JWTSecret string `yaml:"jwtSecret" mapstructure:"jwtSecret"`
		} `yaml:"wintr/auth" mapstructure:"wintr/auth"` //nolint:tagliatelle // Nope.
	}
)
