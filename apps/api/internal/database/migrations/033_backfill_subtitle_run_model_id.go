package migrations

import "database/sql"

func init() {
	Register(&backfillSubtitleRunModelID{
		migrationBase: NewMigrationBase(33, "backfill_subtitle_run_model_id"),
	})
}

// backfillSubtitleRunModelID fills the model_id that every default-model run
// recorded as "" before sub-6-5 (the pipeline snapshotted the CLAUDE_MODEL env
// override, which is empty when the default is in force).
//
// Only completed rows are rewritten: model_id is half of the resume /
// segment-cache identity (FindCompletedRun matches on it), and a completed
// row with "" would have matched — or missed — the wrong model's cache once
// the default changes. Failed/skipped rows carry no reusable output, so their
// "" stays as an honest "unknown" rather than a guess.
//
// "claude-haiku-4-5" is the only default that ever existed before this
// migration (ai.DefaultClaudeModel from story 9R-1 through eval-1); the value
// is spelled out here, not read from the ai package, so the backfill stays
// correct after sub-6-8a moves the default to Sonnet.
type backfillSubtitleRunModelID struct {
	migrationBase
}

const legacyDefaultClaudeModel = "claude-haiku-4-5"

func (m *backfillSubtitleRunModelID) Up(tx *sql.Tx) error {
	_, err := tx.Exec(
		`UPDATE subtitle_runs SET model_id = ? WHERE model_id = '' AND status = 'completed'`,
		legacyDefaultClaudeModel,
	)
	return err
}

func (m *backfillSubtitleRunModelID) Down(tx *sql.Tx) error {
	// Not reversible in general: after Up we cannot tell a backfilled row from
	// a row that genuinely ran on haiku. Leaving the data is the honest Down.
	return nil
}
