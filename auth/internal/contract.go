// SPDX-License-Identifier: ice License 1.0

package internal

// Public API.

const (
	RefreshJwtIssuer            = "ice.io/refresh"
	AccessJwtIssuer             = "ice.io/access"
	RegisteredWithProviderClaim = "registeredWithProvider"
	ProviderFirebase            = "firebase"
	ProviderIce                 = "ice"
)

type (
	Token struct {
		Claims   map[string]any
		UserID   string `json:"userId,omitempty"`
		Role     string `json:"role,omitempty"`
		Email    string `json:"email,omitempty"`
		Provider string
	}
)
