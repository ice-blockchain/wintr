// SPDX-License-Identifier: ice License 1.0

package telegram

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	stdlibtime "time"

	"github.com/stretchr/testify/require"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/terror"
)

const (
	testApplicationYAML = "self"
	testPreviewImageURL = "https://ice-staging.b-cdn.net/profile/default-profile-picture-1.png"
)

// .
var (
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	testChatID = os.Getenv("TELEGRAM_NOTIFICATIONS_TEST_CHAT_ID")
)

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Success_WithPreview(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:          testChatID,
		Text:            "<b>test message</b>",
		PreviewImageURL: testPreviewImageURL,
		BotToken:        cfg.WintrTelegramNotifications.Credentials.Token,
	}
	responder := make(chan error)
	client.Send(ctx, notif, responder)
	require.NoError(t, <-responder)
	close(responder)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Success_WithoutPreview(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:   testChatID,
		Text:     "<b>test message</b>",
		BotToken: cfg.WintrTelegramNotifications.Credentials.Token,
	}
	responder := make(chan error)
	client.Send(ctx, notif, responder)
	require.NoError(t, <-responder)
	close(responder)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Failure_NoBotInfo(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:          testChatID,
		Text:            "<b>test message</b>",
		PreviewImageURL: "https://ice-staging.b-cdn.net/profile/default-profile-picture-1.png",
		BotToken:        "",
	}
	responder := make(chan error)
	client.Send(ctx, notif, responder)
	require.ErrorIs(t, <-responder, ErrTelegramNotificationChatNotFound)
	close(responder)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Failure_TooLongMessage(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	var longText string
	for range 4097 {
		longText += "a"
	}
	notif := &Notification{
		ChatID:   testChatID,
		Text:     longText,
		BotToken: cfg.WintrTelegramNotifications.Credentials.Token,
	}
	responder := make(chan error)
	client.Send(ctx, notif, responder)
	require.ErrorIs(t, <-responder, ErrTelegramNotificationBadRequest)
	close(responder)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Failure_EmptyMessage(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:   testChatID,
		Text:     "",
		BotToken: cfg.WintrTelegramNotifications.Credentials.Token,
	}
	responder := make(chan error)
	client.Send(ctx, notif, responder)
	require.ErrorIs(t, <-responder, ErrTelegramNotificationBadRequest)
	close(responder)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Failure_NoChatId(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:   "",
		Text:     "<b>test message</b>",
		BotToken: cfg.WintrTelegramNotifications.Credentials.Token,
	}
	responder := make(chan error)
	client.Send(ctx, notif, responder)
	require.ErrorIs(t, <-responder, ErrTelegramNotificationBadRequest)
	close(responder)
}

//nolint:funlen,paralleltest // Not to have unpredictable too many attempts error.
func TestClientSendConcurrency_Failure_TooManyAttempts(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)

	client := New(testApplicationYAML)
	wg := new(sync.WaitGroup)
	const concurrency = 100
	wg.Add(concurrency)
	responder := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ { //nolint:intrange // .
		notif := &Notification{
			ChatID:   testChatID,
			Text:     fmt.Sprintf("<b>test message %v</b>", i+1),
			BotToken: cfg.WintrTelegramNotifications.Credentials.Token,
		}
		go func() {
			defer wg.Done()
			innerResponder := make(chan error)
			client.Send(context.Background(), notif, innerResponder)
			responder <- <-innerResponder
			close(innerResponder)
		}()
	}
	wg.Wait()
	close(responder)

	okCounter, tooManyRequestCounter := 0, 0
	for err := range responder {
		if err != nil {
			if errors.Is(err, ErrTelegramNotificationTooManyAttempts) {
				tooManyRequestCounter++
				if tErr := terror.As(err); tErr != nil {
					require.Greater(t, tErr.Data["retry_after"], int64(0))
				}
			}
		} else {
			okCounter++
		}
	}
	require.Greater(t, okCounter, 30)
	require.Greater(t, tooManyRequestCounter, 0)
}
