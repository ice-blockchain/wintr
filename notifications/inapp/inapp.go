// SPDX-License-Identifier: BUSL-1.1

package inapp

import (
	"context"
	"strings"
	"sync"

	stream "github.com/GetStream/stream-go2/v7"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(applicationYamlKey, slug string) Client {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	c := &inApp{}
	cl, err := stream.New(cfg.Credentials.Key, cfg.Credentials.Secret)
	if err != nil {
		log.Panic(errors.Wrapf(err, "unable to init GetStream client"))
	}

	c.client = cl
	c.slug = slug

	return c
}

func (i *inApp) Send(ctx context.Context, data *Parcel) error {
	not, err := i.createNotificationFeed(i.slug, data.UserID)
	if err != nil {
		return errors.Wrapf(err, "unable to create notification feed for %v", data.UserID)
	}

	return errors.Wrap(i.addNotificationActivity(ctx, not, data), "unable to send notification")
}

func (i *inApp) SendMulti(ctx context.Context, parcels []*Parcel) error {
	var wg sync.WaitGroup
	chErr := make(chan error, len(parcels))

	for _, a := range parcels {
		wg.Add(1)
		copyA := a

		go func() {
			defer wg.Done()
			chErr <- i.Send(ctx, copyA)
		}()
	}

	wg.Wait()
	close(chErr)

	var m *multierror.Error
	for e := range chErr {
		m = multierror.Append(m, e)
	}

	return errors.Wrapf(m.ErrorOrNil(), "error sending to multiple notification feeds")
}

func (i *inApp) GetAll(ctx context.Context, userID string) ([]*Parcel, error) {
	/*
		Looks like a bug in getstream lib. NotificationFeed returns wrong struct so we use FlatFeed
		Actually, all stream types are the same Flat type but with different interfaces
	*/
	not, err := i.client.FlatFeed(i.slug, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create flat feed")
	}

	resp, err := not.GetActivities(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get activities for %v", not.ID())
	}

	res := make([]*Parcel, 0, len(resp.Results))
	for c := 0; c < len(resp.Results); c++ {
		m := make(map[string]string, len(resp.Results[c].Extra))
		for k, v := range resp.Results[c].Extra {
			m[k] = v.(string)
		}
		res = append(res, &Parcel{
			ReferenceID: resp.Results[c].ForeignID,
			Action:      resp.Results[c].Verb,
			Actor:       *i.splitID(resp.Results[c].Actor),
			Subject:     *i.splitID(resp.Results[c].Object),
			Data:        m,
		})
	}

	return res, nil
}

func (i *inApp) splitID(data string) *ID {
	r := strings.Split(data, ":")

	if len(r) != 1+1 {
		return &ID{}
	}

	return &ID{
		Type:  r[0],
		Value: r[1],
	}
}
