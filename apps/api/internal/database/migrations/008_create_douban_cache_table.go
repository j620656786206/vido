package migrations

import (
	"database/sql"
	"fmt"
)

// CreateDoubanCacheTable is the migration to create the douban_cache table (Story 3.4)
type CreateDoubanCacheTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateDoubanCacheTable{
		migrationBase: NewMigrationBase(8, "create_douban_cache_table"),
	})
}

// Up creates the douban_cache table for 7-day Douban result caching
func (m *CreateDoubanCacheTable) Up(tx *sql.Tx) error {
	// Create the douban_cache table per Story 3.4 requirements
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS douban_cache (
			id TEXT PRIMARY KEY,
			douban_id TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			title_traditional TEXT,
			original_title TEXT,
			year INTEGER,
			rating REAL,
			rating_count INTEGER,
			director TEXT,
			cast_json TEXT,
			genres_json TEXT,
			countries_json TEXT,
			languages_json TEXT,
			poster_url TEXT,
			summary TEXT,
			summary_traditional TEXT,
			media_type TEXT DEFAULT 'movie',
			runtime INTEGER,
			episodes INTEGER,
			release_date TEXT,
			imdb_id TEXT,
			scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
	`

	if _, err := tx.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create douban_cache table: %w", err)
	}

	// Create index on douban_id for fast lookups
	createDoubanIDIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_douban_cache_douban_id ON douban_cache(douban_id);
	`

	if _, err := tx.Exec(createDoubanIDIndexQuery); err != nil {
		return fmt.Errorf("failed to create douban_id index: %w", err)
	}

	// Create index on title for search lookups
	createTitleIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_douban_cache_title ON douban_cache(title);
	`

	if _, err := tx.Exec(createTitleIndexQuery); err != nil {
		return fmt.Errorf("failed to create title index: %w", err)
	}

	// Create index on expires_at for efficient cleanup operations
	createExpiresIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_douban_cache_expires_at ON douban_cache(expires_at);
	`

	if _, err := tx.Exec(createExpiresIndexQuery); err != nil {
		return fmt.Errorf("failed to create expires_at index: %w", err)
	}

	return nil
}

// Down drops the douban_cache table
func (m *CreateDoubanCacheTable) Down(tx *sql.Tx) error {
	query := `DROP TABLE IF EXISTS douban_cache;`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop douban_cache table: %w", err)
	}

	return nil
}
