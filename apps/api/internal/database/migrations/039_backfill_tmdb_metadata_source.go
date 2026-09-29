package migrations

import "database/sql"

func init() {
	Register(&backfillTMDbMetadataSource{
		migrationBase: NewMigrationBase(39, "backfill_tmdb_metadata_source"),
	})
}

// backfillTMDbMetadataSource (disc-2026-09-unmatched-filter-vs-parse-status
// AC #3) gives a source to rows that already hold a TMDb id but none.
//
// Migration 006 added metadata_source without backfilling it, so a title
// imported before then can carry a TMDb match and a NULL source. 「未匹配」 now
// asks metadata_source whether a row has data at all; without this, such a row
// whose later re-parse failed would be counted as never matched. A TMDb id can
// only have come from TMDb. Rows with a source already are left alone.
type backfillTMDbMetadataSource struct {
	migrationBase
}

func (m *backfillTMDbMetadataSource) Up(tx *sql.Tx) error {
	for _, s := range []string{
		`UPDATE movies SET metadata_source = 'tmdb' WHERE tmdb_id > 0 AND (metadata_source IS NULL OR metadata_source = '')`,
		`UPDATE series SET metadata_source = 'tmdb' WHERE tmdb_id > 0 AND (metadata_source IS NULL OR metadata_source = '')`,
	} {
		if _, err := tx.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func (m *backfillTMDbMetadataSource) Down(tx *sql.Tx) error {
	// Which rows had no source before is not recorded, and a NULL source on a
	// TMDb-matched row was never true; nothing to restore.
	return nil
}
