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
		UserID      UserID                 `json:"userId" example:"users:john"`
		Data        map[string]interface{} `json:"data"`
		ReferenceID string                 `json:"referenceId" example:"users:michael"`
		Action      string                 `json:"action" example:"slaps you with a large trout"`
		Actor       ID                     `json:"actor"`
		Subject     ID                     `json:"subject"`
	}

	ID struct {
		Type  string `json:"type" example:"userId"`
		Value string `json:"value" example:"john"`
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
