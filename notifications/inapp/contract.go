// SPDX-License-Identifier: ice License 1.0

package inapp

import (
	"context"
	stdlibtime "time"

	getstreamio "github.com/GetStream/stream-go2/v7"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/notifications/inapp/internal"
)

// Public API.

const (
	MaxBatchSize = 100
)

type (
	UserID = string
	Token  struct {
		APIKey    string `json:"apiKey,omitempty"`
		APISecret string `json:"apiSecret,omitempty"`
		AppID     string `json:"appId,omitempty"`
	}
	Client interface {
		CreateUserToken(context.Context, UserID) (*Token, error)

		Send(context.Context, *Parcel, ...UserID) error
	}

	Parcel = internal.Parcel
	ID     = internal.ID
)

// Private API.

const (
	requestDeadline = 25 * stdlibtime.Second
)

// .
var (
	errPleaseRetry = errors.New("please retry")
)

type (
	inApp struct {
		client   *getstreamio.Client
		cfg      *config
		feedName string
	}

	config struct {
		WintrInAppNotifications struct {
			Credentials struct {
				Key    string `yaml:"key"`
				AppID  string `yaml:"appId" mapstructure:"appId"`
				Secret string `yaml:"secret"`
			} `yaml:"credentials" mapstructure:"credentials"`
		} `yaml:"wintr/notifications/inapp" mapstructure:"wintr/notifications/inapp"` //nolint:tagliatelle // Nope.
	}
)
