package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupSeriesCountriesTable(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			genres TEXT NOT NULL DEFAULT '[]',
			parse_status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO series (id, title) VALUES ('series-1', 'Test Series');
	`)
	require.NoError(t, err)
	return db
}

func runSeriesCountries(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	m := &addSeriesProductionCountries{migrationBase: NewMigrationBase(40, "add_series_production_countries")}
	require.NoError(t, m.Up(tx))
	require.NoError(t, tx.Commit())
}

func TestAddSeriesProductionCountries_Up(t *testing.T) {
	db := setupSeriesCountriesTable(t)
	defer db.Close()

	runSeriesCountries(t, db)

	var countries sql.NullString
	require.NoError(t, db.QueryRow(`SELECT production_countries FROM series WHERE id = 'series-1'`).Scan(&countries))
	assert.False(t, countries.Valid, "existing rows get NULL, not an empty array — unknown is unknown")

	_, err := db.Exec(`UPDATE series SET production_countries = '[{"iso_3166_1":"CN","name":"China"}]' WHERE id = 'series-1'`)
	require.NoError(t, err)
}

// Running twice (a re-applied migration, or a database that already had the
// column from a manual fix) must be a no-op, not a duplicate-column error.
func TestAddSeriesProductionCountries_IsIdempotent(t *testing.T) {
	db := setupSeriesCountriesTable(t)
	defer db.Close()

	runSeriesCountries(t, db)
	runSeriesCountries(t, db)

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('series') WHERE name = 'production_countries'`).Scan(&n))
	assert.Equal(t, 1, n)
}
