package ledger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrUnbalanced is returned when an entry's lines do not sum to zero.
var ErrUnbalanced = errors.New("entry does not balance")

// ErrUnknownAccount is returned when querying an account with no lines.
var ErrUnknownAccount = errors.New("unknown account")

// AccountID names a ledger account.
type AccountID string

// Line is one side of an entry: a signed amount against one account.
// Sign convention: positive = debit (value in), negative = credit.
type Line struct {
	Account AccountID
	Amount  Money
}

// Entry is one immutable journal entry.
type Entry struct {
	ID        int64
	PostedAt  time.Time
	Reference string // audit hook: "payment:k1", "order:o42"
	Lines     []Line
}

// Journal stores entries and enforces the ledger invariants:
//
//  1. Every entry sums to zero across its lines (per currency).
//  2. Entries are immutable once posted (defensive copies, no mutators).
//  3. Balances are derived by summing lines -- never stored.
//  4. Postings to the journal are serialized (the educational model of
//     per-account ordering; production uses DB transaction semantics).
//
// Educational: in-memory. The production contract differs only in
// durability, ordering source, and permission enforcement -- see
// 25-fintech-with-go/02-double-entry-ledger.md.
type Journal struct {
	mu      sync.Mutex
	entries []Entry
	nextID  int64
}

func NewJournal() *Journal { return &Journal{} }

// Post validates and appends an entry atomically. It is the only way
// money moves through the ledger.
func (j *Journal) Post(_ context.Context, ref string, lines []Line) (Entry, error) {
	if ref == "" {
		return Entry{}, fmt.Errorf("%w: empty reference", ErrInvalidAmount)
	}
	if len(lines) < 2 {
		return Entry{}, fmt.Errorf("entry %q: %w: need at least two lines", ref, ErrUnbalanced)
	}

	// Invariant 1: lines sum to zero, all in one currency.
	total, err := New(0, lines[0].Amount.Currency())
	if err != nil {
		return Entry{}, fmt.Errorf("entry %q line 0: %w", ref, err)
	}
	for i, l := range lines {
		if l.Account == "" {
			return Entry{}, fmt.Errorf("entry %q line %d: empty account", ref, i)
		}
		sum, err := total.Add(l.Amount)
		if err != nil {
			return Entry{}, fmt.Errorf("entry %q line %d: %w", ref, i, err)
		}
		total = sum
	}
	if !total.IsZero() {
		return Entry{}, fmt.Errorf("entry %q: %w (sums to %s)", ref, ErrUnbalanced, total)
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	j.nextID++
	e := Entry{
		ID:        j.nextID,
		PostedAt:  time.Now().UTC(),
		Reference: ref,
		Lines:     append([]Line(nil), lines...), // the journal's own copy
	}
	j.entries = append(j.entries, e)
	returned := e
	returned.Lines = append([]Line(nil), e.Lines...) // a separate copy for the caller
	return returned, nil
}

// PostBalance returns lines split across two accounts: amount in from,
// -amount in to. Convenience for the dominant two-account case.
func (j *Journal) PostBalance(ctx context.Context, ref string, from, to AccountID, m Money) (Entry, error) {
	return j.Post(ctx, ref, []Line{
		{Account: from, Amount: m},
		{Account: to, Amount: m.Negate()},
	})
}

// Balance derives an account balance by summing its lines in entry
// order. Derived, never stored -- the ledger is the source of truth.
func (j *Journal) Balance(_ context.Context, a AccountID) (Money, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	first := true
	var bal Money
	for _, e := range j.entries {
		for _, l := range e.Lines {
			if l.Account != a {
				continue
			}
			if first {
				bal = l.Amount
				first = false
				continue
			}
			sum, err := bal.Add(l.Amount)
			if err != nil {
				return Money{}, fmt.Errorf("balance %s: %w", a, err)
			}
			bal = sum
		}
	}
	if first {
		return Money{}, fmt.Errorf("%w: %s", ErrUnknownAccount, a)
	}
	return bal, nil
}

// Total verifies the global invariant: every entry sums to zero, so
// the sum over all accounts of all currencies must be zero. Rebuild-
// style checks like this are the seed of reconciliation (ch. 03).
func (j *Journal) Total(_ context.Context, currency string) (Money, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	total, err := New(0, currency)
	if err != nil {
		return Money{}, err
	}
	for _, e := range j.entries {
		for _, l := range e.Lines {
			if l.Amount.Currency() != currency {
				continue
			}
			sum, err := total.Add(l.Amount)
			if err != nil {
				return Money{}, err
			}
			total = sum
		}
	}
	return total, nil
}

// Entries returns a copy of the entry list (replay/reconciliation input).
func (j *Journal) Entries() []Entry {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]Entry, len(j.entries))
	copy(out, j.entries)
	return out
}

// ReversalEntry builds the balanced inverse of an entry: the idiomatic
// correction. Post it to undo; never edit history.
func ReversalEntry(orig Entry, ref string) []Line {
	lines := make([]Line, len(orig.Lines))
	for i, l := range orig.Lines {
		lines[i] = Line{Account: l.Account, Amount: l.Amount.Negate()}
	}
	return lines
}
