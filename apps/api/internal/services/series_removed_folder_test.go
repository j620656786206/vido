package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/testutil"
)

// ─── disc-2026-10-series-wrong-version-c ────────────────────────────────────

// main wires the real repository; pin that it satisfies the narrow interfaces
// the scanner and the ingest assert, or the passes would silently never run.
var (
	_ seriesRemovalRepo = (*repository.SeriesRepository)(nil)
	_ seriesRestorer    = (*repository.SeriesRepository)(nil)
)

type fakeSeriesRemovalRepo struct {
	*testutil.MockSeriesRepository
	refs   []repository.SeriesFolderRef
	marked []string
	// back: ids whose episode got a file again while the scan stat-ed.
	back map[string]bool
}

func (f *fakeSeriesRemovalRepo) FindFoldersForRemovalCheck(context.Context) ([]repository.SeriesFolderRef, error) {
	return f.refs, nil
}

func (f *fakeSeriesRemovalRepo) MarkRemovedIfNoFiles(_ context.Context, id string) (bool, error) {
	if f.back[id] {
		return false, nil
	}
	f.marked = append(f.marked, id)
	return true, nil
}

func scannerWithSeriesFolders(t *testing.T, refs []repository.SeriesFolderRef) (*ScannerService, *fakeSeriesRemovalRepo) {
	t.Helper()
	svc, _, _ := setupScannerService(t, nil)
	fake := &fakeSeriesRemovalRepo{MockSeriesRepository: svc.seriesRepo.(*testutil.MockSeriesRepository), refs: refs}
	svc.seriesRepo = fake
	return svc, fake
}

func TestScanner_SeriesFolderGone_IsRemoved(t *testing.T) {
	tv := t.TempDir()
	kept := filepath.Join(tv, "將夜")
	require.NoError(t, os.Mkdir(kept, 0o755)) // still there, even with no video in it
	svc, repo := scannerWithSeriesFolders(t, []repository.SeriesFolderRef{
		{ID: "old", FilePath: filepath.Join(tv, "將夜 (2026)")},
		{ID: "empty-but-there", FilePath: kept},
	})

	n, err := svc.detectRemovedSeries(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"old"}, repo.marked)
}

func TestScanner_SeriesFolderUnderUntrustedRootOrLibrary_IsKept(t *testing.T) {
	tv := filepath.Join(t.TempDir(), "tv")
	svc, repo := scannerWithSeriesFolders(t, []repository.SeriesFolderRef{
		{ID: "root", FilePath: filepath.Join(tv, "將夜")},
		{ID: "lib", FilePath: filepath.Join(t.TempDir(), "elsewhere", "Show"), LibraryID: "lib-tv"},
	})

	n, err := svc.detectRemovedSeries(context.Background(), []string{tv}, map[string]bool{"lib-tv": true})
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, repo.marked, "an unreachable mount is not a deleted show")
}

func TestScanner_SeriesWhoseEpisodeCameBackMidScan_IsNotCounted(t *testing.T) {
	svc, repo := scannerWithSeriesFolders(t, []repository.SeriesFolderRef{{ID: "old", FilePath: filepath.Join(t.TempDir(), "gone")}})
	repo.back = map[string]bool{"old": true}

	n, err := svc.detectRemovedSeries(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, repo.marked)
}

func TestScanner_SeriesRepoWithoutRemovalMethods_IsANoOp(t *testing.T) {
	svc, _, _ := setupScannerService(t, nil)
	n, err := svc.detectRemovedSeries(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Zero(t, n)
}

// AC #4: the folder holds episodes again — the same row comes back.
func TestUpsertSeries_RestoresARemovedShowWhoseFolderIsBack(t *testing.T) {
	db := setupTestDB(t)
	series := repository.NewSeriesRepository(db)
	ctx := context.Background()
	require.NoError(t, series.Create(ctx, &models.Series{ID: "old", Title: "將夜", FilePath: models.NewNullString("/tv/將夜 (2026)")}))
	_, err := db.Exec(`UPDATE series SET is_removed = 1 WHERE id = 'old'`)
	require.NoError(t, err)

	ingest := NewMediaIngestService(series, nil, nil, nil)
	id, err := ingest.UpsertSeries(ctx, SeriesInput{SeriesDir: "/tv/將夜 (2026)", Title: "將夜"})
	require.NoError(t, err)
	assert.Equal(t, "old", id, "the same row, not a new one")

	got, err := series.FindByID(ctx, "old")
	require.NoError(t, err)
	assert.False(t, got.IsRemoved)
}
