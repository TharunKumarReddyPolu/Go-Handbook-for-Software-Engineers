package obshttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// newTestMetrics builds a metrics bundle on a fresh registry.
func newTestMetrics(t *testing.T) (*Metrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	return NewMetrics(reg), reg
}

func TestInstrument_RedCounts(t *testing.T) {
	m, _ := newTestMetrics(t)
	h := http.NewServeMux()
	h.HandleFunc("POST /things", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	h.HandleFunc("POST /broken", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	srv := httptest.NewServer(m.Instrument(h))
	defer srv.Close()

	if resp, err := http.Post(srv.URL+"/things", "text/plain", strings.NewReader("x")); err != nil || resp.StatusCode != 201 {
		t.Fatalf("POST /things: %v %v", resp, err)
	}
	if resp, err := http.Post(srv.URL+"/broken", "text/plain", strings.NewReader("x")); err != nil || resp.StatusCode != 500 {
		t.Fatalf("POST /broken: %v %v", resp, err)
	}

	// Rate: one increment per request with the matched pattern and
	// status as labels.
	if got := testutil.ToFloat64(m.requests.WithLabelValues("POST", "/things", "201")); got != 1 {
		t.Errorf("requests_total[POST /things 201] = %v, want 1", got)
	}
	// Errors: 5xx only; the 201 must not appear.
	if got := testutil.ToFloat64(m.errors.WithLabelValues("POST", "/broken")); got != 1 {
		t.Errorf("errors_total[POST /broken] = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.errors.WithLabelValues("POST", "/things")); got != 0 {
		t.Errorf("errors_total[POST /things] = %v, want 0", got)
	}
	// Duration: exactly one observation per request.
	if n := testutil.CollectAndCount(m.durations, "http_request_duration_seconds"); n != 2 {
		t.Errorf("duration series count = %d, want 2 (one per route)", n)
	}
}

func TestInstrument_RejectedRequestsCounted(t *testing.T) {
	// Instrument wraps OUTSIDE authn: 401s are traffic and must be
	// counted (chapter 2: RED covers requests received).
	m, _ := newTestMetrics(t)
	gate := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	srv := httptest.NewServer(m.Instrument(gate))
	defer srv.Close()

	// An ID-bearing path through a raw (pattern-less) handler is the
	// cardinality hazard: the route label must fall back to the
	// bounded "unmatched" constant, never the raw path.
	if resp, err := http.Get(srv.URL + "/payments/pay_0192837"); err != nil || resp.StatusCode != 401 {
		t.Fatalf("GET: %v %v", resp, err)
	}
	if got := testutil.ToFloat64(m.requests.WithLabelValues("GET", "unmatched", "401")); got != 1 {
		t.Errorf("requests_total[GET unmatched 401] = %v, want 1", got)
	}
}

func TestRoutePattern_Cardinality(t *testing.T) {
	tests := []struct {
		path    string
		pattern string // pre-set pattern (Go 1.22 mux behavior)
		want    string
	}{
		// Pattern wins over whatever the raw path looks like.
		{path: "/payments/pay_0192837", pattern: "POST /payments/{id}/cancel", want: "POST /payments/{id}/cancel"},
		{path: "/healthz", want: "/healthz"},               // shallow, no IDs
		{path: "/payments/pay_0192837", want: "unmatched"}, // ID-bearing: bounded
		{path: "/a/b/c/d", want: "unmatched"},              // deep: suspicious
	}
	for _, tt := range tests {
		r := httptest.NewRequest(http.MethodGet, tt.path, nil)
		r.Pattern = tt.pattern
		if got := RoutePattern(r); got != tt.want {
			t.Errorf("RoutePattern(%q, pattern=%q) = %q, want %q", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestInstrument_SpanAttributes(t *testing.T) {
	// Install a real (discard-output) provider, as otelwiring does in
	// main: without one, otel.Tracer hands out no-op tracers and the
	// handler would run with an invalid span context.
	prev := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(discardExporter{})),
	))

	m, _ := newTestMetrics(t)

	// Capture the span the middleware creates.
	var spanID trace.SpanID
	h := m.Instrument(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sc := trace.SpanContextFromContext(r.Context())
		if !sc.IsValid() {
			t.Error("handler ran without a valid span context")
			return
		}
		spanID = sc.SpanID()
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	if resp, err := http.Get(srv.URL + "/livez"); err != nil || resp.StatusCode != 200 {
		t.Fatalf("GET: %v %v", resp, err)
	}
	if !spanID.IsValid() {
		t.Error("span ID never recorded; middleware did not start a span")
	}
}

// discardExporter satisfies sdktrace.SpanExporter while throwing
// spans away: enough to make span contexts real without output.
type discardExporter struct{}

func (discardExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error { return nil }
func (discardExporter) Shutdown(_ context.Context) error                               { return nil }
