package bank

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// storeCases lists every PaymentStore implementation the unit tier
// must behave identically to. The fake always runs; the db tier
// appends the Postgres store (postgres_test.go). One harness, two
// tiers, one contract (13 Section 4's parity idea).
//
// Each subtest receives a fresh store via build, so cumulative
// effects never leak between cases (the Postgres build re-seeds and
// drops tables; see postgres_test.go).
var storeCases = []struct {
	name  string
	build func(t *testing.T) PaymentStore
}{
	{"fake", func(t *testing.T) PaymentStore {
		t.Helper()
		return NewFakeStore(map[string]int64{"cus_1": 10_000}, "USD")
	}},
}

func runStoreTests(t *testing.T) {
	t.Helper()
	for _, tc := range storeCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("charge success", func(t *testing.T) {
				s := tc.build(t)
				got, err := s.Charge(t.Context(), Payment{CustomerID: "cus_1", AmountMinor: 4_200, Currency: "USD"})
				if err != nil {
					t.Fatalf("Charge: %v", err)
				}
				if got.ID == "" || got.Status != "charged" {
					t.Errorf("payment not materialized: %+v", got)
				}
				acct, err := s.Account(t.Context(), "cus_1")
				if err != nil {
					t.Fatalf("Account: %v", err)
				}
				if acct.BalanceMinor != 5_800 {
					t.Errorf("balance = %d, want 5800 (atomic debit)", acct.BalanceMinor)
				}
			})

			t.Run("overdraft rejected, nothing changed", func(t *testing.T) {
				s := tc.build(t)
				_, err := s.Charge(t.Context(), Payment{CustomerID: "cus_1", AmountMinor: 99_999, Currency: "USD"})
				if !errors.Is(err, ErrInsufficientFunds) {
					t.Fatalf("err = %v, want ErrInsufficientFunds", err)
				}
				acct, err := s.Account(t.Context(), "cus_1")
				if err != nil {
					t.Fatalf("Account: %v", err)
				}
				if acct.BalanceMinor != 10_000 {
					t.Errorf("balance = %d, want unchanged 10000", acct.BalanceMinor)
				}
			})

			t.Run("list newest first, bounded", func(t *testing.T) {
				s := tc.build(t)
				ctx := t.Context()
				for i := 0; i < 5; i++ {
					if _, err := s.Charge(ctx, Payment{CustomerID: "cus_1", AmountMinor: 100, Currency: "USD"}); err != nil {
						t.Fatalf("charge %d: %v", i, err)
					}
				}
				got, err := s.ByCustomer(ctx, "cus_1", 3)
				if err != nil {
					t.Fatalf("ByCustomer: %v", err)
				}
				if len(got) != 3 {
					t.Fatalf("len = %d, want 3 (limit honored)", len(got))
				}
				first, _ := s.ByCustomer(ctx, "cus_1", 1)
				if len(first) == 1 && got[0].ID != first[0].ID {
					t.Errorf("first item of limited list (%s) != newest (%s)", got[0].ID, first[0].ID)
				}
			})

			t.Run("unknown account", func(t *testing.T) {
				s := tc.build(t)
				_, err := s.Charge(t.Context(), Payment{CustomerID: "ghost", AmountMinor: 1, Currency: "USD"})
				if !errors.Is(err, ErrNotFound) {
					t.Fatalf("err = %v, want ErrNotFound", err)
				}
			})

			t.Run("concurrent charges conserve balance", func(t *testing.T) {
				s := tc.build(t)
				const n, each = 50, 100 // total 5000 <= 10000 balance
				var wg sync.WaitGroup
				errs := make(chan error, n)
				for i := 0; i < n; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						_, err := s.Charge(t.Context(), Payment{CustomerID: "cus_1", AmountMinor: each, Currency: "USD"})
						if err != nil {
							errs <- err
						}
					}()
				}
				wg.Wait()
				close(errs)
				for err := range errs {
					t.Error(err)
				}
				acct, err := s.Account(t.Context(), "cus_1")
				if err != nil {
					t.Fatalf("Account: %v", err)
				}
				if acct.BalanceMinor != 10_000-n*each {
					t.Errorf("balance = %d, want %d (no lost updates)", acct.BalanceMinor, 10_000-n*each)
				}
			})
		})
	}
}

func TestStores(t *testing.T) {
	runStoreTests(t)
}

// TestFakeRaceBait floods one fake store from many goroutines without
// waiting: the race detector (CI runs -race) flags any unsynchronized
// access. Deterministic pass/fail, schedule-independent.
func TestFakeRaceBait(t *testing.T) {
	s := NewFakeStore(map[string]int64{"cus_1": 1 << 20}, "USD")
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ctx := t.Context()
			_, _ = s.Charge(ctx, Payment{CustomerID: "cus_1", AmountMinor: 1, Currency: "USD"})
			_, _ = s.ByCustomer(ctx, "cus_1", 10)
			_, _ = s.Account(ctx, "cus_1")
		}(i)
	}
	wg.Wait()
}

// TestWithinTx_LeaseAlwaysReleased pins the transaction-function
// contract from chapter 2 at the unit tier: the fn's failure, panic,
// and success paths all release the lease. Uses the real sql types
// against a registered fake driver.
func TestWithinTx_LeaseAlwaysReleased(t *testing.T) {
	db, drv := newLeakTrackingDB(t)

	// Success path: exactly one commit.
	if err := withinTx(t.Context(), db, func(*sql.Tx) error { return nil }); err != nil {
		t.Fatalf("withinTx: %v", err)
	}
	if got := drv.commits.Load(); got != 1 {
		t.Errorf("commits = %d, want 1", got)
	}

	// Failure path: exactly one rollback, error propagated.
	wantErr := errors.New("boom")
	if err := withinTx(t.Context(), db, func(*sql.Tx) error { return wantErr }); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if got := drv.rollbacks.Load(); got != 1 {
		t.Errorf("rollbacks = %d, want 1", got)
	}

	// Panic path: rollback, then the panic continues.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic did not propagate")
			}
		}()
		_ = withinTx(t.Context(), db, func(*sql.Tx) error { panic("mid-tx") })
	}()
	if got := drv.rollbacks.Load(); got != 2 {
		t.Errorf("rollbacks = %d, want 2 (panic path releases the lease)", got)
	}
}

// --- minimal driver plumbing for the WithinTx test: enough of
// database/sql's driver interface to count commits/rollbacks. This is
// deliberately tiny; real code tests against real Postgres (db tier).

var leakSeq atomic.Int64 // unique driver names per registration

type leakDriver struct {
	commits   atomic.Int64
	rollbacks atomic.Int64
}

func (d *leakDriver) Open(string) (driver.Conn, error) { return &leakConn{d}, nil }

type leakConn struct{ d *leakDriver }

func (c *leakConn) Prepare(query string) (driver.Stmt, error) {
	return nil, fmt.Errorf("not needed")
}
func (c *leakConn) Close() error              { return nil }
func (c *leakConn) Begin() (driver.Tx, error) { return &leakTx{c.d}, nil }

type leakTx struct{ d *leakDriver }

func (t *leakTx) Commit() error   { t.d.commits.Add(1); return nil }
func (t *leakTx) Rollback() error { t.d.rollbacks.Add(1); return nil }

func newLeakTrackingDB(t *testing.T) (*sql.DB, *leakDriver) {
	t.Helper()
	drv := &leakDriver{}
	name := fmt.Sprintf("leak-%d", leakSeq.Add(1))
	sql.Register(name, drv)
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db, drv
}
