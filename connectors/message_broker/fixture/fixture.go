// SPDX-License-Identifier: BUSL-1.1

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
	applicationYamlTestKey := fmt.Sprintf("%v_test", applicationYAMLKey)
	config.MustLoadFromKey(applicationYamlTestKey, &cfg)

	return &testConnector{
		cfg:                &cfg,
		applicationYAMLKey: applicationYamlTestKey,
		order:              order,
		delegate:           fixture.NewConnector("mb", dockerComposeYAMLTemplate, "Successfully started Redpanda!", order, findMessageBrokerPort, nil),
	}
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
	tc.testMessageStore.mx = new(sync.RWMutex)
	tc.testMessageStore.chronologicalMessageList = []RawMessage{}
	processors := make(map[messagebroker.Topic]messagebroker.Processor, len(tc.cfg.MessageBroker.Topics))
	for _, topic := range tc.cfg.MessageBroker.Topics {
		processors[topic.Name] = tc.testMessageStore
	}
	tc.Client = messagebroker.MustConnectAndStartConsuming(ctx, func() {}, tc.applicationYAMLKey, processors)

	return func(cctx context.Context) error {
		return errors.Wrapf(multierror.Append(nil,
			errors.Wrapf(tc.Client.Close(), "failed closing the test message_broker client for %v", tc.applicationYAMLKey),
			errors.Wrapf(cleanUp(cctx), "failed to cleanup message broker connector for %v", tc.applicationYAMLKey)).ErrorOrNil(),
			"failed to cleanup messagebroker test connector")
	}
}

func findMessageBrokerPort(applicationYamlKey string) (int, bool, error) {
	var cfg struct {
		MessageBroker struct {
			CertPath string   `yaml:"certPath"`
			URLs     []string `yaml:"urls"`
		} `yaml:"messageBroker"`
	}
	config.MustLoadFromKey(applicationYamlKey, &cfg)
	if len(cfg.MessageBroker.URLs) == 0 {
		return 0, false, errors.Errorf("invalid/missing application.yaml for `%v`", applicationYamlKey)
	}
	port, err := strconv.Atoi(strings.Split(cfg.MessageBroker.URLs[0], ":")[1])
	if err != nil {
		return 0, false, errors.Wrapf(err, "could not find a valid messageBroker port for `%v`", applicationYamlKey)
	}

	return port, cfg.MessageBroker.CertPath != "", nil
}
