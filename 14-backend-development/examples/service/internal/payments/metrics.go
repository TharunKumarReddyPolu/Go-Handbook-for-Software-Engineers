package payments

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// PaymentMetrics is chapter 2's business signal bundle: what the
// payments product team charts, as opposed to the RED signals the
// platform emits for every HTTP route. Domain-owned, registered on
// the injectable registry; nil-safe so tests without metrics keep
// working (the instrumentation is a wrapper, not a dependency).
type PaymentMetrics struct {
	charged    *prometheus.CounterVec // payments charged, by currency
	failed     *prometheus.CounterVec // business failures (declines), by reason
	chargeOTel metric.Float64Counter  // the same charged count via OTel (ch 3's one-emitter rule)
	tracer     trace.Tracer
}

// NewPaymentMetrics registers the business collectors on reg. The
// OTel meter comes from the global MeterProvider (no-op until main
// configures one, exactly like the tracer).
func NewPaymentMetrics(reg prometheus.Registerer) *PaymentMetrics {
	charged := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "payments",
			Name:      "charged_total",
			Help:      "Payments successfully charged.",
		},
		[]string{"currency"},
	)
	failed := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "payments",
			Name:      "failed_total",
			Help:      "Payments refused for business reasons (not 4xx traffic noise).",
		},
		[]string{"reason"},
	)

	reg.MustRegister(charged, failed)

	otelCharged, _ := otel.Meter("payments").Float64Counter("payments.charged",
		metric.WithDescription("Mirror of payments_charged_total for exemplar linkage (chapter 3)."))

	return &PaymentMetrics{
		charged:    charged,
		failed:     failed,
		chargeOTel: otelCharged,
	}
}

// WithTracer sets the span source; called by main after otelwiring.Init.
func (m *PaymentMetrics) WithTracer(t trace.Tracer) *PaymentMetrics {
	m.tracer = t
	return m
}

// nil-safe emit helpers: instrumentation must never break the
// business path it observes (chapter 2's rule).

func (m *PaymentMetrics) emitCharged(ctx context.Context, currency string, amountMinor int64) {
	if m == nil {
		return
	}
	m.charged.WithLabelValues(currency).Inc()
	if m.chargeOTel != nil {
		m.chargeOTel.Add(ctx, 1, metric.WithAttributes(
			attribute.String("payment.currency", currency),
			attribute.Int64("payment.amount_minor", amountMinor),
		))
	}
}

func (m *PaymentMetrics) emitFailed(ctx context.Context, reason string) {
	if m == nil {
		return
	}
	m.failed.WithLabelValues(reason).Add(1)
}

// chargeSpan is chapter 3's business span: the operation the finance
// team names in dashboards, carrying business attributes.
func (m *PaymentMetrics) chargeSpan(ctx context.Context, in ChargeInput) (context.Context, trace.Span) {
	if m == nil || m.tracer == nil {
		return ctx, trace.SpanFromContext(ctx) // valid no-op span
	}
	ctx, span := m.tracer.Start(ctx, "payments.charge",
		trace.WithAttributes(
			attribute.String("payment.currency", in.Currency),
			attribute.Int64("payment.amount_minor", in.AmountMinor),
		))
	return ctx, span
}
