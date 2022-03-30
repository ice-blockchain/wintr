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
	Partition = int32
	Topic     = string
	Message   struct {
		Headers   map[string]string
		Key       string
		Topic     string
		Value     []byte
		Partition int32
	}
	Client interface {
		io.Closer
		SendMessage(context.Context, *Message, chan<- error)
	}
	Processor interface {
		Process(context.Context, *Message) error
	}
)

// Private API.

const (
	messageBrokerSchemaInitDeadline = 30 * time.Second
	messageBrokerCloseDeadline      = 25 * time.Second
	messageBrokerCommitDeadline     = 25 * time.Second
	messageBrokerConsumeDeadline    = 30 * time.Second
	consumerRecordBufferSize        = 100
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	// | messageBroker manages all operations and is exposed publicly as Client.
	messageBroker struct {
		*concurrentConsumer
		client    *kgo.Client
		admClient *kadm.Client
	}
	// | concurrentConsumer is responsible for managing the state and lifecycle of all partitionConsumers.
	concurrentConsumer struct {
		mx         *sync.Mutex
		consumers  *sync.Map // Is a map[Topic]map[Partition]*partitionConsumer.
		processors map[Topic]Processor
		cancel     context.CancelFunc
	}
	// | partitionConsumer is responsible for processing partition records.
	partitionConsumer struct {
		Processor
		recordsChan chan []*kgo.Record
		topic       Topic
		partition   Partition
		done        bool
		closing     bool
	}
	// | config holds the configuration of this package mounted from `application.yaml`.
	config struct {
		MessageBroker struct {
			ConsumerGroup string   `yaml:"consumerGroup"`
			CertPath      string   `yaml:"certPath"`
			URLs          []string `yaml:"urls"`
			Topics        []struct {
				Name              string        `yaml:"name" json:"name"`
				CleanupPolicy     string        `yaml:"cleanupPolicy" json:"cleanupPolicy"`
				Partitions        uint64        `yaml:"partitions" json:"partitions"`
				ReplicationFactor uint64        `yaml:"replicationFactor" json:"replicationFactor"`
				Retention         time.Duration `yaml:"retention" json:"retention"`
			} `yaml:"topics"`
			CreateTopics bool `yaml:"createTopics"`
		} `yaml:"messageBroker"`
	}
)
