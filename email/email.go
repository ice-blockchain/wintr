// SPDX-License-Identifier: BUSL-1.1

package email

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	appCfg "github.com/ice-blockchain/wintr/config"
)

func New(applicationYAMLKey string) Client {
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	c := &email{}
	c.client = sendgrid.NewSendClient(cfg.Credentials.APIKey)

	return c
}

func (e *email) createCustomEmail(ctx context.Context, parcel *Parcel) *mail.SGMailV3 {
	m := mail.NewV3Mail()
	from := mail.NewEmail(parcel.From.Name, parcel.From.Email)
	m.SetFrom(from)
	m.Subject = parcel.Subject

	person := mail.NewPersonalization()

	to := mail.NewEmail(parcel.To.Name, parcel.To.Email)
	person.AddTos(to)

	for _, c := range parcel.Content {
		content := mail.NewContent(c.Type, c.Data)
		m.AddPersonalizations(person)
		m.AddContent(content)
	}

	return m
}

func (e *email) Send(ctx context.Context, parcel *Parcel) error {
	response, err := e.client.Send(e.createCustomEmail(ctx, parcel))
	if err != nil {
		return errors.Wrapf(err, "error sending email to %v", parcel.To.Email)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var j errorReply
		err := json.Unmarshal([]byte(response.Body), &j)

		if err == nil {
			return errors.New(j.Errors[0].Message)
		}

		return errors.Wrapf(err, "error parsing reply")
	}

	return nil
}

func (e *email) SendMulti(ctx context.Context, parcels []*Parcel) error {
	var wg sync.WaitGroup
	chErr := make(chan error, len(parcels))

	for _, a := range parcels {
		wg.Add(1)
		copyA := a

		go func() {
			defer wg.Done()
			chErr <- e.Send(ctx, copyA)
		}()
	}

	wg.Wait()
	close(chErr)

	var m *multierror.Error
	for e := range chErr {
		m = multierror.Append(m, e)
	}

	return errors.Wrapf(m.ErrorOrNil(), "error sending multiple emails")
}
