package migrations

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// testMigration is a simple migration for testing
type testMigration struct {
	migrationBase
	upCalled   bool
	downCalled bool
	upFunc     func(tx *sql.Tx) error
	downFunc   func(tx *sql.Tx) error
}

func newTestMigration(version int64, name string) *testMigration {
	return &testMigration{
		migrationBase: NewMigrationBase(version, name),
		upFunc: func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)")
			return err
		},
		downFunc: func(tx *sql.Tx) error {
			_, err := tx.Exec("DROP TABLE test_table")
			return err
		},
	}
}

func (m *testMigration) Up(tx *sql.Tx) error {
	m.upCalled = true
	return m.upFunc(tx)
}

func (m *testMigration) Down(tx *sql.Tx) error {
	m.downCalled = true
	return m.downFunc(tx)
}

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

// TestNewRunner verifies runner creation
func TestNewRunner(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	if runner == nil {
		t.Fatal("Expected runner instance, got nil")
	}

	// Verify schema_migrations table was created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check schema_migrations table: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected schema_migrations table to exist, got count: %d", count)
	}
}

// TestNewRunnerWithNilDB verifies error handling for nil database
func TestNewRunnerWithNilDB(t *testing.T) {
	runner, err := NewRunner(nil)
	if err == nil {
		t.Fatal("Expected error for nil database, got nil")
	}
	if runner != nil {
		t.Fatal("Expected nil runner, got non-nil")
	}
}

// TestRegister verifies migration registration
func TestRegister(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	migration := newTestMigration(1, "test_migration")
	err = runner.Register(migration)
	if err != nil {
		t.Fatalf("Failed to register migration: %v", err)
	}

	if len(runner.migrations) != 1 {
		t.Errorf("Expected 1 migration, got %d", len(runner.migrations))
	}
}

// TestRegisterNil verifies nil migration rejection
func TestRegisterNil(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	err = runner.Register(nil)
	if err == nil {
		t.Fatal("Expected error for nil migration, got nil")
	}
}

// TestRegisterDuplicate verifies duplicate version rejection
func TestRegisterDuplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	migration1 := newTestMigration(1, "migration_1")
	migration2 := newTestMigration(1, "migration_2")

	err = runner.Register(migration1)
	if err != nil {
		t.Fatalf("Failed to register first migration: %v", err)
	}

	err = runner.Register(migration2)
	if err == nil {
		t.Fatal("Expected error for duplicate version, got nil")
	}
}

// TestRegisterAll verifies bulk registration
func TestRegisterAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	migrations := []Migration{
		newTestMigration(1, "migration_1"),
		newTestMigration(2, "migration_2"),
		newTestMigration(3, "migration_3"),
	}

	err = runner.RegisterAll(migrations)
	if err != nil {
		t.Fatalf("Failed to register migrations: %v", err)
	}

	if len(runner.migrations) != 3 {
		t.Errorf("Expected 3 migrations, got %d", len(runner.migrations))
	}
}

// TestUp verifies migration execution
func TestUp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	migration := newTestMigration(1, "create_test_table")
	err = runner.Register(migration)
	if err != nil {
		t.Fatalf("Failed to register migration: %v", err)
	}

	ctx := context.Background()
	err = runner.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	if !migration.upCalled {
		t.Error("Expected Up() to be called")
	}

	// Verify migration was recorded
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check migration record: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected migration to be recorded, got count: %d", count)
	}

	// Verify table was created
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check test_table: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected test_table to be created, got count: %d", count)
	}
}

// TestUpWithNoPending verifies no-op when no pending migrations
func TestUpWithNoPending(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	ctx := context.Background()
	err = runner.Up(ctx)
	if err != nil {
		t.Fatalf("Expected no error for no pending migrations, got: %v", err)
	}
}

// TestUpMultipleMigrations verifies multiple migrations run in order
func TestUpMultipleMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	// Register migrations out of order
	migration2 := newTestMigration(2, "migration_2")
	migration2.upFunc = func(tx *sql.Tx) error {
		_, err := tx.Exec("CREATE TABLE table2 (id INTEGER)")
		return err
	}

	migration1 := newTestMigration(1, "migration_1")
	migration1.upFunc = func(tx *sql.Tx) error {
		_, err := tx.Exec("CREATE TABLE table1 (id INTEGER)")
		return err
	}

	err = runner.Register(migration2)
	if err != nil {
		t.Fatalf("Failed to register migration 2: %v", err)
	}
	err = runner.Register(migration1)
	if err != nil {
		t.Fatalf("Failed to register migration 1: %v", err)
	}

	ctx := context.Background()
	err = runner.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Both migrations should be applied
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check migration records: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 migrations to be recorded, got: %d", count)
	}
}

// TestDown verifies migration rollback
func TestDown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	migration := newTestMigration(1, "create_test_table")
	err = runner.Register(migration)
	if err != nil {
		t.Fatalf("Failed to register migration: %v", err)
	}

	ctx := context.Background()

	// Run migration
	err = runner.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	// Rollback migration
	err = runner.Down(ctx)
	if err != nil {
		t.Fatalf("Failed to rollback migration: %v", err)
	}

	if !migration.downCalled {
		t.Error("Expected Down() to be called")
	}

	// Verify migration was removed from records
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check migration record: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected migration to be removed, got count: %d", count)
	}

	// Verify table was dropped
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check test_table: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected test_table to be dropped, got count: %d", count)
	}
}

// TestDownWithNoApplied verifies error when no migrations to rollback
func TestDownWithNoApplied(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	ctx := context.Background()
	err = runner.Down(ctx)
	if err == nil {
		t.Fatal("Expected error for no applied migrations, got nil")
	}
}

// TestStatus verifies migration status reporting
func TestStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	migration1 := newTestMigration(1, "migration_1")
	migration2 := newTestMigration(2, "migration_2")

	err = runner.Register(migration1)
	if err != nil {
		t.Fatalf("Failed to register migration 1: %v", err)
	}
	err = runner.Register(migration2)
	if err != nil {
		t.Fatalf("Failed to register migration 2: %v", err)
	}

	ctx := context.Background()

	// Check status before applying
	statuses, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(statuses))
	}

	for _, status := range statuses {
		if status.Applied {
			t.Errorf("Expected migration %d to not be applied", status.Version)
		}
	}

	// Apply first migration
	err = runner.Register(migration1)
	if err == nil {
		// Already registered
	}

	ctx = context.Background()
	pending, err := runner.getPendingMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) > 0 {
		err = runner.applyMigration(ctx, pending[0])
		if err != nil {
			t.Fatalf("Failed to apply migration: %v", err)
		}
	}

	// Check status after applying one
	statuses, err = runner.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	appliedCount := 0
	for _, status := range statuses {
		if status.Applied {
			appliedCount++
		}
	}

	if appliedCount != 1 {
		t.Errorf("Expected 1 migration to be applied, got %d", appliedCount)
	}
}

// TestTransactionRollbackOnError verifies transaction rollback on error
func TestTransactionRollbackOnError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	// Create a migration that fails
	failingMigration := newTestMigration(1, "failing_migration")
	failingMigration.upFunc = func(tx *sql.Tx) error {
		_, err := tx.Exec("CREATE TABLE test_table (id INTEGER)")
		if err != nil {
			return err
		}
		// Return an error to trigger rollback
		return sql.ErrTxDone
	}

	err = runner.Register(failingMigration)
	if err != nil {
		t.Fatalf("Failed to register migration: %v", err)
	}

	ctx := context.Background()
	err = runner.Up(ctx)
	if err == nil {
		t.Fatal("Expected error from failing migration, got nil")
	}

	// Verify migration was not recorded
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", 1).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check migration record: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected no migration record due to rollback, got count: %d", count)
	}

	// Verify table was not created due to rollback
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check test_table: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected test_table not to exist due to rollback, got count: %d", count)
	}
}

// TestMigrationRecord verifies migration record structure
func TestMigrationRecord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner, err := NewRunner(db)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	migration := newTestMigration(1, "test_migration")
	err = runner.Register(migration)
	if err != nil {
		t.Fatalf("Failed to register migration: %v", err)
	}

	ctx := context.Background()
	err = runner.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	// Query migration record
	var version int64
	var name string
	var appliedAt time.Time

	err = db.QueryRow("SELECT version, name, applied_at FROM schema_migrations WHERE version = ?", 1).Scan(&version, &name, &appliedAt)
	if err != nil {
		t.Fatalf("Failed to query migration record: %v", err)
	}

	if version != 1 {
		t.Errorf("Expected version 1, got %d", version)
	}
	if name != "test_migration" {
		t.Errorf("Expected name 'test_migration', got '%s'", name)
	}
	if appliedAt.IsZero() {
		t.Error("Expected appliedAt to be set")
	}
}
