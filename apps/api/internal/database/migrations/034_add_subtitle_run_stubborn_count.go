package migrations

import "database/sql"

func init() {
	Register(&addSubtitleRunStubbornCount{
		migrationBase: NewMigrationBase(34, "add_subtitle_run_stubborn_count"),
	})
}

// addSubtitleRunStubbornCount records, per completed run, how many cues
// shipped with their ENGLISH original (sub-6-2 AC #3): stubborn_count is the
// quality-gate stubborn cues FR16 always tolerated up to 5% PLUS the cues of
// a chunk whose request failed transiently through every retry (tolerated up
// to 20% together, so a provider hiccup at cue 935 no longer discards the
// 935 cues already paid for); transient_count is that second population on
// its own. transient_count > 0 marks a PARTIAL delivery: the pre-flight lets
// such an item run again (pipeline.go preflightSkip) instead of treating its
// sidecar as final.
//
// NULLABLE on purpose (the migration-032 spent_usd shape): NULL means "not
// counted" — a run that predates this column, or the ASR route, which is
// not the same as "0 English cues". Additive; Rule 15 keeps
// subtitleRunColumns in sync.
type addSubtitleRunStubbornCount struct {
	migrationBase
}

func (m *addSubtitleRunStubbornCount) Up(tx *sql.Tx) error {
	for _, column := range []string{"stubborn_count", "transient_count"} {
		if columnExists(tx, "subtitle_runs", column) {
			continue
		}
		if _, err := tx.Exec("ALTER TABLE subtitle_runs ADD COLUMN " + column + " INTEGER"); err != nil {
			return err
		}
	}
	return nil
}

func (m *addSubtitleRunStubbornCount) Down(tx *sql.Tx) error {
	// The column defaults to NULL and is harmless if left in place; SQLite
	// DROP COLUMN support is version-dependent (mirrors migrations 024/026/032).
	return nil
}
