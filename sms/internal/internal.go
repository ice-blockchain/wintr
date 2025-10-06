// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/twilio/twilio-go"
	twilioopenapimessagingv1 "github.com/twilio/twilio-go/rest/messaging/v1"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(applicationYAMLKey string) (client *twilio.RestClient, messageServicesByCountry map[string]*PhoneNumbersRoundRobinLB) {
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

	client = twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.WintrSMS.Credentials.User,
		Password: cfg.WintrSMS.Credentials.Password,
	})
	client.SetTimeout(requestDeadline)

	return client, initClient(client, &cfg)
}

func initClient(client *twilio.RestClient, cfg *config) map[string]*PhoneNumbersRoundRobinLB {
	const maxPageSize = 1000
	services, err := client.MessagingV1.ListService(new(twilioopenapimessagingv1.ListServiceParams).SetLimit(maxPageSize))
	log.Panic(errors.Wrapf(err, "failed to ListMessageServices")) //nolint:revive // .
	senderPhoneNumbersByCountry := map[string]*PhoneNumbersRoundRobinLB{}
	for i := range services {
		lb := new(PhoneNumbersRoundRobinLB)
		lb.schedulingMessagingServiceSID = *services[i].Sid
		serviceMapped := false
		for country, messagingServiceCfg := range cfg.WintrSMS.MessageServiceSIDs {
			if messagingServiceCfg == lb.schedulingMessagingServiceSID {
				senderPhoneNumbersByCountry[country] = lb
				serviceMapped = true

				break
			}
		}
		if !serviceMapped {
			log.Warn(fmt.Sprintf("message service %v not mapped to any country", lb.schedulingMessagingServiceSID))
			if len(services) == 1 {
				senderPhoneNumbersByCountry["global"] = lb
			} else {
				continue
			}
		}
		phoneNumbers, pErr := client.MessagingV1.ListPhoneNumber(lb.schedulingMessagingServiceSID,
			new(twilioopenapimessagingv1.ListPhoneNumberParams).SetPageSize(maxPageSize))
		log.Panic(errors.Wrapf(pErr, "failed to ListPhoneNumber for %v", lb.schedulingMessagingServiceSID)) //nolint:revive // .
		lb.phoneNumbers = make([]string, 0, len(phoneNumbers))
		for i := range phoneNumbers {
			lb.phoneNumbers = append(lb.phoneNumbers, *(phoneNumbers[i].PhoneNumber))
		}
	}

	return senderPhoneNumbersByCountry
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
