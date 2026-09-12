// Command service is the handbook's complete backend example for
// section 14: config (ch 2), explicit wiring (ch 3), the layered
// payments domain (ch 1) with authn/authz gates (ch 4) and scoped
// logging plus health endpoints (ch 5), running the graceful
// lifecycle from 12 §5.
//
// Layout: main.go here is wiring only; domains live in internal/
// (01-service-layout.md). The store default is in-memory so the
// example runs with zero setup; STORE=postgres exercises the pool
// path with the same domain code.
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

	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/config"
	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/payments"
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

	store, closeStore, err := openStore(ctx, cfg)
	if err != nil {
		logger.Error("store open failed", "err", err)
		os.Exit(1)
	}
	defer closeStore()

	svc := payments.NewService(store, logger)
	handler := payments.NewHandler(svc)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler.Routes(logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := run(ctx, srv, logger, cfg.ShutdownGrace); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
	logger.Info("bye")
}

// run is the extracted lifecycle from 12 §5: serve, signal, drain.
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
// close function participates in shutdown ordering (12 §5: stores
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
