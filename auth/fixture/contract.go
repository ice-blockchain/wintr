// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"github.com/golang-jwt/jwt/v5"
)

const (
	applicationYAMLKey = "self"
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
)
