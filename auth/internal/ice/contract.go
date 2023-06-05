// SPDX-License-Identifier: ice License 1.0

package iceauth

import (
	stdlibtime "time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
	"github.com/ice-blockchain/wintr/time"
)

// Public API.

const (
	JwtIssuer = "ice.io"
)

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrExpiredToken   = errors.New("expired token")
	ErrWrongTypeToken = errors.New("wrong type token")
)

type (
	Client interface {
		VerifyToken(token string) (*internal.Token, error)
		GenerateTokens(now *time.Time, userID, email string, hashCode, seq int64, claims map[string]any) (string, string, error)
		VerifyTokenFields(token string, res jwt.Claims) error
	}

	Token struct {
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
	defaultRole = "app"
)

type (
	auth struct {
		cfg *config
	}

	config struct {
		WintrAuthIce struct {
			JWTSecret             string              `yaml:"jwtSecret" mapstructure:"jwtSecret"`
			RefreshExpirationTime stdlibtime.Duration `yaml:"refreshExpirationTime" mapstructure:"refreshExpirationTime"`
			AccessExpirationTime  stdlibtime.Duration `yaml:"accessExpirationTime" mapstructure:"accessExpirationTime"`
		} `yaml:"wintr/auth/ice" mapstructure:"wintr/auth/ice"` //nolint:tagliatelle // Nope.
	}
)