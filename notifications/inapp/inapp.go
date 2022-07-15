// SPDX-License-Identifier: BUSL-1.1

package inapp

import (
	"context"

	stream "github.com/GetStream/stream-go2/v7"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(applicationYamlKey string) Client {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	c := &inApp{}
	cl, err := stream.New(cfg.Credentials.Key, cfg.Credentials.Secret)
	if err != nil {
		log.Panic(errors.Wrapf(err, "unable to init GetStream client"))
	}

	c.client = cl

	return c
}

func (i *inApp) Send(ctx context.Context, slug, userID string, data *NotificationData) error {
	not, err := i.createNotificationFeed(slug, userID)
	if err != nil {
		return errors.Wrapf(err, "unable to create notification feed")
	}

	return errors.Wrap(i.addNotificationActivity(ctx, not, data), "unable to send notification")
}

func (i *inApp) SendMulti(ctx context.Context, slug, userID string, data []*NotificationData) error {
	not, err := i.createNotificationFeed(slug, userID)
	if err != nil {
		return errors.Wrapf(err, "unable to create notification feed")
	}

	return errors.Wrapf(i.addNotificationActivities(ctx, not, data), "unable to send notifications")
}

func (i *inApp) Get(ctx context.Context, slug, userID string) ([]*NotificationData, error) {
	/*
		Looks like a bug in getstream lib. NotificationFeed returns wrong struct so we use FlatFeed
		Actually, all stream types are the same Flat type but with different interfaces
	*/
	not, err := i.client.FlatFeed(slug, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create flat feed")
	}

	resp, err := not.GetActivities(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get activities for %v", not.ID())
	}

	res := make([]*NotificationData, len(resp.Results))

	for c := 0; c < len(resp.Results); c++ {
		res[c] = &NotificationData{
			Header:   resp.Results[c].Verb,
			ImageURL: resp.Results[c].Extra["imageUrl"].(string),
			BodyText: resp.Results[c].Extra["bodyText"].(string),
		}
	}

	return res, nil
}
