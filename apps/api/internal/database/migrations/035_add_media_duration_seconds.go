package migrations

import "database/sql"

func init() {
	Register(&addMediaDurationSeconds{
		migrationBase: NewMigrationBase(35, "add_media_duration_seconds"),
	})
}

// addMediaDurationSeconds persists the CONTAINER duration ffprobe already
// measures (sub-6-10a AC #1).
//
// The gap this closes: `MediaTechInfo.DurationSeconds` has been parsed on
// every enrichment probe since migration 021 and then thrown away — no column
// held it. So the consent screen priced from TMDb `runtime` alone, and an
// unmatched title (no TMDb row) or any episode (episodes carry no tech-info
// columns at all) fell through to the 45-minute assumption. Production
// screenshot, 2026-09-03: 2,399 rows, every one of them 「片長未知，以 45 分鐘
// 估算」. An estimate that ignores a number we already measured is not an
// estimate, it is a guess wearing one.
//
// Both tables, because both are priced by the same estimator: `movies` gets
// it from enrichment's ffprobe pass, `episodes` from the candidate sweep's
// route probe (which already runs ffprobe per episode).
//
// NULLABLE on purpose (the migration-032 spent_usd / 034 stubborn_count
// shape): NULL means "never measured" — a row that predates this column, or
// one whose probe failed — which is NOT the same as a zero-length file. The
// estimator reads it as "fall through to TMDb runtime", so a NULL is never
// priced as free. Additive; Rule 15 keeps movieSelectColumns / scanMovie and
// the episode equivalents in sync.
type addMediaDurationSeconds struct {
	migrationBase
}

func (m *addMediaDurationSeconds) Up(tx *sql.Tx) error {
	for _, table := range []string{"movies", "episodes"} {
		if columnExists(tx, table, "duration_seconds") {
			continue
		}
		if _, err := tx.Exec("ALTER TABLE " + table + " ADD COLUMN duration_seconds INTEGER"); err != nil {
			return err
		}
	}
	return nil
}

func (m *addMediaDurationSeconds) Down(tx *sql.Tx) error {
	// The column defaults to NULL and is harmless if left in place; SQLite
	// DROP COLUMN support is version-dependent (mirrors migrations 024/026/032/034).
	return nil
}
