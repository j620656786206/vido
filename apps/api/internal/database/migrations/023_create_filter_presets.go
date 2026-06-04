package migrations

import "database/sql"

func init() {
	Register(&createFilterPresets{
		migrationBase: NewMigrationBase(23, "create_filter_presets"),
	})
}

type createFilterPresets struct {
	migrationBase
}

func (m *createFilterPresets) Up(tx *sql.Tx) error {
	// Story 11.4 — Saved filter presets (P2-015).
	// filters holds a JSON string in the URL search-param shape; persisted in
	// SQLite (not localStorage) so presets sync across browser sessions (AC #5).
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS filter_presets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			filters TEXT NOT NULL DEFAULT '{}',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return err
	}

	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_filter_presets_sort_order ON filter_presets(sort_order)`); err != nil {
		return err
	}
	return nil
}

func (m *createFilterPresets) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS filter_presets`)
	return err
}
