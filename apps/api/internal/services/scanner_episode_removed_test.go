package services

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/testutil"
)

// disc-2026-10-episode-rows-outlive-deleted-files: a deleted episode file
// goes back to "no local file"; a folder the scan could not vouch for clears
// nothing (same protection as movies, bugfix-scan-mount-drop-hides-movies).

// main wires the real repository; pin that it satisfies the narrow interface
// the scanner asserts, or the pass would silently never run.
var _ episodeRemovalRepo = (*repository.EpisodeRepository)(nil)

type fakeEpisodeRemovalRepo struct {
	repository.EpisodeRepositoryInterface // unused methods panic if called
	mu                                    sync.Mutex
	refs                                  []repository.EpisodeFileRef
	cleared                               []string
	// moved: ids re-ingested at another path while the scan stat-ed — the
	// compare-and-set write finds nothing to clear.
	moved map[string]bool
}

func (f *fakeEpisodeRemovalRepo) FindFilesForRemovalCheck(context.Context) ([]repository.EpisodeFileRef, error) {
	return f.refs, nil
}

func (f *fakeEpisodeRemovalRepo) ClearMissingFile(_ context.Context, id, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.moved[id] {
		return false, nil
	}
	f.cleared = append(f.cleared, id)
	return true, nil
}

func (f *fakeEpisodeRemovalRepo) FindBySeriesID(context.Context, string) ([]models.Episode, error) {
	return nil, nil
}

func scannerWithEpisodes(t *testing.T, roots []string, refs []repository.EpisodeFileRef) (*ScannerService, *fakeEpisodeRemovalRepo) {
	t.Helper()
	svc, movieRepo, _ := setupScannerService(t, roots)
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Maybe().Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Maybe().Return(nil)
	series := svc.seriesRepo.(*testutil.MockSeriesRepository)
	series.On("List", mock.Anything, mock.Anything).Maybe().Return([]models.Series{}, (*repository.PaginationResult)(nil), nil)
	eps := &fakeEpisodeRemovalRepo{refs: refs}
	svc.SetEpisodeRepo(eps)
	return svc, eps
}

func TestScanner_DeletedEpisodeFileInHealthyRoot_IsCleared(t *testing.T) {
	root := t.TempDir()
	createVideoFiles(t, root, []string{"Arrival.mkv"}) // the folder is healthy
	still := filepath.Join(resolved(t, root), "See.S01E02.mkv")
	require.NoError(t, os.WriteFile(still, []byte("video"), 0o644))
	gone := filepath.Join(resolved(t, root), "See.S01E01.mkv")

	svc, eps := scannerWithEpisodes(t, []string{root}, []repository.EpisodeFileRef{
		{ID: "see-s01e01", FilePath: gone},
		{ID: "see-s01e02", FilePath: still},
	})

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"see-s01e01"}, eps.cleared, "only the episode whose file is gone")
	assert.Equal(t, 1, result.FilesRemoved)
}

func TestScanner_UnreachableRoot_ClearsNoEpisode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tv")
	require.NoError(t, os.Mkdir(root, 0o755))
	gone := filepath.Join(resolved(t, root), "See.S01E01.mkv")
	require.NoError(t, os.RemoveAll(root)) // the mount dropped

	svc, eps := scannerWithEpisodes(t, []string{root}, []repository.EpisodeFileRef{{ID: "see-s01e01", FilePath: gone}})

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Empty(t, eps.cleared)
	assert.Equal(t, 0, result.FilesRemoved)
}

func TestScanner_EmptyMountPoint_ClearsNoEpisode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tv")
	require.NoError(t, os.Mkdir(root, 0o755)) // Docker's view of an unmounted share
	gone := filepath.Join(resolved(t, root), "See", "S1", "See.S01E01.mkv")

	svc, eps := scannerWithEpisodes(t, []string{root}, []repository.EpisodeFileRef{{ID: "see-s01e01", FilePath: gone}})

	_, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Empty(t, eps.cleared)
}

func TestScanner_EpisodeInUntrustedLibrary_IsKept(t *testing.T) {
	svc, eps := scannerWithEpisodes(t, nil, []repository.EpisodeFileRef{
		{ID: "symlinked", FilePath: filepath.Join(t.TempDir(), "elsewhere", "E01.mkv"), LibraryID: "lib-tv"},
	})

	n, err := svc.detectRemovedEpisodeFiles(context.Background(), nil, map[string]bool{"lib-tv": true})
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, eps.cleared, "a folder that is itself a symlink stores TARGET paths — the library is what ties them to it")
}

func TestScanner_EpisodeRepoWithoutRemovalMethods_IsANoOp(t *testing.T) {
	svc, _, _ := setupScannerService(t, nil)
	n, err := svc.detectRemovedEpisodeFiles(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Zero(t, n)
}

// AC #3: an episode re-ingested at a new path mid-scan is not cleared or counted.
func TestScanner_EpisodeMovedMidScan_IsNotCounted(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "See.S01E01.mkv")
	svc, eps := scannerWithEpisodes(t, nil, []repository.EpisodeFileRef{{ID: "see-s01e01", FilePath: gone}})
	eps.moved = map[string]bool{"see-s01e01": true}

	n, err := svc.detectRemovedEpisodeFiles(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, eps.cleared)
}

// AC #7: the scan's removed count covers movies and episodes together.
func TestScanner_RemovedCountCoversMoviesAndEpisodes(t *testing.T) {
	root := t.TempDir()
	createVideoFiles(t, root, []string{"Arrival.mkv"})
	goneMovie := filepath.Join(resolved(t, root), "Dune.mkv")
	goneEpisode := filepath.Join(resolved(t, root), "See.S01E01.mkv")

	svc, eps := scannerWithEpisodes(t, []string{root}, []repository.EpisodeFileRef{{ID: "see-s01e01", FilePath: goneEpisode}})
	movieRepo := svc.movieRepo.(*testutil.MockMovieRepository)
	movieRepo.ExpectedCalls = filterCalls(movieRepo.ExpectedCalls, "FindAllWithFilePath")
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return([]models.Movie{
		{ID: "m1", Title: "Dune", FilePath: models.NewNullString(goneMovie)},
	}, nil)
	movieRepo.On("MarkRemoved", mock.Anything, "m1").Return(nil)

	result, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, result.FilesRemoved)
	assert.Equal(t, []string{"see-s01e01"}, eps.cleared)
}
