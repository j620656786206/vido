package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigration035_AddsNullableDurationToBothMediaTables(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	// A pre-035 shaped insert must still work and read back NULL — "never
	// measured", which the estimator falls through on. If this ever read 0
	// the row would price as a zero-length film, i.e. free.
	_, err := db.Exec(`INSERT INTO movies (id, title, release_date, created_at, updated_at) VALUES ('legacy-movie', 'Ghost', '2020-01-01', ?, ?)`, now, now)
	require.NoError(t, err)
	var legacy sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT duration_seconds FROM movies WHERE id = 'legacy-movie'`).Scan(&legacy))
	assert.False(t, legacy.Valid, "a row written before the column existed reads NULL, never 0")

	_, err = db.Exec(`INSERT INTO movies (id, title, release_date, duration_seconds, created_at, updated_at) VALUES ('measured-movie', 'Dune', '2021-10-22', 9960, ?, ?)`, now, now)
	require.NoError(t, err)
	var movieDuration sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT duration_seconds FROM movies WHERE id = 'measured-movie'`).Scan(&movieDuration))
	assert.True(t, movieDuration.Valid)
	assert.EqualValues(t, 9960, movieDuration.Int64)

	// Episodes are the reason this story exists — they have no other
	// tech-info column at all.
	_, err = db.Exec(`INSERT INTO series (id, title, first_air_date, created_at, updated_at) VALUES ('s1', 'Severance', '2022-02-18', ?, ?)`, now, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, duration_seconds, created_at, updated_at)
		VALUES ('e1', 's1', 1, 1, 3300, ?, ?)`, now, now)
	require.NoError(t, err)
	var episodeDuration sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT duration_seconds FROM episodes WHERE id = 'e1'`).Scan(&episodeDuration))
	assert.True(t, episodeDuration.Valid)
	assert.EqualValues(t, 3300, episodeDuration.Int64)
}

func TestMigration035_UpIsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&addMediaDurationSeconds{}).Up(tx),
		"a second Up on a migrated table must be a no-op, not a duplicate-column error")
	require.NoError(t, tx.Commit())
}

func TestMigration035_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 35 {
			found = true
			assert.Equal(t, "add_media_duration_seconds", m.Name())
		}
	}
	assert.True(t, found, "migration 035 must be registered")
}
