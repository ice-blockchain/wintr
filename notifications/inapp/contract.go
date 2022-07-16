// SPDX-License-Identifier: BUSL-1.1

package inapp

import (
	"context"

	"github.com/GetStream/stream-go2/v7"
)

// Public API.

type (
	UserID = string
	Client interface {
		Send(context.Context, *Parcel) error
		SendMulti(context.Context, []*Parcel) error
		GetAll(context.Context, UserID) ([]*Parcel, error)
	}

	Parcel struct {
		UserID
		Data        map[string]interface{}
		ReferenceID string
		Action      string
		Actor       ID
		Subject     ID
	}

	ID struct {
		Type  string
		Value string
	}
)

// Private API.

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	inApp struct {
		client   *stream.Client
		feedName string
	}

	config struct {
		Credentials struct {
			Key    string `yaml:"key"`
			Secret string `yaml:"secret"`
		} `yaml:"credentials"`
	}
)
