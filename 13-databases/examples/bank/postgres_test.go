//go:build db

// Database-tier tests: real Postgres via TEST_DATABASE_URL. Skips
// cleanly without it, so default CI stays green (the same two-tier
// pattern as 18-kafka-with-go's broker tests). Run locally:
//
//	docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=pw postgres:16
//	TEST_DATABASE_URL=postgres://postgres:pw@localhost:5432/postgres?sslmode=disable \
//	  go test ./13-databases/examples/bank -tags=db -v
package bank

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx"
)

// dbOpen connects, applies the schema, and registers cleanup that
// drops the tables again. One shared database, schema-level isolation:
// sufficient for this example's single-tenant tables.
func dbOpen(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping database-tier tests")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("database unreachable (%v); skipping database-tier tests", err)
	}

	// Drop leftovers from a previous crashed run, then apply.
	for _, stmt := range []string{
		`DROP TABLE IF EXISTS payments`,
		`DROP TABLE IF EXISTS accounts`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("reset schema: %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, Schema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS payments`)
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS accounts`)
	})

	// Seed the account the parity harness expects.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO accounts (customer_id, balance_minor, currency)
		 VALUES ('cus_1', 10000, 'USD')
		 ON CONFLICT (customer_id) DO UPDATE SET balance_minor = 10000`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

func init() {
	// Append the real store to the parity harness's cases: the same
	// tests that pass against the fake must pass against Postgres.
	// build is called per subtest, so each gets re-seeded state via
	// dbOpen's reset + seed.
	storeCases = append(storeCases, struct {
		name  string
		build func(t *testing.T) PaymentStore
	}{
		name: "postgres",
		build: func(t *testing.T) PaymentStore {
			return NewPostgresStore(dbOpen(t))
		},
	})
}

// TestPostgres_ConcurrentChargesConserved is the db-tier race test:
// N goroutines charge concurrently; the balance check in SQL
// serializes them, and the final balance proves no lost updates. This
// is the test that catches an application-side check-then-act rewrite.
func TestPostgres_ConcurrentChargesConserved(t *testing.T) {
	db := dbOpen(t)
	s := NewPostgresStore(db)

	const n, each = 25, 100
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, err := s.Charge(ctx, Payment{
				CustomerID: "cus_1", AmountMinor: each, Currency: "USD",
			})
			if err != nil {
				errs <- fmt.Errorf("charge %d: %w", i, err)
			}
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	acct, err := s.Account(context.Background(), "cus_1")
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	if acct.BalanceMinor != 10_000-n*each {
		t.Errorf("balance = %d, want %d: lost updates under concurrency",
			acct.BalanceMinor, 10_000-n*each)
	}
}
