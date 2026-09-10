# 08 · Concurrency

Go's defining feature, and the section most interviewers and production
incidents live in. Concurrency is built into the language, but correct
concurrency is built into *you*: this section is the training ground.

## Objectives

By the end of this section you can:

- Explain how goroutines differ from threads and why thousands are cheap
- Design channel ownership so bugs cannot compile into existence
- Use context to propagate cancellation through every layer
- Choose between channels, mutexes, and atomics with reasons
- Build worker pools, pipelines, fan-out/fan-in with backpressure and shutdown
- Detect and fix: deadlocks, data races, goroutine leaks, starvation
- Demonstrate a production job processor with retries and graceful shutdown

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Goroutines & channels](01-goroutines-and-channels.md) | The model, ownership, directional channels |
| 2 | [Select & timeouts](02-select-and-timeouts.md) | Multiplexing, the time.After trap |
| 3 | [Context](03-context.md) | Cancellation, deadlines, values |
| 4 | [Sync primitives](04-sync-primitives.md) | Mutex, RWMutex, Once, WaitGroup, atomic, sync.Map |
| 5 | [Patterns](05-patterns.md) | Pools, pipelines, fan-out/in, semaphores, rate limits |
| 6 | [Concurrency vs parallelism](06-concurrency-vs-parallelism.md) | Models, scheduling, what to claim in interviews |
| 7 | [Pitfalls](07-pitfalls.md) | Deadlocks, races, leaks, starvation, the race detector |
| 8 | [FAQ & quick reference](08-faq-notes.md) | Rapid answers, decision tables |

## Examples: one problem, five stages

The `examples/` directory deliberately re-solves the same problem (process
jobs) at increasing production-readiness:

| Stage | Directory | Teaches |
|---|---|---|
| 1 | `examples/01-toy/` | producer/consumer, basic ownership |
| 2 | `examples/02-workerpool/` | bounded workers, results channel |
| 3 | `examples/03-workerpool-cancel/` | context cancellation, leak-free shutdown |
| 4 | `examples/04-backpressure/` | bounded queues, load shedding |
| 5 | `examples/05-jobprocessor/` | retries, graceful shutdown, metrics hooks |

Each stage compiles and has tests: `go test ./08-concurrency/... -race`

## Progress checklist

- [x] Goroutines
- [x] Channels (buffered/unbuffered)
- [x] Channel ownership
- [x] Directional channels
- [x] select
- [x] Context
- [x] sync.Mutex / RWMutex / Once / WaitGroup
- [x] atomic operations
- [x] sync.Map
- [x] Worker pools, fan-out, fan-in, pipelines
- [x] Cancellation, timeouts
- [x] Backpressure, rate limiting, semaphores
- [x] Deadlocks, starvation, race conditions, data races
- [x] Goroutine leaks
- [x] Concurrency vs parallelism
