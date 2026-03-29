package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func setupLibraryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Enable WAL mode and foreign keys
	_, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;")
	require.NoError(t, err)

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE media_libraries (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			content_type TEXT NOT NULL CHECK(content_type IN ('movie', 'series')),
			auto_detect INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE media_library_paths (
			id TEXT PRIMARY KEY,
			library_id TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'unknown',
			last_checked_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (library_id) REFERENCES media_libraries(id) ON DELETE CASCADE
		);
		CREATE TABLE movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			release_date TEXT NOT NULL DEFAULT '',
			genres TEXT NOT NULL DEFAULT '[]',
			library_id TEXT,
			is_removed INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			first_air_date TEXT NOT NULL DEFAULT '',
			genres TEXT NOT NULL DEFAULT '[]',
			library_id TEXT,
			is_removed INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })
	return db
}

func TestMediaLibraryRepository_Create(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{
		Name:        "我的電影",
		ContentType: models.ContentTypeMovie,
	}

	err := repo.Create(ctx, lib)
	require.NoError(t, err)
	assert.NotEmpty(t, lib.ID, "should auto-generate UUID")
	assert.False(t, lib.CreatedAt.IsZero())
}

func TestMediaLibraryRepository_Create_NilError(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, nil)
	assert.Error(t, err)
}

func TestMediaLibraryRepository_GetByID(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))

	found, err := repo.GetByID(ctx, lib.ID)
	require.NoError(t, err)
	assert.Equal(t, lib.ID, found.ID)
	assert.Equal(t, "Movies", found.Name)
	assert.Equal(t, models.ContentTypeMovie, found.ContentType)
}

func TestMediaLibraryRepository_GetByID_NotFound(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrLibraryNotFound))
}

func TestMediaLibraryRepository_GetAll(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie, SortOrder: 1}))
	require.NoError(t, repo.Create(ctx, &models.MediaLibrary{Name: "TV", ContentType: models.ContentTypeSeries, SortOrder: 0}))

	libs, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, libs, 2)
	assert.Equal(t, "TV", libs[0].Name, "should be sorted by sort_order")
}

func TestMediaLibraryRepository_Update(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))

	lib.Name = "My Movies"
	lib.ContentType = models.ContentTypeSeries
	err := repo.Update(ctx, lib)
	require.NoError(t, err)

	found, _ := repo.GetByID(ctx, lib.ID)
	assert.Equal(t, "My Movies", found.Name)
	assert.Equal(t, models.ContentTypeSeries, found.ContentType)
}

func TestMediaLibraryRepository_Update_NotFound(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{ID: "nonexistent", Name: "X", ContentType: models.ContentTypeMovie}
	err := repo.Update(ctx, lib)
	assert.True(t, errors.Is(err, ErrLibraryNotFound))
}

func TestMediaLibraryRepository_Delete(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))

	err := repo.Delete(ctx, lib.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, lib.ID)
	assert.True(t, errors.Is(err, ErrLibraryNotFound))
}

func TestMediaLibraryRepository_Delete_CascadesPaths(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))

	path := &models.MediaLibraryPath{LibraryID: lib.ID, Path: "/media/movies", Status: models.PathStatusUnknown}
	require.NoError(t, repo.AddPath(ctx, path))

	// Delete library should cascade to paths
	require.NoError(t, repo.Delete(ctx, lib.ID))

	paths, err := repo.GetPathsByLibraryID(ctx, lib.ID)
	require.NoError(t, err)
	assert.Empty(t, paths, "paths should be deleted via CASCADE")
}

func TestMediaLibraryRepository_AddPath(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))

	path := &models.MediaLibraryPath{LibraryID: lib.ID, Path: "/media/movies", Status: models.PathStatusAccessible}
	err := repo.AddPath(ctx, path)
	require.NoError(t, err)
	assert.NotEmpty(t, path.ID)
}

func TestMediaLibraryRepository_AddPath_DuplicateError(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))

	path1 := &models.MediaLibraryPath{LibraryID: lib.ID, Path: "/media/movies", Status: models.PathStatusUnknown}
	require.NoError(t, repo.AddPath(ctx, path1))

	path2 := &models.MediaLibraryPath{LibraryID: lib.ID, Path: "/media/movies", Status: models.PathStatusUnknown}
	err := repo.AddPath(ctx, path2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrLibraryPathDuplicate))
}

func TestMediaLibraryRepository_RemovePath(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))

	path := &models.MediaLibraryPath{LibraryID: lib.ID, Path: "/media/movies", Status: models.PathStatusUnknown}
	require.NoError(t, repo.AddPath(ctx, path))

	err := repo.RemovePath(ctx, path.ID)
	require.NoError(t, err)

	paths, _ := repo.GetPathsByLibraryID(ctx, lib.ID)
	assert.Empty(t, paths)
}

func TestMediaLibraryRepository_RemovePath_NotFound(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	err := repo.RemovePath(ctx, "nonexistent")
	assert.True(t, errors.Is(err, ErrLibraryPathNotFound))
}

func TestMediaLibraryRepository_UpdatePathStatus(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))

	path := &models.MediaLibraryPath{LibraryID: lib.ID, Path: "/media/movies", Status: models.PathStatusUnknown}
	require.NoError(t, repo.AddPath(ctx, path))

	err := repo.UpdatePathStatus(ctx, path.ID, models.PathStatusAccessible)
	require.NoError(t, err)

	paths, _ := repo.GetPathsByLibraryID(ctx, lib.ID)
	assert.Equal(t, models.PathStatusAccessible, paths[0].Status)
	assert.NotNil(t, paths[0].LastCheckedAt)
}

func TestMediaLibraryRepository_GetAllWithPathsAndCounts(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	require.NoError(t, repo.Create(ctx, lib))
	require.NoError(t, repo.AddPath(ctx, &models.MediaLibraryPath{LibraryID: lib.ID, Path: "/media/movies"}))

	// Insert a movie with this library_id
	_, err := db.ExecContext(ctx, `INSERT INTO movies (id, title, library_id) VALUES ('m1', 'Test Movie', ?)`, lib.ID)
	require.NoError(t, err)

	results, err := repo.GetAllWithPathsAndCounts(ctx)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 1, results[0].MediaCount)
	assert.Len(t, results[0].Paths, 1)
}

func TestMediaLibraryRepository_Delete_NotFound(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	err := repo.Delete(context.Background(), "nonexistent")
	assert.True(t, errors.Is(err, ErrLibraryNotFound))
}

func TestMediaLibraryRepository_Update_Nil(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	err := repo.Update(context.Background(), nil)
	assert.Error(t, err)
}

func TestMediaLibraryRepository_AddPath_Nil(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	err := repo.AddPath(context.Background(), nil)
	assert.Error(t, err)
}

func TestMediaLibraryRepository_UpdatePathStatus_NotFound(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	err := repo.UpdatePathStatus(context.Background(), "nonexistent", models.PathStatusAccessible)
	assert.True(t, errors.Is(err, ErrLibraryPathNotFound))
}

func TestMediaLibraryRepository_GetAllWithPathsAndCounts_Series(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib := &models.MediaLibrary{Name: "TV", ContentType: models.ContentTypeSeries}
	require.NoError(t, repo.Create(ctx, lib))
	require.NoError(t, repo.AddPath(ctx, &models.MediaLibraryPath{LibraryID: lib.ID, Path: "/media/tv"}))

	_, err := db.ExecContext(ctx, `INSERT INTO series (id, title, library_id) VALUES ('s1', 'Test Show', ?)`, lib.ID)
	require.NoError(t, err)
	// Insert a removed series — should NOT be counted
	_, err = db.ExecContext(ctx, `INSERT INTO series (id, title, library_id, is_removed) VALUES ('s2', 'Removed', ?, 1)`, lib.ID)
	require.NoError(t, err)

	results, err := repo.GetAllWithPathsAndCounts(ctx)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 1, results[0].MediaCount, "should exclude is_removed=1")
}

func TestMediaLibraryRepository_GetAll_Empty(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	libs, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, libs)
}

func TestMediaLibraryRepository_GetAllPaths(t *testing.T) {
	db := setupLibraryTestDB(t)
	repo := NewMediaLibraryRepository(db)
	ctx := context.Background()

	lib1 := &models.MediaLibrary{Name: "Movies", ContentType: models.ContentTypeMovie}
	lib2 := &models.MediaLibrary{Name: "TV", ContentType: models.ContentTypeSeries}
	require.NoError(t, repo.Create(ctx, lib1))
	require.NoError(t, repo.Create(ctx, lib2))
	require.NoError(t, repo.AddPath(ctx, &models.MediaLibraryPath{LibraryID: lib1.ID, Path: "/movies"}))
	require.NoError(t, repo.AddPath(ctx, &models.MediaLibraryPath{LibraryID: lib2.ID, Path: "/tv"}))

	paths, err := repo.GetAllPaths(ctx)
	require.NoError(t, err)
	assert.Len(t, paths, 2)
}
