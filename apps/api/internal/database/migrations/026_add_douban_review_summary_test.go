package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupDoubanCacheTable(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS douban_cache (
			id TEXT PRIMARY KEY,
			douban_id TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO douban_cache (id, douban_id, title, expires_at) VALUES ('c1', '1292052', 'Test', '2099-01-01')`)
	require.NoError(t, err)
	return db
}

func TestAddDoubanReviewSummary_Up(t *testing.T) {
	db := setupDoubanCacheTable(t)
	defer db.Close()

	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &addDoubanReviewSummary{migrationBase: NewMigrationBase(26, "add_douban_review_summary")}
	require.NoError(t, migration.Up(tx))
	require.NoError(t, tx.Commit())

	t.Run("review_summary_json column exists with NULL default", func(t *testing.T) {
		var rs sql.NullString
		err := db.QueryRow(`SELECT review_summary_json FROM douban_cache WHERE douban_id = '1292052'`).Scan(&rs)
		require.NoError(t, err)
		assert.False(t, rs.Valid, "review_summary_json should be NULL")
	})

	t.Run("column is writable", func(t *testing.T) {
		_, err := db.Exec(`UPDATE douban_cache SET review_summary_json = '{"total_comments":5}' WHERE douban_id = '1292052'`)
		require.NoError(t, err)

		var rs string
		require.NoError(t, db.QueryRow(`SELECT review_summary_json FROM douban_cache WHERE douban_id = '1292052'`).Scan(&rs))
		assert.Equal(t, `{"total_comments":5}`, rs)
	})

	t.Run("existing data preserved", func(t *testing.T) {
		var title string
		require.NoError(t, db.QueryRow(`SELECT title FROM douban_cache WHERE douban_id = '1292052'`).Scan(&title))
		assert.Equal(t, "Test", title)
	})
}

func TestAddDoubanReviewSummary_Idempotent(t *testing.T) {
	db := setupDoubanCacheTable(t)
	defer db.Close()

	migration := &addDoubanReviewSummary{migrationBase: NewMigrationBase(26, "add_douban_review_summary")}

	tx1, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx1))
	require.NoError(t, tx1.Commit())

	// Second run must be a no-op (columnExists guard).
	tx2, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx2))
	require.NoError(t, tx2.Commit())
}

func TestAddDoubanReviewSummary_Version(t *testing.T) {
	migration := &addDoubanReviewSummary{migrationBase: NewMigrationBase(26, "add_douban_review_summary")}
	assert.Equal(t, int64(26), migration.Version())
	assert.Equal(t, "add_douban_review_summary", migration.Name())
}
