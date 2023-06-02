// SPDX-License-Identifier: ice License 1.0

package internal

// Public API.

type (
	Token struct {
		Claims   map[string]any
		UserID   string `json:"userId,omitempty"`
		Role     string `json:"role,omitempty"`
		Email    string `json:"email,omitempty"`
		Provider string
	}
)
