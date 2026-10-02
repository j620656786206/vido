package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func runMigration042(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&normalizeGoStringTimestamps{migrationBase: NewMigrationBase(42, "x")}).Up(tx))
	require.NoError(t, tx.Commit())
}

func TestMigration042_RewritesGoStringTimestampsToUTC(t *testing.T) {
	db := newFullyMigratedDB(t) // plain modernc driver: binds time.Time with String(), like production did
	taipei := time.FixedZone("CST", 8*3600)
	written := time.Date(2026, 8, 24, 10, 56, 26, 303854294, taipei)

	_, err := db.Exec(`INSERT INTO settings (key, value, type, created_at, updated_at) VALUES
		('bound', 'v', 'string', ?, ?),
		('literal', 'v', 'string', '2026-08-24 10:56:26.303854294 +0800 CST m=+26.660278123', '2026-08-24 06:13:23.881504585 +0000 UTC m=+27.125705570'),
		('unnamed-zone', 'v', 'string', '2026-03-27 05:43:06 +0800 +0800', CURRENT_TIMESTAMP),
		('rfc3339', 'v', 'string', '2026-07-13T10:55:17.754221086Z', '2026-07-13T10:55:17Z'),
		('junk', 'v', 'string', '2026-07-13 10:55:17 +9999 nonsense', '')`,
		time.Now().In(taipei), written)
	require.NoError(t, err)

	var before sql.NullString
	require.NoError(t, db.QueryRow(`SELECT datetime(created_at) FROM settings WHERE key = 'bound'`).Scan(&before))
	require.False(t, before.Valid, "precondition: the plain driver really stores text SQLite cannot read")

	runMigration042(t, db)

	text := func(key, col string) sql.NullString {
		var s sql.NullString
		require.NoError(t, db.QueryRow(`SELECT CAST(`+col+` AS TEXT) FROM settings WHERE key = ?`, key).Scan(&s))
		return s
	}
	assert.Equal(t, "2026-08-24 02:56:26.303854294Z", text("bound", "updated_at").String)
	assert.Equal(t, "2026-08-24 02:56:26.303854294Z", text("literal", "created_at").String)
	assert.Equal(t, "2026-08-24 06:13:23.881504585Z", text("literal", "updated_at").String)
	assert.Equal(t, "2026-03-26 21:43:06.000000000Z", text("unnamed-zone", "created_at").String)

	// Shapes that are not Go String() stay exactly as they were.
	assert.Equal(t, "2026-07-13T10:55:17.754221086Z", text("rfc3339", "created_at").String)
	assert.Equal(t, "2026-07-13T10:55:17Z", text("rfc3339", "updated_at").String)
	assert.Equal(t, "2026-07-13 10:55:17 +9999 nonsense", text("junk", "created_at").String, "unparseable text is never lost")
	assert.Equal(t, "", text("junk", "updated_at").String)

	// Every rewritten value is now readable by SQLite itself.
	var unreadable int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM settings
		WHERE key IN ('bound', 'literal', 'unnamed-zone')
		  AND (datetime(created_at) IS NULL OR datetime(updated_at) IS NULL)`).Scan(&unreadable))
	assert.Zero(t, unreadable)

	// And Go still scans them into the same instant.
	var back time.Time
	require.NoError(t, db.QueryRow(`SELECT updated_at FROM settings WHERE key = 'bound'`).Scan(&back))
	assert.True(t, back.Equal(written))
}

func TestMigration042_IsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t)
	_, err := db.Exec(`INSERT INTO settings (key, value, type, created_at, updated_at)
		VALUES ('k', 'v', 'string', '2026-08-24 10:56:26.303854294 +0800 CST m=+26.6', '2026-08-24 10:56:26 +0800 CST')`)
	require.NoError(t, err)

	runMigration042(t, db)
	var first string
	require.NoError(t, db.QueryRow(`SELECT created_at || '|' || updated_at FROM settings WHERE key = 'k'`).Scan(&first))
	runMigration042(t, db)
	var second string
	require.NoError(t, db.QueryRow(`SELECT created_at || '|' || updated_at FROM settings WHERE key = 'k'`).Scan(&second))
	assert.Equal(t, first, second)
	assert.Equal(t, "2026-08-24 02:56:26.303854294Z|2026-08-24 02:56:26.000000000Z", first)
}
