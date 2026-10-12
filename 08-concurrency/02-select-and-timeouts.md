# Select & timeouts

## Why Does This Matter?

A goroutine that can only wait on one channel is a goroutine that cannot
time out, cannot cancel, and cannot fail gracefully. `select` is how a
goroutine stays responsive to *multiple* futures at once, and time-based
futures (deadlines, ticks) are how systems stay alive when the world
stops answering.

## Mental Model

`select` is a switch over channel operations: it waits until one case can
proceed, then runs it. With no cases ready, it blocks. With several
ready, it picks **uniformly at random**: deliberately, to prevent
starvation of slower cases. With a `default` case, it never blocks.

```mermaid
stateDiagram-v2
    [*] --> Waiting
    Waiting --> CaseA: chA ready
    Waiting --> CaseB: chB ready
    Waiting --> CaseC: both ready (random pick)
    Waiting --> Default: nothing ready + default present
```

Time is just another channel: `time.After(d)` returns a channel that
receives once after d. A timeout is a select between your operation and
the clock.

## How It Works

```go
select {
case v := <-events:      // wait for the next event
	handle(v)
case <-ticker.C:         // periodic housekeeping
	compact()
case <-ctx.Done():       // cancellation: the case that should always exist
	return ctx.Err()
}
```

The three-part shape above: work, housekeeping, cancellation: is the
skeleton of every long-running goroutine.

### The time.After trap

```go
for {
	select {
	case v := <-ch:
		process(v)
	case <-time.After(5 * time.Second): // resets the timeout on each loop iteration
		return ErrSlowConsumer
	}
}
```

Each loop iteration creates a *new* timer. Since Go 1.23, unreachable
timers can be collected before firing, so they do not accumulate forever;
this is still unnecessary timer churn, and the deadline also *resets*
every time data arrives, so it measures "idle time," not total time.
Correct forms:

```go
// Idle timeout: reuse one timer, stop/reset it.
idle := time.NewTimer(5 * time.Second)
defer idle.Stop()
for {
	select {
	case v := <-ch:
		process(v)
		idle.Stop() // Go 1.23+ synchronous timers need no drain before Reset
		idle.Reset(5 * time.Second)
	case <-idle.C:
		return ErrSlowConsumer
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Total deadline: create it once outside the loop.
timeout := time.After(30 * time.Second)
for {
	select {
	case v := <-ch:
		process(v)
	case <-timeout:
		return ErrBudgetExceeded
	}
}
```

Go 1.23 introduced synchronous timer channels for modules using the
new semantics; Go 1.27 removed the `asynctimerchan` compatibility
setting, so timer channels are now always unbuffered. Timer stops are
more forgiving, but reusing timers remains the right pattern for
correctness and allocation hygiene. See
[meta/versioning.md](../meta/versioning.md).

## Non-blocking operations with default

```go
// Try to send, never block: drop rather than obstruct the producer.
select {
case results <- r:
default:
	dropped.Add(1) // metric: results_dropped_total
}
```

This is the load-shedding primitive: bounded work queues, sampling,
fire-and-forget metrics all use it. The mirror image (`case v := <-ch:
default:`) is a "try receive" for polling designs.

## Basic Example: first of N (fan-in race)

```go
// FirstResponse returns whichever replica answers first, leaking the loser.
func FirstResponse(ctx context.Context, replicas ...func(context.Context) (string, error)) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // cancels losing goroutines: prevents leaks

	type res struct {
		val string
		err error
	}
	ch := make(chan res, len(replicas)) // buffered: no goroutine blocks
	for _, fn := range replicas {
		go func(fn func(context.Context) (string, error)) {
			v, err := fn(ctx)
			ch <- res{v, err}
		}(fn)
	}

	var lastErr error
	for range replicas {
		select {
		case r := <-ch:
			if r.err == nil {
				return r.val, nil
			}
			lastErr = r.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return "", lastErr
}
```

## Real-World Example: heartbeat watchdog

```go
func Watchdog(ctx context.Context, beat <-chan struct{}, maxSilence time.Duration) error {
	t := time.NewTimer(maxSilence)
	defer t.Stop()
	for {
		select {
		case <-beat:
			t.Stop() // Go 1.23+ synchronous timers need no drain before Reset
			t.Reset(maxSilence)
		case <-t.C:
			return errors.New("heartbeat lost") // escalate: restart, failover, alert
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
```

## Production Example: flush-or-deadline batching

A batcher that ships either when the batch fills or when the deadline
hits: the shape of most metric/trace/log shippers:

```go
func (b *Batcher) Run(ctx context.Context) error {
	tick := time.NewTicker(b.maxDelay)
	defer tick.Stop()
	var batch []Event

	for {
		select {
		case ev, ok := <-b.in:
			if !ok {
				return b.flush(ctx, batch) // producer closed: final flush
			}
			batch = append(batch, ev)
			if len(batch) >= b.maxBatch {
				if err := b.flush(ctx, batch); err != nil {
					return err
				}
				batch = batch[:0]
			}
		case <-tick.C:
			if len(batch) > 0 {
				if err := b.flush(ctx, batch); err != nil && ctx.Err() == nil {
					return err
				}
				batch = batch[:0]
			}
		case <-ctx.Done():
			// Shutdown: flush what we have with the remaining budget, or drop.
			return b.flush(context.WithoutCancel(ctx), batch)
		}
	}
}
```

## Common Mistakes

- **`time.After` inside hot loops**: one timer is created per iteration,
  and the timeout resets with each message; for very hot loops, reuse a
  `NewTimer` to reduce churn and make idle-vs-total timeout semantics explicit.
- **Missing the `ctx.Done()` case**: the select blocks forever when the
  caller cancels: a leak.
- **Assuming order**: multiple ready cases are chosen randomly; do not
  build priority into multi-case selects. Priority needs nesting:
  ```go
  select {
  case v := <-high:
      handle(v)
  default:
      select {
      case v := <-high:
          handle(v)
      case v := <-low:
          handle(v)
      }
  }
  ```
- **Forgetting `Stop()` on tickers/timers**: stop timers when their
  lifetime ends; modern timers are garbage-collected when unreachable,
  but explicit Stop still documents ownership and stops tickers.
- **Draining a closed `done` channel in a loop with send**: closing a
  channel broadcasts to all waiters; use it for "done" semantics, not
  single handoff.

## Idiomatic Go

- Every blocking select has a cancellation case.
- `default` is for explicit non-blocking semantics, never "just in case."
- Stop timers/tickers when their behavior should end; under Go 1.23+ timer semantics, `Stop` is unnecessary solely for collecting an unreachable timer.
- If a select grows past ~4 cases, it is usually two responsibilities,
  split the goroutine.

## Performance Considerations

- Empty select (`select {}`) blocks forever: that's a deliberate sleep,
  not a spin.
- `select` with 2 cases costs about the same as a single channel op; the
  cost grows with case count, and the random-pick scan is per-operation.
- `time.After` at high frequency is a measurable allocation source in
  profiles (see [19-performance](../19-performance/)).

## Concurrency Considerations

- `select` is the building block of backpressure: bounded channels +
  `default`-drop or full-block decisions live here.
- Random readiness pick is the *starvation prevention* mechanism; don't
  defeat it with nested priority selects unless you mean it.
- **Pre-Go 1.23 timer folklore**: current synchronous timer channels
  guarantee that after `Stop` or `Reset` returns, receives cannot observe
  stale values from the previous setting. The old stop-and-drain dance
  was needed for pre-1.23 asynchronous timer channels; see the
  [timer compatibility notes](../meta/versioning.md).

## Security Considerations

Timeouts are a security control: every unbounded wait is a slow-loris
resource hold. Attackers exploit missing deadlines; select-with-deadline
is the defense at the goroutine level (the HTTP-level equivalents are in
[12-http-networking](../12-http-networking/) and
[21-security](../21-security/)).

## Testing Strategy

- Use tiny channels as step barriers; assert *reached states*, not sleeps.
- For timers, inject a clock (small `Clock` interface) or shrink
  durations to milliseconds; never sleep-assert.
- Race-detector runs (`-race`) on every test: a select-heavy test suite
  without `-race` is untested.

## Interview Questions

1. *What happens if multiple select cases are ready?*: Uniform random
   choice; why: prevent starvation.
2. *Why might `time.After` in a loop be a problem?*: It creates one
   timer per iteration and resets the timeout on each message; unreachable
   timers are collectible with modern semantics, but hot loops may benefit
   from timer reuse.
3. *How do you make a channel operation non-blocking?*: select with
   default; show drop/shed metrics use.
4. *How would you implement "take the first successful response of 5
   upstreams"?*: The fan-in race example; the grade is in the
   `defer cancel()` and the buffered channel (no leaked losers).

## Practice Exercises

1. Compare a looped `time.After` consumer with a reused `NewTimer` version.
   Measure allocations with `testing.AllocsPerRun` and test whether the
   timeout measures idle time or a total deadline.
2. Implement `OrDone(ctx, ch)`: relays values until ctx cancels or the
   channel closes, leaking nothing. It's a canonical utility; test it.
3. Build the priority-select pair above; then break priority by
   reordering and write a test that catches it statistically.

## Further Reading

- [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines): canonical
- [time.Timer docs](https://pkg.go.dev/time#Timer): the Stop/Reset contract
- [Go 1.23 timer changes](https://go.dev/doc/go1.23): timer channel semantics
- [time.Timer documentation](https://pkg.go.dev/time#Timer): current Stop/Reset contract
