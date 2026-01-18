package migrations

import (
	"database/sql"
	"fmt"
)

// CreateAICacheTable is the migration to create the ai_cache table (Story 3.1)
type CreateAICacheTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateAICacheTable{
		migrationBase: NewMigrationBase(7, "create_ai_cache_table"),
	})
}

// Up creates the ai_cache table for 30-day AI parsing result caching
func (m *CreateAICacheTable) Up(tx *sql.Tx) error {
	// Create the ai_cache table per Story 3.1 requirements
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS ai_cache (
			id TEXT PRIMARY KEY,
			filename_hash TEXT UNIQUE NOT NULL,
			provider TEXT NOT NULL,
			request_prompt TEXT NOT NULL,
			response_json TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
	`

	if _, err := tx.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create ai_cache table: %w", err)
	}

	// Create index on filename_hash for fast lookups
	createHashIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_ai_cache_filename_hash ON ai_cache(filename_hash);
	`

	if _, err := tx.Exec(createHashIndexQuery); err != nil {
		return fmt.Errorf("failed to create filename_hash index: %w", err)
	}

	// Create index on expires_at for efficient cleanup operations
	createExpiresIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_ai_cache_expires_at ON ai_cache(expires_at);
	`

	if _, err := tx.Exec(createExpiresIndexQuery); err != nil {
		return fmt.Errorf("failed to create expires_at index: %w", err)
	}

	return nil
}

// Down drops the ai_cache table
func (m *CreateAICacheTable) Down(tx *sql.Tx) error {
	query := `DROP TABLE IF EXISTS ai_cache;`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop ai_cache table: %w", err)
	}

	return nil
}
