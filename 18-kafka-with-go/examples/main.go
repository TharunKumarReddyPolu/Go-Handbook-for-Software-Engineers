// Command order-processor is the runnable wiring of the Kafka example:
// environment configuration, producer/consumer construction, and the
// graceful-shutdown flow from 08-concurrency stage 5. Domain logic
// lives in service.go; client adapters in producer.go / consumer.go.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func envBrokers() []string {
	b := os.Getenv("KAFKA_BROKERS")
	if b == "" {
		b = "localhost:9092"
	}
	return strings.Split(b, ",")
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	brokers := envBrokers()
	topic := "orders"

	claims := NewMemoryClaims() // production: DB-backed ClaimStore
	svc := NewOrderService(claims, &LoggingApplier{logger: logger})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	consumer := NewConsumer(brokers, "order-processor", topic, svc, logger)
	defer consumer.client.Close()

	producer := NewProducer(brokers)
	defer producer.Close()

	logger.Info("running", "brokers", brokers, "topic", topic)
	if err := consumer.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("consumer exited", "err", err)
		os.Exit(1)
	}
	logger.Info("drained and stopped")
}

// LoggingApplier stands in for the real side effect (persist, charge,
// notify). The fintech section replaces it with a ledger.
type LoggingApplier struct{ logger *slog.Logger }

func (a *LoggingApplier) Apply(_ context.Context, order OrderPayload) error {
	a.logger.Info("applied order", "order_id", order.OrderID, "account", order.AccountID, "amount_minor", order.AmountMinor)
	return nil
}
