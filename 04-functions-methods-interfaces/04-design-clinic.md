# Design clinic: one feature, three ways

## Why Does This Matter?

Abstractions are easier to criticize than to build. The fastest way to
internalize Go's design taste is to watch the same feature get built
three times: once the way class-trained engineers instinctively build
it, once patched into Go idiom, and once the way a Go codebase would
converge on it given time. The feature is deliberately ordinary: a
user-service that validates, stores, and notifies. Every structure in
this chapter exists in real codebases; the names are changed.

## The requirements

- Register a user: validate email, store the record, send a welcome
  notification.
- Notification may fail; registration should still succeed, logged.
- Tests must cover the business rules without a database or mail
  server.
- A second storage backend (Postgres → in-memory) is genuinely
  planned.

## BAD: class-hierarchy Go

The instinct from Java: an abstract base class shape realized as
embedded structs plus an interface taxonomy designed up front.

```go
// BAD: hierarchy thinking, Go syntax.
type AbstractService struct {           // "base class"
    db    *sql.DB
    mailer *SMTPMailer
}

func (a *AbstractService) Register(u *User) error {
    if err := a.preValidate(u); err != nil { return err }
    if err := a.storeUser(u); err != nil { return err }   // hooks?
    return a.sendWelcome(u)
}
func (a *AbstractService) preValidate(u *User) error { return nil }   // template method
func (a *AbstractService) storeUser(u *User) error { return a.db... } // concrete
func (a *AbstractService) sendWelcome(u *User) error { return a.mailer.Send(...) }

type UserService struct { AbstractService }   // "subclass"
func (s *UserService) storeUser(u *User) error { /* custom */ }  // "override"??
```

What goes wrong, concretely:

- **The override does not dispatch**: `AbstractService.Register` calls
  `a.storeUser`, the *embedded* method, not `UserService.storeUser`.
  Embedding has no virtual dispatch; the "template method" pattern
  silently breaks. This is the single most common porting bug.
- **Concrete dependencies baked in**: `*sql.DB` and `*SMTPMailer` are
  untestable without a database and a mail server.
- **Notification failure aborts registration** if it returns error:
  a business rule lost in plumbing.
- **The interface taxonomy is speculative**: `AbstractService`
  anticipates subclasses nobody asked for.

## BETTER: Go syntax, still carrying Java assumptions

The second pass fixes syntax but keeps the shape: interfaces, but
producer-side, wide, and wired through a constructor that takes
everything:

```go
// BETTER: compiles, tests pass, still fights the grain.
type UserServiceInterface interface {       // producer-side, "for flexibility"
    Register(u *User) error
    Validate(u *User) error
    Store(u *User) error
    Notify(u *User) error
    Get(id string) (*User, error)
    Delete(id string) error
}

type UserService struct { db *UserRepo; mail *Mailer }

func NewUserService(db *UserRepo, mail *Mailer) *UserService {
    return &UserService{db: db, mail: mail}
}

func (s *UserService) Register(u *User) error {
    if err := s.Validate(u); err != nil { return err }
    if err := s.Store(u); err != nil { return err }
    if err := s.Notify(u); err != nil { return err }   // still aborts!
    return nil
}
```

Progress: explicit dependencies, interface exists, tests can inject.
Remaining problems:

- **The interface is a mirror of the concrete type** (six methods, one
  implementation): the mock in tests needs six stubs, five of which
  are dead weight in any given test.
- **Producer-side**: `UserRepo` must now import the interface's
  package; the dependency arrow points the wrong way.
- **Business rules entangled**: notification failure aborts
  registration because nobody separated the transactional core from
  the best-effort edges.
- **Positional dependencies** (`*UserRepo`, `*Mailer`) will grow into
  the ten-parameter constructor every big codebase dreads.

## IDIOMATIC: seams where they pay, rules in the domain

```go
// IDIOMATIC: package users (the domain knows its own rules).
type Storer interface {                     // consumer-side, 2 methods
    Get(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, u *User) error
}

type Notifier interface {                   // consumer-side, 1 method
    Welcome(ctx context.Context, u *User) error
}

type Service struct {
    store    Storer
    notifier Notifier
    now      func() time.Time        // time is a dependency too
}

func NewService(s Storer, n Notifier) *Service {
    return &Service{store: s, notifier: n, now: time.Now}
}

func (s *Service) Register(ctx context.Context, email string) (*User, error) {
    u, err := NewUser(email, s.now())          // validation in the constructor
    if err != nil { return nil, err }          // domain error, see 05-errors
    if err := s.store.Save(ctx, u); err != nil {
        return nil, fmt.Errorf("register: %w", err)
    }
    // Best-effort edge: failure is logged, never fails registration.
    if err := s.notifier.Welcome(ctx, u); err != nil {
        slog.Error("welcome notification failed", "user", u.ID, "err", err)
    }
    return u, nil
}
```

What changed and why it matters:

| Decision | BAD | BETTER | IDIOMATIC |
|---|---|---|---|
| Dependencies | concrete, baked in | concrete via constructor | small consumer-side interfaces |
| Validation | template-method hook | method on service | `NewUser` constructor: invalid users unrepresentable |
| Notification rule | aborts on failure | aborts on failure | best-effort edge, logged |
| Interfaces | speculative hierarchy | mirror of the type | two seams, each with a real second use (in-memory double, planned Postgres) |
| Time | global | global | injected `func() time.Time` |
| Test doubles | DB + SMTP required | six-method mock | 3-line inline fakes |

The last row is the tell: the idiomatic version's tests need no
framework, because the interfaces are exactly the seams the tests
need and nothing more. The full runnable version with those tests is
[examples/di](examples/di/).

## What "idiomatic" is not

- **Not zero abstractions**: the interfaces above are load-bearing;
  idiomatic Go is not "no design".
- **Not minimalism as aesthetics**: `now func() time.Time` is one
  field of complexity bought for deterministic time in every test
  involving expiry or timestamps. That trade is real.
- **Not the end of discussion**: the notification edge could be an
  outbox instead of a log line (at-least-once delivery,
  [15-microservices](../15-microservices/README.md)). The clinic's
  shape supports that refactor; the BAD version's does not.

## The generalizable checklist

Run any proposed design through these:

1. **Can the tests run without infrastructure?** If no, find the
   concrete dependency and extract the seam at its consumer.
2. **Does every interface have two users?** (Two implementations, or
   implementation + double.) Otherwise delete it.
3. **Do invalid states cross package boundaries?** If yes, move the
   invariant into a constructor so the type enforces itself.
4. **Can you point at the dependency arrow of every interface?** It
   should point from consumer to interface, with producers unaware.
5. **Are business rules visible in the happy path?** A reader should
   see "validate, save, notify-best-effort" in one glance.

## Common Mistakes

- **Porting template method**: embedding does not dispatch; extract
  the varying step as a function/parameter instead.
- **Interfaces ahead of second use**: the BETTER version's six-method
  interface is the most common real-world shape; it *feels* designed.
- **Business rules in plumbing**: "welcome failure must not fail
  registration" is a domain rule that only shows up when you read the
  happy path as a story.
- **Constructor takes the world**: grows into dependency-injection
  frameworks; the Go answer is Chapter 5's functional options and
  package-level wiring.

## Idiomatic Go

The clinic's pattern generalizes: **concrete first, seams at friction,
domain rules in the domain type, edges best-effort and explicit.**
Every layering question in
[14-backend-development](../14-backend-development/README.md) is this
clinic at larger scale.

## Performance Considerations

The idiomatic version's interface calls and injected `now` cost
nanoseconds: invisible next to the database and network calls the
service exists to make. Optimize data flow, not dispatch
([19-performance](../19-performance/README.md)).

## Concurrency Considerations

The service holds dependencies, not state: safe for concurrent use
exactly as far as its dependencies are. Injected per-request state
lives in `ctx` or parameters, never in fields (the request-scoped
rules are [08-concurrency/03-context](../08-concurrency/03-context.md)).

## Security Considerations

- Validation lives in `NewUser`: invalid emails cannot exist past
  that boundary, which is the only reliable shape for input
  validation ([21-security](../21-security/README.md)).
- The notifier receives a fully validated `*User`, never raw input:
  the trust boundary is the constructor.

## Testing Strategy

The tests this design enables (all in [examples/di](examples/di/)):
happy path with inline fakes; storage failure propagates; notification
failure logged with registration succeeding; validation failures
table-driven. No framework, no sleeps, deterministic clock.

## Interview Questions

1. *Walk me through why the template-method pattern breaks in Go.*:
   Embedding promotes methods but dispatch stays static: the embedded
   type calls its own method, not the outer override.
2. *Where do interfaces go, and what proves they belong?*: Consumer
   side; proof is two real users or a test double with exactly the
   consumer's method set.
3. *How do you keep business rules from leaking into plumbing?*: Put
   them in the domain type's constructor and methods; plumbing
   orchestrates, never decides.
4. *When is injected time worth the field?*: Any logic touching
   expiry, timestamps, or ordering that must be tested
   deterministically; the alternative is flaky sleeps.
5. *A reviewer says the two interfaces are overengineering for one
   storage backend: respond?*: The notifier seam already has two
   users (real + test fake); the store seam has the planned Postgres
   swap; both cost three lines. Compare with the cost of the doubles
   they replace.

## Practice Exercises

1. Extend the clinic: add "resend welcome" to the idiomatic version
   without touching plumbing; note which seams you reused.
2. Take the BAD version and make its notification edge best-effort;
   count the changes and compare with the idiomatic version's diff.
3. Write the six-method BETTER interface's test double by hand; then
   write the two idiomatic fakes. Record which you would maintain.

## Further Reading

- [Go Proverbs](https://go-proverbs.github.io/) (Clear is better than
  clever; A little copying is better than a little dependency)
- [Effective Go: embedding](https://go.dev/doc/effective_go#embedding)
- [Postel's law is not the Go way for internal APIs](https://go.dev/wiki/CodeReviewComments#interfaces)
