// SPDX-License-Identifier: ice License 1.0

package inapp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	stdlibtime "time"

	getstreamio "github.com/GetStream/stream-go2/v7"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/notifications/inapp/internal"
	"github.com/ice-blockchain/wintr/time"
)

func New(applicationYAMLKey, feedName string) Client { //nolint:funlen // .
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	if cfg.WintrInAppNotifications.Credentials.Key == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrInAppNotifications.Credentials.Key = os.Getenv(module + "_INAPP_NOTIFICATIONS_CLIENT_KEY")
		if cfg.WintrInAppNotifications.Credentials.Key == "" {
			cfg.WintrInAppNotifications.Credentials.Key = os.Getenv("INAPP_NOTIFICATIONS_CLIENT_KEY")
		}
	}
	if cfg.WintrInAppNotifications.Credentials.Secret == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrInAppNotifications.Credentials.Secret = os.Getenv(module + "_INAPP_NOTIFICATIONS_CLIENT_SECRET")
		if cfg.WintrInAppNotifications.Credentials.Secret == "" {
			cfg.WintrInAppNotifications.Credentials.Secret = os.Getenv("INAPP_NOTIFICATIONS_CLIENT_SECRET")
		}
	}
	if cfg.WintrInAppNotifications.Credentials.AppID == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrInAppNotifications.Credentials.AppID = os.Getenv(module + "_INAPP_NOTIFICATIONS_CLIENT_APP_ID")
		if cfg.WintrInAppNotifications.Credentials.AppID == "" {
			cfg.WintrInAppNotifications.Credentials.AppID = os.Getenv("INAPP_NOTIFICATIONS_CLIENT_APP_ID")
		}
	}

	cl, err := getstreamio.New(
		cfg.WintrInAppNotifications.Credentials.Key,
		cfg.WintrInAppNotifications.Credentials.Secret,
		getstreamio.WithTimeout(requestDeadline))
	log.Panic(errors.Wrapf(err, "unable to init GetStream client")) //nolint:revive // It's intended.

	inAppClient := &inApp{
		client:   cl,
		cfg:      &cfg,
		feedName: feedName,
	}
	log.Panic(inAppClient.Send(context.Background(), &Parcel{
		Action: "probing_bootstrap",
		Actor: internal.ID{
			Type:  "system",
			Value: "wintr",
		},
		Subject: internal.ID{
			Type:  "system",
			Value: "wintr",
		},
	}, uuid.NewString()))

	return inAppClient
}

func (i *inApp) CreateUserToken(ctx context.Context, userID UserID) (*Token, error) {
	if ctx.Err() != nil {
		return nil, errors.Wrap(ctx.Err(), "context error")
	}
	token, err := i.client.CreateUserToken(userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user token")
	}

	return &Token{
		APIKey:    i.cfg.WintrInAppNotifications.Credentials.Key,
		APISecret: token,
		AppID:     i.cfg.WintrInAppNotifications.Credentials.AppID,
	}, nil
}

func (i *inApp) Send(ctx context.Context, parcel *Parcel, userIDs ...UserID) error {
	return errors.Wrapf(retry(ctx, func() error {
		err := i.send(ctx, parcel, userIDs...)
		if err != nil && !errors.Is(err, errPleaseRetry) {
			return backoff.Permanent(err)
		}

		return err
	}), "permanently failed to send inApp notification %#v", parcel)
}

func retry(ctx context.Context, op func() error) error {
	//nolint:wrapcheck // No need, its just a proxy.
	return backoff.RetryNotify(
		op,
		backoff.WithContext(&backoff.ExponentialBackOff{
			InitialInterval:     100 * stdlibtime.Millisecond, //nolint:mnd,gomnd // .
			RandomizationFactor: 0.5,                          //nolint:mnd,gomnd // .
			Multiplier:          2.5,                          //nolint:mnd,gomnd // .
			MaxInterval:         stdlibtime.Second,
			MaxElapsedTime:      requestDeadline,
			Stop:                backoff.Stop,
			Clock:               backoff.SystemClock,
		}, ctx),
		func(e error, next stdlibtime.Duration) {
			log.Error(errors.Wrapf(e, "wintr/notifications/inapp call failed. retrying in %v... ", next))
		})
}

func (i *inApp) send(ctx context.Context, parcel *Parcel, userIDs ...UserID) error { //nolint:revive,funlen // Its better.
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context error")
	}
	if len(userIDs) == 0 {
		return errors.Errorf("please provide atleast one userID to send to")
	}
	if len(userIDs) > 1 {
		return i.broadcast(ctx, parcel, userIDs)
	}
	userNotificationFeed, err := i.client.NotificationFeed(i.feedName, userIDs[0])
	if err != nil {
		return errors.Wrapf(err, "unable to create notification feed")
	}
	if parcel.Time == nil {
		parcel.Time = time.Now()
	}
	if parcel.ReferenceID == "" {
		if parcel.Subject.Type == "userId" && parcel.Subject.Value == userIDs[0] {
			parcel.ReferenceID = fmt.Sprintf("%s:%s", parcel.Actor.Type, parcel.Actor.Value)
		} else {
			parcel.ReferenceID = fmt.Sprintf("%s:%s", parcel.Subject.Type, parcel.Subject.Value)
		}
	}
	_, err = userNotificationFeed.AddActivity(ctx, *activity(parcel))
	if apiErr, isAPIErr := err.(getstreamio.APIError); isAPIErr { //nolint:errorlint // Its badly handled by the 3rd party.
		if apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode >= http.StatusInternalServerError ||
			(apiErr.Rate.Limit != 0 && apiErr.Rate.Remaining == 0) {
			return errPleaseRetry
		}

		err = errors.Wrapf(err, "unexpected api error while adding activity to notification feed: %#v", apiErr)
	}

	return errors.Wrapf(err, "unexpected error while adding activity to notification feed")
}

func activity(parcel *Parcel) *getstreamio.Activity {
	return &getstreamio.Activity{
		Actor:     fmt.Sprintf("%s:%s", parcel.Actor.Type, parcel.Actor.Value),
		Time:      getstreamio.Time{Time: *parcel.Time.Time},
		Verb:      parcel.Action,
		ForeignID: parcel.ReferenceID,
		Object:    fmt.Sprintf("%s:%s", parcel.Subject.Type, parcel.Subject.Value),
		Extra:     parcel.Data,
	}
}

func (i *inApp) broadcast(ctx context.Context, parcel *Parcel, userIDs []UserID) error { //nolint:funlen // Its better.
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context error")
	}
	if MaxBatchSize < len(userIDs) {
		return errors.Errorf("maxBatchSize is %v, you can't send more at a time", MaxBatchSize)
	}
	if parcel.Time == nil {
		parcel.Time = time.Now()
	}
	if parcel.ReferenceID == "" {
		parcel.ReferenceID = fmt.Sprintf("%s:%s", parcel.Subject.Type, parcel.Subject.Value)
	}
	feeds := make([]getstreamio.Feed, 0, len(userIDs))
	for _, userID := range userIDs {
		userNotificationFeed, err := i.client.NotificationFeed(i.feedName, userID)
		if err != nil {
			return errors.Wrapf(err, "failed to create notification feed for userID %v", userID)
		}
		feeds = append(feeds, userNotificationFeed)
	}
	err := i.client.AddToMany(ctx, *activity(parcel), feeds...)
	if apiErr, isAPIErr := err.(getstreamio.APIError); isAPIErr { //nolint:errorlint // Its badly handled by the 3rd party.
		if apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode >= http.StatusInternalServerError ||
			(apiErr.Rate.Limit != 0 && apiErr.Rate.Remaining == 0) {
			return errPleaseRetry
		}

		err = errors.Wrapf(err, "unexpected api error while broadcasting activity to notification feeds: %#v", apiErr)
	}

	return errors.Wrapf(err, "unexpected error while broadcasting activity to notification feeds")
}
