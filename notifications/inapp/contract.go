// SPDX-License-Identifier: BUSL-1.1

package inapp

import (
	"context"

	"github.com/GetStream/stream-go2/v7"
)

// Public API.

type (
	Client interface {
		Send(context.Context, string, string, *NotificationData) error
		SendMulti(context.Context, string, string, []*NotificationData) error
		Get(context.Context, string, string) ([]*NotificationData, error)
	}

	NotificationData struct {
		Header   string
		ImageURL string
		BodyText string
	}
)

// Private API.

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	inApp struct {
		client *stream.Client
	}

	config struct {
		Credentials struct {
			Key    string `yaml:"key"`
			Secret string `yaml:"secret"`
		} `yaml:"credentials"`
	}
)
