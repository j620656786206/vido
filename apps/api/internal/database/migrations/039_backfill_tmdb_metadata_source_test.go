package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-09-unmatched-filter-vs-parse-status AC #3: rows imported before
// migration 006 added metadata_source carry a TMDb id but no source. Once
// 「未匹配」 reads metadata_source, such a row whose re-parse later failed would
// be counted as having no data at all.
func TestMigration039_BackfillsTMDbSourceOnRowsThatHaveATMDbID(t *testing.T) {
	db := newMigratedDBUpTo(t, 38)
	now := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO movies (id, title, release_date, tmdb_id, metadata_source, parse_status, created_at, updated_at) VALUES
		('m-old-tmdb-null', 'a', '', 603, NULL, 'failed', ?, ?),
		('m-old-tmdb-empty', 'b', '', 604, '', 'success', ?, ?),
		('m-douban', 'c', '', 605, 'douban', 'success', ?, ?),
		('m-manual', 'd', '', NULL, 'manual', 'success', ?, ?),
		('m-no-tmdb', 'e', '', NULL, NULL, 'failed', ?, ?),
		('m-tmdb-zero', 'f', '', 0, NULL, 'failed', ?, ?)`,
		now, now, now, now, now, now, now, now, now, now, now, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO series (id, title, first_air_date, tmdb_id, metadata_source, parse_status, created_at, updated_at) VALUES
		('s-old-tmdb-null', 'g', '', 1399, NULL, 'pending', ?, ?),
		('s-wikipedia', 'h', '', NULL, 'wikipedia', 'success', ?, ?)`, now, now, now, now)
	require.NoError(t, err)

	runner, err := NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(GetAll()))
	require.NoError(t, runner.Up(t.Context()))

	source := func(table, id string) sql.NullString {
		var s sql.NullString
		require.NoError(t, db.QueryRow(`SELECT metadata_source FROM `+table+` WHERE id = ?`, id).Scan(&s))
		return s
	}
	assert.Equal(t, "tmdb", source("movies", "m-old-tmdb-null").String)
	assert.Equal(t, "tmdb", source("movies", "m-old-tmdb-empty").String)
	assert.Equal(t, "douban", source("movies", "m-douban").String, "an existing source is never overwritten")
	assert.Equal(t, "manual", source("movies", "m-manual").String)
	assert.False(t, source("movies", "m-no-tmdb").Valid, "no TMDb id → still no source")
	assert.False(t, source("movies", "m-tmdb-zero").Valid, "tmdb_id 0 is not an id")
	assert.Equal(t, "tmdb", source("series", "s-old-tmdb-null").String)
	assert.Equal(t, "wikipedia", source("series", "s-wikipedia").String)

	// Idempotent: the runner skips applied versions, so run the SQL itself again.
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&backfillTMDbMetadataSource{}).Up(tx))
	require.NoError(t, tx.Commit())
	assert.Equal(t, "tmdb", source("movies", "m-old-tmdb-null").String)
	assert.Equal(t, "douban", source("movies", "m-douban").String)
	assert.False(t, source("movies", "m-no-tmdb").Valid)
}
