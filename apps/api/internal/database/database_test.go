package database

import (
	"context"
	"testing"
	"time"

	"github.com/vido/api/internal/config"
)

// TestNew verifies database creation with valid configuration
func TestNew(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false, // WAL mode not supported for :memory:
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("Expected database instance, got nil")
	}

	if db.conn == nil {
		t.Fatal("Expected connection to be initialized")
	}
}

// TestNewWithNilConfig verifies error handling for nil config
func TestNewWithNilConfig(t *testing.T) {
	db, err := New(nil)
	if err == nil {
		t.Fatal("Expected error for nil config, got nil")
	}
	if db != nil {
		t.Fatal("Expected nil database instance, got non-nil")
	}
}

// TestPing verifies database connection health check
func TestPing(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}
}

// TestPingAfterClose verifies ping fails after close
func TestPingAfterClose(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	err = db.Ping()
	if err == nil {
		t.Fatal("Expected ping to fail after close, got nil error")
	}
}

// TestStats verifies database statistics retrieval
func TestStats(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	stats := db.Stats()

	// Stats should be available even if no queries have been executed
	if stats.MaxOpenConnections != 5 {
		t.Errorf("Expected MaxOpenConnections to be 5, got %d", stats.MaxOpenConnections)
	}
}

// TestClose verifies database can be closed gracefully
func TestClose(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Closing again should not error
	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close database second time: %v", err)
	}
}

// TestConn verifies underlying connection retrieval
func TestConn(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	conn := db.Conn()
	if conn == nil {
		t.Fatal("Expected connection, got nil")
	}

	// Verify we can ping using the connection
	err = conn.PingContext(context.Background())
	if err != nil {
		t.Fatalf("Failed to ping using connection: %v", err)
	}
}

// TestConfigureSQLite verifies SQLite configuration
func TestConfigureSQLite(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify foreign keys are enabled
	var foreignKeys int
	err = db.conn.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("Failed to check foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("Expected foreign_keys to be 1 (enabled), got %d", foreignKeys)
	}

	// Verify cache size
	var cacheSize int
	err = db.conn.QueryRow("PRAGMA cache_size").Scan(&cacheSize)
	if err != nil {
		t.Fatalf("Failed to check cache_size: %v", err)
	}
	if cacheSize != -64000 {
		t.Errorf("Expected cache_size to be -64000, got %d", cacheSize)
	}
}

// TestConnectionPoolSettings verifies connection pool configuration
func TestConnectionPoolSettings(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false,
		MaxOpenConns:    10,
		MaxIdleConns:    3,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != 10 {
		t.Errorf("Expected MaxOpenConnections to be 10, got %d", stats.MaxOpenConnections)
	}
}

// TestWALModeEnabled verifies WAL mode is properly enabled for file-based databases.
// WAL (Write-Ahead Logging) mode provides:
// - Concurrent read performance: Multiple readers can access the database while a writer is active
// - Better write performance: Writes are appended to a WAL file instead of modifying the main database
// - Crash recovery: WAL provides better crash recovery guarantees
// - Reduced disk I/O: Changes are batched in the WAL before being checkpointed to the main database
func TestWALModeEnabled(t *testing.T) {
	// Create a temporary file for the database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test_wal.db"

	cfg := &config.DatabaseConfig{
		Path:            dbPath,
		WALEnabled:      true,
		WALSyncMode:     "NORMAL",
		WALCheckpoint:   1000,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database with WAL mode: %v", err)
	}
	defer db.Close()

	// Verify WAL mode is active
	var journalMode string
	err = db.conn.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected journal_mode to be 'wal', got '%s'", journalMode)
	}

	// Verify synchronous mode
	var synchronous string
	err = db.conn.QueryRow("PRAGMA synchronous").Scan(&synchronous)
	if err != nil {
		t.Fatalf("Failed to query synchronous mode: %v", err)
	}
	// NORMAL synchronous mode = 1
	if synchronous != "1" {
		t.Errorf("Expected synchronous mode to be '1' (NORMAL), got '%s'", synchronous)
	}

	// Verify wal_autocheckpoint
	var walCheckpoint int
	err = db.conn.QueryRow("PRAGMA wal_autocheckpoint").Scan(&walCheckpoint)
	if err != nil {
		t.Fatalf("Failed to query wal_autocheckpoint: %v", err)
	}
	if walCheckpoint != 1000 {
		t.Errorf("Expected wal_autocheckpoint to be 1000, got %d", walCheckpoint)
	}
}

// TestWALModeDisabled verifies the default journal mode when WAL is not enabled
func TestWALModeDisabled(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		WALEnabled:      false,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// For :memory: databases, journal_mode should be 'memory'
	var journalMode string
	err = db.conn.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal_mode: %v", err)
	}
	if journalMode != "memory" {
		t.Errorf("Expected journal_mode to be 'memory' for in-memory database, got '%s'", journalMode)
	}
}
