package migrations

import "database/sql"

func init() {
	Register(&addDoubanRatingFields{
		migrationBase: NewMigrationBase(24, "add_douban_rating_fields"),
	})
}

// addDoubanRatingFields adds denormalized Douban rating columns to the movies
// and series tables (Story 12-1 — dual rating display). The columns mirror the
// TMDb vote_average/vote_count pair so the detail page can render both ratings
// side-by-side with a single fast read (no join against douban_cache).
type addDoubanRatingFields struct {
	migrationBase
}

func (m *addDoubanRatingFields) Up(tx *sql.Tx) error {
	columns := []struct {
		table  string
		column string
		ddl    string
	}{
		{"movies", "douban_id", "ALTER TABLE movies ADD COLUMN douban_id TEXT"},
		{"movies", "douban_rating", "ALTER TABLE movies ADD COLUMN douban_rating REAL"},
		{"movies", "douban_vote_count", "ALTER TABLE movies ADD COLUMN douban_vote_count INTEGER"},
		{"series", "douban_id", "ALTER TABLE series ADD COLUMN douban_id TEXT"},
		{"series", "douban_rating", "ALTER TABLE series ADD COLUMN douban_rating REAL"},
		{"series", "douban_vote_count", "ALTER TABLE series ADD COLUMN douban_vote_count INTEGER"},
	}

	for _, c := range columns {
		if columnExists(tx, c.table, c.column) {
			continue
		}
		if _, err := tx.Exec(c.ddl); err != nil {
			return err
		}
	}

	// Index the Douban subject ID on each table for reverse lookups
	// (e.g., de-duplication when the same Douban subject backs multiple records).
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_movies_douban_id ON movies(douban_id)",
		"CREATE INDEX IF NOT EXISTS idx_series_douban_id ON series(douban_id)",
	}
	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

func (m *addDoubanRatingFields) Down(tx *sql.Tx) error {
	// Drop the indexes; the columns are nullable and harmless if left in place
	// (SQLite DROP COLUMN support is version-dependent, mirrors migration 021).
	for _, idx := range []string{
		"DROP INDEX IF EXISTS idx_movies_douban_id",
		"DROP INDEX IF EXISTS idx_series_douban_id",
	} {
		if _, err := tx.Exec(idx); err != nil {
			return err
		}
	}
	return nil
}
