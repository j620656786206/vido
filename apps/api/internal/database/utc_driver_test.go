package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"

	"github.com/vido/api/internal/config"
)

func newUTCTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := New(&config.DatabaseConfig{
		Path:            filepath.Join(t.TempDir(), "vido.db"),
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
		BusyTimeout:     time.Second,
		CacheSize:       -2000,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Conn().Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, created_at TIMESTAMP, done_at TIMESTAMP, note TEXT, n INTEGER)`)
	require.NoError(t, err)
	return db.Conn()
}

// AC #5: the exact production failure — time.Now() carries a local zone and a
// monotonic reading — must come out as text SQLite's own date functions read.
func TestUTCDriver_TimeNowIsStoredAsSQLiteReadableUTC(t *testing.T) {
	db := newUTCTestDB(t)
	taipei := time.FixedZone("CST", 8*3600)
	now := time.Now().In(taipei) // .In keeps the monotonic reading, like time.Now()

	_, err := db.Exec(`INSERT INTO t (id, created_at) VALUES (1, ?)`, now)
	require.NoError(t, err)

	var text string
	var sqliteDT sql.NullString
	require.NoError(t, db.QueryRow(`SELECT CAST(created_at AS TEXT), datetime(created_at) FROM t WHERE id = 1`).Scan(&text, &sqliteDT))
	assert.True(t, sqliteDT.Valid, "datetime(created_at) must not be NULL, got text %q", text)
	assert.True(t, strings.HasSuffix(text, "Z"), "stored as UTC: %q", text)
	assert.NotContains(t, text, "m=")
	assert.NotContains(t, text, "CST")
	assert.Len(t, text, len("2006-01-02 15:04:05.000000000Z"), "fixed width so text order is time order")
	assert.Equal(t, now.UTC().Format("2006-01-02 15:04:05"), sqliteDT.String)

	var back time.Time
	require.NoError(t, db.QueryRow(`SELECT created_at FROM t WHERE id = 1`).Scan(&back))
	assert.True(t, back.Equal(now), "round trip keeps the instant to the nanosecond: %v vs %v", back, now)
}

// AC #1: Valuer-wrapped times take the same path; NULLs and non-time values
// are untouched.
func TestUTCDriver_ValuersNullsAndOtherValues(t *testing.T) {
	db := newUTCTestDB(t)
	at := time.Date(2026, 8, 24, 10, 56, 26, 303854294, time.FixedZone("CST", 8*3600))

	_, err := db.Exec(`INSERT INTO t (id, created_at, done_at, note, n) VALUES (?, ?, ?, ?, ?)`,
		2, sql.NullTime{Time: at, Valid: true}, sql.NullTime{}, "2026-08-24 10:56:26 +0800 CST", 42)
	require.NoError(t, err)
	var ptr *time.Time
	_, err = db.Exec(`INSERT INTO t (id, created_at) VALUES (3, ?)`, ptr)
	require.NoError(t, err)

	var created string
	var done sql.NullString
	var note string
	var n int
	require.NoError(t, db.QueryRow(`SELECT CAST(created_at AS TEXT), done_at, note, n FROM t WHERE id = 2`).Scan(&created, &done, &note, &n))
	assert.Equal(t, "2026-08-24 02:56:26.303854294Z", created)
	assert.False(t, done.Valid)
	assert.Equal(t, "2026-08-24 10:56:26 +0800 CST", note, "strings are never rewritten — only time.Time values")
	assert.Equal(t, 42, n)

	var nilCreated sql.NullString
	require.NoError(t, db.QueryRow(`SELECT created_at FROM t WHERE id = 3`).Scan(&nilCreated))
	assert.False(t, nilCreated.Valid)
}

// AC #2: rows written before the fix (Go String()) and by SQLite defaults
// still scan into time.Time, and compare correctly against new arguments.
func TestUTCDriver_ReadsOldShapesAndComparesInUTC(t *testing.T) {
	db := newUTCTestDB(t)
	_, err := db.Exec(`INSERT INTO t (id, created_at) VALUES
		(10, '2026-08-24 10:56:26.303854294 +0800 CST m=+26.660278123'),
		(11, '2026-08-24 06:13:23'),
		(12, ?)`, time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	want := map[int]time.Time{
		10: time.Date(2026, 8, 24, 2, 56, 26, 303854294, time.UTC),
		11: time.Date(2026, 8, 24, 6, 13, 23, 0, time.UTC),
		12: time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC),
	}
	for id, w := range want {
		var got time.Time
		require.NoError(t, db.QueryRow(`SELECT created_at FROM t WHERE id = ?`, id).Scan(&got))
		assert.True(t, got.Equal(w), "id %d: %v vs %v", id, got, w)
	}

	// A time argument from a +08:00 clock compares as the same instant.
	var newer int
	cut := time.Date(2026, 8, 24, 13, 30, 0, 0, time.FixedZone("CST", 8*3600)) // 05:30Z
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM t WHERE id IN (11, 12) AND created_at > ?`, cut).Scan(&newer))
	assert.Equal(t, 1, newer, "only 06:13Z is after 05:30Z")
}

// AC #3: BackupService restores through conn.Raw + NewRestore; the wrapper
// must keep exposing it.
func TestUTCDriver_OnlineRestoreStillReachable(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.db")
	srcDB, err := sql.Open("sqlite", src)
	require.NoError(t, err)
	_, err = srcDB.Exec(`CREATE TABLE restored (v TEXT); INSERT INTO restored VALUES ('ok')`)
	require.NoError(t, err)
	require.NoError(t, srcDB.Close())

	db := newUTCTestDB(t)
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	err = conn.Raw(func(dc any) error {
		r, ok := dc.(interface {
			NewRestore(string) (*sqlite.Backup, error)
		})
		require.True(t, ok, "wrapped connection %T must expose NewRestore", dc)
		bk, err := r.NewRestore("file:" + src + "?mode=ro")
		if err != nil {
			return err
		}
		if _, err := bk.Step(-1); err != nil {
			_ = bk.Finish()
			return err
		}
		return bk.Finish()
	})
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	var v string
	require.NoError(t, db.QueryRow(`SELECT v FROM restored`).Scan(&v))
	assert.Equal(t, "ok", v)
}
