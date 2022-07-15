// SPDX-License-Identifier: BUSL-1.1

package inapp

import (
	"context"
	"fmt"

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

func (i *inApp) makeActivity(data *Parcel) stream.Activity {
	m := make(map[string]interface{}, len(data.Data))
	for k, v := range data.Data {
		m[k] = v
	}

	return stream.Activity{
		Actor:     fmt.Sprintf("%s:%s", data.Actor.Type, data.Actor.Value),
		Verb:      data.Action,
		ForeignID: data.ReferenceID,
		Object:    fmt.Sprintf("%s:%s", data.Subject.Type, data.Subject.Value),
		Extra:     m,
	}
}

func (i *inApp) addNotificationActivity(ctx context.Context, notify *stream.NotificationFeed, data *Parcel) error {
	_, err := notify.AddActivity(ctx, i.makeActivity(data))

	return errors.Wrapf(err, "error adding notification activity")
}

func (i *inApp) addNotificationActivities(ctx context.Context, notify *stream.NotificationFeed, data []*Parcel) error {
	activities := make([]stream.Activity, len(data))
	for c, d := range data {
		activities[c] = i.makeActivity(d)
	}

	_, err := notify.AddActivities(ctx, activities...)

	return errors.Wrapf(err, "error adding multiple notification activities")
}
