package migrations

import "database/sql"

func init() {
	Register(&addSubtitleRunHistoryColumns{
		migrationBase: NewMigrationBase(48, "add_subtitle_run_history_columns"),
	})
}

// addSubtitleRunHistoryColumns (disc-2026-11-spend-history-a-ledger) records
// three facts a spend-history page needs and the ledger never kept:
//
//   - estimate_usd: what the 管理字幕 dialog's Estimate would have quoted for
//     this item AT THE MOMENT THE RUN STARTED (resume credit included). It is
//     recomputed server-side, not sent by the client, so a batch item that
//     waited in the queue while the model was changed records the price of
//     the run that actually started, not the one the user was shown earlier.
//   - asr_provider / asr_model: the speech-recognition engine (elevenlabs /
//     openai / self_hosted) and its model id, written only when ASR actually
//     ran on this run.
//   - source: where the subtitle came from — embedded_text (a text track in
//     the file), sidecar (the English subtitle beside the file) or audio
//     (speech recognition). Lanes that produced nothing (skip /
//     no_text_source) leave it NULL. `route` is untouched, so the monthly
//     report that groups by route keeps its numbers.
//
// All four NULLABLE (the 032/034/041 shape): NULL means "not recorded" — a run
// written before this migration is not backfilled. Additive; Rule 15 keeps
// subtitleRunColumns in sync.
type addSubtitleRunHistoryColumns struct {
	migrationBase
}

func (m *addSubtitleRunHistoryColumns) Up(tx *sql.Tx) error {
	for _, c := range []struct{ name, ddl string }{
		{"estimate_usd", "ALTER TABLE subtitle_runs ADD COLUMN estimate_usd REAL"},
		{"asr_provider", "ALTER TABLE subtitle_runs ADD COLUMN asr_provider TEXT"},
		{"asr_model", "ALTER TABLE subtitle_runs ADD COLUMN asr_model TEXT"},
		{"source", "ALTER TABLE subtitle_runs ADD COLUMN source TEXT"},
	} {
		if columnExists(tx, "subtitle_runs", c.name) {
			continue
		}
		if _, err := tx.Exec(c.ddl); err != nil {
			return err
		}
	}
	return nil
}

func (m *addSubtitleRunHistoryColumns) Down(tx *sql.Tx) error {
	// Columns default to NULL and are harmless if left in place (SQLite DROP
	// COLUMN support is version-dependent — mirrors 024/032/034/041).
	return nil
}
