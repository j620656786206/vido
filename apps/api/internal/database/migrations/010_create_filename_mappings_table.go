package migrations

import (
	"database/sql"
	"fmt"
)

// CreateFilenameMappingsTable is the migration to create the filename_mappings table
// for storing learned filename patterns that map to metadata
type CreateFilenameMappingsTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateFilenameMappingsTable{
		migrationBase: NewMigrationBase(10, "create_filename_mappings_table"),
	})
}

// Up creates the filename_mappings table
func (m *CreateFilenameMappingsTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS filename_mappings (
			id TEXT PRIMARY KEY,
			pattern TEXT NOT NULL UNIQUE,
			pattern_type TEXT NOT NULL,
			pattern_regex TEXT,
			fansub_group TEXT,
			title_pattern TEXT,
			metadata_type TEXT NOT NULL,
			metadata_id TEXT NOT NULL,
			tmdb_id INTEGER,
			confidence REAL NOT NULL DEFAULT 1.0,
			use_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_filename_mappings_pattern ON filename_mappings(pattern);
		CREATE INDEX IF NOT EXISTS idx_filename_mappings_fansub_group ON filename_mappings(fansub_group);
		CREATE INDEX IF NOT EXISTS idx_filename_mappings_title_pattern ON filename_mappings(title_pattern);
		CREATE INDEX IF NOT EXISTS idx_filename_mappings_metadata ON filename_mappings(metadata_type, metadata_id);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create filename_mappings table: %w", err)
	}

	return nil
}

// Down drops the filename_mappings table
func (m *CreateFilenameMappingsTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_filename_mappings_metadata;
		DROP INDEX IF EXISTS idx_filename_mappings_title_pattern;
		DROP INDEX IF EXISTS idx_filename_mappings_fansub_group;
		DROP INDEX IF EXISTS idx_filename_mappings_pattern;
		DROP TABLE IF EXISTS filename_mappings;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop filename_mappings table: %w", err)
	}

	return nil
}
