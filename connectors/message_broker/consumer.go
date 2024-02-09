// SPDX-License-Identifier: ice License 1.0

package messagebroker

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/ice-blockchain/wintr/log"
)

func (mb *messageBroker) startConsuming(ctx context.Context, cancel context.CancelFunc) {
	cctx, ccancel := context.WithCancel(ctx)
	mb.concurrentConsumer.cancel = ccancel
	defer func() {
		mb.shutdownConsumerGracefully()
		cancel()
	}()
	defer mb.concurrentConsumer.cancel()
	log.Info("message broker client started consuming...")
	var shouldStop bool
	for !shouldStop && cctx.Err() == nil {
		shouldStop = mb.pollRecords(cctx)
	}
}

func (mb *messageBroker) pollRecords(ctx context.Context) (shouldStop bool) {
	fetches := mb.client.PollRecords(ctx, mb.cfg.MessageBroker.MaxPollRecords)
	if err := fetches.Err0(); err != nil && (errors.Is(err, ctx.Err()) || errors.Is(err, kgo.ErrClientClosed)) {
		return true
	}
	fetches.EachTopic(func(fetchTopic kgo.FetchTopic) {
		partitionConsumers := mb.partitionConsumers(&fetchTopic)
		if partitionConsumers == nil {
			return
		}
		fetchTopic.EachPartition(func(fetchPartition kgo.FetchPartition) {
			if fetchPartition.Err != nil {
				log.Error(errors.Wrap(fetchPartition.Err, "[messageBroker] fetching records failed"), "topic", fetchTopic, "partition", fetchPartition)

				return
			}
			pc := mb.partitionConsumer(&fetchPartition, &fetchTopic, partitionConsumers)
			if pc == nil {
				return
			}
			mb.consumingWg.Add(len(fetchPartition.Records))
			pc.recordsChan <- fetchPartition.Records
		})
	})
	mb.consumingWg.Wait()
	mb.client.AllowRebalance()

	return false
}

func (mb *messageBroker) partitionConsumers(fetchedTopic *kgo.FetchTopic) *sync.Map {
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

	return topicConsumers.(*sync.Map) //nolint:forcetypeassert,revive // We know for sure.
}

func (mb *messageBroker) partitionConsumer(
	fetchedPartition *kgo.FetchPartition, fetchedTopic *kgo.FetchTopic, partitionConsumers *sync.Map,
) *partitionConsumer {
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
			mb.processNotFoundPartitions(fetchedTopic, *fetchedPartition)

			return nil
		}
	}
	if pc.(*partitionConsumer).closing { //nolint:forcetypeassert // We know for sure.
		log.Warn("partition consumer was closing", "partition", fetchedPartition.Partition)
		mb.mx.Lock()
		//nolint:gocritic,staticcheck // Because we just want to make sure we wait for any in progress state changes
		mb.mx.Unlock()
		mb.processNotFoundPartitions(fetchedTopic, *fetchedPartition)

		return nil
	}

	return pc.(*partitionConsumer) //nolint:forcetypeassert,revive // We know for sure.
}

func (mb *messageBroker) processNotFoundPartitions(fetchedTopic *kgo.FetchTopic, fetchPartitions ...kgo.FetchPartition) {
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
		mb.consumingWg.Add(len(records))
		pc.(*partitionConsumer).recordsChan <- records //nolint:forcetypeassert // We know for sure.
	}
	mb.revokePartitions(context.Background(), map[string][]int32{fetchedTopic.Topic: partitions})
}

func (mb *messageBroker) records(fetchedTopic *kgo.FetchTopic, ps []kgo.FetchPartition) (map[int32][]*kgo.Record, []int32) {
	var partitions []int32
	partitionRecords := make(map[int32][]*kgo.Record)
	partitionIterator := func(fetchPartition kgo.FetchPartition) {
		if fetchPartition.Err != nil {
			return
		}
		mb.consumingWg.Add(len(fetchPartition.Records))
		partitionRecords[fetchPartition.Partition] = fetchPartition.Records
		partitions = append(partitions, fetchPartition.Partition)
	}
	if len(ps) != 0 {
		for i := range ps {
			partitionIterator(ps[i])
		}
	} else {
		fetchedTopic.EachPartition(partitionIterator)
	}

	return partitionRecords, partitions
}

func (mb *messageBroker) shutdownConsumerGracefully() { //nolint:funlen,revive // .
	mb.mx.Lock()
	if mb.concurrentConsumer.done {
		mb.mx.Unlock()

		return
	}
	defer mb.client.CloseAllowingRebalance()
	defer mb.mx.Unlock()
	defer log.Info("message broker client stopped consuming")
	defer func() { mb.concurrentConsumer.done = true }()
	ctx, cancel := context.WithTimeout(context.Background(), messageBrokerCloseDeadline)
	defer cancel()
	mb.concurrentConsumer.consumers.Range(func(_, partitionConsumers any) bool {
		partitionConsumers.(*sync.Map).Range(func(_, pc any) bool { //nolint:forcetypeassert // We know for sure.
			pc.(*partitionConsumer).stop(ctx) //nolint:forcetypeassert // We know for sure.

			return true
		})

		return true
	})
	done := true
	cctx, ccancel := context.WithTimeout(context.Background(), messageBrokerCloseDeadline)
	defer ccancel()
	for cctx.Err() == nil {
		mb.concurrentConsumer.consumers.Range(func(_, partitionConsumers any) bool {
			partitionConsumers.(*sync.Map).Range(func(_, pc any) bool { //nolint:forcetypeassert // We know for sure.
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
	ccctx, cccancel := context.WithTimeout(context.Background(), messageBrokerCloseDeadline)
	defer cccancel()
	if err := mb.client.CommitUncommittedOffsets(ccctx); err != nil && !errors.Is(err, kgo.ErrClientClosed) {
		log.Error(errors.Wrap(err, "shutdownConsumerGracefully: closing: committing uncommitted offsets failed"))
	}
	if err := mb.client.Flush(ccctx); err != nil && !errors.Is(err, kgo.ErrClientClosed) {
		log.Error(errors.Wrap(err, "shutdownConsumerGracefully: message broker client flush failed"))
	}
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
func (c *concurrentConsumer) replaceConsumer(ctx context.Context, topic string, partition int32, partitionConsumers *sync.Map, pc any) {
	if pc != nil && !pc.(*partitionConsumer).done { //nolint:forcetypeassert // We know for sure.
		waitForClosingConsumerToFinish(ctx, pc)
	}
	pc = &partitionConsumer{
		concurrentConsumer: c,
		Processor:          c.processors[topic],
		recordsChan:        make(chan []*kgo.Record, consumerRecordBatchBufferSize),
		topic:              topic,
		partition:          partition,
	}
	if partitionCount, ok := c.partitionCountPerTopic.Load(topic); ok {
		pc.(*partitionConsumer).partitionCount = partitionCount.(int32) //nolint:forcetypeassert,errcheck // We know for sure.
	}

	partitionConsumers.Store(partition, pc)
	//nolint:contextcheck // We want to finish up the record processing if the parent context is cancelled(aka graceful shutdown).
	go pc.(*partitionConsumer).consume() //nolint:forcetypeassert // We know for sure.
}

func waitForClosingConsumerToFinish(ctx context.Context, pc any) {
	//nolint:revive // Its a loop.
	for pc.(*partitionConsumer).closing && ctx.Err() == nil && !pc.(*partitionConsumer).done { //nolint:forcetypeassert // We know for sure.
	}
}

func (c *concurrentConsumer) OnPartitionsLost(ctx context.Context, cl *kgo.Client, lost map[string][]int32) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.revokePartitions(ctx, lost)
	cCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second) //nolint:gomnd // .
	defer cancel()
	//nolint:contextcheck // Nope, we're trying to make sure commit happens, even if the parent context is cancelled.
	log.Error(errors.Wrap(cl.CommitUncommittedOffsets(cCtx), "handleLostPartitions: failed to CommitUncommittedOffsets"))
}

func (c *concurrentConsumer) revokePartitions(cctx context.Context, lost map[string][]int32) {
	ctx, cancel := context.WithTimeout(cctx, 25*time.Second) //nolint:gomnd // .
	defer cancel()
	log.Info("some partitions lost/revoked", "lostOrRevoked", lost)
	for topic, partitions := range lost {
		partitionConsumers, topicFound := c.consumers.Load(topic)
		if !topicFound {
			log.Warn("handleLostPartitions: no consumers found for topic", "topic", topic)

			continue
		}
		for _, partition := range partitions {
			if pc, ok := partitionConsumers.(*sync.Map).Load(partition); ok {
				pc.(*partitionConsumer).stop(ctx) //nolint:forcetypeassert // We know for sure.

				continue
			}
			log.Warn("handleLostPartitions: no consumers found for partition", "topic", topic, "partition", partition)
		}
	}
}

func (pc *partitionConsumer) stop(ctx context.Context) {
	if !pc.closing {
		pc.closing = true
		close(pc.recordsChan)
		waitForClosingConsumerToFinish(ctx, pc)
	}
}

func (pc *partitionConsumer) consume() {
	log.Info("started consuming from partition....", "topic", pc.topic, "partition", pc.partition)
	defer log.Info("stopped consuming from partition", "topic", pc.topic, "partition", pc.partition)
	defer func() {
		pc.done = true
	}()

	if pc.consumerTopicConfigs[pc.topic].OneGoroutinePerPartition {
		for records := range pc.recordsChan {
			pc.processRecords(records)
		}
	} else {
		for records := range pc.recordsChan {
			groupedByKey := make(map[string][]*kgo.Record, len(records))
			for _, record := range records {
				groupedByKey[string(record.Key)] = append(groupedByKey[string(record.Key)], record)
			}
			wg := new(sync.WaitGroup)
			wg.Add(len(groupedByKey))
			for k := range groupedByKey {
				go func(key string) {
					defer wg.Done()
					pc.processRecords(groupedByKey[key])
				}(k)
			}
			wg.Wait()
		}
	}
}

func (pc *partitionConsumer) processRecords(records []*kgo.Record) {
	for _, record := range records {
		pc.processRecord(record)
	}
}

func (pc *partitionConsumer) processRecord(record *kgo.Record) {
	pCtx, cancel := context.WithTimeout(context.Background(), messageBrokerProcessRecordDeadline)
	defer cancel()
	defer pc.consumingWg.Done()
	msg := &Message{
		Key:            string(record.Key),
		Value:          record.Value,
		Timestamp:      record.Timestamp,
		Headers:        extractHeaders(record),
		Partition:      pc.partition,
		PartitionCount: pc.partitionCount,
		Topic:          pc.topic,
	}
	log.Error(errors.Wrap(pc.Process(pCtx, msg), "could not process new message"),
		"key", msg.Key,
		"value", string(msg.Value),
		"Timestamp", record.Timestamp,
		"headers", msg.Headers,
		"partition", msg.Partition,
		"partitionCount", msg.PartitionCount,
		"topic", msg.Topic)
}

func extractHeaders(record *kgo.Record) map[string]string {
	headers := make(map[string]string, len(record.Headers))
	if len(record.Headers) != 0 {
		for i := range record.Headers {
			headers[record.Headers[i].Key] = string(record.Headers[i].Value)
		}
	}

	return headers
}
