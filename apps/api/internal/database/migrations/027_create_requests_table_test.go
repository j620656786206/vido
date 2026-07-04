package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupRequestsMigration(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &createRequestsTable{migrationBase: NewMigrationBase(27, "create_requests_table")}
	require.NoError(t, migration.Up(tx))
	require.NoError(t, tx.Commit())
	return db
}

func TestCreateRequestsTable_Up(t *testing.T) {
	db := setupRequestsMigration(t)
	defer db.Close()

	t.Run("inserts a minimal row with defaults", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO requests (id, tmdb_id, media_type, title) VALUES ('r1', 550, 'movie', '鬥陣俱樂部')`)
		require.NoError(t, err)

		var status string
		var fulfilment sql.NullString
		require.NoError(t, db.QueryRow(`SELECT status, fulfilment_source FROM requests WHERE id = 'r1'`).Scan(&status, &fulfilment))
		assert.Equal(t, "pending", status)
		assert.False(t, fulfilment.Valid, "fulfilment_source should default NULL until 13-4 claims the row")
	})

	t.Run("rejects invalid media_type", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO requests (id, tmdb_id, media_type, title) VALUES ('r2', 551, 'series', 'x')`)
		assert.Error(t, err, "media_type CHECK must reject 'series' — requests speak TMDB vocabulary")
	})

	t.Run("rejects invalid status", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO requests (id, tmdb_id, media_type, title, status) VALUES ('r3', 552, 'movie', 'x', 'organizing')`)
		assert.Error(t, err, "status CHECK must reject values outside the 5-value enum")
	})

	t.Run("rejects invalid fulfilment_source", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO requests (id, tmdb_id, media_type, title, fulfilment_source) VALUES ('r4', 553, 'movie', 'x', 'jackett')`)
		assert.Error(t, err, "fulfilment_source CHECK must reject values outside arr|builtin")
	})
}

func TestCreateRequestsTable_ActiveUniqueGuard(t *testing.T) {
	db := setupRequestsMigration(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO requests (id, tmdb_id, media_type, title, status) VALUES ('a1', 550, 'movie', 'x', 'pending')`)
	require.NoError(t, err)

	t.Run("second ACTIVE request for same (tmdb_id, media_type) is rejected", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO requests (id, tmdb_id, media_type, title, status) VALUES ('a2', 550, 'movie', 'x', 'searching')`)
		assert.ErrorContains(t, err, "UNIQUE")
	})

	t.Run("same tmdb_id with different media_type is allowed", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO requests (id, tmdb_id, media_type, title, status) VALUES ('a3', 550, 'tv', 'x', 'pending')`)
		assert.NoError(t, err)
	})

	t.Run("re-request allowed after the prior request fails", func(t *testing.T) {
		_, err := db.Exec(`UPDATE requests SET status = 'failed' WHERE id = 'a1'`)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO requests (id, tmdb_id, media_type, title, status) VALUES ('a4', 550, 'movie', 'x', 'pending')`)
		assert.NoError(t, err, "failed rows must not block a re-request (partial index)")
	})
}

func TestCreateRequestsTable_Down(t *testing.T) {
	db := setupRequestsMigration(t)
	defer db.Close()

	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &createRequestsTable{migrationBase: NewMigrationBase(27, "create_requests_table")}
	require.NoError(t, migration.Down(tx))
	require.NoError(t, tx.Commit())

	var name string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='requests'`).Scan(&name)
	assert.ErrorIs(t, err, sql.ErrNoRows, "requests table should be dropped")
}

func TestCreateRequestsTable_Version(t *testing.T) {
	migration := &createRequestsTable{migrationBase: NewMigrationBase(27, "create_requests_table")}
	assert.Equal(t, int64(27), migration.Version())
	assert.Equal(t, "create_requests_table", migration.Name())
}
