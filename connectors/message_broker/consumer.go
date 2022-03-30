package messagebroker

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/ICE-Blockchain/wintr/log"
)

//nolint:gocognit // Because of the nested x.EachXX, but there's no better way.
func (mb *messageBroker) startConsuming(ctx context.Context, cancel context.CancelFunc) {
	mb.cancel = cancel
	defer mb.close()
	log.Info("message broker client started consuming...")
	for ctx.Err() == nil {
		fetches := mb.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}
		fetches.EachError(func(t string, p int32, err error) {
			log.Error(errors.Wrap(err, "[messageBroker] fetching records failed"), "topic", t, "partition", p)
		})
		fetches.EachTopic(func(t kgo.FetchTopic) {
			if partitionConsumers := mb.partitionConsumers(t); partitionConsumers != nil {
				t.EachPartition(func(p kgo.FetchPartition) {
					if pc := mb.partitionConsumer(p, t, partitionConsumers); pc != nil {
						pc.recordsChan <- p.Records
					}
				})
			}
		})
	}
}

func (mb *messageBroker) partitionConsumers(t kgo.FetchTopic) *sync.Map {
	topicConsumers, foundTopicConsumers := mb.consumers.Load(t.Topic)
	if !foundTopicConsumers {
		mb.mx.Lock()
		//nolint:gocritic // Because we just want to make sure we wait for any in progress state changes
		mb.mx.Unlock()
		topicConsumers, foundTopicConsumers = mb.consumers.Load(t.Topic)
		if !foundTopicConsumers {
			log.Warn("no consumer for topic found", "topic", t.Topic)
			mb.processNotFoundPartitions(t, t.Partitions...)

			return nil
		}
	}

	return topicConsumers.(*sync.Map)
}

func (mb *messageBroker) partitionConsumer(p kgo.FetchPartition, t kgo.FetchTopic, partitionConsumers *sync.Map) *partitionConsumer {
	if p.Err != nil {
		return nil
	}

	pc, foundPartitionConsumer := partitionConsumers.Load(p.Partition)
	if !foundPartitionConsumer {
		mb.mx.Lock()
		//nolint:gocritic // Because we just want to make sure we wait for any in progress state changes
		mb.mx.Unlock()
		pc, foundPartitionConsumer = partitionConsumers.Load(p.Partition)
		if !foundPartitionConsumer {
			log.Warn("no consumer for partition found", "partition", p.Partition)
			mb.processNotFoundPartitions(t, p)

			return nil
		}
	}
	if pc.(*partitionConsumer).closing {
		log.Warn("partition consumer was closing", "partition", p.Partition)
		mb.processNotFoundPartitions(t, p)

		return nil
	}

	return pc.(*partitionConsumer)
}

func (mb *messageBroker) processNotFoundPartitions(t kgo.FetchTopic, ps ...kgo.FetchPartition) {
	mb.mx.Lock()
	defer mb.mx.Unlock()
	partitionRecords, partitions := mb.records(t, ps)
	if len(partitionRecords) == 0 {
		return
	}
	mb.assignPartitions(context.Background(), map[string][]int32{t.Topic: partitions})
	partitionConsumers, _ := mb.consumers.Load(t.Topic)
	for partition, records := range partitionRecords {
		pc, _ := partitionConsumers.(*sync.Map).Load(partition)
		pc.(*partitionConsumer).recordsChan <- records
	}
	mb.revokePartitions(context.Background(), map[string][]int32{t.Topic: partitions})
}

func (mb *messageBroker) records(t kgo.FetchTopic, ps []kgo.FetchPartition) (map[int32][]*kgo.Record, []int32) {
	var partitions []int32
	partitionRecords := make(map[int32][]*kgo.Record)
	partitionIterator := func(p kgo.FetchPartition) {
		if p.Err != nil {
			return
		}
		partitionRecords[p.Partition] = p.Records
		partitions = append(partitions, p.Partition)
	}
	if len(ps) == 0 {
		for _, p := range ps {
			partitionIterator(p)
		}
	} else {
		t.EachPartition(partitionIterator)
	}

	return partitionRecords, partitions
}

func (mb *messageBroker) close() {
	mb.mx.Lock()
	defer mb.mx.Unlock()
	defer mb.cancel()
	defer log.Info("message broker client stopped consuming")
	mb.concurrentConsumer.consumers.Range(func(_, partitionConsumers interface{}) bool {
		partitionConsumers.(*sync.Map).Range(func(_, pc interface{}) bool {
			pc.(*partitionConsumer).close()

			return true
		})

		return true
	})
}

func (mb *messageBroker) closeAndWaitForConsumersToFinishProcessing(ctx context.Context) (err error) {
	mb.close()
	err = errors.Wrap(mb.client.CommitUncommittedOffsets(ctx), "closing: committing uncommitted offsets failed")
	defer func() {
		if err == nil {
			err = ctx.Err()
		}
	}()
	done := true
	for ctx.Err() != nil {
		mb.concurrentConsumer.consumers.Range(func(_, partitionConsumers interface{}) bool {
			partitionConsumers.(*sync.Map).Range(func(_, pc interface{}) bool {
				if !pc.(*partitionConsumer).done {
					done = false
				}

				return pc.(*partitionConsumer).done
			})

			return done
		})
		if done {
			break
		}
	}

	return
}

func (c *concurrentConsumer) OnPartitionsAssigned(ctx context.Context, _ *kgo.Client, assigned map[string][]int32) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.assignPartitions(ctx, assigned)
}

func (c *concurrentConsumer) assignPartitions(ctx context.Context, assigned map[string][]int32) {
	if ctx.Err() != nil {
		return
	}
	log.Info("new partitions assigned", "assigned", assigned)

	for topic, partitions := range assigned {
		partitionConsumers, _ := c.consumers.LoadOrStore(topic, &sync.Map{})
		for _, partition := range partitions {
			pc, loaded := partitionConsumers.(*sync.Map).Load(partition)
			if !loaded || pc.(*partitionConsumer).closing {
				c.replaceConsumer(ctx, topic, partition, partitionConsumers.(*sync.Map), pc)
			}
		}
	}
}

func (c *concurrentConsumer) replaceConsumer(ctx context.Context, topic string, partition int32, partitionConsumers *sync.Map, pc interface{}) {
	if pc != nil && !pc.(*partitionConsumer).done {
		waitForClosingConsumerToFinish(ctx, pc)
	}
	pc = &partitionConsumer{
		Processor:   c.processors[topic],
		recordsChan: make(chan []*kgo.Record, consumerRecordBufferSize),
		topic:       topic,
		partition:   partition,
	}
	partitionConsumers.Store(partition, pc)
	go pc.(*partitionConsumer).consume()
}

func waitForClosingConsumerToFinish(ctx context.Context, pc interface{}) {
	for pc.(*partitionConsumer).closing && ctx.Err() == nil && !pc.(*partitionConsumer).done {
	}
}

func (c *concurrentConsumer) OnPartitionsLost(ctx context.Context, cl *kgo.Client, lost map[string][]int32) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.revokePartitions(ctx, lost)
	cCtx, cancel := context.WithTimeout(context.Background(), messageBrokerCommitDeadline)
	defer cancel()
	log.Error(errors.Wrap(cl.CommitUncommittedOffsets(cCtx), "handleLostPartitions: failed to CommitUncommittedOffsets"))
}

func (c *concurrentConsumer) revokePartitions(_ context.Context, lost map[string][]int32) {
	log.Info("some partitions lost/revoked", "lostOrRevoked", lost)
	for topic, partitions := range lost {
		partitionConsumers, topicFound := c.consumers.Load(topic)
		if !topicFound {
			log.Warn("handleLostPartitions: no consumers found for topic", "topic", topic)

			continue
		}
		for _, partition := range partitions {
			if pc, ok := partitionConsumers.(*sync.Map).Load(partition); ok {
				pc.(*partitionConsumer).close()

				continue
			}
			log.Warn("handleLostPartitions: no consumers found for partition", "topic", topic, "partition", partition)
		}
	}
}

func (pc *partitionConsumer) close() {
	if !pc.closing {
		pc.closing = true
		close(pc.recordsChan)
	}
}

func (pc *partitionConsumer) consume() {
	log.Info("started consuming from partition....", "topic", pc.topic, "partition", pc.partition)
	defer log.Info("stopped consuming from partition", "topic", pc.topic, "partition", pc.partition)
	defer func() { pc.done = true }()

	for records := range pc.recordsChan {
		groupedByKey := make(map[string][]*kgo.Record, len(records))
		for _, record := range records {
			groupedByKey[string(record.Key)] = append(groupedByKey[string(record.Key)], record)
		}
		wg := new(sync.WaitGroup)
		for _, rs := range groupedByKey {
			wg.Add(1)
			go pc.processSameKeyRecords(wg, rs)
		}
		wg.Wait()
	}
}

func (pc *partitionConsumer) processSameKeyRecords(wg *sync.WaitGroup, recordsForSameKey []*kgo.Record) {
	defer wg.Done()
	for _, record := range recordsForSameKey {
		pCtx, cancel := context.WithTimeout(context.Background(), messageBrokerConsumeDeadline)
		m := &Message{
			Key:       string(record.Key),
			Value:     record.Value,
			Headers:   extractHeaders(record),
			Partition: pc.partition,
			Topic:     pc.topic,
		}
		log.Error(errors.Wrap(pc.Process(pCtx, m), "could not process new message"),
			"key", m.Key, "value", string(m.Value), "headers", m.Headers, "partition", m.Partition, "topic", m.Topic)
		cancel()
	}
}

func extractHeaders(record *kgo.Record) map[string]string {
	headers := make(map[string]string, len(record.Headers))
	if len(record.Headers) != 0 {
		for _, header := range record.Headers {
			headers[header.Key] = string(header.Value)
		}
	}

	return headers
}
