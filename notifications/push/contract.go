// SPDX-License-Identifier: ice License 1.0

package push

import (
	"context"
	"io"
	"sync"
	stdlibtime "time"

	fcm "firebase.google.com/go/v4/messaging"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/time"
)

// Public API.

// .
var (
	ErrInvalidDeviceToken = errors.New("device token is invalid")
)

type (
	SubscriptionTopic                                    string
	DeviceToken                                          string
	Notification[TARGET SubscriptionTopic | DeviceToken] struct {
		Data     map[string]string `json:"data,omitempty"`
		Target   TARGET
		Title    string `json:"title,omitempty"`
		Body     string `json:"body,omitempty"`
		ImageURL string `json:"imageUrl,omitempty"`
	}
	DelayedNotification struct {
		*Notification[SubscriptionTopic]
		MinDelay uint `json:"minDelay"`
		MaxDelay uint `json:"maxDelay"`
	}
	Client interface {
		io.Closer

		Send(context.Context, *Notification[DeviceToken], chan<- error)
		Broadcast(context.Context, *Notification[SubscriptionTopic]) error
		BroadcastDelayed(context.Context, *DelayedNotification) error
	}
)

// Private API.

const (
	bufferSizeForEachProcessingGoroutine             = 10
	fcmSendAllBatchSize                              = 500
	requestDeadline                                  = 25 * stdlibtime.Second
	fcmSendAllBufferingDeadline                      = 1 * stdlibtime.Second
	fcmSendAllSlowProcessingMonitoringTickerDeadline = 3 * fcmSendAllBufferingDeadline

	dataOnlyTitle    = "title"
	dataOnlyBody     = "body"
	dataOnlyImageURL = "imageUrl"
	dataOnlyMinDelay = "minDelaySec"
	dataOnlyMaxDelay = "maxDelaySec"
	dataOnlyType     = "type"

	typeDelayedNotification = "delayed"
	priorityHigh            = "high"
)

type (
	push struct {
		client             *fcm.Client
		sink               *pushNotificationsCollectingSink
		applicationYAMLKey string
	}
	pushNotificationsCollectingSink struct {
		applicationYAMLKey        string
		client                    *fcm.Client
		bufferedNotificationsChan chan []*notification
		wg                        *sync.WaitGroup
		mx                        *sync.Mutex
		resetAt                   *time.Time
		bufferedNotifications     []*notification
		closed                    bool
	}
	notificationBatch struct {
		sink                     *pushNotificationsCollectingSink
		devicesWithInvalidTokens map[int]struct{}
		devicesNotified          map[int]struct{}
		devicesFailed            map[int]error
		notifications            []*notification
	}
	notification struct {
		notification *Notification[DeviceToken]
		responder    chan<- error
	}
	config struct {
		WintrPushNotifications struct {
			Credentials struct {
				FilePath    string `yaml:"filePath"`
				FileContent string `yaml:"fileContent"`
			} `yaml:"credentials" mapstructure:"credentials"`
			Concurrency int `yaml:"concurrency"`
		} `yaml:"wintr/notifications/push" mapstructure:"wintr/notifications/push"` //nolint:tagliatelle // Nope.
	}
)
