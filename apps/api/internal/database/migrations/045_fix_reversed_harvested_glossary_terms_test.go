package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func insertGlossary(t *testing.T, db *sql.DB, id, scope, src, zh, source string, confirmed int) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, scope, term_src, term_zh, language, source, confirmed)
		VALUES (?, 'm1', ?, ?, ?, 'zh-Hant', ?, ?)`, id, scope, src, zh, source, confirmed)
	require.NoError(t, err)
}

func glossaryRow(t *testing.T, db *sql.DB, id string) (src, zh string, found bool) {
	t.Helper()
	err := db.QueryRow(`SELECT term_src, term_zh FROM show_glossary WHERE id = ?`, id).Scan(&src, &zh)
	if err == sql.ErrNoRows {
		return "", "", false
	}
	require.NoError(t, err)
	return src, zh, true
}

func TestMigration045_TurnsReversedHarvestedTermsAround(t *testing.T) {
	db := newFullyMigratedDB(t)

	// The two NAS rows (2026-10-08), plus guards.
	insertGlossary(t, db, "r1", "show:1", "馬言者", "The Neigh-sayer", "subtitle", 0)
	insertGlossary(t, db, "r2", "show:1", "風舞市長", "Mayor Winddancer", "subtitle", 0)
	insertGlossary(t, db, "dup", "show:1", "凱勒布瑞博", "Celebrimbor", "subtitle", 0)
	insertGlossary(t, db, "right", "show:1", "Celebrimbor", "凱勒布瑞博", "subtitle", 0)
	insertGlossary(t, db, "mine", "show:1", "林頓", "Lindon", "subtitle", 1) // confirmed: the user's call
	insertGlossary(t, db, "manual", "show:1", "盧恩", "Rhun", "manual", 0)   // not harvested
	insertGlossary(t, db, "ok", "show:1", "Vecna", "維克那", "subtitle", 0)   // already right

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&fixReversedHarvestedGlossaryTerms{}).Up(tx))
	require.NoError(t, tx.Commit())

	src, zh, ok := glossaryRow(t, db, "r1")
	require.True(t, ok)
	assert.Equal(t, [2]string{"The Neigh-sayer", "馬言者"}, [2]string{src, zh})
	src, zh, _ = glossaryRow(t, db, "r2")
	assert.Equal(t, [2]string{"Mayor Winddancer", "風舞市長"}, [2]string{src, zh})

	_, _, ok = glossaryRow(t, db, "dup")
	assert.False(t, ok, "the right-way-round row already exists — the backwards one is the duplicate")
	src, zh, _ = glossaryRow(t, db, "right")
	assert.Equal(t, [2]string{"Celebrimbor", "凱勒布瑞博"}, [2]string{src, zh})

	src, _, _ = glossaryRow(t, db, "mine")
	assert.Equal(t, "林頓", src, "a confirmed row is never touched")
	src, _, _ = glossaryRow(t, db, "manual")
	assert.Equal(t, "盧恩", src, "only harvested rows")
	src, _, _ = glossaryRow(t, db, "ok")
	assert.Equal(t, "Vecna", src)

	// Idempotent: a second run changes nothing.
	tx, err = db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&fixReversedHarvestedGlossaryTerms{}).Up(tx))
	require.NoError(t, tx.Commit())
	src, _, _ = glossaryRow(t, db, "r1")
	assert.Equal(t, "The Neigh-sayer", src)
}

// CR H2: two backwards rows naming the same English term must not trip the
// (scope, term_src NOCASE, language) unique index and abort startup; CR M5: an
// empty rendering is not turned into an empty term.
func TestMigration045_ConvergingRowsAndEmptyRendering(t *testing.T) {
	db := newFullyMigratedDB(t)
	insertGlossary(t, db, "a", "show:1", "馬言者", "The Neigh-sayer", "subtitle", 0)
	insertGlossary(t, db, "b", "show:1", "馬言語者", "the neigh-sayer", "subtitle", 0)
	insertGlossary(t, db, "empty", "show:1", "空的", "", "subtitle", 0)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&fixReversedHarvestedGlossaryTerms{}).Up(tx))
	require.NoError(t, tx.Commit())

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM show_glossary WHERE term_src = 'The Neigh-sayer' COLLATE NOCASE`).Scan(&n))
	assert.Equal(t, 1, n, "one turned around, the other deleted as its duplicate")
	src, _, ok := glossaryRow(t, db, "empty")
	require.True(t, ok)
	assert.Equal(t, "空的", src, "left alone")
}

func TestMigration045_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 45 {
			found = true
			assert.Equal(t, "fix_reversed_harvested_glossary_terms", m.Name())
		}
	}
	assert.True(t, found)
}
