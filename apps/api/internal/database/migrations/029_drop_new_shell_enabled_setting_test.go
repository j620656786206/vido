package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupDropShellFlagMigration(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	require.NoError(t, err)
	return db
}

func runDropShellFlag(t *testing.T, db *sql.DB, up bool) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &dropNewShellEnabledSetting{migrationBase: NewMigrationBase(29, "drop_new_shell_enabled_setting")}
	if up {
		require.NoError(t, migration.Up(tx))
	} else {
		require.NoError(t, migration.Down(tx))
	}
	require.NoError(t, tx.Commit())
}

func TestDropNewShellEnabledSetting_Up(t *testing.T) {
	db := setupDropShellFlagMigration(t)
	defer db.Close()

	t.Run("deletes the stored flag row and nothing else", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO settings (key, value, type) VALUES ('new_shell_enabled', 'true', 'bool')`)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO settings (key, value, type) VALUES ('other_setting', 'keep', 'string')`)
		require.NoError(t, err)

		runDropShellFlag(t, db, true)

		var count int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM settings WHERE key = 'new_shell_enabled'`).Scan(&count))
		assert.Equal(t, 0, count, "flag row should be gone")
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM settings WHERE key = 'other_setting'`).Scan(&count))
		assert.Equal(t, 1, count, "unrelated settings must survive")
	})

	t.Run("is a no-op when the flag was never seeded", func(t *testing.T) {
		runDropShellFlag(t, db, true)
	})
}

func TestDropNewShellEnabledSetting_Down(t *testing.T) {
	db := setupDropShellFlagMigration(t)
	defer db.Close()

	runDropShellFlag(t, db, true)
	runDropShellFlag(t, db, false)

	var value, typ string
	require.NoError(t, db.QueryRow(`SELECT value, type FROM settings WHERE key = 'new_shell_enabled'`).
		Scan(&value, &typ))
	assert.Equal(t, "true", value, "Down restores the post-cutover-1 forced-ON state")
	assert.Equal(t, "bool", typ)
}
