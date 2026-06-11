package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupDoubanRatingTables(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			genres TEXT NOT NULL DEFAULT '[]',
			parse_status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			genres TEXT NOT NULL DEFAULT '[]',
			parse_status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO movies (id, title) VALUES ('movie-1', 'Test Movie');
		INSERT INTO series (id, title) VALUES ('series-1', 'Test Series');
	`)
	require.NoError(t, err)
	return db
}

func TestAddDoubanRatingFields_Up(t *testing.T) {
	db := setupDoubanRatingTables(t)
	defer db.Close()

	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &addDoubanRatingFields{migrationBase: NewMigrationBase(24, "add_douban_rating_fields")}
	require.NoError(t, migration.Up(tx))
	require.NoError(t, tx.Commit())

	t.Run("movies douban columns exist with NULL defaults", func(t *testing.T) {
		var doubanID sql.NullString
		var doubanRating sql.NullFloat64
		var doubanVoteCount sql.NullInt64
		err := db.QueryRow(`SELECT douban_id, douban_rating, douban_vote_count FROM movies WHERE id = 'movie-1'`).
			Scan(&doubanID, &doubanRating, &doubanVoteCount)
		require.NoError(t, err)
		assert.False(t, doubanID.Valid, "douban_id should be NULL")
		assert.False(t, doubanRating.Valid, "douban_rating should be NULL")
		assert.False(t, doubanVoteCount.Valid, "douban_vote_count should be NULL")
	})

	t.Run("series douban columns exist with NULL defaults", func(t *testing.T) {
		var doubanID sql.NullString
		var doubanRating sql.NullFloat64
		var doubanVoteCount sql.NullInt64
		err := db.QueryRow(`SELECT douban_id, douban_rating, douban_vote_count FROM series WHERE id = 'series-1'`).
			Scan(&doubanID, &doubanRating, &doubanVoteCount)
		require.NoError(t, err)
		assert.False(t, doubanID.Valid, "douban_id should be NULL")
		assert.False(t, doubanRating.Valid, "douban_rating should be NULL")
		assert.False(t, doubanVoteCount.Valid, "douban_vote_count should be NULL")
	})

	t.Run("columns are writable", func(t *testing.T) {
		_, err := db.Exec(`UPDATE movies SET douban_id = '1292052', douban_rating = 9.7, douban_vote_count = 2130000 WHERE id = 'movie-1'`)
		require.NoError(t, err)

		var doubanID string
		var doubanRating float64
		var doubanVoteCount int64
		err = db.QueryRow(`SELECT douban_id, douban_rating, douban_vote_count FROM movies WHERE id = 'movie-1'`).
			Scan(&doubanID, &doubanRating, &doubanVoteCount)
		require.NoError(t, err)
		assert.Equal(t, "1292052", doubanID)
		assert.Equal(t, 9.7, doubanRating)
		assert.Equal(t, int64(2130000), doubanVoteCount)
	})

	t.Run("douban_id indexes exist", func(t *testing.T) {
		for _, idx := range []string{"idx_movies_douban_id", "idx_series_douban_id"} {
			var name string
			err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name = ?`, idx).Scan(&name)
			require.NoError(t, err, "index %s should exist", idx)
			assert.Equal(t, idx, name)
		}
	})

	t.Run("existing data preserved", func(t *testing.T) {
		var title string
		require.NoError(t, db.QueryRow(`SELECT title FROM movies WHERE id = 'movie-1'`).Scan(&title))
		assert.Equal(t, "Test Movie", title)
	})
}

func TestAddDoubanRatingFields_Idempotent(t *testing.T) {
	db := setupDoubanRatingTables(t)
	defer db.Close()

	migration := &addDoubanRatingFields{migrationBase: NewMigrationBase(24, "add_douban_rating_fields")}

	tx1, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx1))
	require.NoError(t, tx1.Commit())

	// Second run must be a no-op (columnExists + IF NOT EXISTS guards).
	tx2, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx2))
	require.NoError(t, tx2.Commit())
}

func TestAddDoubanRatingFields_Down(t *testing.T) {
	db := setupDoubanRatingTables(t)
	defer db.Close()

	migration := &addDoubanRatingFields{migrationBase: NewMigrationBase(24, "add_douban_rating_fields")}

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx))
	require.NoError(t, tx.Commit())

	txDown, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Down(txDown))
	require.NoError(t, txDown.Commit())

	for _, idx := range []string{"idx_movies_douban_id", "idx_series_douban_id"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name = ?`, idx).Scan(&name)
		assert.ErrorIs(t, err, sql.ErrNoRows, "index %s should be dropped", idx)
	}
}

func TestAddDoubanRatingFields_Version(t *testing.T) {
	migration := &addDoubanRatingFields{migrationBase: NewMigrationBase(24, "add_douban_rating_fields")}
	assert.Equal(t, int64(24), migration.Version())
	assert.Equal(t, "add_douban_rating_fields", migration.Name())
}
