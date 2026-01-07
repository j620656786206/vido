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
