package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigration034_AddsNullableStubbornCount(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	// A pre-034 shaped insert (no stubborn_count) must still work and read
	// back NULL — "not counted", not 0.
	_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, metadata_hash, glossary_version, prompt_version, model_id, status, cache_enabled, started_at)
		VALUES ('legacy', 'm1', 'movie', 'h', 'g', 'p', 'model', 'completed', 0, ?)`, now)
	require.NoError(t, err)
	var legacy sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT stubborn_count FROM subtitle_runs WHERE id = 'legacy'`).Scan(&legacy))
	assert.False(t, legacy.Valid, "a row written before the column existed reads NULL, never 0")

	_, err = db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, metadata_hash, glossary_version, prompt_version, model_id, status, cache_enabled, started_at, stubborn_count)
		VALUES ('counted', 'm1', 'movie', 'h', 'g', 'p', 'model', 'completed', 0, ?, 12)`, now)
	require.NoError(t, err)
	var counted sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT stubborn_count FROM subtitle_runs WHERE id = 'counted'`).Scan(&counted))
	assert.True(t, counted.Valid)
	assert.EqualValues(t, 12, counted.Int64)
}

func TestMigration034_UpIsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&addSubtitleRunStubbornCount{}).Up(tx), "a second Up on a migrated table must be a no-op, not a duplicate-column error")
	require.NoError(t, tx.Commit())
}

func TestMigration034_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 34 {
			found = true
			assert.Equal(t, "add_subtitle_run_stubborn_count", m.Name())
		}
	}
	assert.True(t, found, "migration 034 must be registered")
}
