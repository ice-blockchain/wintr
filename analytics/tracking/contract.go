// SPDX-License-Identifier: ice License 1.0

package tracking

import (
	"context"
	stdlibtime "time"
)

// Public API.

type (
	Client interface {
		TrackAction(ctx context.Context, userID string, action *Action) error
		SetUserAttributes(ctx context.Context, userID string, attributes map[string]any) error
		DeleteUser(ctx context.Context, userID string) error
	}
	Action struct {
		Attributes map[string]any `json:"attributes,omitempty"`
		Name       string         `json:"name,omitempty"`
	}
)

// Private API.

const (
	requestDeadline = 25 * stdlibtime.Second
)

type (
	tracking struct {
		cfg *config
	}
	config struct {
		Tracking struct {
			Credentials struct {
				AppID  string `yaml:"appId"`
				APIKey string `yaml:"apiKey"`
			} `yaml:"credentials" mapstructure:"credentials"`
			BaseURL string `yaml:"baseUrl" mapstructure:"baseUrl"`
		} `yaml:"wintr/analytics/tracking" mapstructure:"wintr/analytics/tracking"` //nolint:tagliatelle // Nope.
	}
)
