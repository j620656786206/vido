package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newEpisodesTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS episodes (
			id TEXT PRIMARY KEY,
			series_id TEXT NOT NULL,
			tmdb_id INTEGER,
			season_number INTEGER NOT NULL,
			episode_number INTEGER NOT NULL,
			title TEXT,
			file_path TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)
	return db
}

func TestAddEpisodeSubtitleFields_Up(t *testing.T) {
	db := newEpisodesTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, title)
		VALUES ('ep-1', 'series-1', 1, 5, 'Test Episode')`)
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &addEpisodeSubtitleFields{
		migrationBase: NewMigrationBase(25, "add_episode_subtitle_fields"),
	}
	require.NoError(t, migration.Up(tx))
	require.NoError(t, tx.Commit())

	t.Run("subtitle_status defaults to not_searched for existing rows", func(t *testing.T) {
		var status sql.NullString
		err := db.QueryRow(`SELECT subtitle_status FROM episodes WHERE id = 'ep-1'`).Scan(&status)
		require.NoError(t, err)
		assert.True(t, status.Valid, "subtitle_status should be populated by DEFAULT")
		assert.Equal(t, "not_searched", status.String)
	})

	t.Run("subtitle_path and subtitle_language are NULL by default", func(t *testing.T) {
		var path, language sql.NullString
		err := db.QueryRow(`SELECT subtitle_path, subtitle_language FROM episodes WHERE id = 'ep-1'`).
			Scan(&path, &language)
		require.NoError(t, err)
		assert.False(t, path.Valid, "subtitle_path should be NULL")
		assert.False(t, language.Valid, "subtitle_language should be NULL")
	})

	t.Run("existing data preserved after migration", func(t *testing.T) {
		var title string
		err := db.QueryRow(`SELECT title FROM episodes WHERE id = 'ep-1'`).Scan(&title)
		require.NoError(t, err)
		assert.Equal(t, "Test Episode", title)
	})

	t.Run("columns are writable", func(t *testing.T) {
		_, err := db.Exec(`UPDATE episodes SET subtitle_status = 'found',
			subtitle_path = '/media/ep1.zh-Hant.srt', subtitle_language = 'zh-Hant' WHERE id = 'ep-1'`)
		require.NoError(t, err)

		var status, path, language string
		err = db.QueryRow(`SELECT subtitle_status, subtitle_path, subtitle_language
			FROM episodes WHERE id = 'ep-1'`).Scan(&status, &path, &language)
		require.NoError(t, err)
		assert.Equal(t, "found", status)
		assert.Equal(t, "/media/ep1.zh-Hant.srt", path)
		assert.Equal(t, "zh-Hant", language)
	})

	t.Run("subtitle_status index exists", func(t *testing.T) {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index'
			AND name = 'idx_episodes_subtitle_status'`).Scan(&name)
		require.NoError(t, err)
		assert.Equal(t, "idx_episodes_subtitle_status", name)
	})
}

func TestAddEpisodeSubtitleFields_Idempotent(t *testing.T) {
	db := newEpisodesTestDB(t)
	defer db.Close()

	migration := &addEpisodeSubtitleFields{
		migrationBase: NewMigrationBase(25, "add_episode_subtitle_fields"),
	}

	tx1, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx1))
	require.NoError(t, tx1.Commit())

	// Second run must not error (columnExists + IF NOT EXISTS guards).
	tx2, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx2))
	require.NoError(t, tx2.Commit())
}

func TestAddEpisodeSubtitleFields_Version(t *testing.T) {
	migration := &addEpisodeSubtitleFields{
		migrationBase: NewMigrationBase(25, "add_episode_subtitle_fields"),
	}
	assert.Equal(t, int64(25), migration.Version())
	assert.Equal(t, "add_episode_subtitle_fields", migration.Name())
}
