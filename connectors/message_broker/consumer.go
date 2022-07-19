// SPDX-License-Identifier: BUSL-1.1

package messagebroker

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/ice-blockchain/wintr/log"
)

//nolint:revive // Because of the nested x.EachXX, but there's no better way.
func (mb *messageBroker) startConsuming(ctx context.Context, cancel context.CancelFunc) {
	mb.cancel = cancel
	defer mb.shutdownGracefully()
	log.Info("message broker client started consuming...")
	for ctx.Err() == nil {
		fetches := mb.client.PollRecords(ctx, cfg.MessageBroker.MaxPollRecords)
		if fetches.IsClientClosed() {
			return
		}
		var recordsFetched int
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			recordsFetched += len(p.Records)
		})
		mb.consumingWg.Add(recordsFetched)
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
		mb.consumingWg.Wait()
		mb.client.AllowRebalance()
	}
}

func (mb *messageBroker) partitionConsumers(fetchedTopic kgo.FetchTopic) *sync.Map {
	topicConsumers, foundTopicConsumers := mb.consumers.Load(fetchedTopic.Topic)
	if !foundTopicConsumers {
		mb.mx.Lock()
		//nolint:gocritic,staticcheck // Because we just want to make sure we wait for any in progress state changes
		mb.mx.Unlock()
		topicConsumers, foundTopicConsumers = mb.consumers.Load(fetchedTopic.Topic)
		if !foundTopicConsumers {
			log.Warn("no consumer for topic found", "topic", fetchedTopic.Topic)
			mb.processNotFoundPartitions(fetchedTopic, fetchedTopic.Partitions...)

			return nil
		}
	}

	return topicConsumers.(*sync.Map) //nolint:forcetypeassert // We know for sure.
}

func (mb *messageBroker) partitionConsumer(fetchedPartition kgo.FetchPartition, fetchedTopic kgo.FetchTopic, partitionConsumers *sync.Map) *partitionConsumer {
	if fetchedPartition.Err != nil {
		return nil
	}

	pc, foundPartitionConsumer := partitionConsumers.Load(fetchedPartition.Partition)
	if !foundPartitionConsumer {
		mb.mx.Lock()
		//nolint:gocritic,staticcheck // Because we just want to make sure we wait for any in progress state changes
		mb.mx.Unlock()
		pc, foundPartitionConsumer = partitionConsumers.Load(fetchedPartition.Partition)
		if !foundPartitionConsumer {
			log.Warn("no consumer for partition found", "partition", fetchedPartition.Partition)
			mb.processNotFoundPartitions(fetchedTopic, fetchedPartition)

			return nil
		}
	}
	if pc.(*partitionConsumer).closing { //nolint:forcetypeassert // We know for sure.
		log.Warn("partition consumer was closing", "partition", fetchedPartition.Partition)
		mb.processNotFoundPartitions(fetchedTopic, fetchedPartition)

		return nil
	}

	return pc.(*partitionConsumer) //nolint:forcetypeassert // We know for sure.
}

func (mb *messageBroker) processNotFoundPartitions(fetchedTopic kgo.FetchTopic, fetchPartitions ...kgo.FetchPartition) {
	mb.mx.Lock()
	defer mb.mx.Unlock()
	partitionRecords, partitions := mb.records(fetchedTopic, fetchPartitions)
	if len(partitionRecords) == 0 {
		return
	}
	mb.assignPartitions(context.Background(), map[string][]int32{fetchedTopic.Topic: partitions})
	partitionConsumers, _ := mb.consumers.Load(fetchedTopic.Topic)
	for partition, records := range partitionRecords {
		pc, _ := partitionConsumers.(*sync.Map).Load(partition)
		pc.(*partitionConsumer).recordsChan <- records //nolint:forcetypeassert // We know for sure.
	}
	mb.revokePartitions(context.Background(), map[string][]int32{fetchedTopic.Topic: partitions})
}

func (*messageBroker) records(fetchedTopic kgo.FetchTopic, ps []kgo.FetchPartition) (map[int32][]*kgo.Record, []int32) {
	var partitions []int32
	partitionRecords := make(map[int32][]*kgo.Record)
	partitionIterator := func(p kgo.FetchPartition) {
		if p.Err != nil {
			return
		}
		partitionRecords[p.Partition] = p.Records
		partitions = append(partitions, p.Partition)
	}
	if len(ps) != 0 {
		for _, p := range ps {
			partitionIterator(p)
		}
	} else {
		fetchedTopic.EachPartition(partitionIterator)
	}

	return partitionRecords, partitions
}

func (mb *messageBroker) shutdownGracefully() {
	mb.mx.Lock()
	defer mb.mx.Unlock()
	defer mb.cancel()
	defer log.Info("message broker client stopped consuming")
	mb.concurrentConsumer.consumers.Range(func(_, partitionConsumers interface{}) bool {
		partitionConsumers.(*sync.Map).Range(func(_, pc interface{}) bool { //nolint:forcetypeassert // We know for sure.
			pc.(*partitionConsumer).stop() //nolint:forcetypeassert // We know for sure.

			return true
		})

		return true
	})
}

func (mb *messageBroker) closeAndWaitForConsumersToFinishProcessing(ctx context.Context) (err error) {
	mb.shutdownGracefully()
	err = errors.Wrap(mb.client.CommitUncommittedOffsets(ctx), "closing: committing uncommitted offsets failed")
	defer func() {
		if err == nil {
			err = ctx.Err()
		}
	}()
	done := true
	for ctx.Err() != nil {
		mb.concurrentConsumer.consumers.Range(func(_, partitionConsumers interface{}) bool {
			partitionConsumers.(*sync.Map).Range(func(_, pc interface{}) bool { //nolint:forcetypeassert // We know for sure.
				if !pc.(*partitionConsumer).done { //nolint:forcetypeassert // We know for sure.
					done = false
				}

				return pc.(*partitionConsumer).done //nolint:forcetypeassert // We know for sure.
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
			if !loaded || pc.(*partitionConsumer).closing { //nolint:forcetypeassert // We know for sure.
				c.replaceConsumer(ctx, topic, partition, partitionConsumers.(*sync.Map), pc) //nolint:forcetypeassert // We know for sure.
			}
		}
	}
}

//nolint:revive // Intended.
func (c *concurrentConsumer) replaceConsumer(ctx context.Context, topic string, partition int32, partitionConsumers *sync.Map, pc interface{}) {
	if pc != nil && !pc.(*partitionConsumer).done { //nolint:forcetypeassert // We know for sure.
		waitForClosingConsumerToFinish(ctx, pc)
	}
	pc = &partitionConsumer{
		concurrentConsumer: c,
		Processor:          c.processors[topic],
		recordsChan:        make(chan []*kgo.Record, consumerRecordBufferSize),
		topic:              topic,
		partition:          partition,
	}
	if partitionCount, ok := c.partitionCountPerTopic.Load(topic); ok {
		pc.(*partitionConsumer).partitionCount = partitionCount.(int32) //nolint:forcetypeassert,errcheck // We know for sure.
	}

	partitionConsumers.Store(partition, pc)
	go pc.(*partitionConsumer).consume() //nolint:forcetypeassert // We know for sure.
}

func waitForClosingConsumerToFinish(ctx context.Context, pc interface{}) {
	//nolint:revive // Its a loop.
	for pc.(*partitionConsumer).closing && ctx.Err() == nil && !pc.(*partitionConsumer).done { //nolint:forcetypeassert // We know for sure.
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
				pc.(*partitionConsumer).stop() //nolint:forcetypeassert // We know for sure.

				continue
			}
			log.Warn("handleLostPartitions: no consumers found for partition", "topic", topic, "partition", partition)
		}
	}
}

func (pc *partitionConsumer) stop() {
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
		if cfg.MessageBroker.OneGoroutinePerPartition {
			for _, record := range records {
				pc.processRecord(record)
			}

			continue
		}
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
		pc.processRecord(record)
	}
}

func (pc *partitionConsumer) processRecord(record *kgo.Record) {
	pCtx, cancel := context.WithTimeout(context.Background(), messageBrokerConsumeDeadline)
	defer cancel()
	defer pc.consumingWg.Done()
	msg := &Message{
		Key:            string(record.Key),
		Value:          record.Value,
		Headers:        extractHeaders(record),
		Partition:      pc.partition,
		PartitionCount: pc.partitionCount,
		Topic:          pc.topic,
	}
	log.Error(errors.Wrap(pc.Process(pCtx, msg), "could not process new message"),
		"key", msg.Key,
		"value", string(msg.Value),
		"headers", msg.Headers,
		"partition", msg.Partition,
		"partitionCount", msg.PartitionCount,
		"topic", msg.Topic)
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
