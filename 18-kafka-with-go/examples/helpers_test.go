//go:build broker

package main

import (
	"io"
	"log/slog"
	"testing"
)

func discardLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
