// SPDX-License-Identifier: ice License 1.0

package internal

import (
	stdlibtime "time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

// Public API.

const (
	JwtIssuer = "ice.io"
)

// .
var (
	ErrInvalidToken = errors.New("invalid token")
)

type (
	Token struct {
		*jwt.RegisteredClaims
		Custom   *map[string]any `json:"custom,omitempty"`
		Role     string          `json:"role" example:"1"`
		Email    string          `json:"email" example:"jdoe@example.com"`
		HashCode int64           `json:"hashCode,omitempty" example:"12356789"`
		Seq      int64           `json:"seq" example:"1"`
	}
	TokenCreator interface {
		SignedString(token *jwt.Token) (string, error)
		AccessDuration() stdlibtime.Duration
		RefreshDuration() stdlibtime.Duration
	}
	TokenVerifier interface {
		Verify() func(token *jwt.Token) (any, error)
	}
	Secret interface {
		TokenCreator
		TokenVerifier
	}
)

// Private API.

const (
	defaultRole = "app"
)

type (
	config struct {
		WintrServerAuth struct {
			Credentials struct {
				FilePath    string `yaml:"filePath"`
				FileContent string `yaml:"fileContent"`
			} `yaml:"credentials" mapstructure:"credentials"`
			JWTSecret             string              `yaml:"jwtSecret" mapstructure:"jwtSecret"`
			RefreshExpirationTime stdlibtime.Duration `yaml:"refreshExpirationTime" mapstructure:"refreshExpirationTime"`
			AccessExpirationTime  stdlibtime.Duration `yaml:"accessExpirationTime" mapstructure:"accessExpirationTime"`
		} `yaml:"wintr/server/auth" mapstructure:"wintr/server/auth"` //nolint:tagliatelle // Nope.
	}
	iceAuthSecrets struct {
		cfg config
	}
)
