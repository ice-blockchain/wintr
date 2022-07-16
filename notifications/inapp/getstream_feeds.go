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
	return stream.Activity{
		Actor:     fmt.Sprintf("%s:%s", data.Actor.Type, data.Actor.Value),
		Verb:      data.Action,
		ForeignID: data.ReferenceID,
		Object:    fmt.Sprintf("%s:%s", data.Subject.Type, data.Subject.Value),
		Extra:     data.Data,
	}
}

func (i *inApp) addNotificationActivity(ctx context.Context, notify *stream.NotificationFeed, data *Parcel) error {
	_, err := notify.AddActivity(ctx, i.makeActivity(data))

	return errors.Wrapf(err, "error adding notification activity")
}
