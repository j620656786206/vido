package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newSubtitleRunsMigration() *createSubtitleRunsTable {
	return &createSubtitleRunsTable{migrationBase: NewMigrationBase(30, "create_subtitle_runs_table")}
}

func setupSubtitleRunsMigration(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, newSubtitleRunsMigration().Up(tx))
	require.NoError(t, tx.Commit())
	return db
}

func TestCreateSubtitleRunsTable_Up(t *testing.T) {
	db := setupSubtitleRunsMigration(t)
	defer db.Close()

	t.Run("inserts a minimal row with defaults", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type) VALUES ('run1', 'm1', 'movie')`)
		require.NoError(t, err)

		var status, metadataHash, glossaryVersion, promptVersion, modelID string
		var cacheEnabled int
		var completedAt sql.NullString
		require.NoError(t, db.QueryRow(`
			SELECT status, metadata_hash, glossary_version, prompt_version, model_id, cache_enabled, completed_at
			FROM subtitle_runs WHERE id = 'run1'`).
			Scan(&status, &metadataHash, &glossaryVersion, &promptVersion, &modelID, &cacheEnabled, &completedAt))

		assert.Equal(t, "pending", status)
		assert.Equal(t, "", metadataHash, "version-tuple columns default to empty, never NULL")
		assert.Equal(t, "", glossaryVersion)
		assert.Equal(t, "", promptVersion)
		assert.Equal(t, "", modelID)
		assert.Equal(t, 0, cacheEnabled, "cache_enabled is reserved for sub-1-5 and stays 0 here")
		assert.False(t, completedAt.Valid, "completed_at stays NULL until the run finishes")
	})

	t.Run("started_at defaults to CURRENT_TIMESTAMP", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type) VALUES ('run-ts', 'm-ts', 'movie')`)
		require.NoError(t, err)

		var startedAt sql.NullString
		require.NoError(t, db.QueryRow(`SELECT started_at FROM subtitle_runs WHERE id = 'run-ts'`).Scan(&startedAt))
		assert.True(t, startedAt.Valid, "started_at is NOT NULL with a CURRENT_TIMESTAMP default")
		assert.NotEmpty(t, startedAt.String)
	})

	t.Run("rejects TMDB media_type vocabulary", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type) VALUES ('run2', 'm2', 'tv')`)
		assert.Error(t, err, "media_type CHECK must reject 'tv' — subtitle_runs speaks the INTERNAL table vocabulary")
	})

	t.Run("accepts all three internal media grains", func(t *testing.T) {
		for i, grain := range []string{"movie", "series", "episode"} {
			_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type) VALUES (?, ?, ?)`,
				"grain"+string(rune('a'+i)), "mg", grain)
			assert.NoErrorf(t, err, "media_type %q must be accepted — subtitle_status lives on all three tables", grain)
		}
	})

	t.Run("rejects invalid status", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, status) VALUES ('run3', 'm3', 'movie', 'translating')`)
		assert.Error(t, err, "status CHECK must reject values outside the 5-value run enum (translating is a subtitle_status, not a run status)")
	})

	t.Run("rejects a NULL media_id", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_type) VALUES ('run4', 'movie')`)
		assert.Error(t, err, "media_id is NOT NULL")
	})

	t.Run("creates both indexes", func(t *testing.T) {
		rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='subtitle_runs'`)
		require.NoError(t, err)
		defer rows.Close()

		found := map[string]bool{}
		for rows.Next() {
			var name string
			require.NoError(t, rows.Scan(&name))
			found[name] = true
		}
		require.NoError(t, rows.Err())

		assert.True(t, found["idx_subtitle_runs_media"], "resume-lookup index must exist")
		assert.True(t, found["idx_subtitle_runs_status"], "batch-enumeration index must exist")
	})
}

// TestCreateSubtitleRunsTable_LeavesMediaTablesAlone guards Finding 1: the
// extended subtitle_status enum is a Go + frontend contract, and this migration
// must NOT attempt to add a CHECK to movies/series/episodes (SQLite would need a
// full table rebuild — enormous risk for zero requirement).
func TestCreateSubtitleRunsTable_LeavesMediaTablesAlone(t *testing.T) {
	db := setupSubtitleRunsMigration(t)
	defer db.Close()

	for _, table := range []string{"movies", "series", "episodes"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		assert.ErrorIsf(t, err, sql.ErrNoRows,
			"migration 030 must not create or rebuild %s — it creates subtitle_runs only", table)
	}
}

func TestCreateSubtitleRunsTable_Down(t *testing.T) {
	db := setupSubtitleRunsMigration(t)
	defer db.Close()

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, newSubtitleRunsMigration().Down(tx))
	require.NoError(t, tx.Commit())

	var name string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='subtitle_runs'`).Scan(&name)
	assert.ErrorIs(t, err, sql.ErrNoRows, "subtitle_runs table should be dropped")
}

func TestCreateSubtitleRunsTable_Version(t *testing.T) {
	migration := newSubtitleRunsMigration()
	assert.Equal(t, int64(30), migration.Version())
	assert.Equal(t, "create_subtitle_runs_table", migration.Name())
}
