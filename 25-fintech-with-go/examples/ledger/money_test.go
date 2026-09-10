package ledger

import (
	"errors"
	"math"
	"testing"
)

func mustMoney(t *testing.T, amount int64, currency string) Money {
	t.Helper()
	m, err := New(amount, currency)
	if err != nil {
		t.Fatalf("New(%d, %q): %v", amount, currency, err)
	}
	return m
}

func TestNew_RejectsBadCurrency(t *testing.T) {
	for _, c := range []string{"", "us", "USDD", "usd1"} {
		if _, err := New(100, c); !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("New(100, %q) err = %v, want ErrInvalidAmount", c, err)
		}
	}
	if _, err := New(100, "USD"); err != nil {
		t.Errorf("New(100, USD) unexpected error: %v", err)
	}
}

func TestZeroMoneyIsInvalid(t *testing.T) {
	var m Money
	if _, err := m.Add(mustMoney(t, 1, "USD")); !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("zero Money currency = %q; New must be the only constructor", m.Currency())
	}
}

func TestAdd_CurrencyMismatchIsError(t *testing.T) {
	_, err := mustMoney(t, 100, "USD").Add(mustMoney(t, 100, "EUR"))
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("err = %v, want ErrCurrencyMismatch", err)
	}
}

func TestAdd_OverflowWraps(t *testing.T) {
	got, err := mustMoney(t, math.MaxInt64, "USD").Add(mustMoney(t, 1, "USD"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount() != math.MinInt64 {
		t.Errorf("overflow produced %d; document wrapping in your aggregation layer", got.Amount())
	}
}

func TestNegate(t *testing.T) {
	m := mustMoney(t, 2500, "USD")
	if n := m.Negate(); n.Amount() != -2500 || n.Currency() != "USD" {
		t.Errorf("Negate() = %s, want -2500 USD", n)
	}
}

func TestFloatCompare(t *testing.T) {
	// The test that justifies integers: 0.1+0.02 in float64 is not 0.12.
	// Money keeps 10 and 2 as exact integers instead.
	a := mustMoney(t, 10, "USD")
	b := mustMoney(t, 2, "USD")
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Amount() != 12 {
		t.Errorf("10+2 = %d, want 12", sum.Amount())
	}
}
