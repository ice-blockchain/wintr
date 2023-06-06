// SPDX-License-Identifier: ice License 1.0

package messagebroker

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Public API.

type (
	Partition      = int32
	PartitionCount = int32
	Topic          = string
	Message        struct {
		Timestamp      time.Time
		Headers        map[string]string
		Key            string
		Topic          string
		Value          []byte
		Partition      Partition
		PartitionCount PartitionCount
	}
	Client interface {
		io.Closer
		SendMessage(context.Context, *Message, chan<- error)
	}
	Processor interface {
		Process(context.Context, *Message) error
	}

	// Config holds the configuration of this package mounted from `application.yaml`.
	Config struct {
		MessageBroker struct {
			ConsumerGroup      string                 `yaml:"consumerGroup"`
			CertPath           string                 `yaml:"certPath"`
			URLs               []string               `yaml:"urls"` //nolint:tagliatelle // Nope.
			ConsumingTopics    []*ConsumerTopicConfig `yaml:"consumingTopics"`
			ProducingTopics    []*ProducerTopicConfig `yaml:"producingTopics"`
			Topics             []*TopicConfig         `yaml:"topics"`
			CreateTopics       bool                   `yaml:"createTopics"`
			DisableIdempotence bool                   `yaml:"disableIdempotence"`
			MaxPollRecords     int                    `yaml:"maxPollRecords"`
			MaxMessageBytes    int                    `yaml:"maxMessageBytes"`
		} `yaml:"messageBroker"`
	}
	TopicConfig struct {
		Name              string        `yaml:"name" json:"name"`
		CleanupPolicy     string        `yaml:"cleanupPolicy" json:"cleanupPolicy"`
		Partitions        uint64        `yaml:"partitions" json:"partitions"`
		ReplicationFactor uint64        `yaml:"replicationFactor" json:"replicationFactor"`
		Retention         time.Duration `yaml:"retention" json:"retention"`
	}
	ConsumerTopicConfig struct {
		Name                     string `yaml:"name" json:"name"`
		OneGoroutinePerPartition bool   `yaml:"oneGoroutinePerPartition" json:"oneGoroutinePerPartition"`
	}
	ProducerTopicConfig struct {
		Name string `yaml:"name" json:"name"`
	}
)

// Private API.

const (
	messageBrokerSchemaInitDeadline    = 30 * time.Second
	messageBrokerCloseDeadline         = 25 * time.Second
	messageBrokerProcessRecordDeadline = 30 * time.Second
	consumerRecordBatchBufferSize      = 100
)

type (
	// | messageBroker manages all operations and is exposed publicly as Client.
	messageBroker struct {
		cfg *Config
		*concurrentConsumer
		client    *kgo.Client
		admClient *kadm.Client
	}
	// | concurrentConsumer is responsible for managing the state and lifecycle of all partitionConsumers.
	concurrentConsumer struct {
		client                 *kgo.Client
		admClient              *kadm.Client
		consumingWg            *sync.WaitGroup
		mx                     *sync.Mutex
		consumers              *sync.Map // Is a map[Topic]map[Partition]*partitionConsumer.
		processors             map[Topic]Processor
		consumerTopicConfigs   map[Topic]*ConsumerTopicConfig //nolint:structcheck // Wrong.
		partitionCountPerTopic *sync.Map                      // Is a map[Topic]PartitionCount.
		cancel                 context.CancelFunc
		done                   bool
	}
	// | partitionConsumer is responsible for processing partition records.
	partitionConsumer struct {
		Processor
		*concurrentConsumer
		recordsChan    chan []*kgo.Record
		topic          Topic
		partition      Partition
		partitionCount PartitionCount
		done           bool
		closing        bool
	}
)
