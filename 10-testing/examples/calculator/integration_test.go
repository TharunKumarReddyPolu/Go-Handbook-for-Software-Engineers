//go:build integration

// Integration-tier demonstration. This tier needs infrastructure; here a
// "storage service" is simulated over HTTP so the pattern runs anywhere,
// and the file shows the two gates: build tag (this file compiles only
// with -tags=integration) and env-var skip. See 10-testing chapter 3.
package calc

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// testLogger is defined in integration_helpers_test.go and shared by the
// whole package's test files.

func TestIntegration_StorageRoundTrip(t *testing.T) {
	addr := os.Getenv("TEST_STORAGE_ADDR")
	if addr == "" {
		// Fall back to an in-process server so the pattern is runnable
		// everywhere; in a real suite this would be the real service.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"stored":true}`))
		}))
		defer srv.Close()
		addr = srv.URL
	}

	resp, err := http.Post(addr+"/store", "application/json",
		strings.NewReader(`{"expr":"1+2","result":3}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestIntegration_EvalThroughAPI(t *testing.T) {
	h := NewHandler(testLogger(t))
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := srv.Client().Post(srv.URL+"/eval", "application/json",
		strings.NewReader(`{"expr":"20/4"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
