package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupGlossaryMigration(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &createShowGlossaryTable{migrationBase: NewMigrationBase(28, "create_show_glossary_table")}
	require.NoError(t, migration.Up(tx))
	require.NoError(t, tx.Commit())
	return db
}

func TestCreateShowGlossaryTable_Up(t *testing.T) {
	db := setupGlossaryMigration(t)
	defer db.Close()

	t.Run("inserts a minimal row with defaults", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, term_src, term_zh) VALUES ('g1', 'm1', 'Demogorgon', '魔王獸')`)
		require.NoError(t, err)

		var lang, source string
		var confirmed int
		require.NoError(t, db.QueryRow(`SELECT language, source, confirmed FROM show_glossary WHERE id = 'g1'`).
			Scan(&lang, &source, &confirmed))
		assert.Equal(t, "zh-Hant", lang)
		assert.Equal(t, "manual", source)
		assert.Equal(t, 0, confirmed)
	})

	t.Run("rejects invalid source", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, term_src, term_zh, source) VALUES ('g2', 'm1', 'x', 'y', 'bogus')`)
		require.Error(t, err)
	})

	t.Run("enforces unique (media_id, term_src, language)", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, term_src, term_zh) VALUES ('g3', 'm2', 'Vecna', '維克那')`)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO show_glossary (id, media_id, term_src, term_zh) VALUES ('g4', 'm2', 'Vecna', '別的')`)
		require.Error(t, err, "same media+term+language must collide")

		// Different media id is allowed.
		_, err = db.Exec(`INSERT INTO show_glossary (id, media_id, term_src, term_zh) VALUES ('g5', 'm3', 'Vecna', '維克那')`)
		require.NoError(t, err)
	})
}

func TestCreateShowGlossaryTable_Down(t *testing.T) {
	db := setupGlossaryMigration(t)
	defer db.Close()

	tx, err := db.Begin()
	require.NoError(t, err)
	migration := &createShowGlossaryTable{migrationBase: NewMigrationBase(28, "create_show_glossary_table")}
	require.NoError(t, migration.Down(tx))
	require.NoError(t, tx.Commit())

	_, err = db.Exec(`SELECT 1 FROM show_glossary`)
	require.Error(t, err, "table should be gone after Down")
}
