package migrations

import (
	"database/sql"
	"fmt"
)

// CreateParseJobsTable is the migration to create the parse_jobs table
// for tracking download-to-parse queue jobs (Story 4.5)
type CreateParseJobsTable struct {
	migrationBase
}

func init() {
	Register(&CreateParseJobsTable{
		migrationBase: NewMigrationBase(13, "create_parse_jobs_table"),
	})
}

// Up creates the parse_jobs table
func (m *CreateParseJobsTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS parse_jobs (
			id TEXT PRIMARY KEY,
			torrent_hash TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			media_id TEXT,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			UNIQUE(torrent_hash)
		);

		CREATE INDEX IF NOT EXISTS idx_parse_jobs_status ON parse_jobs(status);
		CREATE INDEX IF NOT EXISTS idx_parse_jobs_torrent_hash ON parse_jobs(torrent_hash);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create parse_jobs table: %w", err)
	}

	return nil
}

// Down drops the parse_jobs table
func (m *CreateParseJobsTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_parse_jobs_torrent_hash;
		DROP INDEX IF EXISTS idx_parse_jobs_status;
		DROP TABLE IF EXISTS parse_jobs;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop parse_jobs table: %w", err)
	}

	return nil
}
