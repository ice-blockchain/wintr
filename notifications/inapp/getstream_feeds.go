// SPDX-License-Identifier: BUSL-1.1

package inapp

import (
	"context"

	"github.com/GetStream/stream-go2/v7"
	"github.com/pkg/errors"
)

func (i *inApp) createNotificationFeed(slug, userID string) (*stream.NotificationFeed, error) {
	notify, err := i.client.NotificationFeed(slug, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create notification feed")
	}

	return notify, nil
}

func (i *inApp) makeActivity(id string, data *NotificationData) stream.Activity {
	return stream.Activity{
		Actor:  id,
		Verb:   data.Header,
		Object: "1",
		Extra: map[string]interface{}{
			"imageUrl": data.ImageURL,
			"bodyText": data.BodyText,
		},
	}
}

func (i *inApp) addNotificationActivity(
	ctx context.Context, notify *stream.NotificationFeed, data *NotificationData,
) error {
	_, err := notify.AddActivity(ctx, i.makeActivity(notify.ID(), data))

	return errors.Wrapf(err, "error adding notification activity")
}

func (i *inApp) addNotificationActivities(
	ctx context.Context, notify *stream.NotificationFeed, data []*NotificationData,
) error {
	activities := make([]stream.Activity, len(data))
	for c, d := range data {
		activities[c] = i.makeActivity(notify.ID(), d)
	}

	_, err := notify.AddActivities(ctx, activities...)

	return errors.Wrapf(err, "error adding multiple notification activities")
}
