package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// bugfix-scan-mount-drop-hides-movies: a NAS folder that is briefly
// unreachable must not wipe the library, and files that come back must come
// back into the library.

func resolved(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	require.NoError(t, err)
	abs, err := filepath.Abs(r)
	require.NoError(t, err)
	return abs
}

// AC #2: the whole folder is unreachable this scan.
func TestScanner_UnreachableRoot_MarksNothingRemoved(t *testing.T) {
	root := filepath.Join(t.TempDir(), "movies")
	require.NoError(t, os.Mkdir(root, 0o755))
	moviePath := filepath.Join(resolved(t, root), "Dune.mkv")
	require.NoError(t, os.RemoveAll(root)) // the mount dropped

	svc, movieRepo, _ := setupScannerService(t, []string{root})
	movieRepo.ExpectedCalls = filterCalls(movieRepo.ExpectedCalls, "FindAllWithFilePath")
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return([]models.Movie{
		{ID: "m1", Title: "Dune", FilePath: models.NewNullString(moviePath)},
	}, nil)

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, result.FilesRemoved)
	movieRepo.AssertNotCalled(t, "MarkRemoved", mock.Anything, mock.Anything)
}

// AC #3: the folder exists but is empty (Docker's view of an unmounted share)
// while the library remembers movies under it.
func TestScanner_EmptyMountPoint_MarksNothingRemoved(t *testing.T) {
	root := filepath.Join(t.TempDir(), "movies")
	require.NoError(t, os.Mkdir(root, 0o755))
	moviePath := filepath.Join(resolved(t, root), "Dune.mkv")

	svc, movieRepo, _ := setupScannerService(t, []string{root})
	movieRepo.ExpectedCalls = filterCalls(movieRepo.ExpectedCalls, "FindAllWithFilePath")
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return([]models.Movie{
		{ID: "m1", Title: "Dune", FilePath: models.NewNullString(moviePath)},
	}, nil)

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, result.FilesRemoved)
	movieRepo.AssertNotCalled(t, "MarkRemoved", mock.Anything, mock.Anything)
}

// AC #4: a healthy folder where ONE file was deleted still marks that one.
func TestScanner_OneFileDeletedInHealthyRoot_IsStillMarked(t *testing.T) {
	root := t.TempDir()
	createVideoFiles(t, root, []string{"Arrival.mkv"})
	gonePath := filepath.Join(resolved(t, root), "Dune.mkv")

	svc, movieRepo, _ := setupScannerService(t, []string{root})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)
	movieRepo.ExpectedCalls = filterCalls(movieRepo.ExpectedCalls, "FindAllWithFilePath")
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return([]models.Movie{
		{ID: "m1", Title: "Dune", FilePath: models.NewNullString(gonePath)},
	}, nil)
	movieRepo.On("MarkRemoved", mock.Anything, "m1").Return(nil)

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, result.FilesRemoved)
	movieRepo.AssertCalled(t, "MarkRemoved", mock.Anything, "m1")
}

// AC #1: a movie marked removed whose file is back (unchanged) is restored —
// before this, the unchanged file was "skipped" and stayed hidden forever.
func TestScanner_RemovedMovieWhoseFileIsBack_IsRestored(t *testing.T) {
	root := t.TempDir()
	paths := createVideoFiles(t, root, []string{"Dune.mkv"})
	p := resolved(t, paths[0])
	info, err := os.Stat(p)
	require.NoError(t, err)

	svc, movieRepo, _ := setupScannerService(t, []string{root})
	movieRepo.On("FindByFilePath", mock.Anything, p).Return(&models.Movie{
		ID:        "m1",
		Title:     "Dune",
		FilePath:  models.NewNullString(p),
		FileSize:  models.NewNullInt64(info.Size()),
		UpdatedAt: info.ModTime().Add(1e9), // unchanged since the row was written
		IsRemoved: true,
	}, nil)
	movieRepo.On("RestoreRemoved", mock.Anything, "m1").Return(nil)

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	movieRepo.AssertCalled(t, "RestoreRemoved", mock.Anything, "m1")
	assert.Equal(t, 1, result.FilesUpdated)
	assert.Equal(t, 0, result.FilesSkipped)
}

// Review M1: nested folders (/media and /media/movies both configured) — the
// inner folder's videos were already claimed by the outer one, which must
// not make the inner folder look "empty" and shield a truly deleted film.
func TestScanner_NestedRoots_DeletedFileIsStillMarked(t *testing.T) {
	outer := t.TempDir()
	inner := filepath.Join(outer, "movies")
	createVideoFiles(t, inner, []string{"Arrival.mkv"})
	gonePath := filepath.Join(resolved(t, inner), "Dune.mkv")

	svc, movieRepo, _ := setupScannerService(t, []string{outer, inner})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)
	movieRepo.ExpectedCalls = filterCalls(movieRepo.ExpectedCalls, "FindAllWithFilePath")
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return([]models.Movie{
		{ID: "m1", Title: "Dune", FilePath: models.NewNullString(gonePath)},
	}, nil)
	movieRepo.On("MarkRemoved", mock.Anything, "m1").Return(nil)

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, result.FilesRemoved)
}

// Review M2: a folder whose videos are dead symlinks into an unreachable
// share — the stored paths are the TARGETS, so the path check cannot match;
// the library the movie belongs to does.
func TestScanner_DeadSymlinksIntoShare_MarkNothingRemoved(t *testing.T) {
	share := filepath.Join(t.TempDir(), "nas")
	require.NoError(t, os.Mkdir(share, 0o755))
	target := filepath.Join(resolved(t, share), "Dune.mkv")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	lib := t.TempDir()
	require.NoError(t, os.Symlink(target, filepath.Join(lib, "Dune.mkv")))
	require.NoError(t, os.RemoveAll(share)) // the share dropped; the link is dead

	svc, movieRepo, _ := setupScannerService(t, nil)
	svc.SetLibraryRepo(&fixedLibraries{libs: []models.MediaLibraryWithPaths{{
		MediaLibrary: models.MediaLibrary{ID: "lib-1", ContentType: models.ContentTypeMovie},
		Paths:        []models.MediaLibraryPath{{Path: lib}},
	}}})
	movieRepo.ExpectedCalls = filterCalls(movieRepo.ExpectedCalls, "FindAllWithFilePath")
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return([]models.Movie{
		{ID: "m1", Title: "Dune", FilePath: models.NewNullString(target), LibraryID: models.NewNullString("lib-1")},
	}, nil)

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, result.FilesRemoved)
	movieRepo.AssertNotCalled(t, "MarkRemoved", mock.Anything, mock.Anything)
}

// Restore + the file changed while it was away: restored AND re-parsed,
// counted once.
func TestScanner_RemovedMovieBackWithNewSize_IsRestoredAndReparsed(t *testing.T) {
	root := t.TempDir()
	paths := createVideoFiles(t, root, []string{"Dune.mkv"})
	p := resolved(t, paths[0])

	svc, movieRepo, _ := setupScannerService(t, []string{root})
	movieRepo.On("FindByFilePath", mock.Anything, p).Return(&models.Movie{
		ID:        "m1",
		Title:     "Dune",
		FilePath:  models.NewNullString(p),
		FileSize:  models.NewNullInt64(1), // differs from the file on disk
		IsRemoved: true,
	}, nil)
	movieRepo.On("RestoreRemoved", mock.Anything, "m1").Return(nil)
	movieRepo.On("UpdateScanFileInfo", mock.Anything, "m1", mock.AnythingOfType("int64"), models.ParseStatusPending).Return(nil)

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	movieRepo.AssertCalled(t, "RestoreRemoved", mock.Anything, "m1")
	movieRepo.AssertCalled(t, "UpdateScanFileInfo", mock.Anything, "m1", mock.AnythingOfType("int64"), models.ParseStatusPending)
	assert.Equal(t, 1, result.FilesUpdated)
}

// fixedLibraries serves a fixed library list to the scanner.
type fixedLibraries struct {
	stubLibraryRepo
	libs []models.MediaLibraryWithPaths
}

func (f *fixedLibraries) GetAllWithPathsAndCounts(context.Context) ([]models.MediaLibraryWithPaths, error) {
	return f.libs, nil
}
