package migrations

import "database/sql"

func init() {
	Register(&addSeriesProductionCountries{
		migrationBase: NewMigrationBase(40, "add_series_production_countries"),
	})
}

// addSeriesProductionCountries (sub-7-2b AC #1) gives the series table the
// production_countries column movies have had since migration 006.
//
// Without it a show's countries had nowhere to land, so the translation prompt
// never learned where a series was made — and sub-7-4's「中國內容跳過台灣詞庫」
// rule, which keys on exactly that, silently never applied to a single
// episode. Same shape as the movie column: a JSON array of
// {iso_3166_1, name}, NULL when unknown.
type addSeriesProductionCountries struct {
	migrationBase
}

func (m *addSeriesProductionCountries) Up(tx *sql.Tx) error {
	if columnExists(tx, "series", "production_countries") {
		return nil
	}
	_, err := tx.Exec(`ALTER TABLE series ADD COLUMN production_countries TEXT`)
	return err
}

func (m *addSeriesProductionCountries) Down(tx *sql.Tx) error {
	// Nullable and harmless if left in place (SQLite DROP COLUMN support is
	// version-dependent — mirrors migrations 021/024).
	return nil
}
