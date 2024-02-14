// SPDX-License-Identifier: ice License 1.0

package email

import (
	"context"
	"net/http"
	"os"
	"strings"
	stdlibtime "time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	sendgridrest "github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func init() { //nolint:gochecknoinits // It's the only way to tweak the client.
	sendgrid.DefaultClient = &sendgridrest.Client{HTTPClient: &http.Client{
		Timeout: requestDeadline,
	}}
}

func New(applicationYAMLKey string) Client {
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	if cfg.WintrEmail.Credentials.APIKey == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrEmail.Credentials.APIKey = os.Getenv(module + "_EMAIL_CLIENT_APIKEY")
		if cfg.WintrEmail.Credentials.APIKey == "" {
			cfg.WintrEmail.Credentials.APIKey = os.Getenv("EMAIL_CLIENT_APIKEY")
		}
	}

	emailClient := &email{
		client: sendgrid.NewSendClient(cfg.WintrEmail.Credentials.APIKey),
	}

	log.Panic(emailClient.Send(context.Background(), &Parcel{
		Body: &Body{
			Type: TextPlain,
			Data: "probing bootstrap",
		},
		Subject: "probing bootstrap",
		From: Participant{
			Name:  "ice.io",
			Email: "no-reply@ice.io",
		},
	}, Participant{
		Name:  "ice.io",
		Email: "no-reply@ice.io",
	}))

	return emailClient
}

func (e *email) Send(ctx context.Context, parcel *Parcel, destinations ...Participant) error {
	return errors.Wrapf(retry(ctx, func() error {
		err := e.send(ctx, parcel, destinations...)
		if err != nil && !errors.Is(err, errPleaseRetry) {
			//nolint:wrapcheck // It's a proxy.
			return backoff.Permanent(err)
		}

		return err
	}), "permanently failed to send email to %#v, from %#v, subject %v", destinations, parcel.From, parcel.Subject)
}

func retry(ctx context.Context, op func() error) error {
	//nolint:wrapcheck // No need, its just a proxy.
	return backoff.RetryNotify(
		op,
		//nolint:gomnd // Because those are static configs.
		backoff.WithContext(&backoff.ExponentialBackOff{
			InitialInterval:     100 * stdlibtime.Millisecond,
			RandomizationFactor: 0.5,
			Multiplier:          2.5,
			MaxInterval:         stdlibtime.Second,
			MaxElapsedTime:      requestDeadline,
			Stop:                backoff.Stop,
			Clock:               backoff.SystemClock,
		}, ctx),
		func(e error, next stdlibtime.Duration) {
			log.Error(errors.Wrapf(e, "wintr/email call failed. retrying in %v... ", next))
		})
}

func (e *email) send(ctx context.Context, parcel *Parcel, destinations ...Participant) error { //nolint:revive // Its better.
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context failed")
	}
	if len(destinations) > MaxBatchSize || len(destinations) == 0 {
		return errors.Errorf("please provide a number of destinations between [1..%v]", MaxBatchSize)
	}
	response, err := e.client.SendWithContext(ctx, parcel.sendgridEmail(destinations...))
	if err != nil {
		return errors.Wrapf(err, "error sending email")
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if response.StatusCode == http.StatusTooManyRequests ||
			response.StatusCode >= http.StatusInternalServerError ||
			(response.StatusCode == http.StatusForbidden && strings.Contains(response.Body, "You have exceeded your messaging limits")) {
			return errPleaseRetry
		}

		return errors.Errorf("failed to send email: %v", response.Body)
	}

	return nil
}

func (p *Parcel) sendgridEmail(destinations ...Participant) *mail.SGMailV3 {
	mailObj := new(mail.SGMailV3)
	mailObj.Subject = p.Subject
	mailObj.AddContent(mail.NewContent(string(p.Body.Type), p.Body.Data))
	mailObj.SetFrom(mail.NewEmail(p.From.Name, p.From.Email))
	for i := range destinations {
		personalization := mail.NewPersonalization()
		participant := destinations[i]
		personalization.AddTos(mail.NewEmail(participant.Name, participant.Email))
		if participant.SendAt != nil {
			personalization.SetSendAt(int(participant.SendAt.Unix()))
		}
		mailObj.AddPersonalizations(personalization)
	}

	return mailObj
}
