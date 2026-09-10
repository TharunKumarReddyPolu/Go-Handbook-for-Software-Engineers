package main

import (
	"runtime"
	"testing"
)

func collect(ch <-chan Result) []Result {
	var rs []Result
	for r := range ch {
		rs = append(rs, r)
	}
	return rs
}

func TestRun_AllJobsProcessed(t *testing.T) {
	jobs := []Job{{1, "a"}, {2, "b"}, {3, "c"}, {4, "d"}}
	got := collect(Run(jobs, 3))
	if len(got) != len(jobs) {
		t.Fatalf("got %d results, want %d", len(got), len(jobs))
	}
	seen := map[int]bool{}
	for _, r := range got {
		seen[r.JobID] = true
		if r.Err != nil {
			t.Errorf("job %d: unexpected error %v", r.JobID, r.Err)
		}
	}
	for _, j := range jobs {
		if !seen[j.ID] {
			t.Errorf("job %d missing from results", j.ID)
		}
	}
}

func TestRun_NoGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	_ = collect(Run([]Job{{1, "x"}, {2, "y"}}, 4))
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines before=%d after=%d: leak", before, after)
	}
}

func TestRun_MoreWorkersThanJobs(t *testing.T) {
	if got := collect(Run([]Job{{1, "solo"}}, 8)); len(got) != 1 {
		t.Errorf("got %d results, want 1", len(got))
	}
}
