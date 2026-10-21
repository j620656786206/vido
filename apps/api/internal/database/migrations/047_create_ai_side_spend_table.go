package migrations

import "database/sql"

func init() {
	Register(&createAISideSpendTable{
		migrationBase: NewMigrationBase(47, "create_ai_side_spend_table"),
	})
}

// createAISideSpendTable (backlog-ai-parse-reads-env-keys-only-a) is the
// ledger for paid AI calls made outside a subtitle run: filename parsing,
// fansub parsing and keyword generation. Alexyu 2026-10-21 overturned the
// "unmetered" half of sub-5-1 AC #4 — these calls are now recorded, one row per
// provider call that cost money — while keeping the "uncapped" half: nothing
// here can stop a scan.
//
//   - kind: CHECKed to the three background call types. Subtitle lanes stay in
//     subtitle_runs; this table is not a second place for them.
//   - spent_usd: REAL NOT NULL. A row is written only when the call cost
//     something, so "absent" is never smuggled in as 0.
//   - created_at: NOT NULL with NO default — the app always binds a time.Time,
//     which the sqlite-utc driver writes in the post-042 fixed-width UTC shape
//     (dbtime.Format). A CURRENT_TIMESTAMP default would write a different,
//     shorter text that compares wrongly against the month window's bounds.
//
// Not put in subtitle_runs: its media_id / media_type are NOT NULL and
// media_type only allows movie/series/episode (migration 030).
type createAISideSpendTable struct {
	migrationBase
}

func (m *createAISideSpendTable) Up(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS ai_side_spend (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL CHECK(kind IN ('parse','fansub','keywords')),
			provider TEXT NOT NULL,
			model_id TEXT NOT NULL DEFAULT '',
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			spent_usd REAL NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`); err != nil {
		return err
	}
	_, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_ai_side_spend_created_at ON ai_side_spend(created_at)`)
	return err
}

func (m *createAISideSpendTable) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS ai_side_spend`)
	return err
}
