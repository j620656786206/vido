package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

const (
	// schemaMigrationsTable is the name of the table that tracks applied migrations
	schemaMigrationsTable = "schema_migrations"

	// createSchemaMigrationsTableSQL is the SQL to create the schema_migrations table
	createSchemaMigrationsTableSQL = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
)

// Runner manages database migrations
type Runner struct {
	db         *sql.DB
	migrations []Migration
}

// NewRunner creates a new migration runner
func NewRunner(db *sql.DB) (*Runner, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	runner := &Runner{
		db:         db,
		migrations: make([]Migration, 0),
	}

	// Ensure schema_migrations table exists
	if err := runner.ensureSchemaMigrationsTable(); err != nil {
		return nil, fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return runner, nil
}

// ensureSchemaMigrationsTable creates the schema_migrations table if it doesn't exist
func (r *Runner) ensureSchemaMigrationsTable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, createSchemaMigrationsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to execute CREATE TABLE: %w", err)
	}

	return nil
}

// Register adds a migration to the runner
func (r *Runner) Register(migration Migration) error {
	if migration == nil {
		return fmt.Errorf("migration cannot be nil")
	}

	// Check for duplicate versions
	for _, m := range r.migrations {
		if m.Version() == migration.Version() {
			return fmt.Errorf("migration with version %d already registered", migration.Version())
		}
	}

	r.migrations = append(r.migrations, migration)
	return nil
}

// RegisterAll adds multiple migrations to the runner
func (r *Runner) RegisterAll(migrations []Migration) error {
	for _, m := range migrations {
		if err := r.Register(m); err != nil {
			return err
		}
	}
	return nil
}

// Up runs all pending migrations
func (r *Runner) Up(ctx context.Context) error {
	pending, err := r.getPendingMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(pending) == 0 {
		return nil
	}

	// Sort migrations by version
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version() < pending[j].Version()
	})

	// Apply each pending migration
	for _, migration := range pending {
		if err := r.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version(), migration.Name(), err)
		}
	}

	return nil
}

// Down rolls back the last applied migration
func (r *Runner) Down(ctx context.Context) error {
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Get the last applied migration
	lastApplied := applied[len(applied)-1]

	// Find the migration in our registered migrations
	var migration Migration
	for _, m := range r.migrations {
		if m.Version() == lastApplied.Version {
			migration = m
			break
		}
	}

	if migration == nil {
		return fmt.Errorf("migration %d not found in registered migrations", lastApplied.Version)
	}

	// Rollback the migration
	if err := r.rollbackMigration(ctx, migration); err != nil {
		return fmt.Errorf("failed to rollback migration %d (%s): %w", migration.Version(), migration.Name(), err)
	}

	return nil
}

// applyMigration applies a single migration within a transaction
func (r *Runner) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Apply the migration
	if err = migration.Up(tx); err != nil {
		return fmt.Errorf("migration Up() failed: %w", err)
	}

	// Record the migration in schema_migrations
	if err = r.recordMigration(tx, migration); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// rollbackMigration rolls back a single migration within a transaction
func (r *Runner) rollbackMigration(ctx context.Context, migration Migration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Rollback the migration
	if err = migration.Down(tx); err != nil {
		return fmt.Errorf("migration Down() failed: %w", err)
	}

	// Remove the migration from schema_migrations
	if err = r.removeMigration(tx, migration); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// recordMigration records a migration in the schema_migrations table
func (r *Runner) recordMigration(tx *sql.Tx, migration Migration) error {
	query := `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`
	_, err := tx.Exec(query, migration.Version(), migration.Name(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}
	return nil
}

// removeMigration removes a migration from the schema_migrations table
func (r *Runner) removeMigration(tx *sql.Tx, migration Migration) error {
	query := `DELETE FROM schema_migrations WHERE version = ?`
	_, err := tx.Exec(query, migration.Version())
	if err != nil {
		return fmt.Errorf("failed to delete migration record: %w", err)
	}
	return nil
}

// getPendingMigrations returns migrations that haven't been applied yet
func (r *Runner) getPendingMigrations(ctx context.Context) ([]Migration, error) {
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Create a map of applied migration versions for quick lookup
	appliedVersions := make(map[int64]bool)
	for _, record := range applied {
		appliedVersions[record.Version] = true
	}

	// Find pending migrations
	pending := make([]Migration, 0)
	for _, m := range r.migrations {
		if !appliedVersions[m.Version()] {
			pending = append(pending, m)
		}
	}

	return pending, nil
}

// getAppliedMigrations returns all applied migrations from the schema_migrations table
func (r *Runner) getAppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	query := `SELECT version, name, applied_at FROM schema_migrations ORDER BY version ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	defer rows.Close()

	records := make([]MigrationRecord, 0)
	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.Version, &record.Name, &record.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration records: %w", err)
	}

	return records, nil
}

// Status returns the current migration status
func (r *Runner) Status(ctx context.Context) ([]MigrationStatus, error) {
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Create a map of applied migrations
	appliedMap := make(map[int64]MigrationRecord)
	for _, record := range applied {
		appliedMap[record.Version] = record
	}

	// Build status for all migrations
	statuses := make([]MigrationStatus, 0, len(r.migrations))
	for _, m := range r.migrations {
		status := MigrationStatus{
			Version: m.Version(),
			Name:    m.Name(),
			Applied: false,
		}

		if record, ok := appliedMap[m.Version()]; ok {
			status.Applied = true
			status.AppliedAt = record.AppliedAt
		}

		statuses = append(statuses, status)
	}

	// Sort by version
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Version < statuses[j].Version
	})

	return statuses, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version   int64
	Name      string
	Applied   bool
	AppliedAt time.Time
}
