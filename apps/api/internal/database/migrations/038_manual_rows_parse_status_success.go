package migrations

import "database/sql"

func init() {
	Register(&manualRowsParseStatusSuccess{
		migrationBase: NewMigrationBase(38, "manual_rows_parse_status_success"),
	})
}

// manualRowsParseStatusSuccess (dsr-2b-a AC #3) clears the 失敗/整理中 status
// from rows whose metadata the user already set.
//
// Before dsr-2b-a a Metadata-Editor save wrote metadata_source=manual but left
// parse_status untouched, so every unmatched item a user fixed by hand kept
// reading 失敗. The library's batch re-parse also knocks manual rows back to
// pending. Batch enrichment only picks pending or empty-status rows, and from dsr-2b-a on it
// deliberately leaves manual rows' metadata alone — so nothing else would ever
// clear these. The user's data IS the metadata; the row is resolved.
type manualRowsParseStatusSuccess struct {
	migrationBase
}

func (m *manualRowsParseStatusSuccess) Up(tx *sql.Tx) error {
	for _, s := range []string{
		`UPDATE movies SET parse_status = 'success' WHERE metadata_source = 'manual' AND parse_status IN ('failed', 'pending')`,
		`UPDATE series SET parse_status = 'success' WHERE metadata_source = 'manual' AND parse_status IN ('failed', 'pending')`,
	} {
		if _, err := tx.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func (m *manualRowsParseStatusSuccess) Down(tx *sql.Tx) error {
	// Which rows were failed vs pending before is not recorded, and neither
	// state was true of a row the user had filled in; nothing to restore.
	return nil
}
