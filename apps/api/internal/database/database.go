package database

import (
	"database/sql"
	"fmt"

	"github.com/vido/api/internal/config"
	_ "modernc.org/sqlite"
)

// DB wraps the database connection and provides database operations
type DB struct {
	conn   *sql.DB
	config *config.DatabaseConfig
}

// New creates a new database instance with the given configuration
func New(cfg *config.DatabaseConfig) (*DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config cannot be nil")
	}

	db := &DB{
		config: cfg,
	}

	if err := db.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// connect establishes a connection to the database and configures it
func (db *DB) connect() error {
	connStr := db.config.GetConnectionString()

	conn, err := sql.Open("sqlite", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	db.conn = conn

	// Configure connection pool
	db.conn.SetMaxOpenConns(db.config.MaxOpenConns)
	db.conn.SetMaxIdleConns(db.config.MaxIdleConns)
	db.conn.SetConnMaxLifetime(db.config.ConnMaxLifetime)
	db.conn.SetConnMaxIdleTime(db.config.ConnMaxIdleTime)

	// Apply SQLite-specific settings
	if err := db.configureSQLite(); err != nil {
		db.conn.Close()
		return fmt.Errorf("failed to configure SQLite: %w", err)
	}

	// Verify connection
	if err := db.conn.Ping(); err != nil {
		db.conn.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// configureSQLite applies SQLite-specific PRAGMA settings
func (db *DB) configureSQLite() error {
	pragmas := []string{
		fmt.Sprintf("PRAGMA busy_timeout = %d", db.config.BusyTimeout.Milliseconds()),
		fmt.Sprintf("PRAGMA cache_size = %d", db.config.CacheSize),
		"PRAGMA foreign_keys = ON",
		"PRAGMA temp_store = MEMORY",
	}

	// Enable WAL mode if configured
	if db.config.WALEnabled {
		pragmas = append(pragmas,
			"PRAGMA journal_mode = WAL",
			fmt.Sprintf("PRAGMA synchronous = %s", db.config.WALSyncMode),
			fmt.Sprintf("PRAGMA wal_autocheckpoint = %d", db.config.WALCheckpoint),
		)
	}

	for _, pragma := range pragmas {
		if _, err := db.conn.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute pragma '%s': %w", pragma, err)
		}
	}

	return nil
}

// Conn returns the underlying sql.DB connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Close closes the database connection gracefully
func (db *DB) Close() error {
	if db.conn == nil {
		return nil
	}

	if err := db.conn.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}

// Ping verifies the database connection is alive
func (db *DB) Ping() error {
	if db.conn == nil {
		return fmt.Errorf("database connection is nil")
	}

	if err := db.conn.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Stats returns database connection pool statistics
func (db *DB) Stats() sql.DBStats {
	if db.conn == nil {
		return sql.DBStats{}
	}
	return db.conn.Stats()
}
