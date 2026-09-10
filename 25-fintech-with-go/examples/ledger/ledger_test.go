package ledger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

var ctx = context.Background()

func TestPost_RejectsUnbalanced(t *testing.T) {
	j := NewJournal()
	_, err := j.Post(ctx, "bad", []Line{
		{Account: "customer_cash", Amount: mustMoney(t, 100, "USD")},
		// missing the balancing -100 side
	})
	if !errors.Is(err, ErrUnbalanced) {
		t.Fatalf("err = %v, want ErrUnbalanced", err)
	}
}

func TestPost_RejectsSingleLine(t *testing.T) {
	j := NewJournal()
	_, err := j.Post(ctx, "bad", []Line{
		{Account: "customer_cash", Amount: mustMoney(t, 100, "USD")},
	})
	if !errors.Is(err, ErrUnbalanced) {
		t.Fatalf("err = %v, want ErrUnbalanced", err)
	}
}

func TestPost_RejectsMixedCurrencies(t *testing.T) {
	j := NewJournal()
	_, err := j.Post(ctx, "bad", []Line{
		{Account: "customer_cash", Amount: mustMoney(t, 100, "USD")},
		{Account: "revenue", Amount: mustMoney(t, -100, "EUR")},
	})
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("err = %v, want ErrCurrencyMismatch", err)
	}
}

func TestPostBalanced(t *testing.T) {
	j := NewJournal()
	e, err := j.PostBalance(ctx, "payment:k1", "customer_cash", "revenue",
		mustMoney(t, 2500, "USD"))
	if err != nil {
		t.Fatalf("PostBalance: %v", err)
	}
	if len(e.Lines) != 2 || e.ID == 0 || e.Reference != "payment:k1" {
		t.Errorf("entry = %+v", e)
	}

	cash, err := j.Balance(ctx, "customer_cash")
	if err != nil {
		t.Fatal(err)
	}
	if cash.Amount() != 2500 {
		t.Errorf("customer_cash = %s, want 2500 USD", cash)
	}
	rev, _ := j.Balance(ctx, "revenue")
	if rev.Amount() != -2500 {
		t.Errorf("revenue = %s, want -2500 USD", rev)
	}
}

func TestEntriesAreImmutable(t *testing.T) {
	j := NewJournal()
	e, _ := j.PostBalance(ctx, "x:1", "a", "b", mustMoney(t, 500, "USD"))

	// Mutating the returned copy must not affect the journal.
	e.Lines[0].Amount = mustMoney(t, 999999, "USD")

	bal, _ := j.Balance(ctx, "a")
	if bal.Amount() != 500 {
		t.Errorf("journal balance = %s after caller mutated a copy; defensive copy failed", bal)
	}
}

func TestUnknownAccount(t *testing.T) {
	j := NewJournal()
	if _, err := j.Balance(ctx, "ghost"); !errors.Is(err, ErrUnknownAccount) {
		t.Errorf("err = %v, want ErrUnknownAccount", err)
	}
}

func TestReversalLeavesBalancesUnchanged(t *testing.T) {
	j := NewJournal()
	orig, _ := j.PostBalance(ctx, "payment:k2", "customer_cash", "revenue",
		mustMoney(t, 1000, "USD"))

	if _, err := j.Post(ctx, "reversal:k2", ReversalEntry(orig, "reversal:k2")); err != nil {
		t.Fatalf("reversal: %v", err)
	}

	cash, _ := j.Balance(ctx, "customer_cash")
	rev, _ := j.Balance(ctx, "revenue")
	if cash.Amount() != 0 || rev.Amount() != 0 {
		t.Errorf("after reversal: cash=%s revenue=%s, want 0/0", cash, rev)
	}
}

func TestConcurrentPosts_StayBalanced(t *testing.T) {
	j := NewJournal()
	const n = 100
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Go(func() {
			_, _ = j.Post(ctx, fmt.Sprintf("tx-%d", i), []Line{
				{Account: "customer_cash", Amount: mustMoney(t, 1, "USD")},
				{Account: "revenue", Amount: mustMoney(t, -1, "USD")},
			})
		})
	}
	wg.Wait()

	cash, _ := j.Balance(ctx, "customer_cash")
	rev, _ := j.Balance(ctx, "revenue")
	if cash.Amount() != n || rev.Amount() != -n {
		t.Errorf("post-race lost money: cash=%s revenue=%s, want %d/%d", cash, rev, n, -n)
	}
	if total, _ := j.Total(ctx, "USD"); !total.IsZero() {
		t.Errorf("global total = %s, want zero (invariant 1)", total)
	}
	// Run with -race: the mutex discipline is part of the contract.
}

func TestReplayRebuildsBalances(t *testing.T) {
	j := NewJournal()
	for i := 0; i < 10; i++ {
		_, err := j.PostBalance(ctx, fmt.Sprintf("p:%d", i), "cash", "revenue",
			mustMoney(t, int64(10+i), "USD"))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Rebuild a fresh journal from serialized entries: the reconciliation
	// smoke test -- balances must match exactly.
	fresh := NewJournal()
	for _, e := range j.Entries() {
		if _, err := fresh.Post(ctx, e.Reference, e.Lines); err != nil {
			t.Fatalf("replay %q: %v", e.Reference, err)
		}
	}
	want, _ := j.Balance(ctx, "cash")
	got, _ := fresh.Balance(ctx, "cash")
	if got != want {
		t.Errorf("replayed balance = %s, want %s", got, want)
	}
}
