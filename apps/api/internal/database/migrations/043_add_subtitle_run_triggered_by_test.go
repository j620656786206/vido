package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func runMigration043(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&addSubtitleRunTriggeredBy{migrationBase: NewMigrationBase(43, "x")}).Up(tx))
	require.NoError(t, tx.Commit())
}

func TestMigration043_AddsNullableTriggeredByColumn(t *testing.T) {
	db := newFullyMigratedDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	assert.True(t, columnExists(tx, "subtitle_runs", "triggered_by"))
	require.NoError(t, tx.Rollback())

	// A row written without the column stays NULL — old runs are not guessed at.
	_, err = db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, status, started_at)
		VALUES ('r1', 'm1', 'movie', 'completed', '2026-10-01 00:00:00')`)
	require.NoError(t, err)
	var triggeredBy sql.NullString
	require.NoError(t, db.QueryRow(`SELECT triggered_by FROM subtitle_runs WHERE id = 'r1'`).Scan(&triggeredBy))
	assert.False(t, triggeredBy.Valid)
}

func TestMigration043_IsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t) // the chain already ran 043 once
	runMigration043(t, db)      // a second run must be a no-op, not "duplicate column"

	tx, err := db.Begin()
	require.NoError(t, err)
	assert.True(t, columnExists(tx, "subtitle_runs", "triggered_by"))
	require.NoError(t, tx.Rollback())
}
