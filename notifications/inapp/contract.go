// SPDX-License-Identifier: BUSL-1.1

package inapp

import (
	"context"

	"github.com/GetStream/stream-go2/v7"
)

// Public API.

type (
	Client interface {
		Send(context.Context, string, *Parcel) error
		SendMulti(context.Context, string, []*Parcel) error
		GetAll(context.Context, string) ([]*Parcel, error)
	}

	Parcel struct {
		Data        map[string]string
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
		client *stream.Client
		slug   string
	}

	config struct {
		Credentials struct {
			Key    string `yaml:"key"`
			Secret string `yaml:"secret"`
		} `yaml:"credentials"`
	}
)
