// SPDX-License-Identifier: ice License 1.0

package messagebroker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"

	appCfg "github.com/ice-blockchain/wintr/config"
	"github.com/ice-blockchain/wintr/log"
)

func MustConnect(ctx context.Context, applicationYAMLKey string) Client {
	var cfg Config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)

	mb := &messageBroker{cfg: &cfg}
	mb.connectCreateAndValidateTopics(ctx)

	return mb
}

func MustConnectAndStartConsuming(ctx context.Context, cancel context.CancelFunc, applicationYAMLKey string, processors ...Processor) Client {
	var cfg Config
	appCfg.MustLoadFromKey(applicationYAMLKey, &cfg)
	if len(processors) == 0 {
		log.Panic(errors.New("at least one processor required if you want to start consuming"))
	}
	processorPerTopicMappings := make(map[Topic]Processor, len(processors))
	for index, processor := range processors {
		processorPerTopicMappings[cfg.MessageBroker.ConsumingTopics[index].Name] = processor
	}
	mb := &messageBroker{
		cfg: &cfg,
		concurrentConsumer: &concurrentConsumer{
			mx:                     new(sync.Mutex),
			consumers:              new(sync.Map),
			processors:             processorPerTopicMappings,
			consumingWg:            new(sync.WaitGroup),
			partitionCountPerTopic: new(sync.Map),
		},
	}
	mb.connectCreateAndValidateTopics(ctx, mb.consumerOpts(processorPerTopicMappings)...)
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
		kgo.ConsumerGroup(mb.cfg.MessageBroker.ConsumerGroup),
		kgo.OnPartitionsRevoked(mb.OnPartitionsLost),
		kgo.OnPartitionsLost(mb.OnPartitionsLost),
		kgo.OnPartitionsAssigned(mb.OnPartitionsAssigned),
		kgo.BlockRebalanceOnPoll(),
	}
}

func (mb *messageBroker) connectCreateAndValidateTopics(ctx context.Context, additionalOpts ...kgo.Opt) {
	if err := mb.connect(); err != nil {
		log.Panic(errors.Wrap(err, "failed to connect to message broker"))
	}

	adminCtx, cancelCtx := context.WithTimeout(ctx, messageBrokerSchemaInitDeadline)
	defer cancelCtx()
	mb.createAndValidateTopics(adminCtx)

	if len(additionalOpts) != 0 {
		mb.client.Close()
		if err := mb.connect(additionalOpts...); err != nil {
			log.Panic(errors.Wrap(err, "failed to connect to message broker"))
		}
	}
}

func (mb *messageBroker) connect(additionalOpts ...kgo.Opt) error { //nolint:funlen // .
	opts := []kgo.Opt{
		kgo.SeedBrokers(mb.cfg.MessageBroker.URLs...),
		kgo.WithLogger(mb),
		kgo.ProducerBatchCompression(kgo.ZstdCompression(), kgo.Lz4Compression(), kgo.SnappyCompression(), kgo.NoCompression()),
		kgo.ProducerOnDataLossDetected(func(topic string, partition int32) {
			log.Error(errors.New("message broker producer data loss detected"), "topic", topic, "partition", partition)
		}),
	}
	if len(additionalOpts) != 0 {
		opts = append(opts, additionalOpts...)
	}
	if mb.cfg.MessageBroker.DisableIdempotence {
		opts = append(opts, kgo.DisableIdempotentWrite())
	}
	if mb.cfg.MessageBroker.CertPath != "" {
		log.Info("enabling TLS for message broker")
		tlsConfig, err := mb.buildMessageBrokerTLS()
		if err != nil {
			return errors.Wrap(err, "could not build TLS for the message broker")
		}
		opts = append(opts, kgo.DialTLSConfig(tlsConfig))
	}
	if mb.cfg.MessageBroker.MaxMessageBytes == 0 {
		mb.cfg.MessageBroker.MaxMessageBytes = 10485760
	}
	opts = append(opts, kgo.ProducerBatchMaxBytes(int32(mb.cfg.MessageBroker.MaxMessageBytes)))
	log.Info("connecting to MessageBroker...", "URLs", mb.cfg.MessageBroker.URLs)
	var err error
	mb.client, err = kgo.NewClient(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to connect to MessageBroker")
	}
	mb.admClient = kadm.NewClient(mb.client)
	if mb.concurrentConsumer != nil {
		mb.concurrentConsumer.client = mb.client
		mb.concurrentConsumer.admClient = mb.admClient
	}

	return nil
}

func (mb *messageBroker) buildMessageBrokerTLS() (*tls.Config, error) {
	caCertPool := x509.NewCertPool()
	caCert, err := os.ReadFile(mb.cfg.MessageBroker.CertPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading message broker TLS certificate %v", mb.cfg.MessageBroker.CertPath)
	}
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Panic(errors.New("failed to AppendCertsFromPEM file"))
	}
	var accessCerts []tls.Certificate
	if mb.cfg.MessageBroker.AccessKeyPath != "" && mb.cfg.MessageBroker.AccessCertPath != "" {
		keypair, loadErr := tls.LoadX509KeyPair(mb.cfg.MessageBroker.AccessCertPath, mb.cfg.MessageBroker.AccessKeyPath)
		if loadErr != nil {
			log.Panic(errors.Wrapf(loadErr, "failed to load access (key,cert) pair at (`%v`,`%v`)",
				mb.cfg.MessageBroker.AccessKeyPath, mb.cfg.MessageBroker.AccessCertPath))
		}
		accessCerts = []tls.Certificate{keypair}
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: accessCerts,
		RootCAs:      caCertPool,
	}, nil
}

func (mb *messageBroker) createAndValidateTopics(ctx context.Context) {
	log.Info("creating and/or validate topics...", "URLs", mb.cfg.MessageBroker.URLs)
	allTopics := make(map[Topic]struct{}, len(mb.cfg.MessageBroker.Topics)+len(mb.cfg.MessageBroker.ConsumingTopics)+len(mb.cfg.MessageBroker.ProducingTopics))
	for _, tpc := range mb.cfg.MessageBroker.Topics {
		if mb.cfg.MessageBroker.CreateTopics {
			mb.createTopic(ctx, tpc)
		}
		allTopics[tpc.Name] = struct{}{}
	}
	if mb.concurrentConsumer != nil {
		mb.consumerTopicConfigs = make(map[Topic]*ConsumerTopicConfig, len(mb.cfg.MessageBroker.ConsumingTopics))
		for _, consumingTopic := range mb.cfg.MessageBroker.ConsumingTopics {
			allTopics[consumingTopic.Name] = struct{}{}
			mb.consumerTopicConfigs[consumingTopic.Name] = consumingTopic
		}
	}
	for _, producingTopic := range mb.cfg.MessageBroker.ProducingTopics {
		allTopics[producingTopic.Name] = struct{}{}
	}
	topics := make([]string, 0, len(allTopics))
	for topic := range allTopics {
		topics = append(topics, topic)
	}
	mb.validateTopics(ctx, topics)
}

func (mb *messageBroker) createTopic(ctx context.Context, tpc *TopicConfig) {
	p := tpc.CleanupPolicy
	if p == "" {
		p = "compact"
	}
	configs := map[string]*string{
		"cleanup.policy":      kadm.StringPtr(p),
		"retention.ms":        kadm.StringPtr(strconv.Itoa(int(tpc.Retention.Milliseconds()))),
		"delete.retention.ms": kadm.StringPtr(strconv.Itoa(int(tpc.Retention.Milliseconds()))),
		"segment.ms":          kadm.StringPtr(strconv.Itoa(int(tpc.Retention.Milliseconds()))),
	}
	log.Info("trying to create topic", "topic", tpc)
	createTopicsResponse, cErr := mb.admClient.CreateTopics(ctx, int32(tpc.Partitions), int16(tpc.ReplicationFactor), configs, tpc.Name)
	if cErr != nil {
		log.Panic(errors.Wrap(cErr, "could not create topic"), "topic", tpc)
	}
	if createTopicsResponse[tpc.Name].Err != nil && !errors.Is(createTopicsResponse[tpc.Name].Err, kerr.TopicAlreadyExists) {
		log.Panic(errors.Wrap(createTopicsResponse[tpc.Name].Err, "could not create topic"), "topic", tpc)
	}
	if createTopicsResponse[tpc.Name].Err != nil && errors.Is(createTopicsResponse[tpc.Name].Err, kerr.TopicAlreadyExists) {
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
		for i := range sortedTopicDetails {
			mb.partitionCountPerTopic.Store(sortedTopicDetails[i].Topic, int32(len(sortedTopicDetails[i].Partitions)))
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
	if mb == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), messageBrokerCloseDeadline)
	defer cancel()
	if mb.concurrentConsumer != nil {
		if !mb.concurrentConsumer.done {
			mb.concurrentConsumer.cancel()
			for !mb.concurrentConsumer.done && ctx.Err() == nil { //nolint:revive // That's intended, its a blocking wait.
			}
		}
	} else {
		log.Error(errors.Wrap(mb.client.Flush(ctx), "message broker client flush failed"))
		mb.client.Close()
	}

	return nil
}

func (*messageBroker) Level() kgo.LogLevel {
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

func (*messageBroker) Log(level kgo.LogLevel, msg string, keyValEnumeration ...any) {
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
	case kgo.LogLevelNone:
	default:
	}
}
