// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"context"
	"os"
	"strings"

	getstreamio "github.com/GetStream/stream-go2/v7"
	"github.com/pkg/errors"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/notifications/inapp/internal"
	"github.com/ice-blockchain/wintr/time"
)

func GetAllInAppNotifications(ctx context.Context, applicationYAMLKey, feedName, userID string) ([]*internal.Parcel, error) {
	client := newGetStreamIOClient(applicationYAMLKey)
	res, err := getNotificationFeedActivities(ctx, client, feedName, userID)
	if err != nil {
		return nil, err
	}
	if len(res) != 0 {
		return res, nil
	}

	return getFlatFeedActivities(ctx, client, feedName, userID)
}

func getNotificationFeedActivities(ctx context.Context, client *getstreamio.Client, feedName, userID string) ([]*internal.Parcel, error) {
	feed, err := client.NotificationFeed(feedName, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create notifications feed")
	}

	resp, err := feed.GetActivities(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get activities for %v", feed.ID())
	}

	res := make([]*internal.Parcel, 0, len(resp.Results))
	for c := 0; c < len(resp.Results); c++ {
		notifResults := &resp.Results[c]
		for i := range notifResults.Activities {
			respActivity := notifResults.Activities[i]
			parcel := new(internal.Parcel)
			parcel.Action = respActivity.Verb
			parcel.ReferenceID = respActivity.ForeignID
			parcel.Time = time.New(respActivity.Time.Time)
			parcel.Actor = parseID(respActivity.Actor)
			parcel.Subject = parseID(respActivity.Object)
			parcel.Data = respActivity.Extra
			res = append(res, parcel)
		}
	}

	return res, nil
}

func getFlatFeedActivities(ctx context.Context, client *getstreamio.Client, feedName, userID string) ([]*internal.Parcel, error) {
	feed, err := client.FlatFeed(feedName, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create flat feed")
	}

	resp, err := feed.GetActivities(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get activities for %v", feed.ID())
	}
	res := make([]*internal.Parcel, 0, len(resp.Results))
	for c := 0; c < len(resp.Results); c++ {
		respActivity := &resp.Results[c]
		parcel := new(internal.Parcel)
		parcel.Action = respActivity.Verb
		parcel.ReferenceID = respActivity.ForeignID
		parcel.Time = time.New(respActivity.Time.Time)
		parcel.Actor = parseID(respActivity.Actor)
		parcel.Subject = parseID(respActivity.Object)
		parcel.Data = respActivity.Extra
		res = append(res, parcel)
	}

	return res, nil
}

func parseID(data string) internal.ID {
	var id internal.ID
	parts := strings.Split(data, ":")

	if len(parts) != 1+1 {
		id.Value = data
	} else {
		id.Type = parts[0]
		id.Value = parts[1]
	}

	return id
}

func newGetStreamIOClient(applicationYAMLKey string) *getstreamio.Client {
	var cfg struct {
		WintrInAppNotifications struct {
			Credentials struct {
				Key    string `yaml:"key"`
				Secret string `yaml:"secret"`
			} `yaml:"credentials" mapstructure:"credentials"`
		} `yaml:"wintr/notifications/inapp" mapstructure:"wintr/notifications/inapp"` //nolint:tagliatelle // Nope.
	}
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
	cl, err := getstreamio.New(
		cfg.WintrInAppNotifications.Credentials.Key,
		cfg.WintrInAppNotifications.Credentials.Secret)
	log.Panic(errors.Wrapf(err, "unable to init GetStream client"))

	return cl
}
