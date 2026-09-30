package migrations

import "database/sql"

func init() {
	Register(&addSubtitleRunLedgerColumns{
		migrationBase: NewMigrationBase(41, "add_subtitle_run_ledger_columns"),
	})
}

// addSubtitleRunLedgerColumns (sub-7-6a) turns subtitle_runs into a ledger the
// monthly spend page can read without guessing:
//
//   - route: which lane the item took — deliver_direct / convert_then_deliver
//     (free, an existing Chinese track), translate (LLM), asr (speech
//     recognition, with or without a translate leg), skip / no_text_source.
//     Until now only a log line knew this, so "how much did skipping save"
//     could not be answered from the table.
//   - cache_hit_cues: how many cues of a translate run were served from the
//     segment cache instead of the model. Also only ever logged before.
//   - batch_id: the consent batch this run belonged to, so a batch receipt
//     ("本次 $0.53 · 844 句") can be summed from the rows after the fact.
//
// All three NULLABLE (the 032/034 shape): NULL means "not recorded" — a run
// written before this migration, or a route on which the figure is not
// measured (cache hits on a deliver route) — which is not the same as 0 /
// no batch. Additive; Rule 15 keeps subtitleRunColumns in sync.
type addSubtitleRunLedgerColumns struct {
	migrationBase
}

func (m *addSubtitleRunLedgerColumns) Up(tx *sql.Tx) error {
	for _, c := range []struct{ name, ddl string }{
		{"route", "ALTER TABLE subtitle_runs ADD COLUMN route TEXT"},
		{"cache_hit_cues", "ALTER TABLE subtitle_runs ADD COLUMN cache_hit_cues INTEGER"},
		{"batch_id", "ALTER TABLE subtitle_runs ADD COLUMN batch_id TEXT"},
	} {
		if columnExists(tx, "subtitle_runs", c.name) {
			continue
		}
		if _, err := tx.Exec(c.ddl); err != nil {
			return err
		}
	}
	// The monthly aggregation groups by batch; the per-media index already
	// covers the receipt lookups a single item needs.
	_, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_subtitle_runs_batch_id ON subtitle_runs(batch_id)`)
	return err
}

func (m *addSubtitleRunLedgerColumns) Down(tx *sql.Tx) error {
	// Columns default to NULL and are harmless if left in place (SQLite DROP
	// COLUMN support is version-dependent — mirrors 024/032/034).
	_, err := tx.Exec(`DROP INDEX IF EXISTS idx_subtitle_runs_batch_id`)
	return err
}
