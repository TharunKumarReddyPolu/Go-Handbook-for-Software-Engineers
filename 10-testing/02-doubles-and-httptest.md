# Test doubles & httptest

## Why Does This Matter?

Units under test depend on databases, APIs, clocks, and queues. Test
doubles replace those dependencies with fast, controllable stand-ins —
but the *kind* of stand-in determines what your test proves. This
chapter gives you the vocabulary, the decision rules, and the httptest
patterns for the dependency every web service has: HTTP.

## Mental Model

| Double | What it is | Proves |
|---|---|---|
| **Fake** | Working lightweight implementation (in-memory store) | The system works against a *behavioral* contract |
| **Stub** | Hard-coded responses for calls made during the test | Your code reacts correctly to specific return values |
| **Mock** | Expects specific calls; fails on unexpected ones (or records them for assertion) | *Interactions* — that your code asked the right questions |
| **Spy** | Real call wrapped with recording | Both behavior and how it was invoked |

The ordering rule that prevents most double-ology debates:

> **Prefer fakes > stubs > mocks.** The further down you go, the more
> your tests verify *implementation* instead of *behavior*, and the more
> they break during refactors.

Mocks are for interactions that ARE the requirement ("payment was
submitted exactly once"). Everything else deserves a fake.

## Interfaces at the right width

Doubles live behind interfaces, and interface width decides everything:

```go
// TOO WIDE — mocks HTTP: you end up asserting on transport details
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// RIGHT WIDTH — mocks intent: what the domain needs
type CardCharger interface {
	Charge(ctx context.Context, amountMinor int64, token string) (ChargeID, error)
}
```

Declare interfaces **where they are consumed** (consumer-side), not
where implemented. The payments service defines `CardCharger`; the
Stripe adapter satisfies it without importing the service. This is why
Go doesn't need a mocking *framework* to decouple — the language did the
decoupling.

## Fakes — the primary double

```go
// store.go — the real dependency's contract, defined by its consumer
type UserStore interface {
	Get(ctx context.Context, id string) (User, error)
	Put(ctx context.Context, u User) error
}

// fake_test.go — a working in-memory implementation
type fakeStore struct {
	mu     sync.Mutex
	users  map[string]User
	getErr error // injection point for failure-path tests
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]User{}}
}

func (f *fakeStore) Get(ctx context.Context, id string) (User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return User{}, f.getErr
	}
	u, ok := f.users[id]
	if !ok {
		return User{}, ErrNotFound // match the real error contract!
	}
	return u, nil
}

func (f *fakeStore) Put(ctx context.Context, u User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[u.ID] = u
	return nil
}
```

The critical line is the comment on `ErrNotFound`: a fake must honor the
real dependency's error contract, or tests pass while production 500s.

## Stubs and mock-style checks — minimal, inline

Go's culture prefers small hand-rolled doubles over frameworks:

```go
// Stub: returns a scripted response, records nothing else.
type stubCharger struct {
	resp func() error
}
func (s *stubCharger) Charge(ctx context.Context, a int64, t string) (ChargeID, error) {
	return "", s.resp()
}

// Mock-style: records calls for interaction assertions.
type recordingCharger struct {
	mu      sync.Mutex
	charges []struct{ amount int64; token string }
	err     error
}
func (c *recordingCharger) Charge(ctx context.Context, amount int64, token string) (ChargeID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.charges = append(c.charges, struct{ amount int64; token string }{amount, token})
	return "ch_1", c.err
}
```

Then the test asserts on the *recording*:

```go
func TestCheckout_ChargesOnce(t *testing.T) {
	rc := &recordingCharger{}
	svc := &CheckoutService{charger: rc}

	err := svc.Checkout(ctx, Order{TotalMinor: 4200, Token: "tok_1"})
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if n := len(rc.charges); n != 1 { // the interaction IS the spec
		t.Fatalf("charges = %d, want 1", n)
	}
	if rc.charges[0].amount != 4200 || rc.charges[0].token != "tok_1" {
		t.Errorf("charge args = %+v, want amount 4200 token tok_1", rc.charges[0])
	}
}
```

When a codebase grows past a dozen doubles, `matryer/moq` (generate
mocks from interfaces) or `uber-go/mock` are common; the *decision rules
above don't change* — generation is just mechanical convenience.

## httptest — testing HTTP without servers

### Testing your handler

```go
// api.go — the layer under test
func NewHandler(svc *CheckoutService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /checkout", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TotalMinor int64  `json:"total_minor"`
			Token      string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if err := svc.Checkout(r.Context(), Order{TotalMinor: req.TotalMinor, Token: req.Token}); err != nil {
			http.Error(w, "checkout failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
	return mux
}
```

```go
// api_test.go
func TestCheckoutHandler(t *testing.T) {
	rc := &recordingCharger{}
	h := NewHandler(&CheckoutService{charger: rc})

	body := `{"total_minor": 4200, "token": "tok_1"}`
	req := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if n := len(rc.charges); n != 1 {
		t.Errorf("charges = %d, want 1", n)
	}
}
```

`httptest.NewRequest` + `httptest.NewRecorder` run the handler
*synchronously in-process* — no ports, no TLS, no flakes. This is the
default for handler tests.

### Testing your HTTP *client*

`httptest.NewServer` runs a real local server your client code can call:

```go
func TestFetchUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/42" {
			t.Errorf("path = %s", r.URL.Path) // assert on the request your client sent
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"id":"42","name":"Ada"}`)
	}))
	defer srv.Close()

	got, err := FetchUser(context.Background(), srv.URL+"/users/42")
	if err != nil {
		t.Fatalf("FetchUser: %v", err)
	}
	if got.Name != "Ada" {
		t.Errorf("name = %q", got.Name)
	}
}
```

Use `NewServer` for client code; use `NewRecorder` for handler code.
`httptest.NewUnstartedServer` + `srv.TLS = ...` + `srv.StartTLS()`
covers TLS paths when needed.

### Injecting failure into clients

```go
// RoundTripper stub — the client-side interface seam
type failingTransport struct{ status int }
func (f failingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("connection reset")
}

client := &http.Client{Transport: failingTransport{}} // inject into client code via a seam
```

RoundTripper is the client-side equivalent of the store interface —
narrow, standard, and mockable without frameworks.

## Common Mistakes

- **Interface declared by the provider, 14 methods wide** — the mock
  implements a cathedral; every test stubs 12 methods it doesn't care
  about. Split the interface at the consumer.
- **Fake with different error semantics** than the real dependency —
  the highest-value bug source in double-land. Mirror sentinels and
  wrap behavior.
- **Asserting mock call counts that aren't requirements** ("cached.Get
  called exactly twice") — locks in implementation; the refactor tax
  follows.
- **httptest server without Close()** — leaks listeners; port
  exhaustion in big suites.
- **Using `time.Now()` in code with no seam, then sleeping in tests** —
  inject a `now func() time.Time` field; delete the sleeps.

## Idiomatic Go

- Doubles live in `_test.go` files beside their consumers, not a
  `testutil` dumping ground (a shared `testutil` package is fine for
  genuinely cross-cutting helpers — keep it small).
- Constructor injection via struct fields: `svc := &Service{store: s,
  charger: c}` — no DI framework, visible wiring.
- Integration point of the real dependency is tested in the
  integration tier (ch. 03); the unit tier proves *your* logic.

## Performance Considerations

- In-process handler tests run in microseconds; server tests add
  localhost round-trips (~50-200µs) — fine for dozens, heavy for
  thousands.
- Fakes are the fastest double; network-protocol fakes (e.g., miniredis)
  trade fidelity for speed — decide per tier.

## Concurrency Considerations

- Doubles must be as thread-safe as the real thing — a fake store
  without a mutex will (correctly) trip `-race` in parallel tests, and
  that's a *good* outcome: fix the fake.
- `httptest` handler assertions run on the server's goroutine; use
  `t.Errorf` (not Fatal) from handlers, or channel the failure back.

## Security Considerations

- Test doubles embedding real tokens/keys is an incident waiting to
  happen; generate fake credentials with obviously fake formats.
- `httptest.NewServer` binds localhost only — but do not point test
  clients at real third-party URLs; it makes suites network-dependent
  and can trigger real charges.

## Testing Strategy

The doubles' tests themselves follow table-driven style; interaction
assertions stay minimal (the interaction must be a requirement). The
repo's CI runs every test with `-race`, which is what makes hand-rolled
thread-safe doubles verifiable.

## Interview Questions

1. *Fake vs mock — when does each earn its place?* — Behavioral contract
   vs required interaction; grades on preferring fakes.
2. *Where do you declare the interface for a dependency, and why?* —
   Consumer-side; decouples without imports, enables doubles.
3. *Test an HTTP client that must retry on 503.* — httptest server that
   fails N times then succeeds; assert the final result and (if it's a
   requirement) the request count.
4. *How do you test time-based logic (cache TTL)?* — Inject a clock
   interface/func; never sleep.

## Practice Exercises

1. Build `fakeStore` with context-cancellation support (Get/Put return
   ctx.Err() when ctx is done); prove a service honors cancellation
   through the fake.
2. Write the retry-on-503 client test with a httptest server that
   counts requests; assert both recovery and retry count.
3. Replace a mock-heavy test of your own with a fake; count the lines
   the refactor deleted from the test.

## Further Reading

- [httptest package](https://pkg.go.dev/net/http/httptest) — the three constructors and when
- [go-cmp](https://github.com/google/go-cmp) — the common diff library
- [Keep tests on a tight leash](https://www.uber.com/blog/ghosts-of-tested-code/) — interaction-test economics
