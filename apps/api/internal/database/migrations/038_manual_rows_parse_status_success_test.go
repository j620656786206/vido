package migrations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dsr-2b-a AC #3: rows the user edited before the fix are stuck at
// metadata_source=manual + parse_status failed/pending — batch enrichment only
// picks pending rows and (now) leaves manual rows alone, so nothing else would
// ever clear them.
func TestMigration038_ManualRowsStuckFailedOrPendingBecomeSuccess(t *testing.T) {
	db := newMigratedDBUpTo(t, 37)
	now := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO movies (id, title, release_date, metadata_source, parse_status, created_at, updated_at) VALUES
		('m-manual-failed', '我填的', '', 'manual', 'failed', ?, ?),
		('m-manual-pending', '我填的2', '', 'manual', 'pending', ?, ?),
		('m-tmdb-failed', 'x.mkv', '', 'tmdb', 'failed', ?, ?),
		('m-nosource-pending', 'y.mkv', '', NULL, 'pending', ?, ?)`, now, now, now, now, now, now, now, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO series (id, title, first_air_date, metadata_source, parse_status, created_at, updated_at) VALUES
		('s-manual-failed', '我填的影集', '', 'manual', 'failed', ?, ?),
		('s-douban-pending', 'z', '', 'douban', 'pending', ?, ?)`, now, now, now, now)
	require.NoError(t, err)

	runner, err := NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(GetAll()))
	require.NoError(t, runner.Up(t.Context()))

	status := func(table, id string) string {
		var s string
		require.NoError(t, db.QueryRow(`SELECT parse_status FROM `+table+` WHERE id = ?`, id).Scan(&s))
		return s
	}
	assert.Equal(t, "success", status("movies", "m-manual-failed"))
	assert.Equal(t, "success", status("movies", "m-manual-pending"))
	assert.Equal(t, "failed", status("movies", "m-tmdb-failed"), "only rows the user owns")
	assert.Equal(t, "pending", status("movies", "m-nosource-pending"))
	assert.Equal(t, "success", status("series", "s-manual-failed"))
	assert.Equal(t, "pending", status("series", "s-douban-pending"))

	// Idempotent: the runner skips applied versions, so run the SQL itself again.
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&manualRowsParseStatusSuccess{}).Up(tx))
	require.NoError(t, tx.Commit())
	assert.Equal(t, "success", status("movies", "m-manual-failed"))
	assert.Equal(t, "failed", status("movies", "m-tmdb-failed"))
}
