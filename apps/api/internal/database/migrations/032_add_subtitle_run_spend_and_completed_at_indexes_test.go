package migrations

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newFullyMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	runner, err := NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func TestMigration032_AddsSpendColumnsAndIndexes(t *testing.T) {
	db := newFullyMigratedDB(t)

	// Columns exist and accept writes.
	_, err := db.Exec(`INSERT INTO subtitle_runs (id, media_id, media_type, metadata_hash, glossary_version, prompt_version, model_id, status, cache_enabled, started_at, spent_usd, budget_usd)
		VALUES ('r1', 'm1', 'movie', 'h', 'g', 'p', 'model', 'completed', 0, ?, 0.42, 5.0)`, time.Now().UTC())
	require.NoError(t, err)

	for _, idx := range []string{"idx_subtitle_runs_completed_at", "idx_parse_jobs_completed_at"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?`, idx).Scan(&name)
		require.NoError(t, err, idx)
	}
}

// TestMigration032_NormalizesLocalOffsetTimestamps exercises the Go-side
// rewrite directly: the shipped chain runs against an empty table, so the
// normalization path needs a seeded local-offset row and a manual call.
func TestMigration032_NormalizesLocalOffsetTimestamps(t *testing.T) {
	db := newFullyMigratedDB(t)

	local := time.Date(2026, 8, 26, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	_, err := db.Exec(`INSERT INTO parse_jobs (id, torrent_hash, file_path, file_name, status, retry_count, created_at, updated_at, completed_at)
		VALUES ('legacy', 'h', '/x', 'x', 'completed', 0, ?, ?, ?)`, local, local, local)
	require.NoError(t, err)

	// Sanity: the driver stored the local offset verbatim.
	var raw string
	require.NoError(t, db.QueryRow(`SELECT CAST(completed_at AS TEXT) FROM parse_jobs WHERE id = 'legacy'`).Scan(&raw))
	require.Contains(t, raw, "+0800")

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, normalizeParseJobTimestampsToUTC(tx))
	require.NoError(t, tx.Commit())

	for _, col := range []string{"created_at", "updated_at", "completed_at"} {
		var text string
		require.NoError(t, db.QueryRow(`SELECT CAST(`+col+` AS TEXT) FROM parse_jobs WHERE id = 'legacy'`).Scan(&text))
		assert.True(t, strings.Contains(text, "+0000") || strings.HasSuffix(text, "Z"), "%s not UTC: %q", col, text)
		// 10:00 +08:00 == 02:00 UTC — the instant survives the rewrite.
		assert.Contains(t, text, "02:00:00", col)
	}
}

func TestParseNonUTCTimeText(t *testing.T) {
	ts, needs := parseNonUTCTimeText("2026-08-26 10:00:00 +0800 CST")
	require.True(t, needs)
	assert.Equal(t, "2026-08-26 02:00:00 +0000 UTC", ts.UTC().String())

	_, needs = parseNonUTCTimeText("2026-08-26 02:00:00 +0000 UTC")
	assert.False(t, needs, "already-UTC text must not churn")

	_, needs = parseNonUTCTimeText("")
	assert.False(t, needs)

	_, needs = parseNonUTCTimeText("not a time")
	assert.False(t, needs, "unparseable text is left untouched")
}
