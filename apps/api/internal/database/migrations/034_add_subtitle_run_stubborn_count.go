package migrations

import "database/sql"

func init() {
	Register(&addSubtitleRunStubbornCount{
		migrationBase: NewMigrationBase(34, "add_subtitle_run_stubborn_count"),
	})
}

// addSubtitleRunStubbornCount records, per completed run, how many cues
// shipped with their ENGLISH original (sub-6-2 AC #3): the quality-gate
// stubborn cues FR16 always tolerated up to 5%, plus — new in sub-6-2 — the
// cues of a chunk whose request failed transiently through every retry,
// tolerated up to 20% in total so a provider hiccup at cue 935 no longer
// discards the 935 cues already paid for.
//
// NULLABLE on purpose (the migration-032 spent_usd shape): NULL means "run
// predates this column", which is not the same as "0 English cues" — a
// pre-034 completed run may well carry a handful of quality-stubborn lines
// that nobody counted. Additive; Rule 15 keeps subtitleRunColumns in sync.
type addSubtitleRunStubbornCount struct {
	migrationBase
}

func (m *addSubtitleRunStubbornCount) Up(tx *sql.Tx) error {
	if columnExists(tx, "subtitle_runs", "stubborn_count") {
		return nil
	}
	_, err := tx.Exec("ALTER TABLE subtitle_runs ADD COLUMN stubborn_count INTEGER")
	return err
}

func (m *addSubtitleRunStubbornCount) Down(tx *sql.Tx) error {
	// The column defaults to NULL and is harmless if left in place; SQLite
	// DROP COLUMN support is version-dependent (mirrors migrations 024/026/032).
	return nil
}
