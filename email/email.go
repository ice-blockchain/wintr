// SPDX-License-Identifier: BUSL-1.1

package email

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	appCfg "github.com/ice-blockchain/wintr/config"
)

func New(applicationYAMLKey string) Client {
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	if cfg.Credentials.APIKey == "" {
		cfg.Credentials.APIKey = os.Getenv("EMAIL_CLIENT_APIKEY")
	}

	return &email{
		client: sendgrid.NewSendClient(cfg.Credentials.APIKey),
	}
}

func (*email) createCustomEmail(parcel *Parcel) *mail.SGMailV3 {
	mailParcel := mail.NewV3Mail()
	from := mail.NewEmail(parcel.From.Name, parcel.From.Email)
	mailParcel.SetFrom(from)
	mailParcel.Subject = parcel.Subject

	person := mail.NewPersonalization()

	to := mail.NewEmail(parcel.To.Name, parcel.To.Email)
	person.AddTos(to)

	for _, c := range parcel.Content {
		content := mail.NewContent(c.Type, c.Data)
		mailParcel.AddPersonalizations(person)
		mailParcel.AddContent(content)
	}

	return mailParcel
}

func (e *email) Send(_ context.Context, parcel *Parcel) error {
	response, err := e.client.Send(e.createCustomEmail(parcel))
	if err != nil {
		return errors.Wrapf(err, "error sending email to %v", parcel.To.Email)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var j errorReply
		err = json.Unmarshal([]byte(response.Body), &j)

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
