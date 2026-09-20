package bank

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgresStore implements PaymentStore over database/sql + the pgx
// stdlib driver. The overdraft guard lives in SQL (`balance >= $1`),
// so the invariant holds under concurrency without check-then-act
// races (chapter 2's optimistic pattern).
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore wraps an opened pool. Pool construction and ping
// are the caller's (main's) job: see 13 Section 1 OpenDB.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// withinTx is the transaction-function pattern from 13 Section 2: commit on
// success, rollback on any failure or panic, lease always released.
func withinTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Schema is the versioned migration this store requires (chapter 3:
// migrations are ordered, versioned, recorded; here it is one table
// pair applied by the db-tier tests).
const Schema = `
CREATE TABLE IF NOT EXISTS accounts (
	customer_id   text PRIMARY KEY,
	balance_minor bigint  NOT NULL CHECK (balance_minor >= 0),
	currency      text    NOT NULL
);

CREATE TABLE IF NOT EXISTS payments (
	id           bigserial PRIMARY KEY,
	customer_id  text NOT NULL REFERENCES accounts (customer_id),
	amount_minor bigint NOT NULL CHECK (amount_minor > 0),
	currency     text   NOT NULL,
	status       text   NOT NULL,
	created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS payments_customer_created
	ON payments (customer_id, created_at DESC);
`

func (s *PostgresStore) Charge(ctx context.Context, p Payment) (Payment, error) {
	var out Payment
	err := withinTx(ctx, s.db, func(tx *sql.Tx) error {
		// Debit guarded by the balance predicate: a row that fails the
		// guard matches zero rows, which is how SQL says "overdraft".
		res, err := tx.ExecContext(ctx,
			`UPDATE accounts SET balance_minor = balance_minor - $1
			 WHERE customer_id = $2 AND balance_minor >= $1`,
			p.AmountMinor, p.CustomerID)
		if err != nil {
			return fmt.Errorf("debit: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrInsufficientFunds
		}
		err = tx.QueryRowContext(ctx,
			`INSERT INTO payments (customer_id, amount_minor, currency, status)
			 VALUES ($1, $2, $3, 'charged')
			 RETURNING id, status`,
			p.CustomerID, p.AmountMinor, p.Currency,
		).Scan(&out.ID, &out.Status)
		if err != nil {
			return fmt.Errorf("insert payment: %w", err)
		}
		out.CustomerID = p.CustomerID
		out.AmountMinor = p.AmountMinor
		out.Currency = p.Currency
		return nil
	})
	if err != nil {
		return Payment{}, err
	}
	return out, nil
}

func (s *PostgresStore) ByCustomer(ctx context.Context, customerID string, limit int) ([]Payment, error) {
	if limit <= 0 || limit > 100 {
		limit = 100 // list methods always bound themselves (13 Section 4)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, customer_id, amount_minor, currency, status
		 FROM payments WHERE customer_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2`, customerID, limit)
	if err != nil {
		return nil, fmt.Errorf("query payments: %w", err)
	}
	defer rows.Close()

	var out []Payment
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.AmountMinor, &p.Currency, &p.Status); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payments: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) Account(ctx context.Context, customerID string) (Account, error) {
	var a Account
	err := s.db.QueryRowContext(ctx,
		`SELECT customer_id, balance_minor, currency
		 FROM accounts WHERE customer_id = $1`, customerID,
	).Scan(&a.CustomerID, &a.BalanceMinor, &a.Currency)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, fmt.Errorf("account %q: %w", customerID, ErrNotFound)
	}
	if err != nil {
		return Account{}, fmt.Errorf("query account: %w", err)
	}
	return a, nil
}
