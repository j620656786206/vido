package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Transaction represents a database transaction
type Transaction struct {
	tx *sql.Tx
}

// BeginTx starts a new database transaction with the given context and options
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	tx, err := db.conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &Transaction{tx: tx}, nil
}

// Begin starts a new database transaction with default options
func (db *DB) Begin() (*Transaction, error) {
	return db.BeginTx(context.Background(), nil)
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	if t.tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	if t.tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	if err := t.tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// Tx returns the underlying sql.Tx
func (t *Transaction) Tx() *sql.Tx {
	return t.tx
}

// ExecContext executes a query without returning any rows
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	result, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

// Exec executes a query without returning any rows using background context
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}

// QueryContext executes a query that returns rows
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return rows, nil
}

// Query executes a query that returns rows using background context
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if db.conn == nil {
		return nil
	}

	return db.conn.QueryRowContext(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row using background context
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

// GetWALMode returns the current journal mode (should be "wal" if WAL is enabled)
func (db *DB) GetWALMode() (string, error) {
	if db.conn == nil {
		return "", fmt.Errorf("database connection is nil")
	}

	var mode string
	err := db.conn.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		return "", fmt.Errorf("failed to get journal mode: %w", err)
	}

	return mode, nil
}

// GetSyncMode returns the current synchronous mode
func (db *DB) GetSyncMode() (string, error) {
	if db.conn == nil {
		return "", fmt.Errorf("database connection is nil")
	}

	var mode int
	err := db.conn.QueryRow("PRAGMA synchronous").Scan(&mode)
	if err != nil {
		return "", fmt.Errorf("failed to get synchronous mode: %w", err)
	}

	// Map integer values to string names
	modes := map[int]string{
		0: "OFF",
		1: "NORMAL",
		2: "FULL",
		3: "EXTRA",
	}

	if modeName, ok := modes[mode]; ok {
		return modeName, nil
	}

	return fmt.Sprintf("UNKNOWN(%d)", mode), nil
}

// Checkpoint performs a WAL checkpoint operation
func (db *DB) Checkpoint() error {
	if db.conn == nil {
		return fmt.Errorf("database connection is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := db.conn.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return fmt.Errorf("failed to checkpoint WAL: %w", err)
	}

	return nil
}

// IsWALEnabled checks if WAL mode is currently enabled
func (db *DB) IsWALEnabled() (bool, error) {
	mode, err := db.GetWALMode()
	if err != nil {
		return false, err
	}

	return mode == "wal", nil
}
