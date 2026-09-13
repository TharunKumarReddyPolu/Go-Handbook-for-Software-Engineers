// Package obshttp is the observability middleware from section 20:
// metrics (RED, Prometheus format), distributed tracing (OTel), and
// the log-correlation tie between them. Platform layer: knows HTTP,
// not the domain.
//
// The RED method (chapter 2): every instrumented request produces
// exactly one Rate increment, one Errors increment, and one Duration
// observation. Route() uses the matched pattern ("/payments/{id}"),
// never the raw path, so cardinality stays bounded (chapter 2's
// cardinality table).
package obshttp

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Metrics is the RED bundle. One instance per service; handlers take
// it by pointer, never construct globals.
type Metrics struct {
	requests  *prometheus.CounterVec
	errors    *prometheus.CounterVec
	durations *prometheus.HistogramVec
}

// NewMetrics registers the RED collectors on reg (pass
// prometheus.DefaultRegisterer for the common case, a fresh
// Registry in tests).
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "http",
			Name:      "requests_total",
			Help:      "Requests received.",
		}, []string{"method", "route", "status"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "http",
			Name:      "errors_total",
			Help:      "Requests answered with 5xx.",
		}, []string{"method", "route"}),
		durations: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "http",
			Name:      "request_duration_seconds",
			Help:      "End-to-end request latency.",
			// Wide enough to see both the fast path and the
			// timeout ceiling; buckets are a contract: changing
			// them silently breaks every dashboard and SLO
			// computation built on them (chapter 2).
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		}, []string{"method", "route"}),
	}
	reg.MustRegister(m.requests, m.errors, m.durations)
	return m
}

// Instrument wraps next with RED metrics and an OTel span. It runs
// OUTSIDE authn (metrics cover the requests you receive, including
// the rejected ones: 401s and 404s are traffic too).
func (m *Metrics) Instrument(next http.Handler) http.Handler {
	tracer := otel.Tracer("obshttp")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := RoutePattern(r)
		sw := &statusWriter{ResponseWriter: w}
		start := time.Now()

		ctx, span := tracer.Start(r.Context(), r.Method+" "+route,
			trace.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("url.path", r.URL.Path),
				attribute.String("http.route", route),
			))
		defer span.End()

		next.ServeHTTP(sw, r.WithContext(ctx))

		status := sw.Status()
		m.requests.WithLabelValues(r.Method, route, status).Inc()
		m.durations.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		if code := sw.code(); code >= 500 {
			m.errors.WithLabelValues(r.Method, route).Inc()
			span.SetStatus(codes.Error, http.StatusText(code))
		}
		span.SetAttributes(attribute.String("http.response.status_code", status))
	})
}

// statusWriter records the status code and write time. It is the
// same pattern as httpmw.statusWriter; it lives here too because the
// observability layer must not depend on a specific service's
// platform package (chapter 1's one-way rule).
type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (c *statusWriter) WriteHeader(code int) {
	if !c.written {
		c.status, c.written = code, true
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *statusWriter) Write(b []byte) (int, error) {
	if !c.written {
		c.status, c.written = http.StatusOK, true
	}
	return c.ResponseWriter.Write(b)
}

func (c *statusWriter) Status() string {
	return strconv.Itoa(c.code())
}

func (c *statusWriter) code() int {
	if c.status == 0 {
		return http.StatusOK
	}
	return c.status
}

// RoutePattern returns the matched pattern for cardinality-safe
// labels. Go 1.22+ muxes record it; for unmatched requests (404 from
// the mux itself, health checks mounted bare) it falls back to the
// literal path only when the path has no IDs in it, and to the
// bounded constant "unmatched" otherwise. The failure mode to avoid:
// /payments/pay_0192837 becoming a label value (chapter 2's table).
func RoutePattern(r *http.Request) string {
	if p := r.Pattern; p != "" {
		return p
	}
	path := r.URL.Path
	if strings.Count(path, "/") <= 2 && !strings.ContainsAny(path, "0123456789") {
		return path
	}
	return "unmatched"
}
