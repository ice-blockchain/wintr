// SPDX-License-Identifier: ice License 1.0

package telegram

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	stdlibtime "time"

	"github.com/stretchr/testify/require"

	appcfg "github.com/ice-blockchain/wintr/config"
)

const (
	testApplicationYAML = "self"
	testPreviewImageURL = "https://ice-staging.b-cdn.net/profile/default-profile-picture-1.png"
)

// .
var (
	//nolint:gochecknoglobals // Because its loaded once, at runtime.
	testChatID string
)

func TestMain(m *testing.M) {
	module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(testApplicationYAML, "-", "_"), "/", "_"))
	testChatID = os.Getenv(module + "_TELEGRAM_NOTIFICATIONS_TEST_CHAT_ID")
	if strings.TrimSpace(testChatID) == "" {
		testChatID = os.Getenv("TELEGRAM_NOTIFICATIONS_TEST_CHAT_ID")
	}

	os.Exit(m.Run())
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Success_WithPreviewAndButton(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:          testChatID,
		Text:            "<b>test message</b> <a href=\"https://ice.io\">check ice page</a>",
		PreviewImageURL: testPreviewImageURL,
		BotToken:        cfg.WintrTelegramNotifications.Credentials.BotToken,
		Buttons: []struct {
			Text string `json:"text,omitempty"`
			URL  string `json:"url,omitempty"`
		}{
			{
				Text: "test button",
				URL:  "https://ice.io",
			},
		},
	}
	require.NoError(t, client.Send(ctx, notif))
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Success_WithoutPreviewAndButton(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:   testChatID,
		Text:     "<b>test message</b>",
		BotToken: cfg.WintrTelegramNotifications.Credentials.BotToken,
	}
	require.NoError(t, client.Send(ctx, notif))
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Failure_NoBotInfo(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:          testChatID,
		Text:            "<b>test message</b>",
		PreviewImageURL: "https://ice-staging.b-cdn.net/profile/default-profile-picture-1.png",
		BotToken:        "",
	}
	require.ErrorIs(t, client.Send(ctx, notif), ErrTelegramNotificationChatNotFound)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Failure_TooLongMessage(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}

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
		BotToken: cfg.WintrTelegramNotifications.Credentials.BotToken,
	}
	require.ErrorIs(t, client.Send(ctx, notif), ErrTelegramNotificationBadRequest)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Failure_EmptyMessage(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:   testChatID,
		Text:     "",
		BotToken: cfg.WintrTelegramNotifications.Credentials.BotToken,
	}
	require.ErrorIs(t, client.Send(ctx, notif), ErrTelegramNotificationBadRequest)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSend_Failure_NoChatId(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" {
		t.Skip()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	notif := &Notification{
		ChatID:   "",
		Text:     "<b>test message</b>",
		BotToken: cfg.WintrTelegramNotifications.Credentials.BotToken,
	}
	require.ErrorIs(t, client.Send(ctx, notif), ErrTelegramNotificationBadRequest)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientSendConcurrency_Success_TooManyAttempts(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*stdlibtime.Second)
	defer cancel()

	client := New(testApplicationYAML)
	wg := new(sync.WaitGroup)
	const concurrency = 100
	wg.Add(concurrency)
	responder := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ { //nolint:intrange // .
		notif := &Notification{
			ChatID:   testChatID,
			Text:     fmt.Sprintf("<b>test message %v</b>", i+1),
			BotToken: cfg.WintrTelegramNotifications.Credentials.BotToken,
		}
		go func() {
			defer wg.Done()
			responder <- client.Send(ctx, notif)
		}()
	}
	wg.Wait()
	close(responder)
	for err := range responder {
		require.NoError(t, err)
	}
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientGetUpdates_Success(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	upd, err := client.GetUpdates(ctx, &GetUpdatesArg{
		BotToken:       cfg.WintrTelegramNotifications.Credentials.BotToken,
		AllowedUpdates: []string{"message"},
		Limit:          1,
		Offset:         0,
	})
	require.NoError(t, err)
	require.Empty(t, upd)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientGetUpdates_Success_WrongOffset(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	upd, err := client.GetUpdates(ctx, &GetUpdatesArg{
		BotToken:       cfg.WintrTelegramNotifications.Credentials.BotToken,
		AllowedUpdates: []string{"message"},
		Limit:          1,
		Offset:         11111111111111111,
	})
	require.NoError(t, err)
	require.Empty(t, upd)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientGetUpdates_Success_WrongAllowedUpdates(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	upd, err := client.GetUpdates(ctx, &GetUpdatesArg{
		BotToken:       cfg.WintrTelegramNotifications.Credentials.BotToken,
		AllowedUpdates: []string{"wrong"},
		Limit:          1,
		Offset:         0,
	})
	require.NoError(t, err)
	require.Empty(t, upd)
}

//nolint:paralleltest // Not to have unpredictable too many attempts error.
func TestClientGetUpdates_Failure_NoBotTokenProvided(t *testing.T) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		t.Skip()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	client := New(testApplicationYAML)
	upd, err := client.GetUpdates(ctx, &GetUpdatesArg{
		BotToken:       "",
		AllowedUpdates: []string{"message"},
		Limit:          1,
		Offset:         0,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTelegramNotificationChatNotFound)
	require.Empty(t, upd)
}

func BenchmarkBufferedClientSendWithPreview(b *testing.B) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		b.Skip()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	client := New(testApplicationYAML)
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			notif := &Notification{
				ChatID:          testChatID,
				Text:            "<b>test message</b>",
				PreviewImageURL: testPreviewImageURL,
				BotToken:        cfg.WintrTelegramNotifications.Credentials.BotToken,
			}
			require.NoError(b, client.Send(ctx, notif))
		}
	})
}

func BenchmarkBufferedClientSendNoPreview(b *testing.B) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		b.Skip()
	}

	client := New(testApplicationYAML)
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			notif := &Notification{
				ChatID:   testChatID,
				Text:     "<b>test message</b>",
				BotToken: cfg.WintrTelegramNotifications.Credentials.BotToken,
			}
			require.NoError(b, client.Send(ctx, notif))
		}
	})
}

func BenchmarkGetUpdates(b *testing.B) {
	var cfg config
	appcfg.MustLoadFromKey(testApplicationYAML, &cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	if cfg.WintrTelegramNotifications.Credentials.BotToken == "" || testChatID == "" {
		b.Skip()
	}
	client := New(testApplicationYAML)
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.GetUpdates(ctx, &GetUpdatesArg{
				BotToken:       cfg.WintrTelegramNotifications.Credentials.BotToken,
				AllowedUpdates: []string{"message"},
				Limit:          1,
				Offset:         0,
			})
			require.NoError(b, err)
		}
	})
}
