// producer.go adapts franz-go for the order-events processor. The
// durability/batching choices mirror 18-kafka-with-go/03-producer-consumer.md;
// the domain logic stays in service.go, unaware of this file.
package main

import (
	"context"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// NewProducer builds the durability-critical producer:
//   - acks=all: writes are durable when all in-sync replicas have them
//   - zstd compression: order-of-magnitude network reduction on JSON payloads
//   - bounded buffer: MaxBufferedRecords is the backpressure surface
//
// Config errors panic at startup (programmer error, fail fast); delivery
// errors surface per-record through ProduceWithResult (operational).
func NewProducer(brokers []string) *kgo.Client {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.ProducerBatchCompression(kgo.ZstdCompression()),
		kgo.ProducerBatchMaxBytes(1<<20), // 1 MiB batches
		kgo.MaxBufferedRecords(10_000),   // bounded: backpressure, not queue-forever
		kgo.RecordDeliveryTimeout(15*time.Second),
	)
	if err != nil {
		panic(err)
	}
	return cl
}

// ProduceWithResult sends one record and waits for its delivery outcome,
// bounded by the caller's context. The callback runs on the client's
// delivery goroutine; the channel handoff makes the wait ctx-aware.
func ProduceWithResult(ctx context.Context, cl *kgo.Client, topic, key string, value []byte) error {
	ch := make(chan error, 1)
	cl.Produce(ctx, &kgo.Record{Topic: topic, Key: []byte(key), Value: value},
		func(r *kgo.Record, err error) { ch <- err })
	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
