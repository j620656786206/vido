package migrations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration050_ClearsOnlyUndecidedStamps(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()
	for i, r := range []struct {
		id      string
		burned  any
		checked any
	}{
		{"undecided", nil, now},     // stamped without a verdict → retried
		{"decided", "zh-Hans", now}, // has a verdict → kept
		{"never", nil, nil},         // never checked → stays NULL
	} {
		_, err := db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, subtitle_status, burned_in_subtitle, burned_in_checked_at, created_at, updated_at)
			VALUES (?, 's', 1, ?, 'not_searched', ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, r.id, i+1, r.burned, r.checked)
		require.NoError(t, err)
	}

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&retryUndecidedBurnedInChecks{}).Up(tx))
	require.NoError(t, tx.Commit())

	stamped := func(id string) bool {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM episodes WHERE id = ? AND burned_in_checked_at IS NOT NULL`, id).Scan(&n))
		return n == 1
	}
	assert.False(t, stamped("undecided"), "an undecided season is retried on the next round")
	assert.True(t, stamped("decided"), "a season with a verdict keeps its stamp")
	assert.False(t, stamped("never"))
}
