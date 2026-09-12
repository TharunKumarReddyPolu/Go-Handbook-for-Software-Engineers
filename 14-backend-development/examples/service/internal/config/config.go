// Package config is chapter 2's loader: typed at the edge, validated
// once at boot, immutable after. Tests pass a map via Load's function
// parameter; main passes os.Getenv.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Kind selects the store implementation at wiring time (main). It is
// composition data, not an environment check: every value has tests
// on both sides.
type Kind string

const (
	KindMemory   Kind = "memory"
	KindPostgres Kind = "postgres"
)

// Secret wraps a credential so formatting and logging redact it.
// The raw value is available through Value() at the single point it
// is consumed (the constructor that needs it).
type Secret string

func (s Secret) Value() string { return string(s) }
func (s Secret) String() string {
	if s == "" {
		return ""
	}
	return "[REDACTED]"
}
func (s Secret) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

// Config is the service's entire configuration surface. Fields map
// mechanically to env names.
type Config struct {
	Addr           string        // ADDR
	Store          Kind          // STORE
	DatabaseURL    Secret        // DATABASE_URL (required for postgres)
	ShutdownGrace  time.Duration // SHUTDOWN_GRACE
	RequestTimeout time.Duration // REQUEST_TIMEOUT
	PoolMaxOpen    int32         // POOL_MAX_OPEN
	LogLevel       string        // LOG_LEVEL (debug|info|warn|error)
}

// Load parses and validates. All violations are joined so an operator
// fixes every typo in one deploy.
func Load(environ func(string) string) (Config, error) {
	var c Config
	var errs []error

	c.Addr = env(environ, "ADDR", ":8080")
	c.Store = Kind(env(environ, "STORE", string(KindMemory)))
	c.DatabaseURL = Secret(environ("DATABASE_URL"))
	c.ShutdownGrace = mustDuration(environ, "SHUTDOWN_GRACE", 25*time.Second, &errs)
	c.RequestTimeout = mustDuration(environ, "REQUEST_TIMEOUT", 10*time.Second, &errs)
	c.PoolMaxOpen = mustInt32(environ, "POOL_MAX_OPEN", 25, &errs)
	c.LogLevel = env(environ, "LOG_LEVEL", "info")

	switch c.Store {
	case KindMemory, KindPostgres:
	default:
		errs = append(errs, fmt.Errorf("STORE: %q is not one of memory|postgres", c.Store))
	}
	if c.Store == KindPostgres && c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required when STORE=postgres"))
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %q is not one of debug|info|warn|error", c.LogLevel))
	}
	if c.RequestTimeout <= 0 || c.RequestTimeout > time.Minute {
		errs = append(errs, fmt.Errorf("REQUEST_TIMEOUT: %s is outside (0, 1m]", c.RequestTimeout))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return c, nil
}

// Summary renders the safe, loggable view: no secrets, ever.
func (c Config) Summary() string {
	return fmt.Sprintf("addr=%s store=%s grace=%s timeout=%s pool=%d log=%s db=%s",
		c.Addr, c.Store, c.ShutdownGrace, c.RequestTimeout, c.PoolMaxOpen, c.LogLevel, c.DatabaseURL)
}

func env(get func(string) string, key, def string) string {
	if v := get(key); v != "" {
		return v
	}
	return def
}

func mustDuration(get func(string) string, key string, def time.Duration, errs *[]error) time.Duration {
	raw := get(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %q is not a duration (try %q)", key, raw, def))
		return def
	}
	return d
}

func mustInt32(get func(string) string, key string, def int32, errs *[]error) int32 {
	raw := get(key)
	if raw == "" {
		return def
	}
	var v int32
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %q is not an integer", key, raw))
		return def
	}
	return v
}

// osGetenv adapts os.Getenv to the Load seam (single call site).
func osGetenv(k string) string { return os.Getenv(k) }

// LoadOS is main's entry point.
func LoadOS() (Config, error) { return Load(osGetenv) }
