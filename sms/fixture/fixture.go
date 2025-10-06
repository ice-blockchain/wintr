// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twilio/twilio-go"
	twilioclient "github.com/twilio/twilio-go/client"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/ice-blockchain/wintr/sms/internal"
)

//nolint:gochecknoglobals // We're using lazy stateless singletons for the whole testing runtime.
var (
	globalClient           *twilio.RestClient
	globalToPhoneNumbersLB map[string]*internal.PhoneNumbersRoundRobinLB
	singleton              = new(sync.Once)
)

func client() *twilio.RestClient {
	singleton.Do(func() {
		globalClient, globalToPhoneNumbersLB = internal.New("_")
	})

	return globalClient
}

func toPhoneNumbersLB() *internal.PhoneNumbersRoundRobinLB {
	singleton.Do(func() {
		globalClient, globalToPhoneNumbersLB = internal.New("_")
	})
	var messagingService *internal.PhoneNumbersRoundRobinLB
	for _, ms := range globalToPhoneNumbersLB {
		messagingService = ms

		break
	}

	return messagingService
}

func AssertSMSCode(ctx context.Context, tb testing.TB, toNumber, expectedCode string, parse func(body string) string) {
	tb.Helper()
	assert.Equal(tb, expectedCode, GetSMSCode(ctx, tb, toNumber, parse))
}

func GetSMSCode(ctx context.Context, tb testing.TB, toNumber string, parse func(body string) string) string {
	tb.Helper()
	twilioClient := client()
	isSent := false
	for !isSent && ctx.Err() == nil {
		smsList := getTwilioSMSListByNumber(twilioClient, toNumber)

		for i := range smsList {
			sms := smsList[i]
			if *sms.Direction != "outbound-api" {
				continue
			}
			code := parse(*sms.Body)
			isSent = *sms.Status == "sent" || *sms.Status == "delivered"
			if isSent {
				assert.Nil(tb, sms.ErrorCode)

				return code
			}
		}
		time.Sleep(100 * time.Millisecond) //nolint:mnd,gomnd // Not a magic.
	}
	require.Fail(tb, "SMS code was not sent")

	return ""
}

func getTwilioSMSListByNumber(twi *twilio.RestClient, toNumber string) []openapi.ApiV2010Message {
	to := new(string)
	*to = toNumber

	res, err := twi.Api.ListMessage(&openapi.ListMessageParams{To: to})
	if err != nil {
		return nil
	}

	return res
}

func ClearSMSQueue(phoneNumber string) error {
	twilioClient := client()
	smsList := getTwilioSMSListByNumber(twilioClient, phoneNumber)

	for i := range smsList {
		if err := twilioClient.Api.DeleteMessage(*smsList[i].Sid, nil); err != nil {
			//nolint:errorlint // errors.As(err,*twilioclient.TwilioRestError) doesn't seem to work.
			if tErr, ok := err.(*twilioclient.TwilioRestError); ok && tErr.Code == 20009 && tErr.Status == 409 {
				time.Sleep(5 * time.Second) //nolint:mnd,gomnd // .
				err = twilioClient.Api.DeleteMessage(*smsList[i].Sid, nil)
			}
			if err == nil {
				continue
			}

			return errors.Wrap(err, "failed to delete message using twilio API")
		}
	}

	return nil
}

func TestingPhoneNumber() string {
	return toPhoneNumbersLB().PhoneNumber()
}
