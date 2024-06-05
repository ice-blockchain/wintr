// SPDX-License-Identifier: ice License 1.0

package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/terror"
)

//nolint:funlen // .
func New(applicationYAMLKey string) Client {
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.Token == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrTelegramNotifications.Credentials.Token = os.Getenv(module + "_TELEGRAM_NOTIFICATIONS_CREDENTIALS_TOKEN")
	}
	if cfg.WintrTelegramNotifications.Credentials.Token == "" {
		cfg.WintrTelegramNotifications.Credentials.Token = os.Getenv("TELEGRAM_NOTIFICATIONS_CREDENTIALS_TOKEN")
	}
	if cfg.WintrTelegramNotifications.BaseURL == "" {
		cfg.WintrTelegramNotifications.BaseURL = os.Getenv("TELEGRAM_NOTIFICATIONS_BASE_URL")
		if cfg.WintrTelegramNotifications.BaseURL == "" {
			cfg.WintrTelegramNotifications.BaseURL = "https://api.telegram.org"
		}
		if strings.HasSuffix(strings.TrimSpace(cfg.WintrTelegramNotifications.BaseURL), "/") {
			cfg.WintrTelegramNotifications.BaseURL = cfg.WintrTelegramNotifications.BaseURL[:len(cfg.WintrTelegramNotifications.BaseURL)-1]
		}
	}
	if cfg.WintrTelegramNotifications.ParseMode == "" {
		cfg.WintrTelegramNotifications.ParseMode = os.Getenv("TELEGRAM_NOTIFICATIONS_PARSE_MODE")
		if cfg.WintrTelegramNotifications.ParseMode == "" {
			cfg.WintrTelegramNotifications.ParseMode = "HTML"
		}
	}
	cl := &telegramNotification{
		cfg: &cfg,
	}
	responder := make(chan error)
	cl.Send(context.Background(), &Notification{ChatID: "test", Text: "test", BotToken: cfg.WintrTelegramNotifications.Credentials.Token}, responder)
	err := <-responder
	if err != nil && !errors.Is(err, ErrTelegramNotificationBadRequest) {
		log.Panic(err)
	}
	close(responder)

	return cl
}

//nolint:funlen,gocognit,revive // .
func (t *telegramNotification) Send(ctx context.Context, notif *Notification, responseChan chan<- error) {
	if ctx.Err() != nil {
		if responseChan != nil {
			responseChan <- errors.Wrap(ctx.Err(), "context error")
		}

		return
	}
	go func(ctx context.Context) {
		sendMsgCtx, cancel := context.WithTimeout(ctx, requestDeadline)
		defer cancel()
		msg, err := t.buildTelegramMessage(notif)
		if err != nil {
			if responseChan != nil {
				responseChan <- errors.Wrapf(err, "can't build telegram message for:%#v", notif)
			}
		}
		resp, sErr := t.send(sendMsgCtx, notif.BotToken, msg)
		if sErr != nil {
			if responseChan != nil {
				responseChan <- errors.Wrapf(sErr, "can't call send() function for:%#v", notif)
			}

			return
		}
		var rErr error
		if !resp.Ok {
			switch resp.ErrorCode {
			case 400: //nolint:gomnd,mnd // .
				rErr = ErrTelegramNotificationBadRequest
			case 404: //nolint:gomnd,mnd // .
				rErr = ErrTelegramNotificationChatNotFound
			case 429: //nolint:gomnd,mnd // .
				rErr = terror.New(ErrTelegramNotificationTooManyAttempts, map[string]any{"retry_after": resp.Parameters.RetryAfter})
			default:
				rErr = ErrTelegramNotificationUnexpected
			}
		}
		if responseChan != nil {
			responseChan <- errors.Wrapf(rErr, "can't send telegram message:%#v, errorCode:%v, description:%v", notif, resp.ErrorCode, resp.Description)
		} else {
			log.Error(rErr)
		}
	}(ctx)
}

func (t *telegramNotification) buildTelegramMessage(notif *Notification) (jsonVal string, err error) {
	ts := telegramMessage{
		ChatID:    notif.ChatID,
		Text:      notif.Text,
		ParseMode: t.cfg.WintrTelegramNotifications.ParseMode,
		LinkPreviewOptions: struct {
			URL           string `json:"url" example:"https://ice-staging.b-cdn.net/profile/default-profile-picture-1.png"`
			IsDisabled    bool   `json:"is_disabled" example:"false"`    //nolint:tagliatelle // It's telegram API.
			ShowAboveText bool   `json:"show_above_text" example:"true"` //nolint:tagliatelle // It's telegram API.
		}{
			IsDisabled:    t.cfg.WintrTelegramNotifications.LinkPreviewOptions.IsDisabled,
			URL:           notif.PreviewImageURL,
			ShowAboveText: true,
		},
		DisableNotification: notif.DisableNotification,
	}
	val, err := json.Marshal(ts)
	if err != nil {
		return "", errors.Wrapf(err, "can't send telegram push notification:%#v", notif)
	}

	return string(val), err
}

//nolint:revive // .
func (t *telegramNotification) send(ctx context.Context, botToken, jsonVal string) (tResp *telegramAPIResponse, err error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "context error")
	}
	reader := strings.NewReader(jsonVal)
	apiURL := fmt.Sprintf("%v/bot%v/sendMessage", t.cfg.WintrTelegramNotifications.BaseURL, botToken)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, reader)
	if err != nil {
		return nil, errors.Wrap(err, "can't send telegram push notification")
	}
	request.Header.Set("Content-Type", contentType)
	resp, err := new(http.Client).Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "can't send request")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "can't read body")
	}
	var response telegramAPIResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, errors.Wrapf(err, "can't parse telegram body:%v", string(body))
	}

	return &response, nil
}
