# Graceful shutdown

## Why Does This Matter?

`log.Fatal(srv.ListenAndServe())` ends every tutorial, and it teaches
the one habit that causes deploy-time errors: when you kill that
process mid-deploy, in-flight requests die mid-response. Clients see
connection resets, retries multiply load, and Kubernetes marks the
next pod unready too. Graceful shutdown is a small amount of code with
an outsized reliability return, and the stdlib provides all of it.

## Mental Model

Shutdown is a handshake with three parties, in the right order:

```mermaid
sequenceDiagram
    participant K as stop signal (SIGTERM)
    participant S as server
    participant H as in-flight handlers
    participant D as dependencies
    K->>S: signal
    S->>K: stop accepting new conns
    Note over S: readiness flips false (LB/K8s stops routing)
    K->>S: grace period starts
    S->>H: Serve returns; contexts canceled
    H->>D: finish writes, flush, ack
    H-->>S: handlers return
    S->>K: exit 0 (or force after grace)
```

The order that matters: **stop receiving work first, then finish the
work you have.** Reversing the order (canceling contexts before
deregistering from the load balancer) guarantees errors.

## Syntax / API: the full lifecycle

```go
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("GET /readyz", handleReady) // sees the atomic flag

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		log.Fatalf("server failed to start: %v", err) // not shutdown
	case <-ctx.Done(): // signal received
	}

	// 1. Flip readiness first: the LB stops sending work.
	atomic.StoreInt32(&ready, 0)

	// 2. Stop accepting; wait for in-flight with a grace period.
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), 25*time.Second) // < K8s terminationGracePeriod
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("forced close after grace period: %v", err)
		_ = srv.Close() // still-running handlers are abandoned
	}

	// 3. Release other resources AFTER the server drained.
	store.Close()
	log.Println("bye")
}
```

What each method actually does:

- **`Shutdown(ctx)`**: closes listeners immediately (no new conns),
  then waits for *idle* connections to close and active connections to
  become idle, until the context deadline. Returns the error when it
  gives up. WebSockets and streaming responses are **not** idle, so
  they block `Shutdown` until your grace period: close them yourself
  (track them in a registry) or bound them with timeouts.
- **`Close()`**: hard kill, everything. Only after `Shutdown` failed.
- **`RegisterOnShutdown(f)`**: hooks for background workers to stop
  before the drain finishes.

## Basic Example: the readiness flag

```go
var ready atomic.Bool // init true; flipped false on shutdown

func handleReady(w http.ResponseWriter, _ *http.Request) {
	if !ready.Load() {
		http.Error(w, "draining", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintln(w, "ok")
}
```

Kubernetes (and any sane LB) stops routing to the pod the moment
`/readyz` fails, which means the grace period is spent serving real
traffic, not new requests that arrived after SIGTERM.

## Real-World Example: shutdown ordering with dependencies

Dependencies close in **reverse acquisition order**, and the HTTP
server is usually acquired first:

1. flip readiness,
2. `srv.Shutdown` (drain requests),
3. drain/flush async machinery (Kafka producers, batchers: their
   `Close` flushes; see
   [18-kafka-with-go](../18-kafka-with-go/)),
4. close the database pool.

Closing the pool before draining the server converts every in-flight
request into an error; that is the bug that "we added graceful
shutdown" was supposed to fix.

The same `signal.NotifyContext` pattern governs background loops: a
worker selecting on `ctx.Done()` (as in
[08 §3](../08-concurrency/03-context.md)) stops accepting new items
and finishes the current one, which is the identical handshake at
goroutine scale.

## Production Example: what the runnable example does

`examples/api/` wires the full lifecycle: signal context, readiness
flag, `Shutdown` with a grace period, and a test that sends SIGTERM to
the process and asserts (a) the listener stops accepting immediately,
(b) an in-flight slow request still completes with a 200, (c) exit
happens inside the grace period. Those three assertions are the
contract; CI runs them on every commit.

## Common Mistakes

- **Calling `srv.Close()` instead of `Shutdown`** "because Shutdown
  hangs": it hangs because something (usually a websocket or a handler
  without timeouts) is stuck. Find that; do not swap in the hard kill.
- **Grace period ≥ the platform's kill timeout.** Kubernetes sends
  SIGKILL at `terminationGracePeriodSeconds`; a 30-second grace inside
  a 30-second termination window means SIGKILL always wins and you
  get none of the benefit. Grace < termination, strictly.
- **Forgetting readiness, or checking it before the listener binds.**
  A pod that is ready before it listens sends traffic into a void;
  the readiness flag and the listener must come up together.
- **Pre-Bind SIGTERM race**: `signal.NotifyContext` before the
  listener starts means a signal during startup is handled cleanly
  instead of killing a half-initialized process.
- **Long-lived connections (websockets, SSE) never going idle**, so
  `Shutdown` waits the full grace period every deploy. Track and close
  them on shutdown.
- **`log.Fatal` in the serve goroutine**: it calls `os.Exit` and skips
  every deferred cleanup, including `Shutdown`. Send errors to a
  channel; let `main` decide.

## Idiomatic Go

- `signal.NotifyContext` over raw signal channels: the context
  composes with everything else.
- One shutdown function per process, built from reverse-ordered
  closers; test it the way you test any function.
- Exit code 0 after graceful completion; nonzero only on real
  failures (orchestrators restart on nonzero, which you do not want
  during a planned stop).

## Performance Considerations

- Shutdown cost is per-connection bookkeeping; the deploy-time cost
  that matters is connection establishment: with `Keep-Alive` and
  proper pool settings (chapter 4), new pods warm up in milliseconds.
- Probes every second on a hot endpoint is wasted work; readiness
  should check cheap local state, not dependencies (a dependency blip
  then removes you from rotation for a problem you did not have).

## Concurrency Considerations

- `Shutdown` waits on connection state, not on your goroutines. Any
  goroutine a handler spawned must be tied to the request context or
  to an `OnShutdown` hook, or it survives the drain and dies with the
  process mid-write.
- The readiness flag must be atomic or mutex-guarded: it is read by
  every probe and written by the shutdown path, a textbook race
  otherwise.

## Security Considerations

- Shut down cleanly so audits and connection logs complete; a SIGKILL
  mid-request leaves half-written access records.
- Health endpoints must not leak dependency topology or versions;
  "ok"/"draining" is enough for everyone (metrics in
  [20-observability](../20-observability/) carry the detail).

## Testing Strategy

- In-process: start `httptest.NewServer`, fire a slow handler, call
  `srv.Shutdown` from the test, assert the slow response completed.
  No signals needed, fully deterministic.
- Process-level: the runnable example's SIGTERM test. Slower, but it
  proves the actual binary behaves.
- Chaos the ordering: a test that flips readiness *after* shutdown
  (the wrong order) documents why the flag exists.

## Interview Questions

1. Walk through everything that happens between SIGTERM and process
   exit in your last service.
2. Why flip readiness before calling `Shutdown`?
3. What happens to websocket connections during `Shutdown`, and what
   do you do about it?
4. Grace period vs `terminationGracePeriodSeconds`: how do you pick?
5. A handler spawned a goroutine writing to Kafka: who stops it, when?

## Practice Exercises

1. Add graceful shutdown to a server you have, then kill it with
   `kill -TERM` mid-request-storm and count the client errors: zero
   is the target.
2. Write the in-process shutdown test from the Testing Strategy
   section.
3. Make readiness reflect a dependency check, then reason (in a
   comment) about what happens to the service during that
   dependency's next blip. Revert it to local state afterward.

## Further Reading

- [http.Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
- [signal.NotifyContext](https://pkg.go.dev/os/signal#NotifyContext)
- [Kubernetes container lifecycle](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/)
