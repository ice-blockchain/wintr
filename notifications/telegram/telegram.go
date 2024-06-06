// SPDX-License-Identifier: ice License 1.0

package telegram

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	stdlibtime "time"

	"github.com/goccy/go-json"
	"github.com/imroc/req/v3"
	"github.com/pkg/errors"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func init() { //nolint:gochecknoinits // It's the only way to tweak the client.
	req.DefaultClient().SetJsonMarshal(json.Marshal)
	req.DefaultClient().SetJsonUnmarshal(json.Unmarshal)
	req.DefaultClient().GetClient().Timeout = requestDeadline
}

//nolint:funlen // .
func New(applicationYAMLKey string) Client {
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrTelegramNotifications.Credentials.BotToken = os.Getenv(module + "_TELEGRAM_NOTIFICATIONS_CREDENTIALS_BOT_TOKEN")
	}
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" {
		cfg.WintrTelegramNotifications.Credentials.BotToken = os.Getenv("TELEGRAM_NOTIFICATIONS_CREDENTIALS_BOT_TOKEN")
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
	if err := cl.Send(
		context.Background(), &Notification{ChatID: "test", Text: "test", BotToken: cfg.WintrTelegramNotifications.Credentials.BotToken},
	); err != nil && !errors.Is(err, ErrTelegramNotificationBadRequest) {
		log.Panic(err)
	}

	return cl
}

func (t *telegramNotification) Send(ctx context.Context, notif *Notification) error {
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context error")
	}
	msg, err := t.buildTelegramMessage(notif)
	if err != nil {
		return errors.Wrapf(err, "can't build telegram message for:%#v", notif)
	}
	url := fmt.Sprintf("%v/bot%v/sendMessage", t.cfg.WintrTelegramNotifications.BaseURL, notif.BotToken)
	if sErr := t.post(ctx, url, msg); sErr != nil {
		return errors.Wrapf(sErr, "can't call send() function for:%#v", notif)
	}

	return nil
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

func (t *telegramNotification) post(ctx context.Context, url, body string) error {
	newReq := t.buildHTTPRequest(ctx)
	newReq = newReq.SetBodyJsonString(body)
	resp, err := newReq.Post(url)
	if err != nil || resp.IsErrorState() {
		if err == nil {
			respBody, pErr := resp.ToString()
			if pErr != nil {
				return errors.Wrapf(pErr, "notifications/telegram post `%v` failed, body:%#v, [1]unable to read response body", url, body)
			}
			var rErr error
			switch resp.GetStatusCode() {
			case 400: //nolint:gomnd,mnd // .
				rErr = ErrTelegramNotificationBadRequest
			case 404: //nolint:gomnd,mnd // .
				rErr = ErrTelegramNotificationChatNotFound
			default:
				rErr = ErrTelegramNotificationUnexpected
			}

			return errors.Wrapf(rErr, "notifications/telegram post `%v` failed, body:%#v, response: %v", url, body, respBody)
		}

		return errors.Wrapf(err, "notifications/telegram post `%v` failed, body:%#v", url, body)
	}
	if _, err = resp.ToString(); err != nil {
		return errors.Wrapf(err, "notifications/telegram post `%v` failed, body:%#v, [2]unable to read response body", url, body)
	}

	return nil
}

//nolint:mnd,gomnd // Static config.
func (*telegramNotification) buildHTTPRequest(ctx context.Context) *req.Request {
	return req.
		SetContext(ctx).
		SetRetryBackoffInterval(10*stdlibtime.Millisecond, 1*stdlibtime.Second). //nolint:mnd,gomnd // .
		SetRetryHook(func(resp *req.Response, err error) {
			switch {
			case err != nil:
				log.Error(errors.Wrapf(err, "failed to send telegram notification, retrying... "))
			case resp.GetStatusCode() == http.StatusTooManyRequests:
				log.Error(errors.New("rate limit for telegram notification reached, retrying... "))
			case resp.GetStatusCode() >= http.StatusInternalServerError:
				log.Error(errors.New("failed to send telegram notification[internal server error], retrying... "))
			}
		}).
		SetRetryCount(25).
		SetRetryCondition(func(resp *req.Response, err error) bool {
			return (err != nil && (resp.GetStatusCode() == http.StatusTooManyRequests || resp.GetStatusCode() >= http.StatusInternalServerError) ||
				resp.IsErrorState() && resp.GetStatusCode() == http.StatusTooManyRequests)
		}).
		SetContentType("application/json").
		SetHeader("Accept", "application/json")
}
