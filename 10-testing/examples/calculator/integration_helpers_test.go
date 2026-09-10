//go:build integration

package calc

import (
	"io"
	"log/slog"
	"testing"
)

// testLogger returns a quiet structured logger for integration tests.
// The t.Helper() contract applies to fixture helpers too; t.Cleanup
// owns resources.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
