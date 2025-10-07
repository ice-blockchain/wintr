// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/twilio/twilio-go"
	twilioopenapimessagingv1 "github.com/twilio/twilio-go/rest/messaging/v1"

	appcfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func New(applicationYAMLKey string) (client *twilio.RestClient, messageServicesByCountry map[string]*MessagingService) {
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

func initClient(client *twilio.RestClient, cfg *config) map[string]*MessagingService {
	const maxPageSize = 1000
	services, err := client.MessagingV1.ListService(new(twilioopenapimessagingv1.ListServiceParams).SetLimit(maxPageSize))
	log.Panic(errors.Wrapf(err, "failed to ListMessageServices")) //nolint:revive // .
	senderPhoneNumbersByCountry := map[string]*MessagingService{}
	for i := range services {
		lb := new(MessagingService)
		lb.messagingServiceSID = *services[i].Sid
		serviceMapped := false
		for country, messagingServiceCfg := range cfg.WintrSMS.MessageServiceSIDs {
			if messagingServiceCfg == lb.messagingServiceSID {
				senderPhoneNumbersByCountry[country] = lb
				serviceMapped = true

				break
			}
		}
		if !serviceMapped {
			log.Warn(fmt.Sprintf("message service %v not mapped to any country", lb.messagingServiceSID))
			if len(services) > 1 {
				continue
			}
			senderPhoneNumbersByCountry["global"] = lb
		}
	}

	return senderPhoneNumbersByCountry
}

func (lb *MessagingService) MessageServiceSID() string {
	return lb.messagingServiceSID
}
