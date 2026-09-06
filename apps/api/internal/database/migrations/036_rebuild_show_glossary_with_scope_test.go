package migrations

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// migrateUpTo applies every registered migration with Version() <= v — so a
// test can seed the 028-shaped table and then run 036 on real data.
func migrateUpTo(t *testing.T, v int64) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	runner, err := NewRunner(db)
	require.NoError(t, err)
	var subset []Migration
	for _, m := range GetAll() {
		if m.Version() <= v {
			subset = append(subset, m)
		}
	}
	require.NoError(t, runner.RegisterAll(subset))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func run036(t *testing.T, db *sql.DB) {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&rebuildShowGlossaryWithScope{migrationBase: NewMigrationBase(36, "rebuild_show_glossary_with_scope")}).Up(tx))
	require.NoError(t, tx.Commit())
}

func seedShow(t *testing.T, db *sql.DB, kind, id string, tmdbID any) {
	t.Helper()
	now := time.Now().UTC()
	switch kind {
	case "series":
		_, err := db.Exec(`INSERT INTO series (id, title, first_air_date, tmdb_id, created_at, updated_at) VALUES (?, ?, '2016-07-15', ?, ?, ?)`, id, "show "+id, tmdbID, now, now)
		require.NoError(t, err)
	case "movie":
		_, err := db.Exec(`INSERT INTO movies (id, title, release_date, tmdb_id, created_at, updated_at) VALUES (?, ?, '2021-10-22', ?, ?, ?)`, id, "film "+id, tmdbID, now, now)
		require.NoError(t, err)
	}
}

func seedOldTerm(t *testing.T, db *sql.DB, id, mediaID, termSrc, termZh, source string, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, term_src, term_zh, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, mediaID, termSrc, termZh, source, createdAt, createdAt)
	require.NoError(t, err)
}

func scopeOf(t *testing.T, db *sql.DB, id string) string {
	t.Helper()
	var scope string
	require.NoError(t, db.QueryRow(`SELECT scope FROM show_glossary WHERE id = ?`, id).Scan(&scope))
	return scope
}

// AC #1 / AC #5(a): the backfill splits rows by whether their show matched TMDb.
func TestMigration036_BackfillSplitsTMDbAndLocal(t *testing.T) {
	db := migrateUpTo(t, 35)
	seedShow(t, db, "series", "s-matched", int64(66732))
	seedShow(t, db, "series", "s-unmatched", nil)
	seedShow(t, db, "movie", "m-matched", int64(27205))
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	seedOldTerm(t, db, "g1", "s-matched", "Vecna", "維克那", "subtitle", base)
	seedOldTerm(t, db, "g2", "s-unmatched", "Kylo", "凱羅", "manual", base)
	seedOldTerm(t, db, "g3", "m-matched", "Cobb", "柯布", "metadata", base)
	seedOldTerm(t, db, "g4", "ghost-id", "Nobody", "無名", "manual", base)

	run036(t, db)

	assert.Equal(t, "tmdb:tv:66732", scopeOf(t, db, "g1"))
	assert.Equal(t, "local:s-unmatched", scopeOf(t, db, "g2"), "a series with no TMDb id keeps a local drawer")
	assert.Equal(t, "tmdb:movie:27205", scopeOf(t, db, "g3"))
	assert.Equal(t, "local:ghost-id", scopeOf(t, db, "g4"), "an id nothing knows still keeps its terms")

	// media_id survives as the audit column.
	var mediaID string
	require.NoError(t, db.QueryRow(`SELECT media_id FROM show_glossary WHERE id = 'g1'`).Scan(&mediaID))
	assert.Equal(t, "s-matched", mediaID)
}

// AC #5(a): NOCASE + trim collapse duplicates, the EARLIEST row wins.
func TestMigration036_CollapsesCaseAndWhitespaceDuplicates_KeepingTheEarliest(t *testing.T) {
	db := migrateUpTo(t, 35)
	seedShow(t, db, "series", "s1", int64(66732))
	seedOldTerm(t, db, "old", "s1", "Demogorgon", "魔王獸", "subtitle", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	seedOldTerm(t, db, "newer", "s1", "demogorgon", "德摩高根", "manual", time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC))
	seedOldTerm(t, db, "padded", "s1", "  Demogorgon ", "有空白", "manual", time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC))
	seedOldTerm(t, db, "other-show", "s2", "demogorgon", "別劇", "manual", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))

	run036(t, db)

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM show_glossary WHERE scope = 'tmdb:tv:66732'`).Scan(&n))
	assert.Equal(t, 1, n, "three spellings of one term collapse to one row")
	var id, termSrc, termZh string
	require.NoError(t, db.QueryRow(`SELECT id, term_src, term_zh FROM show_glossary WHERE scope = 'tmdb:tv:66732'`).Scan(&id, &termSrc, &termZh))
	assert.Equal(t, "old", id, "the earliest row is the survivor")
	assert.Equal(t, "Demogorgon", termSrc)
	assert.Equal(t, "魔王獸", termZh)

	var other int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM show_glossary WHERE scope = 'local:s2'`).Scan(&other))
	assert.Equal(t, 1, other, "the same term in ANOTHER show is untouched")
}

// AC #1: the source CHECK is gone — all five values (and, at the SQL level,
// anything) can be written; the closed set now lives in models.Validate.
func TestMigration036_DropsSourceCheck_AndTrimsTermSrc(t *testing.T) {
	db := migrateUpTo(t, 36)
	for _, src := range []string{"subtitle", "metadata", "manual", "official_subtitle", "community"} {
		_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, scope, term_src, term_zh, source) VALUES (?, 'm', 'local:m', ?, 'x', ?)`, "id-"+src, "t-"+src, src)
		require.NoError(t, err, src)
	}

	// The unique index is NOCASE on term_src.
	_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, scope, term_src, term_zh) VALUES ('u1', 'm', 'local:m', 'Vecna', 'a')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO show_glossary (id, media_id, scope, term_src, term_zh) VALUES ('u2', 'm', 'local:m', 'VECNA', 'b')`)
	require.Error(t, err, "same drawer, same term in another case must collide")
	_, err = db.Exec(`INSERT INTO show_glossary (id, media_id, scope, term_src, term_zh) VALUES ('u3', 'm', 'tmdb:tv:1', 'VECNA', 'b')`)
	require.NoError(t, err, "a different drawer is a different key")
}

func TestMigration036_UpIsIdempotent(t *testing.T) {
	db := migrateUpTo(t, 36)
	run036(t, db)
	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'show_glossary%'`).Scan(&n))
	assert.Equal(t, 1, n, "no leftover _v2 table after a second Up")
}

func TestMigration036_DownRestoresThe028Shape(t *testing.T) {
	db := migrateUpTo(t, 36)
	_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, scope, term_src, term_zh, source) VALUES ('c1', 'm', 'tmdb:tv:1', 'Vecna', 'a', 'community')`)
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&rebuildShowGlossaryWithScope{migrationBase: NewMigrationBase(36, "rebuild_show_glossary_with_scope")}).Down(tx))
	require.NoError(t, tx.Commit())

	_, err = db.Exec(`SELECT scope FROM show_glossary`)
	require.Error(t, err, "scope column is gone")
	var source string
	require.NoError(t, db.QueryRow(`SELECT source FROM show_glossary WHERE id = 'c1'`).Scan(&source))
	assert.Equal(t, "manual", source, "a source the 028 CHECK cannot express is folded to manual")
	_, err = db.Exec(`INSERT INTO show_glossary (id, media_id, term_src, term_zh, source) VALUES ('c2', 'm', 'x', 'y', 'community')`)
	require.Error(t, err, "the CHECK is back")
}

func TestMigration036_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 36 {
			found = true
			assert.Equal(t, "rebuild_show_glossary_with_scope", m.Name())
		}
	}
	assert.True(t, found, "migration 036 must be registered")
}
