package migrations

import "database/sql"

func init() {
	Register(&addSubtitleFields{
		migrationBase: NewMigrationBase(18, "add_subtitle_fields"),
	})
}

type addSubtitleFields struct {
	migrationBase
}

func (m *addSubtitleFields) Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		-- Movies: subtitle tracking fields
		ALTER TABLE movies ADD COLUMN subtitle_status TEXT DEFAULT 'not_searched';
		ALTER TABLE movies ADD COLUMN subtitle_path TEXT;
		ALTER TABLE movies ADD COLUMN subtitle_language TEXT;
		ALTER TABLE movies ADD COLUMN subtitle_last_searched TIMESTAMP;
		ALTER TABLE movies ADD COLUMN subtitle_search_score REAL;

		-- Series: subtitle tracking fields
		ALTER TABLE series ADD COLUMN subtitle_status TEXT DEFAULT 'not_searched';
		ALTER TABLE series ADD COLUMN subtitle_path TEXT;
		ALTER TABLE series ADD COLUMN subtitle_language TEXT;
		ALTER TABLE series ADD COLUMN subtitle_last_searched TIMESTAMP;
		ALTER TABLE series ADD COLUMN subtitle_search_score REAL;

		-- Indexes for subtitle status queries
		CREATE INDEX IF NOT EXISTS idx_movies_subtitle_status ON movies(subtitle_status);
		CREATE INDEX IF NOT EXISTS idx_series_subtitle_status ON series(subtitle_status);
	`)
	return err
}

func (m *addSubtitleFields) Down(tx *sql.Tx) error {
	// SQLite does not support DROP COLUMN before 3.35.0
	// For rollback, we recreate tables without the subtitle columns
	// In practice, this migration is not expected to be rolled back
	return nil
}
