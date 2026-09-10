# Money & payments in Go

## Why Does This Matter?

Most payment-system bugs are not exotic distributed-systems failures —
they are money represented wrong, retries applied twice, and errors
swallowed at boundaries. This chapter fixes the representation problem
permanently and builds the idempotent payment pattern that every later
chapter assumes.

## Mental Model

```text
money = (amount, currency)   — always together, always integers
retries are certain           — therefore idempotency is mandatory
failure is the normal path    — design for it, don't handle it "if it happens"
```

## Money representation — the non-negotiables

### 1. Integers in minor units. Never floats.

```go
// WRONG — binary floats cannot represent 0.1 exactly; drift compounds
var amount float64 = 19.99
amount += 0.01 // 20.000000000000004 on many platforms

// RIGHT — minor units (cents, pence, satoshi, agorot...)
var amountMinor int64 = 1999
amountMinor += 1 // 2000
```

Floats break money three ways: representation error (`0.1` isn't
representable), addition drift, and comparison unpredictability. Every
real system that used float64 for money has a bug list to prove it.

### 2. Currency rides with amount. Always.

```go
// WRONG — add USD to EUR because both are "int64 at a boundary"
total := usdTotal + eurTotal

// RIGHT — the type makes it unrepresentable
total, err := money.Add(usdTotal, eurTotal) // returns error: currency mismatch
```

### 3. Overflow is a design input.

int64 minor units hold ~92 quadrillion cents — far beyond any single
transaction, but *summation* over millions of rows can overflow; your
aggregation layer decides the width (int64 vs big.Int) explicitly.

## The Money type

```go
// examples/ledger/money.go
package ledger

import (
	"errors"
	"fmt"
)

// ErrCurrencyMismatch is returned when operations combine currencies.
var ErrCurrencyMismatch = errors.New("currency mismatch")

// ErrInvalidAmount is returned for malformed amounts.
var ErrInvalidAmount = errors.New("invalid amount")

// Money is an amount in minor units with its currency. The zero value
// is invalid on purpose: New is the only constructor, so a Money value
// is always well-formed (see 01-go-fundamentals zero-values chapter for
// why breaking that rule here is the safer trade).
type Money struct {
	amount   int64
	currency string
}

// New validates and constructs. Currency is the ISO-4217 code; amount
// is minor units (cents for USD).
func New(amount int64, currency string) (Money, error) {
	if currency == "" || len(currency) != 3 {
		return Money{}, fmt.Errorf("%w: currency %q", ErrInvalidAmount, currency)
	}
	return Money{amount: amount, currency: currency}, nil
}

func (m Money) Amount() int64    { return m.amount }
func (m Money) Currency() string { return m.currency }

// Add combines two monies; mismatched currencies are an error, not a bug.
func (m Money) Add(o Money) (Money, error) {
	if m.currency != o.currency {
		return Money{}, fmt.Errorf("%w: %s + %s", ErrCurrencyMismatch, m.currency, o.currency)
	}
	return Money{amount: m.amount + o.amount, currency: m.currency}, nil
}

// Negate returns -m (for the debit/credit pairing in ledger entries).
func (m Money) Negate() Money { return Money{amount: -m.amount, currency: m.currency} }

func (m Money) IsZero() bool { return m.amount == 0 }
func (m Money) IsPositive() bool { return m.amount > 0 }

func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.amount, m.currency) // educational: formatting rules per currency live elsewhere
}
```

Design notes that matter in review:

- The zero value is *invalid*, forcing construction through `New` — a
  deliberate exception to the zero-value-usability rule, because a
  silently-zero money value is a loss.
- Amounts are unexported; nothing can build a mixed-currency sum by
  struct literal.
- Comparisons compare currency first; equality includes currency.

## Idempotent payments — the durable claim pattern

Section 18 introduced claim/apply/release against an in-memory store.
For payments, the store is durable (a database) and the pattern
tightens:

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Payment Service
    participant DB as Ledger DB
    participant P as PSP

    C->>S: POST /payments (Idempotency-Key: k1)
    S->>DB: BEGIN; INSERT claim k1 (ON CONFLICT DO NOTHING)
    alt claim won
        S->>P: charge(amount)
        P-->>S: auth code
        S->>DB: INSERT entry (same tx); COMMIT
        S-->>C: 201 payment_id
    else claim exists
        S->>DB: SELECT result for k1
        S-->>C: original result (201 or error), same status
    end
```

The rules that make this correct:

1. **Claim and result live in the same transaction** as the money
   movement. A payment recorded without its claim, or a claim without
   its payment, is a reconciliation finding.
2. **Replays return the original result** — same status, same body
   shape. A client that retries a *succeeded* payment must not get a
   fresh 201 with a new ID (that reads as a second payment).
3. **Timeouts vs errors**: a PSP timeout is *unknown*, not failure. The
   claim stays pending; a background job resolves it against the PSP
   (query by idempotency key) before anyone retries. Treating unknown
   as failure is how double charges happen.
4. **Keys are scoped per operation** (payment creation vs capture are
   different key namespaces).

## Retries — the classification applied to money

The [errors section's Retryable](../05-errors/02-error-design.md) gets
stricter where money moves:

| PSP response | Meaning | Action |
|---|---|---|
| Declined (insufficient funds, etc.) | final for this attempt | no retry; record; return to caller |
| Network timeout | unknown | do NOT blind-retry; reconcile via query |
| 5xx from PSP | unknown/possibly processed | query-before-retry; then retry with same key |
| 4xx validation | final | no retry; fix input |

The regex: **money never retries blind.** Either the operation is
idempotent-keyed (PSP sees the same key and collapses it — Stripe-style,
see Further Reading) or it queries for the outcome first. Both, done
right, compose.

## Basic Example — a payment service boundary

```go
// Educational: the shape, not an audited implementation.
type PaymentService struct {
	claims  ClaimStore   // durable, transactional
	psp     PSP          // payment service provider client
	entries *Journal     // the double-entry journal (ch. 02)
}

func (s *PaymentService) CreatePayment(ctx context.Context, req CreatePayment) (PaymentResult, error) {
	if req.IdempotencyKey == "" {
		return PaymentResult{}, errors.New("idempotency key required")
	}

	// Durable claim in the same tx as the eventual entry.
	won, prior, err := s.claims.Claim(ctx, req.IdempotencyKey, req)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("claim: %w", err)
	}
	if !won {
		return prior, nil // replay: return the original result verbatim
	}

	auth, err := s.psp.Charge(ctx, req.Amount, req.Source)
	if err != nil {
		var decl *DeclinedError
		if errors.As(err, &decl) {
			// Final: record the decline as the claim's result. No retry.
			s.claims.Resolve(ctx, req.IdempotencyKey, declinedResult(decl))
			return PaymentResult{}, err
		}
		// Unknown (timeout/5xx): leave the claim pending for reconciliation.
		return PaymentResult{}, fmt.Errorf("charge outcome unknown: %w", err)
	}

	if err := s.entries.Post(ctx, paymentEntries(req, auth)); err != nil {
		// Money entry failed after a successful charge: DO NOT retry the
		// charge. Flag for the reconciliation job (ch. 03).
		s.claims.MarkNeedsReconcile(ctx, req.IdempotencyKey, auth.ID)
		return PaymentResult{}, fmt.Errorf("post entries for %s: %w", auth.ID, err)
	}
	return successResult(auth), nil
}
```

Note what is deliberately missing: no `for attempt := range` retry loop
around the charge. Money retries are *workflow* (claims + queries +
reconciliation), not loops.

## Common Mistakes

- **float64 anywhere near money** — including "just for display";
  formatting from minor units at the edge.
- **Idempotency keys optional** — a client without a key cannot retry
  safely; the API makes the key required.
- **Replaying errors as fresh errors** — a replayed decline must return
  the same decline; fresh error text leaks processing state.
- **Treating timeouts as declines** — the double-charge machine; the
  pending-claim workflow exists for exactly this.
- **Rounding during currency conversion inside business logic** —
  conversions are their own logged, audited operation with explicit
  rounding rules.
- **Storing money as string in JSON APIs** — parse to int64 minor
  units at the boundary; strings invite locale errors.

## Idiomatic Go (money-shaped)

- `Money` is a value type; pass by value, no pointers — it is 24-ish
  bytes and identity never matters.
- Ledger entries are immutable once posted: no update API exists, which
  the type system encourages by unexporting mutators.
- Errors carry the operation and identifier (`post entries for
  auth_x`), per the [errors section](../05-errors/02-error-design.md).

## Performance Considerations

- Money math is integer math: nanoseconds. Hot paths are hot because of
  I/O and locking, never arithmetic.
- Ledger posting throughput is DB-bound; batch inserts of entry lines
  and serialize per-account ordering (ch. 02/03).
- High-throughput payment rails (thousands TPS) are a *sharding +
  batching* problem — see [19-performance](../19-performance/) for the
  measurement discipline before believing any scaling claim.

## Concurrency Considerations

- Claims must be atomic across concurrent replays — the DB's unique
  constraint is the arbiter, not application-level checks.
- Account balance invariants under concurrency are ledger-internal
  (single-writer per account or row locks); `ledger_test.go` includes
  the concurrent-posting test that pins this.

## Security Considerations

- Idempotency keys are credentials-adjacent: scope them per customer,
  validate before lookup, rate-limit key probing (see
  [21-security](../21-security/)).
- PSP credentials live in a secret manager, never in config files;
  request signing and mTLS per the PSP's contract.
- Audit logs (ch. 04) are append-only and access-controlled; a payment
  system without tamper-evident history is a compliance incident.

## Testing Strategy

- Property tests: random valid operations against the ledger keep
  total-balance invariants (`ledger_test.go` demonstrates).
- Concurrency tests under `-race`: parallel postings, parallel claim
  replays.
- Idempotency tests: same key twice → same result, single side effect;
  failure-then-retry path; unknown-outcome path (timeouts) stays
  pending.
- Reconciliation tests: injected "charge succeeded but entry failed"
  is *found* by the reconciliation query, not by hope.

## Interview Questions

1. *Why are floats wrong for money? Show the failure.* — 0.1
   representation, addition drift; the candidate who mentions
   decimal-vs-minor-units tradeoffs (fixed-point vs arbitrary
   precision) grades highest.
2. *Design idempotent payment creation.* — The sequence diagram; grade
   on: claim+result same tx, replay-returns-original, unknown-outcome
   handling.
3. *A charge succeeded but your DB write failed. What happens next?* —
   No blind retry; reconcile; the answer that invents a compensating
   "refund" before reconciliation loses points.
4. *Your PSP charges twice on a timeout. Whose bug?* — Both sides:
   your client retried without the idempotency key (or before
   reconciliation), the PSP collapsed or didn't per its contract; the
   fix is workflow, not blame.

## Practice Exercises

1. Write `Mul(ratio float64, round Rounding) (Money, error)` for
   currency conversion with explicit rounding modes, and a property
   test that `Add`/`Mul` never produce negative amounts from positive
   inputs.
2. Implement the pending-claim workflow with a fake PSP that times out
   once, then exposes a query endpoint; prove reconciliation resolves
   it exactly once.
3. Fuzz `Money.Add`/`Negate` for overflow behavior; document where
   int64 stops being enough and what the design says to do.

## Further Reading

- [Stripe: idempotency keys](https://docs.stripe.com/api/idempotent_requests) — the industry reference contract
- [ISO 4217 currency codes](https://www.iso.org/iso-4217-currency-codes.html) — minor-unit reality (JPY has 0 decimals!)
- [Go blog: constants](https://go.dev/blog/constants) — why untyped constants ease integer money math
