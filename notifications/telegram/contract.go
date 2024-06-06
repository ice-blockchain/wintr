// SPDX-License-Identifier: ice License 1.0

package telegram

import (
	"context"
	stdlibtime "time"

	"github.com/pkg/errors"
)

// Public API.

type (
	Client interface {
		Send(ctx context.Context, notif *Notification) error
	}
	Notification struct {
		ChatID              string `json:"chatId,omitempty"`
		Text                string `json:"text,omitempty"`
		PreviewImageURL     string `json:"previewImageUrl,omitempty"`
		BotToken            string `json:"botToken,omitempty"`
		DisableNotification bool   `json:"disableNotification,omitempty"`
	}
)

// Private API.

const (
	requestDeadline = 25 * stdlibtime.Second
)

var (
	ErrTelegramNotificationChatNotFound = errors.New("chat not found")
	ErrTelegramNotificationBadRequest   = errors.New("bad request")
	ErrTelegramNotificationUnexpected   = errors.New("unexpected")
)

type (
	telegramNotification struct {
		cfg *config
	}
	telegramMessage struct {
		ChatID             string `json:"chat_id" example:"111"` //nolint:tagliatelle // It's telegram API.
		Text               string `json:"text" example:"hello world"`
		ParseMode          string `json:"parse_mode" example:"HTML"` //nolint:tagliatelle // It's telegram API.
		LinkPreviewOptions struct {
			URL           string `json:"url" example:"https://ice-staging.b-cdn.net/profile/default-profile-picture-1.png"`
			IsDisabled    bool   `json:"is_disabled" example:"false"`    //nolint:tagliatelle // It's telegram API.
			ShowAboveText bool   `json:"show_above_text" example:"true"` //nolint:tagliatelle // It's telegram API.
		} `json:"link_preview_options"` //nolint:tagliatelle // It's telegram API.
		DisableNotification bool `json:"disable_notification" example:"true"` //nolint:tagliatelle // It's telegram API.
	}
	config struct {
		WintrTelegramNotifications struct {
			Credentials struct {
				BotToken string `yaml:"botToken"`
			} `yaml:"credentials" mapstructure:"credentials"`
			ParseMode          string `yaml:"parseMode"`
			BaseURL            string `yaml:"baseUrl"`
			LinkPreviewOptions struct {
				IsDisabled bool `yaml:"isDisabled"`
			} `yaml:"linkPreviewOptions"`
		} `yaml:"wintr/notifications/telegram" mapstructure:"wintr/notifications/telegram"` //nolint:tagliatelle // Nope.
	}
)
