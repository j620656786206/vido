package migrations

import "database/sql"

func init() {
	Register(&createExploreBlocks{
		migrationBase: NewMigrationBase(22, "create_explore_blocks"),
	})
}

type createExploreBlocks struct {
	migrationBase
}

func (m *createExploreBlocks) Up(tx *sql.Tx) error {
	// Story 10.3 — Custom homepage discover blocks (P2-002)
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS explore_blocks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			content_type TEXT NOT NULL CHECK(content_type IN ('movie', 'tv')),
			genre_ids TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT '',
			region TEXT NOT NULL DEFAULT '',
			sort_by TEXT NOT NULL DEFAULT '',
			max_items INTEGER NOT NULL DEFAULT 20,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return err
	}

	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_explore_blocks_sort_order ON explore_blocks(sort_order)`); err != nil {
		return err
	}
	return nil
}

func (m *createExploreBlocks) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS explore_blocks`)
	return err
}
