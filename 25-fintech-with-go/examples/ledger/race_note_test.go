package ledger

import "testing"

// TestRaceDetectorNote documents the race-detector workflow: the
// concurrent invariant tests above are designed for `go test -race`,
// which requires CGO. This machine lacks a C toolchain, so local runs
// skip race mode; CI (GitHub Actions ubuntu runners) executes the full
// suite with -race on every push. Do not weaken the concurrent tests
// to pass locally -- they exist for the CI race run.
func TestRaceDetectorNote(t *testing.T) {
	t.Log("run with race detection in CI: go test ./... -race")
}
