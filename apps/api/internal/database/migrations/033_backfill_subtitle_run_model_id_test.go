package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// The shipped chain runs against an empty table, so the backfill is exercised
// by seeding rows and calling Up directly (the migration-032 test pattern).
func TestMigration033_BackfillsOnlyEmptyCompletedRows(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	insert := func(id, model, status string) {
		t.Helper()
		_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, metadata_hash, glossary_version, prompt_version, model_id, status, cache_enabled, started_at)
			VALUES (?, 'm1', 'movie', 'h', 'g', 'p', ?, ?, 0, ?)`, id, model, status, now)
		require.NoError(t, err)
	}
	insert("legacy-completed", "", "completed")
	insert("legacy-failed", "", "failed")
	insert("sonnet-completed", "claude-sonnet-5", "completed")

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&backfillSubtitleRunModelID{}).Up(tx))
	require.NoError(t, tx.Commit())

	modelOf := func(id string) string {
		t.Helper()
		var m string
		require.NoError(t, db.QueryRow(`SELECT model_id FROM subtitle_runs WHERE id = ?`, id).Scan(&m))
		return m
	}
	require.Equal(t, legacyDefaultClaudeModel, modelOf("legacy-completed"), "empty completed row is backfilled")
	require.Equal(t, "", modelOf("legacy-failed"), "failed rows keep their honest unknown")
	require.Equal(t, "claude-sonnet-5", modelOf("sonnet-completed"), "explicit models are never touched")
}

func TestMigration033_IsRegisteredInOrder(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 33 {
			found = true
			require.Equal(t, "backfill_subtitle_run_model_id", m.Name())
		}
	}
	require.True(t, found, "migration 33 must be registered")
	var _ *sql.DB // keep the sql import honest for the helper signature above
}
