# 20 · Observability

**Status: in depth.** Five chapters, each tied to the working code in
[14-backend-development/examples/service](../14-backend-development/examples/service/),
which carries the section's instrumentation: RED metrics, tracing
middleware, `/metrics`, and OTel wiring with correct shutdown
ordering. Kafka-specific metric designs live in
[18 Section 4](../18-kafka-with-go/04-observability-tuning.md); pprof
mechanics live in [19 Section 1](../19-performance/01-measure-first.md).

## Chapters

1. **[Structured logging](01-structured-logging.md)**: slog in
   production: events not sentences, request scoping, redaction in
   the type, the levels budget
2. **[Metrics](02-metrics.md)**: counter/gauge/histogram, RED for
   services, business metrics for the domain, and the cardinality
   discipline that keeps both affordable
3. **[Tracing & OpenTelemetry](03-tracing-and-otel.md)**: spans
   through context, propagation, sampling, and init-first/flush-last
   wiring
4. **[SLOs, burn rates & alerting](04-slos-and-alerting.md)**: the
   error budget, symptom-based pages, two burn-rate tiers
5. **[Debugging production incidents](05-incident-debugging.md)**:
   the signal-to-question table, the evidence commands, rollback vs
   debug ordering

## The wired example

The section's code ships inside the Section 14 service so one example
serves both: `obshttp` (RED + spans middleware), `otelwiring`
(process-global provider with a no-op default), and `payments`
metrics (business signals). Run it and scrape it:

```bash
go run ./14-backend-development/examples/service &
curl -s localhost:8080/metrics | head        # RED + business metrics
LOG_LEVEL=debug go run ./14-backend-development/examples/service   # JSON logs
```

With `OTEL_EXPORTER_OTLP_ENDPOINT` set, spans export to any OTLP
collector; unset, the tracer is a no-op and nothing else changes.

## The one-page contract

| Question | Signal | Chapter |
|---|---|---|
| What happened to this request? | Correlated JSON logs | 1 |
| How is the system behaving? | RED + business metrics | 2 |
| Where did the request go across services? | Traces | 3 |
| Are users suffering? | SLIs vs SLOs, burn rates | 4 |
| What do I do right now? | Runbook + evidence commands | 5 |
