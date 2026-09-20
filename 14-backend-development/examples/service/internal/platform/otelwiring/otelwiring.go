// Package otelwiring is chapter 3: process-global tracer setup,
// collapsed into an Init/shutdown pair. The tracer lives in a global
// registry (the OTel design), so the discipline is: init once in
// main, before any request can start; shut down last (after the HTTP
// server drains) so the final spans reach the collector.
package otelwiring

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

// ShutdownFn flushes and stops the tracer provider. Never nil: the
// no-op setup returns a no-op shutdown, so callers need no nil checks
// on the hot path (main's defer runs unconditionally).
type ShutdownFn func(context.Context) error

// Init configures the global tracer provider. OTEL_EXPORTER_OTLP_ENDPOINT
// selects the collector (SDK default localhost:4318); when it is
// empty, a no-op provider is installed so the service runs with zero
// infrastructure and zero code branches. OTEL_SDK_DISABLED=1 forces
// the no-op path even with an endpoint set (the kill switch).
func Init(ctx context.Context, serviceName, version string) (ShutdownFn, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" || os.Getenv("OTEL_SDK_DISABLED") == "1" {
		// No-op: still call Shutdown for symmetry; it is cheap.
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracehttp.New(ctx) // env-configured endpoint, batching
	if err != nil {
		return nil, fmt.Errorf("otel exporter: %w", err)
	}

	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		))
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	// Sampler: 100% in dev, head-sample in prod by env. Sampling is
	// a correctness tradeoff: errors worth tracing may be unsampled
	// (chapter 3's table); tail-sampling at the collector is the fix
	// when that matters.
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio()))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter), // async export: never spans in the request path
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)

	// W3C traceparent propagation (the default, made explicit): the
	// inbound middleware reads upstream traces; outbound clients
	// write them.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		// Bounded flush: a hung collector must not hang shutdown
		// (12 Section 5's grace-period discipline applies to exporters too).
		flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return errors.Join(tp.Shutdown(flushCtx))
	}, nil
}

func ratio() float64 {
	switch v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v {
	case "", "1":
		return 1.0
	default:
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err != nil || f <= 0 || f > 1 {
			return 1.0
		}
		return f
	}
}
