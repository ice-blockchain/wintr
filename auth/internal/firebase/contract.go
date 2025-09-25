// SPDX-License-Identifier: ice License 1.0

package firebaseauth //nolint:revive //.

import (
	"context"

	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth/internal"
)

// Public API.

var (
	ErrUserNotFound = errors.New("user not found")
	ErrConflict     = errors.New("change conflicts with another user")
	ErrForbidden    = errors.New("forbidden")
)

type (
	Client interface {
		VerifyToken(ctx context.Context, token string) (*internal.Token, error)
		UpdateCustomClaims(ctx context.Context, userID string, customClaims map[string]any) error
		DeleteUser(ctx context.Context, userID string) error
		UpdateEmail(ctx context.Context, userID, email string) error
		GetUser(ctx context.Context, userID string) (*firebaseauth.UserRecord, error)
		GetUserByEmail(ctx context.Context, email string) (*firebaseauth.UserRecord, error)
	}
)

// Private API.
const (
	passwordSignInProvider = "password" //nolint:gosec // Not an actual password.
)

type (
	auth struct {
		client             *firebaseauth.Client
		allowEmailPassword bool
	}

	config struct {
		WintrAuthFirebase struct {
			Credentials struct {
				FilePath    string `yaml:"filePath"`
				FileContent string `yaml:"fileContent"`
			} `yaml:"credentials" mapstructure:"credentials"`
			AllowEmailPassword bool `yaml:"allowEmailPassword"`
		} `yaml:"wintr/auth/firebase" mapstructure:"wintr/auth/firebase"` //nolint:tagliatelle // Nope.
	}
)
