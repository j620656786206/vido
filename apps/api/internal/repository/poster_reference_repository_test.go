package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/database/migrations"
	_ "modernc.org/sqlite"
)

func setupPosterRefDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

// bugfix-poster-orphan-sweep AC #2: every row that points at an upload counts —
// soft-removed ones too (restoring them must bring the poster back).
func TestPosterReferenceRepository_ListUploadedPosterPaths(t *testing.T) {
	db := setupPosterRefDB(t)
	exec := func(q string, args ...any) {
		_, err := db.Exec(q, args...)
		require.NoError(t, err)
	}
	exec(`INSERT INTO movies (id, release_date, title, poster_path) VALUES ('m1','2020-01-01','A','/posters/m1.jpg?v=1')`)
	exec(`INSERT INTO movies (id, release_date, title, poster_path) VALUES ('m2','2020-01-01','B','/posters/m2.jpg')`)
	exec(`INSERT INTO movies (id, release_date, title, poster_path, is_removed) VALUES ('m3','2020-01-01','C','/posters/m3.jpg', 1)`)
	exec(`INSERT INTO movies (id, release_date, title, poster_path) VALUES ('m4','2020-01-01','D','/abc123.jpg')`)
	exec(`INSERT INTO movies (id, release_date, title, poster_path) VALUES ('m5','2020-01-01','E','https://img.douban.com/x.jpg')`)
	exec(`INSERT INTO movies (id, release_date, title) VALUES ('m6','2020-01-01','F')`)
	// A pasted absolute URL of this app's own poster route also points at an upload.
	exec(`INSERT INTO movies (id, release_date, title, poster_path) VALUES ('m7','2020-01-01','G','http://nas:8080/api/v1/posters/m2.jpg')`)
	exec(`INSERT INTO series (id, first_air_date, title, poster_path) VALUES ('s1','2020-01-01','S','/posters/s1.jpg?v=9')`)
	exec(`INSERT INTO series (id, first_air_date, title, poster_path, is_removed) VALUES ('s2','2020-01-01','T','/posters/s2.jpg', 1)`)

	got, err := NewPosterReferenceRepository(db).ListUploadedPosterPaths(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"/posters/m1.jpg?v=1", "/posters/m2.jpg", "/posters/m3.jpg",
		"http://nas:8080/api/v1/posters/m2.jpg",
		"/posters/s1.jpg?v=9", "/posters/s2.jpg",
	}, got)
}

func TestPosterReferenceRepository_QueryErrorIsReturned(t *testing.T) {
	db := setupPosterRefDB(t)
	require.NoError(t, db.Close())
	_, err := NewPosterReferenceRepository(db).ListUploadedPosterPaths(context.Background())
	assert.Error(t, err)
}
