package migrations

import "database/sql"

func init() {
	Register(&addSubtitleRunTriggeredBy{
		migrationBase: NewMigrationBase(43, "add_subtitle_run_triggered_by"),
	})
}

// addSubtitleRunTriggeredBy (infra-optin-usage-report-a1) records WHO started a
// run: "auto" (the free-lane AutoGenerator after a scan, or the request-
// completion trigger after a download) or "manual" (anything a user clicked).
// The opt-in usage report counts subtitles Vido produced on its own in the last
// 7 days; until now nothing in the table could tell the two apart — batch_id is
// set only for consent batches, so an automatic run and a single manual run
// both had it empty.
//
// Named triggered_by, not trigger: TRIGGER is an SQLite keyword.
//
// NULLABLE (the 032/034/041 shape): NULL means "recorded before this column
// existed". Old rows are NOT backfilled — guessing would put invented numbers
// into the north-star metric.
type addSubtitleRunTriggeredBy struct {
	migrationBase
}

func (m *addSubtitleRunTriggeredBy) Up(tx *sql.Tx) error {
	if columnExists(tx, "subtitle_runs", "triggered_by") {
		return nil
	}
	_, err := tx.Exec(`ALTER TABLE subtitle_runs ADD COLUMN triggered_by TEXT`)
	return err
}

func (m *addSubtitleRunTriggeredBy) Down(tx *sql.Tx) error {
	// The column defaults to NULL and is harmless if left in place (SQLite DROP
	// COLUMN support is version-dependent — mirrors 024/032/034/041).
	return nil
}
