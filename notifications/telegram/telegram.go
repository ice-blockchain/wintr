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

//nolint:funlen,gocognit,revive // .
func New(applicationYAMLKey string) Client {
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if strings.TrimSpace(cfg.WintrTelegramNotifications.Credentials.BotToken) == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrTelegramNotifications.Credentials.BotToken = os.Getenv(module + "_TELEGRAM_NOTIFICATIONS_CREDENTIALS_BOT_TOKEN")
		if strings.TrimSpace(cfg.WintrTelegramNotifications.Credentials.BotToken) == "" {
			cfg.WintrTelegramNotifications.Credentials.BotToken = os.Getenv("TELEGRAM_NOTIFICATIONS_CREDENTIALS_BOT_TOKEN")
		}
	}
	if strings.TrimSpace(cfg.WintrTelegramNotifications.BaseURL) == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrTelegramNotifications.BaseURL = os.Getenv(module + "_TELEGRAM_NOTIFICATIONS_BASE_URL")
		if strings.TrimSpace(cfg.WintrTelegramNotifications.BaseURL) == "" {
			cfg.WintrTelegramNotifications.BaseURL = os.Getenv("TELEGRAM_NOTIFICATIONS_BASE_URL")
			if strings.TrimSpace(cfg.WintrTelegramNotifications.BaseURL) == "" {
				cfg.WintrTelegramNotifications.BaseURL = "https://api.telegram.org"
			}
		}
		if strings.HasSuffix(strings.TrimSpace(cfg.WintrTelegramNotifications.BaseURL), "/") {
			cfg.WintrTelegramNotifications.BaseURL = cfg.WintrTelegramNotifications.BaseURL[:len(cfg.WintrTelegramNotifications.BaseURL)-1]
		}
	}
	if strings.TrimSpace(cfg.WintrTelegramNotifications.ParseMode) == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrTelegramNotifications.ParseMode = os.Getenv(module + "_TELEGRAM_NOTIFICATIONS_PARSE_MODE")
		if strings.TrimSpace(cfg.WintrTelegramNotifications.ParseMode) == "" {
			cfg.WintrTelegramNotifications.ParseMode = os.Getenv("TELEGRAM_NOTIFICATIONS_PARSE_MODE")
			if strings.TrimSpace(cfg.WintrTelegramNotifications.ParseMode) == "" {
				cfg.WintrTelegramNotifications.ParseMode = "HTML"
			}
		}
	}
	cl := &telegramNotification{
		cfg: &cfg,
	}
	if err := cl.Send(
		context.Background(), &Notification{ChatID: "test", Text: "test", BotToken: cfg.WintrTelegramNotifications.Credentials.BotToken},
	); err != nil && !errors.Is(err, ErrTelegramNotificationChatNotFound) {
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
	if _, sErr := t.post(ctx, url, msg); sErr != nil {
		return errors.Wrapf(sErr, "can't call send() function for:%#v", notif)
	}

	return nil
}

func (t *telegramNotification) GetUpdates(ctx context.Context, arg *GetUpdatesArg) (updates []*Update, err error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "context error")
	}
	ts := getUpdatesMessage{
		AllowedUpdates: arg.AllowedUpdates,
		Limit:          arg.Limit,
		Offset:         arg.Offset,
	}
	val, err := json.Marshal(ts)
	if err != nil {
		return nil, errors.Wrapf(err, "can't send telegram push notification:%#v", arg)
	}
	url := fmt.Sprintf("%v/bot%v/getUpdates", t.cfg.WintrTelegramNotifications.BaseURL, arg.BotToken)
	resp, sErr := t.post(ctx, url, string(val))
	if sErr != nil {
		return nil, errors.Wrapf(sErr, "can't call getUpdates function for:%#v", arg)
	}
	var parsed struct {
		Result []*Update `json:"result"`
		OK     bool      `json:"ok"`
	}
	if err = json.Unmarshal([]byte(resp), &parsed); err != nil {
		return nil, errors.Wrapf(err, "unmarshalling response for telegram failed, response: %v", resp)
	}

	return parsed.Result, nil
}

//nolint:funlen // .
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
	if len(notif.Buttons) > 0 {
		for ix := range notif.Buttons {
			ts.ReplyMarkup.InlineKeyboard = append(ts.ReplyMarkup.InlineKeyboard, []struct {
				Text string `json:"text" example:"some text"`
				URL  string `json:"url" example:"https://ice.io"`
			}{
				{
					Text: notif.Buttons[ix].Text,
					URL:  notif.Buttons[ix].URL,
				},
			})
		}
	}
	if notif.ReplyMessageID > 0 {
		ts.ReplyParameters.MessageID = notif.ReplyMessageID
	}
	val, err := json.Marshal(ts)
	if err != nil {
		return "", errors.Wrapf(err, "can't send telegram push notification:%#v", notif)
	}

	return string(val), err
}

//nolint:funlen // .
func (t *telegramNotification) post(ctx context.Context, url, body string) (response string, err error) {
	newReq := t.buildHTTPRequest(ctx)
	newReq = newReq.SetBodyJsonString(body)
	resp, err := newReq.Post(url)
	if err != nil || resp.IsErrorState() {
		if err == nil {
			respBody, pErr := resp.ToString()
			if pErr != nil {
				return "", errors.Wrapf(pErr, "notifications/telegram post `%v` failed, body:%#v, [1]unable to read response body", url, body)
			}
			var rErr error
			switch resp.GetStatusCode() {
			case 400: //nolint:gomnd,mnd // .
				rErr = ErrTelegramNotificationBadRequest
			case 403: //nolint:gomnd,mnd // .
				rErr = ErrTelegramNotificationForbidden
			case 404: //nolint:gomnd,mnd // .
				rErr = ErrTelegramNotificationChatNotFound
			case 409: //nolint:gomnd,mnd // .
				rErr = ErrTelegramBotConflict
			default:
				return "", errors.Wrapf(rErr, "notifications/telegram post `%v` failed, unexpected error, body:%#v, response: %v", url, body, respBody)
			}

			return "", errors.Wrapf(rErr, "notifications/telegram post `%v` failed, body:%#v, response: %v", url, body, respBody)
		}

		return "", errors.Wrapf(err, "notifications/telegram post `%v` failed, body:%#v", url, body)
	}
	if response, err = resp.ToString(); err != nil {
		return "", errors.Wrapf(err, "notifications/telegram post `%v` failed, body:%#v, [2]unable to read response body", url, body)
	}

	return response, nil
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
			return (err != nil ||
				(resp.IsErrorState() && resp.GetStatusCode() != http.StatusBadRequest &&
					resp.GetStatusCode() != http.StatusForbidden && resp.GetStatusCode() != http.StatusNotFound))
		}).
		SetContentType("application/json").
		SetHeader("Accept", "application/json")
}
