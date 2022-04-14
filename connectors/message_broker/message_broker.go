// SPDX-License-Identifier: BUSL-1.1

package messagebroker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"

	appCfg "github.com/ICE-Blockchain/wintr/config"
	"github.com/ICE-Blockchain/wintr/log"
)

func MustConnect(ctx context.Context, applicationYamlKey string) Client {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)

	mb := &messageBroker{}
	mb.connectCreateAndValidateTopics(ctx)

	return mb
}

func MustConnectAndStartConsuming(ctx context.Context, cancel context.CancelFunc, applicationYamlKey string, processors map[Topic]Processor) Client {
	appCfg.MustLoadFromKey(applicationYamlKey, &cfg)
	if len(processors) == 0 {
		log.Panic(errors.New("alteast one processor required if you want to start consuming"))
	}
	mb := &messageBroker{concurrentConsumer: &concurrentConsumer{
		mx:                     new(sync.Mutex),
		consumers:              new(sync.Map),
		processors:             processors,
		consumingWg:            new(sync.WaitGroup),
		partitionCountPerTopic: new(sync.Map),
	}}
	mb.connectCreateAndValidateTopics(ctx, mb.consumerOpts(processors)...)
	go mb.startConsuming(ctx, cancel)

	return mb
}

func (mb *messageBroker) consumerOpts(processors map[Topic]Processor) []kgo.Opt {
	topics := make([]string, 0, len(processors))
	for topic := range processors {
		topics = append(topics, topic)
	}

	return []kgo.Opt{
		kgo.ConsumeTopics(topics...),
		kgo.FetchIsolationLevel(kgo.ReadUncommitted()),
		kgo.ConsumerGroup(cfg.MessageBroker.ConsumerGroup),
		kgo.OnPartitionsRevoked(mb.OnPartitionsLost),
		kgo.OnPartitionsLost(mb.OnPartitionsLost),
		kgo.OnPartitionsAssigned(mb.OnPartitionsAssigned),
		kgo.BlockRebalanceOnPoll(),
		kgo.AutoCommitCallback(func(_ *kgo.Client, rq *kmsg.OffsetCommitRequest, rp *kmsg.OffsetCommitResponse, err error) {
			if err != nil {
				log.Error(errors.Wrap(err, "failed to autocommit offsets"), "request", rq, "response", rp)
			} else {
				log.Debug("auto committed offsets", "request", rq, "response", rp)
			}
		}),
	}
}

func (mb *messageBroker) connectCreateAndValidateTopics(ctx context.Context, additionalOpts ...kgo.Opt) {
	if err := mb.connect(additionalOpts...); err != nil {
		log.Panic(errors.Wrap(err, "failed to connect to message broker"))
	}

	adminCtx, cancelCtx := context.WithTimeout(ctx, messageBrokerSchemaInitDeadline)
	defer cancelCtx()
	mb.createAndValidateTopics(adminCtx)
}

func (mb *messageBroker) connect(additionalOpts ...kgo.Opt) error {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.MessageBroker.URLs...),
		kgo.WithLogger(mb),
		kgo.ProducerOnDataLossDetected(func(topic string, partition int32) {
			log.Error(errors.New("message broker producer data loss detected"), "topic", topic, "partition", partition)
		}),
	}
	if len(additionalOpts) != 0 {
		opts = append(opts, additionalOpts...)
	}
	if cfg.MessageBroker.CertPath != "" {
		log.Info("enabling TLS for message broker")
		tlsConfig, err := buildMessageBrokerTLS()
		if err != nil {
			return errors.Wrap(err, "could not build TLS for the message broker")
		}
		opts = append(opts, kgo.DialTLSConfig(tlsConfig))
	}
	log.Info("connecting to MessageBroker...", "URLs", cfg.MessageBroker.URLs)
	var err error
	mb.client, err = kgo.NewClient(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to connect to MessageBroker")
	}
	mb.admClient = kadm.NewClient(mb.client)

	return nil
}

func buildMessageBrokerTLS() (*tls.Config, error) {
	caCertPool := x509.NewCertPool()
	caCert, err := os.ReadFile(cfg.MessageBroker.CertPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading message broker TLS certificate %v", cfg.MessageBroker.CertPath)
	}
	caCertPool.AppendCertsFromPEM(caCert)

	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    caCertPool,
	}, nil
}

func (mb *messageBroker) createAndValidateTopics(ctx context.Context) []string {
	log.Info("creating and/or validate topics...", "URLs", cfg.MessageBroker.URLs)
	ts := make([]string, 0, len(cfg.MessageBroker.Topics))
	for _, tpc := range cfg.MessageBroker.Topics {
		if cfg.MessageBroker.CreateTopics {
			mb.createTopic(ctx, tpc)
		}
		ts = append(ts, tpc.Name)
	}
	mb.validateTopics(ctx, ts)

	return ts
}

//nolint:gofumpt // To be removed when golangci-lint works properly with go1.18.
func (mb *messageBroker) createTopic(ctx context.Context, tpc struct {
	Name              string        `yaml:"name" json:"name"`
	CleanupPolicy     string        `yaml:"cleanupPolicy" json:"cleanupPolicy"`
	Partitions        uint64        `yaml:"partitions" json:"partitions"`
	ReplicationFactor uint64        `yaml:"replicationFactor" json:"replicationFactor"`
	Retention         time.Duration `yaml:"retention" json:"retention"`
}) {
	p := tpc.CleanupPolicy
	if p == "" {
		p = "delete"
	}
	configs := map[string]*string{
		"cleanup.policy":      kadm.StringPtr(p),
		"retention.ms":        kadm.StringPtr(strconv.Itoa(int(tpc.Retention.Milliseconds()))),
		"delete.retention.ms": kadm.StringPtr(strconv.Itoa(int(tpc.Retention.Milliseconds()))),
		"segment.ms":          kadm.StringPtr(strconv.Itoa(int(tpc.Retention.Milliseconds()))),
	}
	log.Info("trying to create topic", "topic", tpc)
	r, cErr := mb.admClient.CreateTopics(ctx, int32(tpc.Partitions), int16(tpc.ReplicationFactor), configs, tpc.Name)
	if cErr != nil {
		log.Panic(errors.Wrap(cErr, "could not create topic"), "topic", tpc)
	}
	if r[tpc.Name].Err != nil && !errors.Is(r[tpc.Name].Err, kerr.TopicAlreadyExists) {
		log.Panic(errors.Wrap(r[tpc.Name].Err, "could not create topic"), "topic", tpc)
	}
	if r[tpc.Name].Err != nil && errors.Is(r[tpc.Name].Err, kerr.TopicAlreadyExists) {
		log.Info("topic already exists, so we`re ok", "topic", tpc)
	}
}

func (mb *messageBroker) validateTopics(ctx context.Context, topics []string) {
	mb.validateTopicListing(ctx, topics)
	mb.validateTopicConfigDescribing(ctx, topics)
}

func (mb *messageBroker) validateTopicListing(ctx context.Context, topics []string) {
	r, cErr := mb.admClient.ListTopics(ctx, topics...)
	if cErr != nil {
		log.Panic(errors.Wrap(cErr, "could not list topics"))
	}
	sortedTopicDetails := r.Sorted()
	if len(sortedTopicDetails) == 0 {
		log.Panic(errors.New("could not list topic because nothing was found"))
	}
	if sortedTopicDetails[0].Err != nil {
		log.Panic(errors.Wrap(sortedTopicDetails[0].Err, "could not list topic"), "topics", sortedTopicDetails)
	}
	log.Info(fmt.Sprintf("topics %v found! ", topics), "topics", sortedTopicDetails)
	if mb.concurrentConsumer != nil && mb.partitionCountPerTopic != nil {
		for _, detail := range sortedTopicDetails {
			mb.partitionCountPerTopic.Store(detail.Topic, int32(len(detail.Partitions)))
		}
	}
}

func (mb *messageBroker) validateTopicConfigDescribing(ctx context.Context, topics []string) {
	rc, cErr := mb.admClient.DescribeTopicConfigs(ctx, topics...)
	if cErr != nil {
		log.Panic(errors.Wrap(cErr, "could not describe topic configs"))
	}
	if len(rc) == 0 {
		log.Panic(errors.New("could not describe topic configs nothing was found"))
	}
	if rc[0].Err != nil {
		log.Panic(errors.Wrap(rc[0].Err, "could not describe topic configs"), "topic-configs", rc)
	}
	log.Info(fmt.Sprintf("topic configs %v found! ", topics), "topic-configs", rc)
}

func (mb *messageBroker) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), messageBrokerCloseDeadline)
	defer cancel()
	var err1 error
	if mb.concurrentConsumer != nil {
		err1 = errors.Wrap(mb.closeAndWaitForConsumersToFinishProcessing(ctx), "waiting for consumers to finish failed")
	}
	err2 := errors.Wrap(mb.client.Flush(ctx), "message broker client flush failed")
	mb.client.Close()
	var err error
	if err1 != nil && err2 != nil {
		err = multierror.Append(err1, err2)
	} else {
		if err1 != nil {
			err = err1
		}
		if err2 != nil {
			err = err2
		}
	}

	return errors.Wrap(err, "failed to close message broker client")
}

func (mb *messageBroker) Level() kgo.LogLevel {
	switch log.Level() {
	case "trace":
		return kgo.LogLevelDebug
	case "debug":
		return kgo.LogLevelDebug
	case "info":
		return kgo.LogLevelInfo
	case "warn":
		return kgo.LogLevelWarn
	case "error":
		return kgo.LogLevelError
	case "fatal":
		return kgo.LogLevelError
	case "panic":
		return kgo.LogLevelError
	default:
		return kgo.LogLevelNone
	}
}

func (mb *messageBroker) Log(level kgo.LogLevel, msg string, keyValEnumeration ...interface{}) {
	switch level {
	case kgo.LogLevelError:
		log.Error(errors.New(msg), keyValEnumeration...)

		return
	case kgo.LogLevelWarn:
		log.Warn(msg, keyValEnumeration...)

		return
	case kgo.LogLevelInfo:
		log.Info(msg, keyValEnumeration...)

		return
	case kgo.LogLevelDebug:
		log.Debug(msg, keyValEnumeration...)

		return
	default:
	}
}
