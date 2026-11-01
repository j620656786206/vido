package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigration048_AddsNullableHistoryColumns(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	// A pre-048 shaped insert still works and reads back NULL — old rows are
	// not backfilled, and "not recorded" is never 0 / "".
	_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, metadata_hash, glossary_version, prompt_version, model_id, status, cache_enabled, started_at)
		VALUES ('legacy', 'm1', 'movie', 'h', 'g', 'p', 'model', 'completed', 0, ?)`, now)
	require.NoError(t, err)
	var estimate sql.NullFloat64
	var provider, model, source sql.NullString
	require.NoError(t, db.QueryRow(`SELECT estimate_usd, asr_provider, asr_model, source FROM subtitle_runs WHERE id = 'legacy'`).
		Scan(&estimate, &provider, &model, &source))
	assert.False(t, estimate.Valid, "a row written before the column existed has no estimate")
	assert.False(t, provider.Valid)
	assert.False(t, model.Valid)
	assert.False(t, source.Valid)

	_, err = db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, metadata_hash, glossary_version, prompt_version, model_id, status, cache_enabled, started_at, estimate_usd, asr_provider, asr_model, source)
		VALUES ('history', 'm1', 'movie', 'h', 'g', 'p', 'model', 'completed', 0, ?, 0.42, 'elevenlabs', 'scribe_v2', 'audio')`, now)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT estimate_usd, asr_provider, asr_model, source FROM subtitle_runs WHERE id = 'history'`).
		Scan(&estimate, &provider, &model, &source))
	assert.InDelta(t, 0.42, estimate.Float64, 1e-9)
	assert.Equal(t, "elevenlabs", provider.String)
	assert.Equal(t, "scribe_v2", model.String)
	assert.Equal(t, "audio", source.String)
}

func TestMigration048_UpIsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&addSubtitleRunHistoryColumns{}).Up(tx), "a second Up on a migrated table must be a no-op")
	require.NoError(t, tx.Commit())
}

func TestMigration048_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 48 {
			found = true
			assert.Equal(t, "add_subtitle_run_history_columns", m.Name())
		}
	}
	assert.True(t, found, "migration 048 must be registered")
}
