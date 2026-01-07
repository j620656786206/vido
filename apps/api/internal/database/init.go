package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vido/api/internal/config"
)

// Initialize creates and initializes the database with the given configuration
// It ensures the database directory exists, creates the database file, and verifies WAL mode
func Initialize(cfg *config.DatabaseConfig) (*DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config cannot be nil")
	}

	// Ensure database directory exists
	if err := ensureDatabaseDir(cfg); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Create database connection
	db, err := New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Verify WAL mode if enabled in config
	if cfg.WALEnabled {
		if err := verifyWALMode(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("WAL mode verification failed: %w", err)
		}
	}

	return db, nil
}

// ensureDatabaseDir creates the database directory if it doesn't exist
func ensureDatabaseDir(cfg *config.DatabaseConfig) error {
	dbDir := cfg.GetDatabaseDir()

	// Check if directory already exists
	info, err := os.Stat(dbDir)
	if err == nil {
		// Directory exists, verify it's actually a directory
		if !info.IsDir() {
			return fmt.Errorf("database path exists but is not a directory: %s", dbDir)
		}
		return nil
	}

	// If error is not "not exists", return it
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat database directory: %w", err)
	}

	// Create directory with appropriate permissions
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	return nil
}

// verifyWALMode checks that WAL mode is actually enabled on the database
func verifyWALMode(db *DB) error {
	isWAL, err := db.IsWALEnabled()
	if err != nil {
		return fmt.Errorf("failed to check WAL mode: %w", err)
	}

	if !isWAL {
		mode, _ := db.GetWALMode()
		return fmt.Errorf("WAL mode is not enabled (current mode: %s)", mode)
	}

	return nil
}

// GetDatabaseFilePath returns the absolute path to the database file
func GetDatabaseFilePath(cfg *config.DatabaseConfig) (string, error) {
	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute database path: %w", err)
	}
	return absPath, nil
}

// DatabaseExists checks if the database file already exists
func DatabaseExists(cfg *config.DatabaseConfig) (bool, error) {
	info, err := os.Stat(cfg.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	// Verify it's a regular file
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("database path exists but is not a regular file: %s", cfg.Path)
	}

	return true, nil
}
