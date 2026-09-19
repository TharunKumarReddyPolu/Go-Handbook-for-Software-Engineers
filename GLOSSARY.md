# Glossary

Definitions used consistently across the handbook. When a chapter and this
file disagree, this file wins and the chapter should be fixed.

## Concurrency

**Goroutine**: A lightweight, independently scheduled function running
concurrently with others. Started with `go f()`. Costs a few KB of stack
that grows on demand; thousands are routine.

**Channel**: A typed conduit for passing values between goroutines with
happens-before guarantees. Unbuffered channels synchronize sender and
receiver; buffered channels decouple them up to capacity.

**Buffered channel**: A channel with capacity; sends succeed while buffer
space remains. Backpressure appears when the buffer fills.

**Channel ownership**: The discipline that one goroutine creates, writes,
and closes a channel; receivers only read. Prevents most channel bugs.

**Directional channel**: A channel type restricted to send-only
(`chan<- T`) or receive-only (`<-chan T`), enforced at compile time.

**select**: Wait on multiple channel operations, choosing whichever is
ready; with `default`, becomes non-blocking.

**Context**: Carries cancellation signals, deadlines, and request-scoped
values across API boundaries and goroutines. The standard mechanism for
timeout and shutdown.

**Data race**: Two goroutines access the same memory concurrently with at
least one write, without synchronization. Undefined behavior in Go.
Detected by `-race`.

**Race condition**: Broader: correctness depends on timing or
interleaving. Every data race is a race condition; not every race condition
involves a data race.

**Deadlock**: Goroutines blocked forever waiting on each other. Go's
runtime detects total deadlocks of all goroutines and panics; partial
deadlocks hang.

**Starvation**: A goroutine perpetually fails to acquire a resource
because others monopolize it.

**Goroutine leak**: A goroutine blocked forever because its exit path was
never taken. Accumulates memory and descriptors.

**Fan-out / fan-in**: Distributing work across multiple goroutines
(fan-out); merging results into one consumer (fan-in).

**Backpressure**: Signaling producers to slow down when consumers cannot
keep up; typically bounded buffers + blocking sends, or explicit
acknowledgment.

**Semaphore**: A counting primitive limiting concurrent access to a
resource; in Go usually a buffered channel of tokens.

**Happens-before**: The memory-model ordering guarantee that makes one
goroutine's writes visible to another (e.g., a channel send happens-before
the corresponding receive).

## Memory and runtime

**Stack**: Per-goroutine memory for local variables and call frames;
grows/shrinks automatically. Cheap to allocate.

**Heap**: Shared memory for values whose lifetime exceeds their frame.
Allocation and collection cost more than stack.

**Escape analysis**: The compiler deciding whether a value can stay on
the stack or must "escape" to the heap. Inspect with
`go build -gcflags=-m`.

**Garbage collector (GC)**: Go's concurrent, tri-color mark-and-sweep
collector that reclaims unreachable heap memory. Tuned for low pause
times; pressure (allocation rate) matters more than live set.

**GOGC**: The old knob controlling GC target as a percentage of live
heap; superseded in practice by GOMEMLIMIT.

**GOMEMLIMIT**: A soft memory limit for the runtime; makes Go safe to run
in containers with fixed memory.

**GOMAXPROCS**: Maximum OS threads simultaneously executing Go code.
Defaults to the CPU count and, since Go 1.25, is cgroup-aware in
containers. Set it explicitly when the runtime cannot infer the quota.

**Scheduler**: The runtime component distributing goroutines over OS
threads using an M:N model with work stealing.

**P, M, G**: Scheduler's three abstractions: logical processor (P), OS
thread (M), goroutine (G).

**Pointer semantics / value semantics**: Whether assignment shares the
underlying value (pointers, slices, maps) or copies it (numbers, structs
of copyable fields).

## Language

**Interface**: A set of method signatures. Satisfied implicitly; values
carry (type, value) pairs internally.

**Method set**: The set of methods available on a type or its pointer,
determining which interfaces it satisfies. Value receivers join both sets;
pointer receivers join only the pointer's.

**Embedding**: Composing types by nesting them, which promotes methods
without inheritance or subtyping.

**Type assertion**: Extracting a concrete type from an interface value:
`v, ok := x.(T)`.

**Type switch**: A switch over the dynamic type of an interface value.

**Generics**: Type parameters on functions and types (Go 1.18) and on
methods (Go 1.27), constrained by interface-like type sets.

**Constraint**: The interface-like requirement a type argument must
satisfy, e.g. `[T constraints.Ordered]`.

**Zero value**: The value a type takes without initialization: 0, "",
false, nil, or the zero struct. Go guarantees it exists and is safe.

**Sentinel error**: A predeclared error value compared with `errors.Is`
(`io.EOF`, `sql.ErrNoRows`).

**Error wrapping**: Adding context to an error while preserving the chain
for `errors.Is`/`errors.As`, using `%w`.

## Modules and tooling

**Module**: A versioned unit of dependency management rooted at a
`go.mod` file.

**go.sum**: Cryptographic checksums pinning dependency content for
reproducible builds.

**internal**: A package path element restricting imports to the parent
tree; the compiler-enforced visibility tool.

**Workspace**: A `go.work` file letting multiple local modules develop
together without replace directives.

**Race detector**: The `-race` build mode instrumenting memory accesses
to find data races at runtime.

**pprof**: The profiling formats and the tool that reads them: CPU, heap,
goroutine, mutex, and block profiles.

**PGO (profile-guided optimization)**: Feeding a CPU profile into the
build so the compiler optimizes hot paths.

**govulncheck**: Scans dependencies for known vulnerabilities reachable
from your code's call graph.

## Data and backend

**Idempotency**: Producing the same result when an operation repeats;
essential wherever retries exist. Payment systems key it with idempotency
tokens.

**At-least-once delivery**: Guarantee that messages arrive at least once,
with possible duplicates; consumers must deduplicate.

**At-most-once delivery**: Guarantee that messages arrive no more than
once, with possible losses.

**Exactly-once semantics**: The marketing term for "no duplicates, no
losses"; achievable only end-to-end with idempotent or transactional
consumers, never by a transport alone.

**Consumer group**: Kafka's load-balancing unit: partitions are divided
among group members; each partition has one consumer per group.

**Partition**: Kafka's unit of parallelism and ordering; ordering is
guaranteed within, never across, partitions.

**Offset**: A consumer's position in a partition's log.

**Backpressure (systems)**: End-to-end flow control: reject, queue, or
degrade when downstream cannot keep up.

**Circuit breaker**: A client-side guard that fails fast after repeated
dependency failures, letting it recover.

**Saga**: A sequence of local transactions with compensating actions for
rollback across services.

**Outbox pattern**: Writing domain events to a database table in the same
transaction as the state change; a relay publishes them, solving the
dual-write problem.

**Double-entry bookkeeping**: Recording every transaction as balanced
debits and credits; the foundation of financial ledgers.

**Reconciliation**: Proving that two independent records of the same
money agree; the fintech discipline that catches what invariants miss.

**Eventual consistency**: Replicas converge given time and no new
updates; reads may be stale without coordination.

**CAP theorem**: Under a network partition, a distributed store must
choose consistency or availability.

**Quorum**: The number of replicas that must agree for a read or write to
succeed (e.g., W+R>N for strong-ish consistency).
