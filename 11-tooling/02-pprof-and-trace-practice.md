# pprof & trace in practice

## Why Does This Matter?

[19 §1](../19-performance/01-measure-first.md) teaches the
profiles: what each one is, how to read a flamegraph, measure
before optimizing. This chapter is the operational layer: mounting
the endpoints safely, capturing under load, automating the
capture, and continuous profiling as a production practice. The
gap between "I can run pprof" and "I can profile the incident at
3 a.m." is exactly this chapter ([20
§5](../20-observability/05-incident-debugging.md) is the incident
flow that depends on it).

## Mental Model

Profiling has three production problems beyond the mechanics:

1. **Access**: the profile endpoints are inside the cluster,
   behind the ingress, on a pod that autoscales.
2. **Timing**: the interesting moment is under load, often
   mid-incident, when nobody wants to hand-run commands.
3. **Comparison**: a profile without a baseline is trivia;
   before/after pairs are evidence.

Solve all three once, in tooling, and profiling becomes a habit
instead of an archaeology project.

## How It Works

**Mounting the endpoints**: `net/http/pprof` registers on
`DefaultServeMux` ([19 §1](../19-performance/01-measure-first.md)'s
mechanics). The production shape is a dedicated internal port:

```go
func serveDebug(addr string) *http.Server {
    mux := http.NewServeMux()
    mux.Handle("/debug/pprof/", http.DefaultServeMux) // mount, don't import-and-expose
    return &http.Server{Addr: addr, Handler: mux,
        ReadHeaderTimeout: 5 * time.Second}
}
// main: go serveDebug(":6060").ListenAndServe() with the lifecycle from [22 §2]
```

The internal port is reachable from `kubectl port-forward` and
blocked at the ingress by construction ([21 §1](../21-security/01-threat-model-and-validation.md)'s
exposure rules): profiles are code layout plus data shapes.

**Capturing under load**, the repeatable one-liner set (this
repo's guidance: script them once):

```bash
kubectl port-forward pod/svc-xyz 6060:6060 &
go tool pprof -seconds=30 -output=cpu.pb.gz http://localhost:6060/debug/pprof/profile
go tool pprof -output=heap.pb.gz http://localhost:6060/debug/pprof/heap
curl -s localhost:6060/debug/pprof/goroutine?debug=2 > goroutines.txt
```

**Comparison** is benchstat's job for benchmarks and `pprof -top
-diff_base` for profiles: the delta view turns "here is a hot
function" into "this function got 3x hotter than the baseline
capture" ([19 §1](../19-performance/01-measure-first.md)'s
before/after discipline, automated).

## Syntax / API

The profile matrix, operational notes beyond [19 §1](../19-performance/01-measure-first.md)'s
introduction:

| Profile | Cost under load | Duration guidance |
|---|---|---|
| CPU (`profile?seconds=N`) | sampling, ~cheap | 30s under representative load |
| Heap (`heap`) | point-in-time, cheap | any time; capture pairs |
| Goroutine (`goroutine?debug=2`) | cheap; output is large | any time; grep by block state |
| Execution trace (`trace?seconds=N`) | expensive | 1-5s windows only |
| Block/mutex | near-free, needs enabling | enable permanently in prod ([19 §3](../19-performance/03-concurrency-performance.md)) |

Enable block and mutex profiling at boot in every service: they
are near-free and answer contention questions that only show up
under real traffic:

```go
runtime.SetBlockProfileRate(1_000_000)   // 1 event per ms blocked
runtime.SetMutexProfileFraction(100)     // 1 in 100 contends
```

## Basic Example

The capture script that makes 3 a.m. survivable (commit it, name
it, teach it):

```bash
#!/usr/bin/env bash
# profile.sh <pod> [seconds]: captures the incident evidence set.
POD=${1:?pod required}; SECS=${2:-30}
kubectl port-forward "$POD" 6060:6060 & PF=$!; sleep 1
curl -s localhost:6060/debug/pprof/goroutine?debug=2 > "goroutines-$(date +%s).txt"
go tool pprof -seconds="$SECS" -raw http://localhost:6060/debug/pprof/profile > "cpu-$(date +%s).pb"
go tool pprof -raw http://localhost:6060/debug/pprof/heap > "heap-$(date +%s).pb"
kill $PF
```

Capture before rollback ([20
§5](../20-observability/05-incident-debugging.md)'s ordering
rule): the profiles are the postmortem's raw material.

## Real-World Example

Continuous profiling turns the evidence problem around: instead of
capturing when something breaks, capture always, at low cost, and
diff across deploys. The Go ecosystem options: Parca, Pyroscope
(Grafana), and Google Cloud Profiler all consume `pprof` format
from a small agent. The operational pattern: 1-5% CPU overhead
(measure it), profiles pushed per-minute, and the deploy marker
([22 §6](../22-production-go/06-releases-and-rollbacks.md)'s
version label) on each profile: the "p99 regressed in v42" query
becomes one diff instead of a repro ([19
§4](../19-performance/04-compiler-and-pgo.md)'s PGO pipeline
consumes the same profiles: the infrastructure pays for itself).

## Production Example

**The profiling readiness checklist** for a service ([22
§2](../22-production-go/02-server-lifecycle.md)'s boot order
includes it):

- [ ] `net/http/pprof` mounted on an internal port (not the public mux)
- [ ] Block/mutex profiling enabled at boot
- [ ] `profile.sh` committed; the on-call knows it exists
- [ ] Runtime metrics exported ([20 §2](../20-observability/02-metrics.md)): NumGoroutine, GC pauses
- [ ] Continuous profiler (or scheduled captures) with deploy labels
- [ ] Dashboard links in the runbook ([20 §5](../20-observability/05-incident-debugging.md)'s table)

**The autoscaling wrinkle**: the pod you want to profile is
terminating or new pods appear mid-capture. Profile a *stable*
pod (pin by name, not service DNS), and for fleet-wide questions,
aggregate continuous profiles rather than sampling one pod: one
pod's hot path may be a shard-key anomaly ([16
§4](../16-distributed-systems/04-quorums-sharding.md)'s hot-key
caution).

## Common Mistakes

| Mistake | Consequence | Do instead |
|---|---|---|
| pprof on the public mux | The internet profiles your service | Internal port + network policy |
| Capturing once, conclusions forever | Profiles are load-shaped | Capture under representative load; diff against baseline |
| 30s execution traces | Multi-GB artifacts, stalled handlers | 1-5s trace windows |
| Profiling one pod of a skewed fleet | You profile the anomaly | Continuous profiling aggregated fleet-wide |
| Skipping block/mutex profiling | Contention invisible until it hurts | Enable at boot; near-free |
| Profiles without deploy labels | Cannot correlate regressions | Version label on every capture |

## Idiomatic Go

- `net/http/pprof` imported for side effects in a debug-only
  binary or mux: the stdlib way; no third-party profiler needed
  for the basics.
- Profiles are protobufs (`gzip`d): store, diff, and ship them as
  data, not screenshots.
- The `runtime/pprof` programmatic API for tests: profile a load
  test in CI ([19 §1](../19-performance/01-measure-first.md)'s
  harness) and fail on allocation regressions.

## Performance Considerations

CPU profiling samples at 100Hz by default: negligible. Heap
profiling samples allocations (set via `runtime.MemProfileRate`);
the default samples enough. The expensive one is the execution
trace: seconds only. Continuous profilers add 1-5%: budget it,
measure it ([19 §1](../19-performance/01-measure-first.md)'s
rule applies to the measurement infrastructure itself).

## Concurrency Considerations

The goroutine profile is the concurrency instrument: `debug=2`
dumps every goroutine's stack with its blocking state
(`chan receive`, `semacquire`, `IO wait`): the leak and deadlock
diagnosis ([20 §5](../20-observability/05-incident-debugging.md)'s
workhorse; [09 §5](../09-memory-runtime/05-memory-leaks.md)'s
shape-1 detection). Block/mutex profiles quantify what the dump
shows qualitatively: which lock, how long, whose stack.

## Security Considerations

Profiles expose: code layout (function names, hot paths), data
shapes (heap contents' types), and goroutine stacks (sometimes
with strings). Internal-only exposure, RBAC on the debug port,
and no profile endpoints on public ingress ([21
§1](../21-security/01-threat-model-and-validation.md)). Postmortem
artifacts (profiles, dumps) can contain PII in stack arguments:
treat incident artifacts with the same care as logs ([20
§1](../20-observability/01-structured-logging.md)'s redaction).

## Testing Strategy

- Profile-enabled load tests in CI for hot paths ([19
  §1](../19-performance/01-measure-first.md)): assert
  allocs/op ceilings with `testing.AllocsPerRun` and `-benchmem`
  regressions via benchstat ([10 §4](../10-testing/04-benchmarks-coverage-fuzzing.md)).
- A smoke test that the debug server answers `/debug/pprof/` and
  that the public router does not route it.
- Chaos rehearsal: the game day captures profiles mid-incident
  ([22 §4](../22-production-go/04-dependency-failures.md)'s game
  days): the script gets tested when it matters.

## Interview Questions

1. Design the profiling setup for a service running 200 pods:
   access, capture, aggregation, correlation with deploys.
2. When is a single-pod CPU profile misleading?
3. What do block and mutex profiles add that CPU and heap do not?
4. How does continuous profiling change the incident and the PGO
   pipeline?

## Practice Exercises

1. Add the internal debug port + block/mutex profiling to the
   Section 14 service; write the test proving the public router
   cannot reach `/debug/pprof/`.
2. Commit `profile.sh` to a project; run it against a load-tested
   service; produce one diff report against a baseline capture.
3. Evaluate a continuous profiler locally (Parca or Pyroscope
   with the Section 14 service); measure its overhead; annotate a
   deploy marker.

## Further Reading

- [net/http/pprof documentation](https://pkg.go.dev/net/http/pprof)
- [Go diagnostics: profiling](https://go.dev/doc/diagnostics)
- [Pyroscope for Go](https://grafana.com/docs/pyroscope/)
- [PPGO: continuous profiling concepts](https://www.pprof.org/)
