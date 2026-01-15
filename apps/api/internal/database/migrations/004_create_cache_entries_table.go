package migrations

import (
	"database/sql"
	"fmt"
)

// CreateCacheEntriesTable is the migration to create the cache_entries table
type CreateCacheEntriesTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateCacheEntriesTable{
		migrationBase: NewMigrationBase(4, "create_cache_entries_table"),
	})
}

// Up creates the cache_entries table with TTL support
func (m *CreateCacheEntriesTable) Up(tx *sql.Tx) error {
	// Create the cache_entries table
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS cache_entries (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := tx.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create cache_entries table: %w", err)
	}

	// Create index on type for efficient ClearByType operations
	createTypeIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_cache_entries_type ON cache_entries(type);
	`

	if _, err := tx.Exec(createTypeIndexQuery); err != nil {
		return fmt.Errorf("failed to create type index: %w", err)
	}

	// Create index on expires_at for efficient ClearExpired operations
	createExpiresIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at ON cache_entries(expires_at);
	`

	if _, err := tx.Exec(createExpiresIndexQuery); err != nil {
		return fmt.Errorf("failed to create expires_at index: %w", err)
	}

	return nil
}

// Down drops the cache_entries table
func (m *CreateCacheEntriesTable) Down(tx *sql.Tx) error {
	query := `
		DROP TABLE IF EXISTS cache_entries;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop cache_entries table: %w", err)
	}

	return nil
}
