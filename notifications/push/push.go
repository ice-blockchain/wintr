// SPDX-License-Identifier: BUSL-1.1

package push

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

func New(ctx context.Context, applicationYamlKey string) Client {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(cfg.FCMCredentialsFile))
	if err != nil {
		log.Panic(errors.Wrap(err, "can't get firebase.NewApp"))
	}
	cl, err := app.Messaging(ctx)
	if err != nil {
		log.Panic(errors.Wrap(err, "can't get firebase messaging.Client"))
	}

	c := &push{}
	c.Client = cl

	return c
}

func (p *push) Send(ctx context.Context, dn *Parcel) (string, error) {
	resp, err := p.Client.Send(ctx, p.buildFCMMessage(dn))
	if err != nil {
		return "", errors.Wrapf(err, "failed to send push notification")
	}

	return resp, nil
}

func (p *push) SendMulti(ctx context.Context, parcels []*Parcel) (*messaging.BatchResponse, error) {
	m := p.buildFCMMessages(parcels)

	batchR, err := p.Client.SendAll(ctx, m)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send multiple push notifications")
	}

	return batchR, nil
}

func (p *push) buildFCMMessages(devNotifications []*Parcel) []*messaging.Message {
	m := make([]*messaging.Message, 0, len(devNotifications))

	for _, dn := range devNotifications {
		m = append(m, p.buildFCMMessage(dn))
	}

	return m
}

func (p *push) buildFCMMessage(dn *Parcel) *messaging.Message {
	return &messaging.Message{
		Data:  dn.Notification.Data,
		Token: dn.Device.Token,
		Notification: &messaging.Notification{
			Title:    dn.Notification.Title,
			Body:     dn.Notification.Body,
			ImageURL: dn.Notification.ImageURL,
		},
	}
}
