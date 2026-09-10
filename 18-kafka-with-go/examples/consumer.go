// consumer.go adapts franz-go for the order-events processor: the
// poll -> process -> commit loop with process-then-commit ordering
// (at-least-once), bounded batches, and clean drain on cancellation.
// The discipline documented in 18-kafka-with-go/03-producer-consumer.md.
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Consumer wires the transport-free Handler into a Kafka consumer group.
type Consumer struct {
	client      *kgo.Client
	h           Handler
	logger      *slog.Logger
	maxAttempts int
	dlqTopic    string
}

// NewConsumer builds a group consumer. Config errors panic at startup.
func NewConsumer(brokers []string, group, topic string, h Handler, logger *slog.Logger) *Consumer {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
		kgo.FetchMaxBytes(16<<20),
		// Process-then-commit is the default contract here; auto-commit
		// stays OFF so offsets only advance when the loop below commits.
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(), // drain-before-commit across rebalances
	)
	if err != nil {
		panic(err)
	}
	return &Consumer{
		client:      cl,
		h:           h,
		logger:      logger,
		maxAttempts: 3,
		dlqTopic:    topic + ".DLQ",
	}
}

// Run consumes until ctx is cancelled or the client closes. The loop
// shape: bounded fetch, per-partition processing (preserves ordering
// within a partition), commit after the batch completes.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		fetches := c.client.PollRecords(ctx, 64)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			c.drainAndCommit(ctx)
			return ctx.Err()
		}
		fetches.EachError(func(t string, p int32, err error) {
			c.logger.ErrorContext(ctx, "fetch error", "topic", t, "partition", p, "err", err)
		})

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			c.processPartition(ctx, p)
		})

		// Commit AFTER processing: at-least-once. BlockRebalanceOnPoll
		// guarantees no partition is revoked between poll and commit.
		c.client.CommitRecords(ctx, recordsOf(fetches)...)
	}
}

// processPartition runs bounded retries per record, then DLQs exhausts.
func (c *Consumer) processPartition(ctx context.Context, p kgo.FetchTopicPartition) {
	for _, rec := range p.Records {
		msg := adapt(rec)
		var lastErr error
		for attempt := 0; attempt < c.maxAttempts; attempt++ {
			err := c.h.Process(ctx, msg)
			if err == nil {
				lastErr = nil
				break
			}
			var term *TerminalError
			if asTerminal(err, &term) { // retries cannot fix this
				lastErr = err
				break
			}
			lastErr = err
			select {
			case <-time.After(backoff(attempt)):
			case <-ctx.Done():
				return // cancel beats retry; uncommitted work reprocesses
			}
		}
		if lastErr != nil {
			c.toDeadLetter(ctx, msg, lastErr)
		}
	}
}

func (c *Consumer) toDeadLetter(ctx context.Context, msg Message, cause error) {
	headers := []kgo.RecordHeader{
		{Key: "dlq_error", Value: []byte(cause.Error())},
		{Key: "dlq_at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
		{Key: "dlq_service", Value: []byte("order-processor")},
	}
	for k, v := range msg.Headers {
		headers = append(headers, kgo.RecordHeader{Key: "orig_" + k, Value: []byte(v)}) // preserve identity for replay
	}
	cl := c.client
	cl.Produce(ctx, &kgo.Record{
		Topic:   c.dlqTopic,
		Key:     []byte(msg.Key),
		Value:   msg.Value,
		Headers: headers,
	}, func(r *kgo.Record, err error) {
		if err != nil {
			c.logger.ErrorContext(ctx, "DLQ publish failed", "key", msg.Key, "err", err)
		}
	})
}

// drainAndCommit gives in-flight work a final bounded commit window on
// shutdown. The service-level handler contract (no commits while work
// is in flight) holds because BlockRebalanceOnPoll serializes here.
func (c *Consumer) drainAndCommit(ctx context.Context) {
	dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	c.client.CommitMarkedOffsets(dctx)
}

func adapt(rec *kgo.Record) Message {
	headers := make(map[string]string, len(rec.Headers))
	for _, h := range rec.Headers {
		headers[h.Key] = string(h.Value)
	}
	return Message{Key: string(rec.Key), Value: rec.Value, Headers: headers}
}

// backoff spaces retries; linear here is deliberate: bounded, simple,
// and jitter is unnecessary at three attempts within one partition.
func backoff(attempt int) time.Duration {
	return time.Duration(attempt+1) * 100 * time.Millisecond
}

// asTerminal mirrors errors.As for the TerminalError contract.
func asTerminal(err error, target **TerminalError) bool {
	for err != nil {
		if t, ok := err.(*TerminalError); ok {
			*target = t
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func recordsOf(fetches kgo.Fetches) []*kgo.Record {
	var rs []*kgo.Record
	fetches.EachPartition(func(p kgo.FetchTopicPartition) {
		rs = append(rs, p.Records...)
	})
	return rs
}
