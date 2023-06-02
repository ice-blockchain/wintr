// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	stdlibtime "time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	jwtIssuer = "ice.io"
)

type (
	IceToken struct {
		*jwt.RegisteredClaims
		Custom   *map[string]any `json:"custom,omitempty"`
		Role     string          `json:"role" example:"1"`
		Email    string          `json:"email" example:"jdoe@example.com"`
		HashCode int64           `json:"hashCode,omitempty" example:"12356789"`
		Seq      int64           `json:"seq" example:"1"`
	}

	fixtureIceAuth struct {
		RefreshExpirationTime stdlibtime.Duration
		AccessExpirationTime  stdlibtime.Duration
	}
)
