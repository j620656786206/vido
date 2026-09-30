package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigration041_AddsNullableLedgerColumns(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	// A pre-041 shaped insert must still work and read back NULL — "not
	// recorded", never "" / 0.
	_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, metadata_hash, glossary_version, prompt_version, model_id, status, cache_enabled, started_at)
		VALUES ('legacy', 'm1', 'movie', 'h', 'g', 'p', 'model', 'completed', 0, ?)`, now)
	require.NoError(t, err)
	var route, batchID sql.NullString
	var cacheHits sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT route, cache_hit_cues, batch_id FROM subtitle_runs WHERE id = 'legacy'`).Scan(&route, &cacheHits, &batchID))
	assert.False(t, route.Valid, "a row written before the column existed has no route")
	assert.False(t, cacheHits.Valid, "cache hits were never measured on it")
	assert.False(t, batchID.Valid)

	_, err = db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, metadata_hash, glossary_version, prompt_version, model_id, status, cache_enabled, started_at, route, cache_hit_cues, batch_id)
		VALUES ('ledger', 'm1', 'movie', 'h', 'g', 'p', 'model', 'completed', 0, ?, 'translate', 120, 'batch-9')`, now)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT route, cache_hit_cues, batch_id FROM subtitle_runs WHERE id = 'ledger'`).Scan(&route, &cacheHits, &batchID))
	assert.Equal(t, "translate", route.String)
	assert.EqualValues(t, 120, cacheHits.Int64)
	assert.Equal(t, "batch-9", batchID.String)
}

func TestMigration041_UpIsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&addSubtitleRunLedgerColumns{}).Up(tx), "a second Up on a migrated table must be a no-op")
	require.NoError(t, tx.Commit())
}

func TestMigration041_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 41 {
			found = true
			assert.Equal(t, "add_subtitle_run_ledger_columns", m.Name())
		}
	}
	assert.True(t, found, "migration 041 must be registered")
}
