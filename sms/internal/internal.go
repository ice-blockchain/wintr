// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"os"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/twilio/twilio-go"
	twilioopenapi "github.com/twilio/twilio-go/rest/api/v2010"
	twilioopenapimessagingv1 "github.com/twilio/twilio-go/rest/messaging/v1"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(applicationYAMLKey string) (*twilio.RestClient, *PhoneNumbersRoundRobinLB) {
	var cfg config
	appcfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	if cfg.WintrSMS.Credentials.User == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrSMS.Credentials.User = os.Getenv(module + "_SMS_CLIENT_USER")
		if cfg.WintrSMS.Credentials.User == "" {
			cfg.WintrSMS.Credentials.User = os.Getenv("SMS_CLIENT_USER")
		}
	}
	if cfg.WintrSMS.Credentials.Password == "" {
		module := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(applicationYAMLKey, "-", "_"), "/", "_"))
		cfg.WintrSMS.Credentials.Password = os.Getenv(module + "_SMS_CLIENT_PASSWORD")
		if cfg.WintrSMS.Credentials.Password == "" {
			cfg.WintrSMS.Credentials.Password = os.Getenv("SMS_CLIENT_PASSWORD")
		}
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.WintrSMS.Credentials.User,
		Password: cfg.WintrSMS.Credentials.Password,
	})
	client.SetTimeout(requestDeadline)

	return client, new(PhoneNumbersRoundRobinLB).init(client)
}

func (lb *PhoneNumbersRoundRobinLB) init(client *twilio.RestClient) *PhoneNumbersRoundRobinLB {
	const maxPageSize = 1000
	allPhoneNumbersOwned, err := client.Api.ListIncomingPhoneNumber(new(twilioopenapi.ListIncomingPhoneNumberParams).SetPageSize(maxPageSize))
	log.Panic(errors.Wrapf(err, "failed to ListIncomingPhoneNumber")) //nolint:revive // That's intended.

	lb.phoneNumbers = make([]string, 0, len(allPhoneNumbersOwned))
	for i := range allPhoneNumbersOwned {
		lb.phoneNumbers = append(lb.phoneNumbers, *(allPhoneNumbersOwned[i].PhoneNumber))
	}
	services, err := client.MessagingV1.ListService(new(twilioopenapimessagingv1.ListServiceParams).SetLimit(1))
	log.Panic(errors.Wrapf(err, "failed to ListMessageServices"))
	if len(services) > 0 {
		lb.schedulingMessagingServiceSID = *services[0].Sid
	}

	return lb
}

func (lb *PhoneNumbersRoundRobinLB) PhoneNumber() string {
	return lb.phoneNumbers[atomic.AddUint64(&lb.currentIndex, 1)%uint64(len(lb.phoneNumbers))]
}

func (lb *PhoneNumbersRoundRobinLB) SchedulingMessageServiceSID() string {
	return lb.schedulingMessagingServiceSID
}

func (lb *PhoneNumbersRoundRobinLB) PhoneNumbers() []string {
	return lb.phoneNumbers[:] //nolint:gocritic // We need to clone it.
}
