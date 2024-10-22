// SPDX-License-Identifier: ice License 1.0

package internal

// Public API.

const (
	RefreshJwtIssuer            = "ice.io/refresh"
	AccessJwtIssuer             = "ice.io/access"
	MetadataIssuer              = "ice.io/metadata"
	RegisteredWithProviderClaim = "registeredWithProvider"
	ProviderFirebase            = "firebase"
	ProviderIce                 = "ice"
	FirebaseIDClaim             = "firebaseId"
	IceIDClaim                  = "iceId"
)

type (
	Token struct {
		Claims   map[string]any
		UserID   string `json:"userId,omitempty"`
		Role     string `json:"role,omitempty"`
		Email    string `json:"email,omitempty"`
		Tenant   string `json:"tenant,omitempty"`
		Provider string
	}
)
