package migrations

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	Register(&addEpisodeBurnedInSubtitle{
		migrationBase: NewMigrationBase(49, "add_episode_burned_in_subtitle"),
	})
}

// addEpisodeBurnedInSubtitle (disc-2026-11-burned-in-subs-b1) stores what the
// burned-in check read off an episode's picture, and lets its model call be
// recorded as side spend.
//
//   - episodes.burned_in_subtitle: zh-Hant / zh-Hans / zh / none. NULL = never
//     checked, or checked without a decision (frames could not be grabbed, the
//     reply could not be read) — "undecided" is never written as "none".
//     A column of its own, not an entry in subtitle_tracks: the sweep rewrites
//     that column whenever it re-probes the file.
//   - episodes.burned_in_checked_at: when the value above was written.
//   - ai_side_spend.kind gains 'burned_in_check'. SQLite cannot alter a CHECK,
//     so the table is rebuilt with every row and the created_at index kept.
//
// Both columns NULLABLE with no default; existing rows are untouched.
type addEpisodeBurnedInSubtitle struct {
	migrationBase
}

func (m *addEpisodeBurnedInSubtitle) Up(tx *sql.Tx) error {
	for _, c := range []struct{ name, ddl string }{
		{"burned_in_subtitle", "ALTER TABLE episodes ADD COLUMN burned_in_subtitle TEXT"},
		{"burned_in_checked_at", "ALTER TABLE episodes ADD COLUMN burned_in_checked_at TIMESTAMP"},
	} {
		if columnExists(tx, "episodes", c.name) {
			continue
		}
		if _, err := tx.Exec(c.ddl); err != nil {
			return err
		}
	}
	return rebuildAISideSpend(tx, "'parse','fansub','keywords','burned_in_check'", "burned_in_check")
}

// Down drops the 'burned_in_check' rows (the old CHECK cannot hold them) and
// restores the 047 kind set. The episode columns stay — NULL and harmless,
// like 024/032/034/041/048.
func (m *addEpisodeBurnedInSubtitle) Down(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM ai_side_spend WHERE kind = 'burned_in_check'`); err != nil {
		return err
	}
	return rebuildAISideSpendDown(tx)
}

// rebuildAISideSpend recreates ai_side_spend with `kinds` as its CHECK list,
// unless the live table's DDL already names `marker` (a second Up is a no-op).
func rebuildAISideSpend(tx *sql.Tx, kinds, marker string) error {
	var ddl string
	if err := tx.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'ai_side_spend'`).Scan(&ddl); err != nil {
		return fmt.Errorf("read ai_side_spend schema: %w", err)
	}
	if strings.Contains(ddl, "'"+marker+"'") {
		return nil
	}
	for _, stmt := range []string{
		`CREATE TABLE ai_side_spend_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL CHECK(kind IN (` + kinds + `)),
			provider TEXT NOT NULL,
			model_id TEXT NOT NULL DEFAULT '',
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			spent_usd REAL NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`INSERT INTO ai_side_spend_new (id, kind, provider, model_id, input_tokens, output_tokens, spent_usd, created_at)
		 SELECT id, kind, provider, model_id, input_tokens, output_tokens, spent_usd, created_at FROM ai_side_spend`,
		`DROP INDEX IF EXISTS idx_ai_side_spend_created_at`,
		`DROP TABLE ai_side_spend`,
		`ALTER TABLE ai_side_spend_new RENAME TO ai_side_spend`,
		`CREATE INDEX IF NOT EXISTS idx_ai_side_spend_created_at ON ai_side_spend(created_at)`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("rebuild ai_side_spend: %w", err)
		}
	}
	return nil
}

func rebuildAISideSpendDown(tx *sql.Tx) error {
	var ddl string
	if err := tx.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'ai_side_spend'`).Scan(&ddl); err != nil {
		return fmt.Errorf("read ai_side_spend schema: %w", err)
	}
	if !strings.Contains(ddl, "'burned_in_check'") {
		return nil
	}
	// Rebuild to the 047 list by forcing the marker check to miss.
	return rebuildAISideSpend(tx, "'parse','fansub','keywords'", "\x00")
}
