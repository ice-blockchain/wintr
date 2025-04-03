// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/connectors/fixture"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/log"
)

func NewTestConnector(applicationYAMLKey string, order int) TestConnector {
	var cfg messagebroker.Config
	applicationYAMLTestKey := fmt.Sprintf("%v_test", applicationYAMLKey)
	config.MustLoadFromKey(applicationYAMLTestKey, &cfg)

	tc := &testConnector{
		cfg:                &cfg,
		applicationYAMLKey: applicationYAMLTestKey,
		order:              order,
	}
	tc.delegate = fixture.NewConnector("mb", dockerComposeYAMLTemplate, "Successfully started Redpanda!", order, tc.findMessageBrokerPort, nil)

	return tc
}

func (tc *testConnector) Order() int {
	return tc.order
}

func (tc *testConnector) Setup(ctx context.Context) fixture.ContextErrClose {
	cleanUp := tc.delegate.Setup(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(cleanUp(ctx), "failed to cleanup message_broker connector due to premature panic"))
			log.Panic(e)
		}
	}()
	tc.testMessageStore = new(testMessageStore)
	tc.testMessageStore.mx = new(sync.RWMutex)                    //nolint:staticcheck // .
	tc.testMessageStore.chronologicalMessageList = []RawMessage{} //nolint:staticcheck // .
	processors := make([]messagebroker.Processor, 0, len(tc.cfg.MessageBroker.Topics))
	for range tc.cfg.MessageBroker.Topics {
		processors = append(processors, tc.testMessageStore)
	}
	tc.Client = messagebroker.MustConnectAndStartConsuming(ctx, func() {}, tc.applicationYAMLKey, processors...)

	return func(cctx context.Context) error {
		return errors.Wrapf(multierror.Append(nil,
			errors.Wrapf(tc.Client.Close(), "failed closing the test message_broker client for %v", tc.applicationYAMLKey),
			errors.Wrapf(cleanUp(cctx), "failed to cleanup message broker connector for %v", tc.applicationYAMLKey)).ErrorOrNil(),
			"failed to cleanup messagebroker test connector")
	}
}

func (tc *testConnector) findMessageBrokerPort() (int, bool, error) {
	if len(tc.cfg.MessageBroker.URLs) == 0 {
		return 0, false, errors.Errorf("invalid/missing application.yaml for `%v`", tc.applicationYAMLKey)
	}
	port, err := strconv.Atoi(strings.Split(tc.cfg.MessageBroker.URLs[0], ":")[1])
	if err != nil {
		return 0, false, errors.Wrapf(err, "could not find a valid messageBroker port for `%v`", tc.applicationYAMLKey)
	}

	return port, tc.cfg.MessageBroker.CertPath != "", nil
}
