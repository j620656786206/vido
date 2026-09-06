package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// newMigratedDBUpTo runs every registered migration whose version is <= max,
// so a test can stage pre-migration rows before running the one under test.
func newMigratedDBUpTo(t *testing.T, max int) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	runner, err := NewRunner(db)
	require.NoError(t, err)
	for _, m := range GetAll() {
		if m.Version() <= int64(max) {
			require.NoError(t, runner.Register(m))
		}
	}
	require.NoError(t, runner.Up(t.Context()))
	return db
}

func TestMigration037_MarksTableBackfilledAndForeignIDsCleared(t *testing.T) {
	db := newMigratedDBUpTo(t, 36)
	now := time.Now().UTC()

	// Two scopes already seeded by PR #397, one harvested-only scope.
	for _, row := range [][3]string{
		{"tmdb:tv:1396", "Walter White", "metadata"},
		{"tmdb:tv:1396", "Jesse", "metadata"},
		{"tmdb:movie:550", "Tyler", "metadata"},
		{"tmdb:movie:27205", "Cobb", "subtitle"},
		{"local:abc", "Foo", "metadata"},
	} {
		_, err := db.Exec(`INSERT INTO show_glossary (id, media_id, scope, term_src, term_zh, source, created_at, updated_at)
			VALUES (?, 'm', ?, ?, 'x', ?, ?, ?)`, row[0]+row[1], row[0], row[1], row[2], now, now)
		require.NoError(t, err)
	}
	// A Douban-matched movie and a TMDb-matched one; a Wikipedia-matched series.
	_, err := db.Exec(`INSERT INTO movies (id, title, release_date, tmdb_id, metadata_source, created_at, updated_at) VALUES
		('m-douban', '讓子彈飛', '2010-12-16', 3742360, 'douban', ?, ?),
		('m-tmdb', 'Fight Club', '1999-10-15', 550, 'tmdb', ?, ?)`, now, now, now, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO series (id, title, first_air_date, tmdb_id, metadata_source, created_at, updated_at) VALUES
		('s-wiki', 'X', '2020-01-01', 12345, 'wikipedia', ?, ?),
		('s-tmdb', 'Breaking Bad', '2008-01-20', 1396, 'tmdb', ?, ?)`, now, now, now, now)
	require.NoError(t, err)

	runner, err := NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(GetAll()))
	require.NoError(t, runner.Up(t.Context()))

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM glossary_seed_marks`).Scan(&n))
	assert.Equal(t, 2, n, "only shared scopes with metadata terms get a mark")
	var seeded int
	require.NoError(t, db.QueryRow(`SELECT seeded FROM glossary_seed_marks WHERE scope = 'tmdb:tv:1396'`).Scan(&seeded))
	assert.Equal(t, 2, seeded)

	var id sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT tmdb_id FROM movies WHERE id = 'm-douban'`).Scan(&id))
	assert.False(t, id.Valid, "a Douban subject id is not a TMDb id")
	require.NoError(t, db.QueryRow(`SELECT tmdb_id FROM movies WHERE id = 'm-tmdb'`).Scan(&id))
	assert.EqualValues(t, 550, id.Int64)
	require.NoError(t, db.QueryRow(`SELECT tmdb_id FROM series WHERE id = 's-wiki'`).Scan(&id))
	assert.False(t, id.Valid)
	require.NoError(t, db.QueryRow(`SELECT tmdb_id FROM series WHERE id = 's-tmdb'`).Scan(&id))
	assert.EqualValues(t, 1396, id.Int64)

	// Idempotent.
	require.NoError(t, runner.Up(t.Context()))
}
