// SPDX-License-Identifier: ice License 1.0

package telegram

import (
	"context"
	stdlibtime "time"

	"github.com/pkg/errors"
)

// Public API.

const (
	BotCommandType = "bot_command"
)

type (
	Client interface {
		Send(ctx context.Context, notif *Notification) error
		GetUpdates(ctx context.Context, arg *GetUpdatesArg) (updates []*Update, err error)
	}
	Button struct {
		Text         string `json:"text,omitempty"`
		URL          string `json:"url,omitempty"`
		CallbackData string `json:"callbackData,omitempty"`
	}
	Notification struct {
		ChatID              string   `json:"chatId,omitempty"`
		Text                string   `json:"text,omitempty"`
		PreviewImageURL     string   `json:"previewImageUrl,omitempty"`
		BotToken            string   `json:"botToken,omitempty"`
		Buttons             []Button `json:"buttons,omitempty"`
		ReplyMessageID      int64    `json:"replyMessageId,omitempty"`
		DisableNotification bool     `json:"disableNotification,omitempty"`
	}
	GetUpdatesArg struct {
		BotToken       string   `json:"botToken,omitempty"`
		AllowedUpdates []string `json:"allowedUpdates,omitempty"`
		Limit          int64    `json:"limit,omitempty"`
		Offset         int64    `json:"offset,omitempty"`
	}
	Message struct {
		Entities []struct {
			Type string `json:"type,omitempty"`
		} `json:"entities,omitempty"`
		Text string `json:"text,omitempty"`
		From struct {
			LanguageCode string `json:"language_code,omitempty"` //nolint:tagliatelle // It's telegram API.
			Username     string `json:"username,omitempty"`
			ID           int64  `json:"id,omitempty"`
			IsBot        bool   `json:"is_bot,omitempty"` //nolint:tagliatelle // It's telegram API.
		} `json:"from,omitempty"`
		MessageID int64 `json:"message_id,omitempty"` //nolint:tagliatelle // It's telegram API.
		Date      int64 `json:"date,omitempty"`
	}
	CallbackQuery struct {
		Data     string `json:"data,omitempty"`
		ID       string `json:"id,omitempty"`
		Entities []struct {
			Type string `json:"type,omitempty"`
		} `json:"entities,omitempty"`
		Text string `json:"text,omitempty"`
		From struct {
			Username     string `json:"username,omitempty"`
			LanguageCode string `json:"language_code,omitempty"` //nolint:tagliatelle // It's telegram API.
			ID           int64  `json:"id,omitempty"`
			IsBot        bool   `json:"is_bot,omitempty"` //nolint:tagliatelle // It's telegram API.
		} `json:"from,omitempty"`
	}
	Update struct {
		Message       *Message       `json:"message,omitempty"`
		CallbackQuery *CallbackQuery `json:"callback_query,omitempty"` //nolint:tagliatelle // It's telegram API.
		UpdateID      int64          `json:"update_id,omitempty"`      //nolint:tagliatelle // It's telegram API.
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
		ChatID             string `json:"chat_id,omitempty" example:"111"` //nolint:tagliatelle // It's telegram API.
		Text               string `json:"text,omitempty" example:"hello world"`
		ParseMode          string `json:"parse_mode,omitempty" example:"HTML"` //nolint:tagliatelle // It's telegram API.
		LinkPreviewOptions struct {
			URL           string `json:"url,omitempty" example:"https://ice-staging.b-cdn.net/profile/default-profile-picture-1.png"`
			IsDisabled    bool   `json:"is_disabled,omitempty" example:"false"`    //nolint:tagliatelle // It's telegram API.
			ShowAboveText bool   `json:"show_above_text,omitempty" example:"true"` //nolint:tagliatelle // It's telegram API.
		} `json:"link_preview_options"` //nolint:tagliatelle // It's telegram API.
		ReplyMarkup struct {
			InlineKeyboard [][]struct {
				Text         string `json:"text" example:"some text"`
				URL          string `json:"url,omitempty" example:"https://ice.io"`
				CallbackData string `json:"callback_data,omitempty" example:"1"` //nolint:tagliatelle // It's telegram API.
			} `json:"inline_keyboard,omitempty"` //nolint:tagliatelle // It's telegram API.
		} `json:"reply_markup,omitempty"` //nolint:tagliatelle // It's telegram API.
		ReplyParameters struct {
			MessageID int64 `json:"message_id,omitempty"` //nolint:tagliatelle // It's telegram API.
		} `json:"reply_parameters"` //nolint:tagliatelle // It's telegram API.
		DisableNotification bool `json:"disable_notification,omitempty" example:"true"` //nolint:tagliatelle // It's telegram API.
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
