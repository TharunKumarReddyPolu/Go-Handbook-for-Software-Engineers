package config

import (
	"strings"
	"testing"
	"time"
)

func mapEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load(mapEnv(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":8080" || cfg.Store != KindMemory || cfg.LogLevel != "info" {
		t.Errorf("defaults wrong: %+v", cfg)
	}
	if cfg.ShutdownGrace != 25*time.Second || cfg.RequestTimeout != 10*time.Second {
		t.Errorf("duration defaults wrong: %+v", cfg)
	}
	if cfg.PoolMaxOpen != 25 {
		t.Errorf("pool default = %d, want 25", cfg.PoolMaxOpen)
	}
}

func TestLoad_BatchedErrors(t *testing.T) {
	_, err := Load(mapEnv(map[string]string{
		"STORE":           "sqlite",  // not a kind
		"SHUTDOWN_GRACE":  "25",      // missing unit
		"REQUEST_TIMEOUT": "10x",     // not a duration
		"LOG_LEVEL":       "verbose", // not allowlisted
		"POOL_MAX_OPEN":   "many",    // not an integer
		"DATABASE_URL":    "",        // fine for memory
	}))
	if err == nil {
		t.Fatal("want batched validation error")
	}
	// Chapter 2's rule: every violation in one message, one deploy.
	for _, want := range []string{"STORE", "SHUTDOWN_GRACE", "REQUEST_TIMEOUT", "LOG_LEVEL", "POOL_MAX_OPEN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %s, got: %v", want, err)
		}
	}
}

func TestLoad_PostgresRequiresDSN(t *testing.T) {
	_, err := Load(mapEnv(map[string]string{"STORE": "postgres"}))
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("err = %v, want DATABASE_URL required", err)
	}

	cfg, err := Load(mapEnv(map[string]string{
		"STORE":        "postgres",
		"DATABASE_URL": "postgres://u:s3cret@db:5432/app",
	}))
	if err != nil {
		t.Fatalf("Load with DSN: %v", err)
	}
	// The summary is the log surface: the password must never appear.
	if strings.Contains(cfg.Summary(), "s3cret") {
		t.Error("Summary leaked the secret")
	}
	if cfg.DatabaseURL.Value() != "postgres://u:s3cret@db:5432/app" {
		t.Error("Value() should give the constructor the real DSN")
	}
	if got := cfg.DatabaseURL.LogValue().String(); got != "[REDACTED]" {
		t.Errorf("LogValue = %q, want redacted", got)
	}
	if got := cfg.DatabaseURL.String(); strings.Contains(got, "s3cret") {
		t.Errorf("String leaked: %s", got)
	}
}

func TestLoad_DurationUnitsHonest(t *testing.T) {
	// "10" without a unit is the classic footgun; the loader must
	// reject it rather than guess milliseconds.
	_, err := Load(mapEnv(map[string]string{"REQUEST_TIMEOUT": "10"}))
	if err == nil {
		t.Fatal("unitless duration must be rejected")
	}
}
