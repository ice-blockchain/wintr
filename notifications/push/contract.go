// SPDX-License-Identifier: BUSL-1.1

package push

import (
	"context"
	"firebase.google.com/go/v4/messaging"
)

// Public API.

type (
	Parcel struct {
		Notification Notification `json:"notification,omitempty"`
		Device       Device       `json:"device,omitempty"`
		UserID       string       `json:"userId,omitempty"`
	}
	Device struct {
		ID    string `json:"id,omitempty"`
		Token string `json:"token,omitempty"`
	}
	Notification struct {
		Data     map[string]string `json:"data,omitempty"`
		Title    string            `json:"title,omitempty"`
		Body     string            `json:"body,omitempty"`
		ImageURL string            `json:"imageUrl,omitempty"`
	}

	Client interface {
		Send(context.Context, *Parcel) (string, error)
		SendMulti(context.Context, []*Parcel) (*messaging.BatchResponse, error)
	}
)

// Private API.

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	push struct {
		Client *messaging.Client
	}

	config struct {
		FCMCredentialsFile string `yaml:"fcmCredentialsFile" json:"fcmCredentialsFile"`
	}
)
