package migrations

import (
	"database/sql"
	"fmt"
)

// CreateWikipediaCacheTable is the migration to create the wikipedia_cache table (Story 3.5)
type CreateWikipediaCacheTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateWikipediaCacheTable{
		migrationBase: NewMigrationBase(9, "create_wikipedia_cache_table"),
	})
}

// Up creates the wikipedia_cache table for 7-day Wikipedia result caching
func (m *CreateWikipediaCacheTable) Up(tx *sql.Tx) error {
	// Create the wikipedia_cache table per Story 3.5 requirements
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS wikipedia_cache (
			id TEXT PRIMARY KEY,
			query TEXT NOT NULL,
			page_title TEXT NOT NULL,
			title TEXT NOT NULL,
			original_title TEXT,
			year INTEGER,
			director TEXT,
			cast_json TEXT,
			genres_json TEXT,
			summary TEXT,
			image_url TEXT,
			media_type TEXT DEFAULT 'movie',
			confidence REAL DEFAULT 0.5,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
	`

	if _, err := tx.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create wikipedia_cache table: %w", err)
	}

	// Create index on query for fast lookups
	createQueryIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_wikipedia_cache_query ON wikipedia_cache(query);
	`

	if _, err := tx.Exec(createQueryIndexQuery); err != nil {
		return fmt.Errorf("failed to create query index: %w", err)
	}

	// Create index on page_title for lookups by Wikipedia page title
	createPageTitleIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_wikipedia_cache_page_title ON wikipedia_cache(page_title);
	`

	if _, err := tx.Exec(createPageTitleIndexQuery); err != nil {
		return fmt.Errorf("failed to create page_title index: %w", err)
	}

	// Create index on title for search lookups
	createTitleIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_wikipedia_cache_title ON wikipedia_cache(title);
	`

	if _, err := tx.Exec(createTitleIndexQuery); err != nil {
		return fmt.Errorf("failed to create title index: %w", err)
	}

	// Create index on expires_at for efficient cleanup operations
	createExpiresIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_wikipedia_cache_expires_at ON wikipedia_cache(expires_at);
	`

	if _, err := tx.Exec(createExpiresIndexQuery); err != nil {
		return fmt.Errorf("failed to create expires_at index: %w", err)
	}

	return nil
}

// Down drops the wikipedia_cache table
func (m *CreateWikipediaCacheTable) Down(tx *sql.Tx) error {
	query := `DROP TABLE IF EXISTS wikipedia_cache;`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop wikipedia_cache table: %w", err)
	}

	return nil
}
