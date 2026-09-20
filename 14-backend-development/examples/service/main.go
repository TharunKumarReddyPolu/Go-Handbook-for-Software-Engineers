// Command service is the handbook's complete backend example for
// section 14, now carrying section 20's observability: RED metrics
// and tracing middleware on every request, business metrics on the
// payments domain, a /metrics endpoint, and OTel wiring with correct
// shutdown ordering (flush spans AFTER the server drains).
//
// Layout: main.go here is wiring only; domains live in internal/
// (01-service-layout.md). The store default is in-memory so the
// example runs with zero setup; OTEL_EXPORTER_OTLP_ENDPOINT empty
// means the tracer is a no-op: observability must never require
// infrastructure to boot (chapter 3's zero-branches rule).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/config"
	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/payments"
	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/platform/obshttp"
	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/platform/otelwiring"
)

func main() {
	cfg, err := config.LoadOS()
	if err != nil {
		slog.Error("configuration invalid", "err", err)
		os.Exit(1)
	}
	logger := newLogger(cfg.LogLevel)
	logger.Info("starting", "config", cfg.Summary())

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Observability wiring (section 20 ch 3): tracer provider first
	// (before any request can start a span), registry + business
	// metrics next, shutdown function held for the LAST defer.
	shutdownTracing, err := otelwiring.Init(ctx, "payments-service", "dev")
	if err != nil {
		logger.Error("tracing init failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		// Flush after the server drains: the final requests' spans
		// must reach the collector. Bounded by otelwiring (5s).
		if err := shutdownTracing(context.Background()); err != nil {
			logger.Error("trace flush failed", "err", err)
		}
	}()

	reg := prometheus.NewRegistry() // NOT the default registry: a
	// private registry exports exactly this service's metrics, never
	// a dependency's (chapter 2's registry hygiene).
	payMetrics := payments.NewPaymentMetrics(reg)
	red := obshttp.NewMetrics(reg)

	store, closeStore, err := openStore(ctx, cfg)
	if err != nil {
		logger.Error("store open failed", "err", err)
		os.Exit(1)
	}
	defer closeStore()

	svc := payments.NewService(store, logger).WithMetrics(payMetrics)
	handler := payments.NewHandler(svc)

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.Handle("/", handler.Routes(logger, red))

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := run(ctx, srv, logger, cfg.ShutdownGrace); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
	logger.Info("bye")
}

// run is the extracted lifecycle from 12 Section 5: serve, signal, drain.
// Extracted so the in-process shutdown test can exercise the exact
// production path.
func run(ctx context.Context, srv *http.Server, log *slog.Logger, grace time.Duration) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced close after grace period", "err", err)
		_ = srv.Close()
	}
	return nil
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
}

// openStore builds the store behind the config switch (chapter 3:
// environments differ by config, not code paths). The returned
// close function participates in shutdown ordering (12 Section 5: stores
// close after the HTTP server drains).
func openStore(ctx context.Context, cfg config.Config) (payments.PaymentStore, func(), error) {
	switch cfg.Store {
	case config.KindPostgres:
		// Real SQL path: 13-databases's bank example carries the full
		// Postgres implementation with its own two-tier tests. The
		// example service keeps a stub error here so it runs anywhere;
		// wiring it is the capstone's exercise (28-projects).
		_ = ctx
		return nil, func() {}, fmt.Errorf("postgres store: wire sqldb.Open + bank.NewPostgresStore here (see 13-databases)")
	default:
		return payments.NewMemoryStore(), func() {}, nil
	}
}
