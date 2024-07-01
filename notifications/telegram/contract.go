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
		GetUpdates(ctx context.Context, arg *GetUpdatesArg) (updates []*Update, err error)
	}
	Notification struct {
		ChatID          string `json:"chatId,omitempty"`
		Text            string `json:"text,omitempty"`
		PreviewImageURL string `json:"previewImageUrl,omitempty"`
		BotToken        string `json:"botToken,omitempty"`
		Buttons         []struct {
			Text string `json:"text,omitempty"`
			URL  string `json:"url,omitempty"`
		}
		ReplyMessageID      int64 `json:"replyMessageId,omitempty"`
		DisableNotification bool  `json:"disableNotification,omitempty"`
	}
	GetUpdatesArg struct {
		BotToken       string   `json:"botToken,omitempty"`
		AllowedUpdates []string `json:"allowedUpdates,omitempty"`
		Limit          int64    `json:"limit,omitempty"`
		Offset         int64    `json:"offset,omitempty"`
	}
	Update struct {
		Message struct {
			Text string `json:"text,omitempty"`
			From struct {
				Username string `json:"username,omitempty"`
				ID       int64  `json:"id,omitempty"`
				IsBot    bool   `json:"is_bot,omitempty"` //nolint:tagliatelle // It's telegram API.
			} `json:"from,omitempty"`
			MessageID int64 `json:"message_id,omitempty"` //nolint:tagliatelle // It's telegram API.
			Date      int64 `json:"date,omitempty"`
		} `json:"message,omitempty"`
		UpdateID int64 `json:"update_id,omitempty"` //nolint:tagliatelle // It's telegram API.
	}
)

// Private API.

const (
	requestDeadline = 25 * stdlibtime.Second
)

var (
	ErrTelegramNotificationChatNotFound = errors.New("chat not found")
	ErrTelegramNotificationBadRequest   = errors.New("bad request")
	ErrTelegramNotificationForbidden    = errors.New("forbidden")
	ErrTelegramBotConflict              = errors.New("conflict")
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
		ReplyMarkup struct {
			InlineKeyboard [][]struct {
				Text string `json:"text" example:"some text"`
				URL  string `json:"url" example:"https://ice.io"`
			} `json:"inline_keyboard,omitempty"` //nolint:tagliatelle // It's telegram API.
		} `json:"reply_markup,omitempty"` //nolint:tagliatelle // It's telegram API.
		ReplyParameters struct {
			MessageID int64 `json:"message_id"` //nolint:tagliatelle // It's telegram API.
		} `json:"reply_parameters"` //nolint:tagliatelle // It's telegram API.
		DisableNotification bool `json:"disable_notification" example:"true"` //nolint:tagliatelle // It's telegram API.
	}
	getUpdatesMessage struct {
		AllowedUpdates []string `json:"allowed_updates,omitempty"` //nolint:tagliatelle // It's telegram API.
		Limit          int64    `json:"limit,omitempty"`
		Offset         int64    `json:"offset,omitempty"`
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
