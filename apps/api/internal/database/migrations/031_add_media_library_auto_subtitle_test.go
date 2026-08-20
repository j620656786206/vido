package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupMediaLibrariesTable(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS media_libraries (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			auto_detect INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO media_libraries (id, name, content_type) VALUES ('lib-1', '我的電影', 'movie')`)
	require.NoError(t, err)
	return db
}

func runAutoSubtitleMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &addMediaLibraryAutoSubtitle{migrationBase: NewMigrationBase(31, "add_media_library_auto_subtitle")}
	require.NoError(t, migration.Up(tx))
	require.NoError(t, tx.Commit())
}

// TestAddMediaLibraryAutoSubtitle_DefaultsOff is the cost-safety assertion, not
// a schema formality: a library that pre-dates this feature must not start
// processing anything on the next scan.
func TestAddMediaLibraryAutoSubtitle_DefaultsOff(t *testing.T) {
	db := setupMediaLibrariesTable(t)
	defer db.Close()

	runAutoSubtitleMigration(t, db)

	var autoSubtitle bool
	require.NoError(t, db.QueryRow(`SELECT auto_subtitle FROM media_libraries WHERE id = 'lib-1'`).Scan(&autoSubtitle))
	assert.False(t, autoSubtitle, "an existing library must be OFF after the migration — opt-in is never inherited")
}

func TestAddMediaLibraryAutoSubtitle_NewRowsDefaultOff(t *testing.T) {
	db := setupMediaLibrariesTable(t)
	defer db.Close()
	runAutoSubtitleMigration(t, db)

	_, err := db.Exec(`INSERT INTO media_libraries (id, name, content_type) VALUES ('lib-2', '我的影集', 'series')`)
	require.NoError(t, err)

	var autoSubtitle bool
	require.NoError(t, db.QueryRow(`SELECT auto_subtitle FROM media_libraries WHERE id = 'lib-2'`).Scan(&autoSubtitle))
	assert.False(t, autoSubtitle, "a library created without naming the column must be OFF")
}

func TestAddMediaLibraryAutoSubtitle_IsIdempotent(t *testing.T) {
	db := setupMediaLibrariesTable(t)
	defer db.Close()

	runAutoSubtitleMigration(t, db)
	assert.NotPanics(t, func() { runAutoSubtitleMigration(t, db) },
		"re-running must be a no-op — migrations replay on every boot")
}

func TestAddMediaLibraryAutoSubtitle_DownIsSafe(t *testing.T) {
	db := setupMediaLibrariesTable(t)
	defer db.Close()
	runAutoSubtitleMigration(t, db)

	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &addMediaLibraryAutoSubtitle{migrationBase: NewMigrationBase(31, "add_media_library_auto_subtitle")}
	require.NoError(t, migration.Down(tx))
	require.NoError(t, tx.Commit())
}
