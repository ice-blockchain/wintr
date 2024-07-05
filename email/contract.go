// SPDX-License-Identifier: ice License 1.0

package email

import (
	"context"
	stdlibtime "time"

	"github.com/pkg/errors"
	"github.com/sendgrid/sendgrid-go"

	"github.com/ice-blockchain/wintr/time"
)

// Public API.

const (
	MaxBatchSize = 1000
)

const (
	TextHTML     ContentType = "text/html"
	TextPlain    ContentType = "text/plain"
	TextMarkdown ContentType = "text/markdown"
)

type (
	ContentType string

	Participant struct {
		SendAt             *time.Time
		SubstitutionFields map[string]string
		Name               string
		Email              string
	}

	Body struct {
		Type ContentType
		Data string
	}

	Parcel struct {
		From    Participant
		Body    *Body
		Subject string
	}

	Client interface {
		Send(ctx context.Context, parcel *Parcel, participants ...Participant) error
	}
)

// Private API.

const (
	requestDeadline = 25 * stdlibtime.Second
)

// .
var (
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	cfg            Config
	errPleaseRetry = errors.New("please retry")
)

type (
	email struct {
		client *sendgrid.Client
	}

	Config struct {
		WintrEmail struct {
			Credentials struct {
				APIKey string `yaml:"apiKey" mapstructure:"apiKey"`
			} `yaml:"credentials" mapstructure:"credentials"`
		} `yaml:"wintr/email" mapstructure:"wintr/email"` //nolint:tagliatelle // Nope.
	}
)
