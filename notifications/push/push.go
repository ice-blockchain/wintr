// SPDX-License-Identifier: ice License 1.0

package push

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	stdlibtime "time"

	firebase "firebase.google.com/go/v4"
	fcm "firebase.google.com/go/v4/messaging"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	firebaseoption "google.golang.org/api/option"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(applicationYAMLKey string) Client { //nolint:funlen,gocognit,revive // .
	var cfg config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if cfg.WintrPushNotifications.Credentials.FileContent == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrPushNotifications.Credentials.FileContent = os.Getenv(fmt.Sprintf("%s_PUSH_NOTIFICATIONS_CREDENTIALS_FILE_CONTENT", module))
		if cfg.WintrPushNotifications.Credentials.FileContent == "" {
			cfg.WintrPushNotifications.Credentials.FileContent = os.Getenv("PUSH_NOTIFICATIONS_CREDENTIALS_FILE_CONTENT")
		}
		if cfg.WintrPushNotifications.Credentials.FileContent == "" {
			cfg.WintrPushNotifications.Credentials.FileContent = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if !strings.HasPrefix(strings.TrimSpace(cfg.WintrPushNotifications.Credentials.FileContent), "{") {
				cfg.WintrPushNotifications.Credentials.FileContent = ""
			}
		}
	}
	if cfg.WintrPushNotifications.Credentials.FilePath == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrPushNotifications.Credentials.FilePath = os.Getenv(fmt.Sprintf("%s_PUSH_NOTIFICATIONS_CREDENTIALS_FILE_PATH", module))
		if cfg.WintrPushNotifications.Credentials.FilePath == "" {
			cfg.WintrPushNotifications.Credentials.FilePath = os.Getenv("PUSH_NOTIFICATIONS_CREDENTIALS_FILE_PATH")
		}
		if cfg.WintrPushNotifications.Credentials.FilePath == "" {
			cfg.WintrPushNotifications.Credentials.FilePath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if strings.HasPrefix(strings.TrimSpace(cfg.WintrPushNotifications.Credentials.FilePath), "{") {
				cfg.WintrPushNotifications.Credentials.FilePath = ""
			}
		}
	}
	if cfg.WintrPushNotifications.Concurrency == 0 {
		cfg.WintrPushNotifications.Concurrency = (1 + 1) * runtime.GOMAXPROCS(-1)
	}
	var credentialsOption firebaseoption.ClientOption
	if cfg.WintrPushNotifications.Credentials.FileContent != "" {
		credentialsOption = firebaseoption.WithCredentialsJSON([]byte(cfg.WintrPushNotifications.Credentials.FileContent))
	}
	if cfg.WintrPushNotifications.Credentials.FilePath != "" {
		credentialsOption = firebaseoption.WithCredentialsFile(cfg.WintrPushNotifications.Credentials.FilePath)
	}
	firebaseApp, err := firebase.NewApp(context.Background(), nil, credentialsOption)
	log.Panic(errors.Wrapf(err, "[%v] failed to build firebase app ", applicationYAMLKey)) //nolint:revive // That's intended.
	fcmClient, err := firebaseApp.Messaging(context.Background())
	log.Panic(errors.Wrapf(err, "[%v] failed to build FCM messaging.Client", applicationYAMLKey))

	return newClient(applicationYAMLKey, fcmClient, &cfg)
}

func newClient(applicationYAMLKey string, fcmClient *fcm.Client, cfg *config) Client {
	cl := &push{
		applicationYAMLKey: applicationYAMLKey,
		client:             fcmClient,
		sink: &pushNotificationsCollectingSink{
			applicationYAMLKey:        applicationYAMLKey,
			client:                    fcmClient,
			bufferedNotificationsChan: make(chan []*notification, cfg.WintrPushNotifications.Concurrency*bufferSizeForEachProcessingGoroutine),
			wg:                        new(sync.WaitGroup),
			mx:                        new(sync.Mutex),
		},
	}
	for i := 0; i < cfg.WintrPushNotifications.Concurrency; i++ {
		go cl.sink.startProcessing()
	}
	go cl.sink.monitorSlowProcessing()

	responder := make(chan error)
	cl.Send(context.Background(), &Notification[DeviceToken]{
		Target: DeviceToken(uuid.NewString()),
		Title:  "probing bootstrap",
		Body:   "probing bootstrap",
	}, responder)
	if err := <-responder; err == nil || !errors.Is(err, ErrInvalidDeviceToken) {
		if err == nil {
			log.Panic(errors.New("unexpected success"))
		}
		log.Panic(err)
	}

	return cl
}

func (p *push) Close() error {
	log.Info("push, started shutdown", "package", p.applicationYAMLKey)
	defer log.Info("push, finished shutdown", "package", p.applicationYAMLKey)
	p.sink.stop()

	return nil
}

func (p *push) Send(ctx context.Context, notif *Notification[DeviceToken], responseChan chan<- error) {
	if ctx.Err() != nil {
		if responseChan != nil {
			responseChan <- errors.Wrap(ctx.Err(), "context error")
		}

		return
	}
	p.sink.accept(notif, responseChan) //nolint:contextcheck // Don't need it, for now.
}

func (p *push) Broadcast(ctx context.Context, notification *Notification[SubscriptionTopic]) error {
	return errors.Wrapf(retry(ctx, func() error {
		_, err := p.client.Send(ctx, &fcm.Message{
			Data: notification.Data,
			Notification: &fcm.Notification{
				Title:    notification.Title,
				Body:     notification.Body,
				ImageURL: notification.ImageURL,
			},
			Topic: string(notification.Target),
		})

		return err //nolint:wrapcheck // No need to do that, it's wrapped outside.
	}), "[%v] permanently failed to broadcast %#v", p.applicationYAMLKey, notification)
}

func retry(ctx context.Context, op func() error) error {
	//nolint:wrapcheck // No need, its just a proxy.
	return backoff.RetryNotify(
		op,
		//nolint:gomnd // Because those are static configs.
		backoff.WithContext(&backoff.ExponentialBackOff{
			InitialInterval:     300 * stdlibtime.Millisecond,
			RandomizationFactor: 0.5,
			Multiplier:          2.5,
			MaxInterval:         2 * stdlibtime.Second,
			MaxElapsedTime:      requestDeadline,
			Stop:                backoff.Stop,
			Clock:               backoff.SystemClock,
		}, ctx),
		func(e error, next stdlibtime.Duration) {
			log.Error(errors.Wrapf(e, "wintr/notifications/push call failed. retrying in %v... ", next))
		})
}
