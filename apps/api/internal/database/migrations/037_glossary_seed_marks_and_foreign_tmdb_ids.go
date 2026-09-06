package migrations

import "database/sql"

func init() {
	Register(&glossarySeedMarksAndForeignTMDbIDs{
		migrationBase: NewMigrationBase(37, "glossary_seed_marks_and_foreign_tmdb_ids"),
	})
}

// glossarySeedMarksAndForeignTMDbIDs does two small, related things for the
// seed-on-first-resolve seam (backlog-glossary-seed-existing-library-and-
// parse-queue, the CR round on it).
//
//  1. `glossary_seed_marks` — one row per shared glossary scope that has been
//     seeded from TMDb credits. The durable "already seeded?" answer used to
//     be EXISTS(source='metadata') on the live terms, which meant a user who
//     deleted every seeded term as junk got them all re-planted on the next
//     restart. A mark survives the deletion. Backfilled from the scopes that
//     PR #397 already seeded, so they are not fetched again either.
//
//  2. NULL out `tmdb_id` on movies/series rows whose metadata_source is
//     douban or wikipedia. Those writers stored the PROVIDER's numeric id in
//     the TMDb column (a Douban subject id parses as an int just fine), so
//     the row resolved to `tmdb:movie:<doubanID>` — some unrelated film's
//     shared glossary drawer — and, with seeding on resolve, would have
//     fetched that film's cast. The writers are fixed alongside; this cleans
//     the rows they already wrote. tmdb_id on such a row was never a TMDb
//     id, so nothing real is lost.
type glossarySeedMarksAndForeignTMDbIDs struct {
	migrationBase
}

func (m *glossarySeedMarksAndForeignTMDbIDs) Up(tx *sql.Tx) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS glossary_seed_marks (
			scope TEXT PRIMARY KEY,
			seeded INTEGER NOT NULL DEFAULT 0,
			seeded_at TIMESTAMP NOT NULL
		)`,
		`INSERT OR IGNORE INTO glossary_seed_marks (scope, seeded, seeded_at)
			SELECT scope, COUNT(*), MIN(created_at) FROM show_glossary
			WHERE source = 'metadata' AND scope LIKE 'tmdb:%'
			GROUP BY scope`,
		`UPDATE movies SET tmdb_id = NULL WHERE metadata_source IN ('douban', 'wikipedia') AND tmdb_id IS NOT NULL`,
		`UPDATE series SET tmdb_id = NULL WHERE metadata_source IN ('douban', 'wikipedia') AND tmdb_id IS NOT NULL`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func (m *glossarySeedMarksAndForeignTMDbIDs) Down(tx *sql.Tx) error {
	// The nulled ids were never TMDb ids; nothing to restore.
	_, err := tx.Exec(`DROP TABLE IF EXISTS glossary_seed_marks`)
	return err
}
