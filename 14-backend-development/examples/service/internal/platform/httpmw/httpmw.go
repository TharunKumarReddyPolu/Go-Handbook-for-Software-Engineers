// Package httpmw is the platform middleware from 12 Section 2: the chain
// constructor plus the three every service needs. The scoped-logger
// middleware (chapter 5) attaches a request-scoped slog to the
// context; handlers pull it via log.FromContext. Section 20 adds
// trace correlation: the logger carries trace/span IDs when a span
// is active.
package httpmw

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type middleware func(http.Handler) http.Handler

// Chain applies middleware in reverse so the first listed is
// outermost: Chain(h, a, b) == a(b(h)).
func Chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// --- scoped logger (chapter 5) ---

type ctxKey int

const loggerKey ctxKey = 0

// FromContext returns the request-scoped logger; falls back to the
// default so late-stage wiring mistakes log instead of panic.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// ScopedLogger attaches a logger carrying the request ID to the
// context and echoes the ID in the response header. When a trace
// context exists (section 20's Instrument runs outside this, so a
// span is usually present), the trace and span IDs join the log
// line: the correlation that turns "find the logs for this trace"
// into one grep.
func ScopedLogger(base *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := fmt.Sprintf("req_%d", time.Now().UnixNano())
			logger := base.With("request_id", id)
			if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
				logger = logger.With("trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
			}
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r.WithContext(
				context.WithValue(r.Context(), loggerKey, logger)))
		})
	}
}

// --- access log: one line per request ---

// statusWriter records whether and what was written (12 Section 2).
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (c *statusWriter) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *statusWriter) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	return c.ResponseWriter.Write(b)
}

// AccessLog emits exactly one structured line per request.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		FromContext(r.Context()).InfoContext(r.Context(), "http",
			"method", r.Method, "path", r.URL.Path,
			"status", sw.status, "dur", time.Since(start).String())
	})
}

// --- recovery ---

// Recover converts handler panics into 500s, never rewriting a
// response that already committed (12 Section 2's countingWriter rule).
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		defer func() {
			if rec := recover(); rec != nil {
				FromContext(r.Context()).ErrorContext(r.Context(), "panic",
					"rec", rec, "path", r.URL.Path)
				if sw.status == 0 {
					http.Error(sw, "internal error", http.StatusInternalServerError)
				}
			}
		}()
		next.ServeHTTP(sw, r)
	})
}
