// SPDX-License-Identifier: BUSL-1.1

package email

import (
	"context"

	"github.com/sendgrid/sendgrid-go"
)

// Public API.

const (
	ContentTypeTextHTML  = "text/html"
	ContentTypeTextPlain = "text/plain"
	ContentTypeMarkdown  = "text/markdown"
)

type (
	Participant struct {
		Name  string
		Email string
	}

	Content struct {
		Type string
		Data string
	}

	Parcel struct {
		From    Participant
		To      Participant
		Subject string
		Content []Content
	}

	Client interface {
		Send(context.Context, *Parcel) error
		SendMulti(context.Context, []*Parcel) error
	}
)

// Private API.

//
var (
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	cfg config
)

type (
	email struct {
		client *sendgrid.Client
	}

	config struct {
		Credentials struct {
			APIKey string `json:"apiKey"`
		} `yaml:"credentials"`
	}

	errorReply struct {
		Errors []struct {
			Field   interface{} `json:"field"`
			Help    interface{} `json:"help"`
			Message string      `json:"message"`
		} `json:"errors"`
	}
)
