package migrations

import "database/sql"

func init() {
	Register(&addIsRemovedField{
		migrationBase: NewMigrationBase(19, "add_is_removed_field"),
	})
}

type addIsRemovedField struct {
	migrationBase
}

func (m *addIsRemovedField) Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		-- Movies: soft-delete flag for removed files (Story 7-2)
		ALTER TABLE movies ADD COLUMN is_removed INTEGER NOT NULL DEFAULT 0;

		-- Series: soft-delete flag for removed files (Story 7-2)
		ALTER TABLE series ADD COLUMN is_removed INTEGER NOT NULL DEFAULT 0;

		-- Indexes for filtering removed files
		CREATE INDEX IF NOT EXISTS idx_movies_is_removed ON movies(is_removed);
		CREATE INDEX IF NOT EXISTS idx_series_is_removed ON series(is_removed);
	`)
	return err
}

func (m *addIsRemovedField) Down(tx *sql.Tx) error {
	// SQLite does not support DROP COLUMN before 3.35.0
	// In practice, this migration is not expected to be rolled back
	return nil
}
