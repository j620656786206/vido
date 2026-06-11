package migrations

import "database/sql"

func init() {
	Register(&addEpisodeSubtitleFields{
		migrationBase: NewMigrationBase(25, "add_episode_subtitle_fields"),
	})
}

// addEpisodeSubtitleFields adds per-episode subtitle tracking columns to the
// episodes table (Story 12-2 — season/episode accordion with subtitle status).
// Migration 018 added the same trio to movies and series; this extends it down
// to the episode grain so the detail-page accordion can show a subtitle status
// indicator per episode. Existing episodes default to 'not_searched'.
type addEpisodeSubtitleFields struct {
	migrationBase
}

func (m *addEpisodeSubtitleFields) Up(tx *sql.Tx) error {
	columns := []struct {
		table  string
		column string
		ddl    string
	}{
		{"episodes", "subtitle_status", "ALTER TABLE episodes ADD COLUMN subtitle_status TEXT DEFAULT 'not_searched'"},
		{"episodes", "subtitle_path", "ALTER TABLE episodes ADD COLUMN subtitle_path TEXT"},
		{"episodes", "subtitle_language", "ALTER TABLE episodes ADD COLUMN subtitle_language TEXT"},
	}

	for _, c := range columns {
		if columnExists(tx, c.table, c.column) {
			continue
		}
		if _, err := tx.Exec(c.ddl); err != nil {
			return err
		}
	}

	// Index subtitle_status for "episodes needing subtitles" scans
	// (mirrors idx_series_subtitle_status from migration 018).
	if _, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_episodes_subtitle_status ON episodes(subtitle_status)"); err != nil {
		return err
	}

	return nil
}

func (m *addEpisodeSubtitleFields) Down(tx *sql.Tx) error {
	// Drop the index; the columns are nullable and harmless if left in place
	// (SQLite DROP COLUMN support is version-dependent, mirrors migrations 021/024).
	if _, err := tx.Exec("DROP INDEX IF EXISTS idx_episodes_subtitle_status"); err != nil {
		return err
	}
	return nil
}
