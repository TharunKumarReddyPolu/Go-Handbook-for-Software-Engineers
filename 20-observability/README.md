# 20 · Observability

**Status: outline — chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Metric designs already exist in
[18 §4](../18-kafka-with-go/04-observability-tuning.md) (Kafka
services) and
[25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md)
(reconciliation alerts).

## Planned chapters

1. **Structured logging** — slog in production: levels, fields,
   sampling, log-once discipline from
   [05 §2](../05-errors/02-error-design.md)
2. **Metrics** — the four Go metric shapes (counter, gauge, histogram,
   summary); RED metrics for services,
   USE for resources
3. **Tracing** — OpenTelemetry in Go: spans through context
   ([08 §3](../08-concurrency/03-context.md)), correlation IDs,
   exemplar links between traces and metrics
4. **SLOs, SLIs, SLAs** — the SRE stack: choosing indicators,
   error budgets, burn-rate alerting
5. **Alerting that works** — symptom-based alerts, runbooks as code,
   the dashboard-questions framing from
   [26 §4](../26-go-interview-preparation/04-senior-scenarios.md)
6. **Debugging production incidents** — the Go-specific flow: pprof
   endpoints, goroutine dumps, execution traces under load
   ([19 §1](../19-performance/01-measure-first.md) pairs)
7. **Worked example** — one service instrumented end to end: logs,
   metrics, traces, SLOs, and the dashboard that answers its runbook
