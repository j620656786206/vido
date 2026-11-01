package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigration049_AddsNullableBurnedInColumns(t *testing.T) {
	db := newFullyMigratedDB(t)

	_, err := db.Exec(`INSERT INTO series (id, title, first_air_date) VALUES ('s1', 'Show', '2020-01-01')`)
	require.NoError(t, err)
	// A pre-049 shaped insert still works and reads back NULL: "never checked"
	// is never "none".
	_, err = db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number) VALUES ('e1', 's1', 1, 1)`)
	require.NoError(t, err)
	var burned sql.NullString
	var checkedAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT burned_in_subtitle, burned_in_checked_at FROM episodes WHERE id = 'e1'`).Scan(&burned, &checkedAt))
	assert.False(t, burned.Valid)
	assert.False(t, checkedAt.Valid)

	_, err = db.Exec(`UPDATE episodes SET burned_in_subtitle = 'zh-Hans', burned_in_checked_at = ? WHERE id = 'e1'`, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT burned_in_subtitle FROM episodes WHERE id = 'e1'`).Scan(&burned))
	assert.Equal(t, "zh-Hans", burned.String)
}

func TestMigration049_SideSpendAcceptsBurnedInCheckAndKeepsRows(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO ai_side_spend (kind, provider, model_id, spent_usd, created_at) VALUES ('burned_in_check', 'claude', 'claude-sonnet-5', 0.0023, ?)`, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ai_side_spend (kind, provider, spent_usd, created_at) VALUES ('bogus', 'claude', 1, ?)`, now)
	assert.Error(t, err, "the kind set is still closed")

	var idx int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_ai_side_spend_created_at'`).Scan(&idx))
	assert.Equal(t, 1, idx, "the created_at index survives the rebuild")
}

func TestMigration049_RebuildKeepsExistingSideSpendRows(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO ai_side_spend (kind, provider, model_id, input_tokens, output_tokens, spent_usd, created_at) VALUES ('parse', 'claude', 'm', 10, 2, 0.01, ?)`, now)
	require.NoError(t, err)

	// Roll back to the 047 shape and forward again: the parse row survives both.
	tx, err := db.Begin()
	require.NoError(t, err)
	m := &addEpisodeBurnedInSubtitle{}
	require.NoError(t, m.Down(tx))
	require.NoError(t, m.Up(tx))
	require.NoError(t, tx.Commit())

	var kind string
	var in, out int64
	require.NoError(t, db.QueryRow(`SELECT kind, input_tokens, output_tokens FROM ai_side_spend`).Scan(&kind, &in, &out))
	assert.Equal(t, "parse", kind)
	assert.Equal(t, int64(10), in)
	assert.Equal(t, int64(2), out)
}

func TestMigration049_UpIsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&addEpisodeBurnedInSubtitle{}).Up(tx), "a second Up on a migrated database must be a no-op")
	require.NoError(t, tx.Commit())
}

func TestMigration049_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 49 {
			found = true
			assert.Equal(t, "add_episode_burned_in_subtitle", m.Name())
		}
	}
	assert.True(t, found, "migration 049 must be registered")
}
