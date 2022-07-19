// SPDX-License-Identifier: BUSL-1.1

package inapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/GetStream/stream-go2/v7"
	"github.com/pkg/errors"
)

func (i *inApp) createNotificationFeed(feedName string, userID UserID) (*stream.NotificationFeed, error) {
	notify, err := i.client.NotificationFeed(feedName, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create notification feed")
	}

	return notify, nil
}

func (*inApp) makeActivity(data *Parcel) stream.Activity {
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

func (i *ID) parse(data string) *ID {
	parts := strings.Split(data, ":")

	if len(parts) != 1+1 {
		i.Value = data
	} else {
		i.Type = parts[0]
		i.Value = parts[1]
	}

	return i
}

func (p *Parcel) activityToParcel(a *stream.Activity) *Parcel {
	p.ReferenceID = a.ForeignID
	p.Action = a.Verb
	p.Actor = *new(ID).parse(a.Actor)
	p.Subject = *new(ID).parse(a.Object)
	p.Data = a.Extra

	return p
}
