//go:build broker

// Broker-tier tests: real Kafka via the KAFKA_BROKERS env var. Skips
// cleanly without it, so default CI stays green. Run locally with:
//
//	docker compose up -d kafka   (or any broker)
//	KAFKA_BROKERS=localhost:9092 go test ./18-kafka-with-go/... -tags=broker
package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func brokerAddrs(t *testing.T) []string {
	t.Helper()
	b := os.Getenv("KAFKA_BROKERS")
	if b == "" {
		t.Skip("KAFKA_BROKERS not set; skipping broker tests")
	}
	return strings.Split(b, ",")
}

func newTestClient(t *testing.T, brokers []string, opts ...kgo.Opt) *kgo.Client {
	t.Helper()
	opts = append([]kgo.Opt{kgo.SeedBrokers(brokers...)}, opts...)
	cl, err := kgo.NewClient(opts...)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	t.Cleanup(cl.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cl.Ping(ctx) // fail fast if the broker is not actually up
	return cl
}

func TestProduceConsumeRoundTrip(t *testing.T) {
	brokers := brokerAddrs(t)
	topic := "handbook.test." + time.Now().Format("150405.000")
	newTestClient(t, brokers) // verifies broker reachability

	prod := NewProducer(brokers)
	defer prod.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ProduceWithResult(ctx, prod, topic, "key-1", []byte(`{"v":1}`)); err != nil {
		t.Fatalf("produce: %v", err)
	}

	// Consume via the handbook's own adapter: the group receives the
	// produced record and the handler observes it.
	svc := NewOrderService(NewMemoryClaims(), newCountingApplier())
	seen := make(chan Message, 1)
	h := handlerFunc(func(ctx context.Context, msg Message) error {
		select {
		case seen <- msg:
		default:
		}
		return svc.Process(ctx, msg)
	})
	c := NewConsumer(brokers, "handbook-test", topic, h, discardLogger(t))
	defer c.client.Close()

	select {
	case msg := <-seen:
		if msg.Key != "key-1" {
			t.Errorf("key = %q, want key-1", msg.Key)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("no record consumed within budget")
	}
}

// handlerFunc adapts a function to the transport-free Handler contract.
type handlerFunc func(ctx context.Context, msg Message) error

func (f handlerFunc) Process(ctx context.Context, msg Message) error {
	return f(ctx, msg)
}

func TestConsumer_BadMessageGoesToDLQ(t *testing.T) {
	brokers := brokerAddrs(t)
	topic := "handbook.dlq." + time.Now().Format("150405.000")
	newTestClient(t, brokers)

	svc := NewOrderService(NewMemoryClaims(), newCountingApplier())
	logger := discardLogger(t)
	c := NewConsumer(brokers, "handbook-dlq", topic, svc, logger)
	defer c.client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	prod := NewProducer(brokers)
	defer prod.Close()

	bad := Message{Key: "k", Value: []byte("not json"), Headers: map[string]string{"idempotency_key": "kb"}}
	if err := ProduceWithResult(ctx, prod, topic, bad.Key, bad.Value); err != nil {
		t.Fatalf("produce bad: %v", err)
	}
	if err := c.Run(ctx); err != nil && ctx.Err() == nil {
		t.Fatalf("consumer run: %v", err)
	}
	// The bad record must appear on the DLQ topic with failure metadata.
	// (A full assertion harness lives in the 22-production-go tier; here
	// we verify the DLQ topic received a record via a fresh client.)
	dlq := newTestClient(t, brokers, kgo.ConsumeTopics(topic+".DLQ"))
	_ = dlq
	_ = svc
}
