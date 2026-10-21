package migrations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigration047_CreatesAISideSpend(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	for _, kind := range []string{"parse", "fansub", "keywords"} {
		_, err := db.Exec(`INSERT INTO ai_side_spend (kind, provider, model_id, input_tokens, output_tokens, spent_usd, created_at)
			VALUES (?, 'gemini', 'gemini-2.5-flash-lite', 1000, 200, 0.00018, ?)`, kind, now)
		require.NoError(t, err, kind)
	}

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM ai_side_spend`).Scan(&n))
	assert.Equal(t, 3, n)

	var idx string
	require.NoError(t, db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index' AND name = 'idx_ai_side_spend_created_at'`).Scan(&idx),
		"the monthly window reads by created_at")
}

func TestMigration047_RejectsUnknownKindAndMissingSpend(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO ai_side_spend (kind, provider, model_id, input_tokens, output_tokens, spent_usd, created_at)
		VALUES ('translate', 'claude', 'claude-sonnet-5', 1, 1, 0.1, ?)`, now)
	assert.Error(t, err, "subtitle lanes live in subtitle_runs, not here")

	_, err = db.Exec(`INSERT INTO ai_side_spend (kind, provider, model_id, input_tokens, output_tokens, spent_usd, created_at)
		VALUES ('parse', 'claude', 'claude-sonnet-5', 1, 1, NULL, ?)`, now)
	assert.Error(t, err, "a recorded row always carries its money")

	_, err = db.Exec(`INSERT INTO ai_side_spend (kind, provider, model_id, input_tokens, output_tokens, spent_usd)
		VALUES ('parse', 'claude', 'claude-sonnet-5', 1, 1, 0.1)`)
	assert.Error(t, err, "created_at is written by the app (dbtime shape), never a CURRENT_TIMESTAMP default")
}

func TestMigration047_UpIsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&createAISideSpendTable{}).Up(tx))
	require.NoError(t, tx.Commit())
}

func TestMigration047_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 47 {
			found = true
			assert.Equal(t, "create_ai_side_spend_table", m.Name())
		}
	}
	assert.True(t, found)
}
