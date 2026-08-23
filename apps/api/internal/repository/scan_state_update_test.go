package repository

// bugfix-wide-update-stale-copy-other-callers — the single-intent writers.
//
// Each test seeds a row, changes an UNRELATED column underneath through the
// subtitle pipeline's own narrow writer (the concurrent write the wide Update
// used to clobber), calls the narrow writer under test, and asserts that the
// unrelated write survived and only the intended columns moved.

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// createScanStateMovie seeds the row WITHOUT the concurrent subtitle write, so
// a test can load a copy first and let the write land afterwards (the real
// detectRemovedFiles sequence).
func createScanStateMovie(t *testing.T) (*MovieRepository, context.Context) {
	t.Helper()
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewMovieRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &models.Movie{
		ID:          "mv-scan",
		Title:       "原標題",
		FilePath:    models.NewNullString("/media/mv-scan.mkv"),
		FileSize:    models.NewNullInt64(100),
		ParseStatus: models.ParseStatusSuccess,
		PosterPath:  models.NewNullString("/posters/old.jpg"),
	}))
	return repo, ctx
}

// deliverSubtitle is the concurrent write every test must preserve.
func deliverSubtitle(t *testing.T, repo *MovieRepository, ctx context.Context) {
	t.Helper()
	require.NoError(t, repo.UpdateSubtitleGenerationStatus(ctx,
		"mv-scan", models.SubtitleStatusFound, "/media/mv-scan.zh-Hant.srt", "zh-Hant"))
}

func seedScanStateMovie(t *testing.T) (*MovieRepository, context.Context) {
	t.Helper()
	repo, ctx := createScanStateMovie(t)
	deliverSubtitle(t, repo, ctx)
	return repo, ctx
}

func assertMovieSubtitleSurvived(t *testing.T, repo *MovieRepository, ctx context.Context) *models.Movie {
	t.Helper()
	got, err := repo.FindByID(ctx, "mv-scan")
	require.NoError(t, err)
	assert.Equal(t, models.SubtitleStatusFound, got.SubtitleStatus, "the concurrent subtitle write must survive a narrow write")
	assert.Equal(t, "/media/mv-scan.zh-Hant.srt", got.SubtitlePath.String)
	return got
}

func TestMovieUpdateScanFileInfo_WritesOnlyFileSizeAndParseStatus(t *testing.T) {
	repo, ctx := seedScanStateMovie(t)

	require.NoError(t, repo.UpdateScanFileInfo(ctx, "mv-scan", 4096, models.ParseStatusPending))

	got := assertMovieSubtitleSurvived(t, repo, ctx)
	assert.Equal(t, int64(4096), got.FileSize.Int64)
	assert.Equal(t, models.ParseStatusPending, got.ParseStatus)
	assert.Equal(t, "原標題", got.Title)
	assert.Equal(t, "/posters/old.jpg", got.PosterPath.String)
	assert.False(t, got.IsRemoved)
}

func TestMovieMarkRemoved_WritesOnlyIsRemoved(t *testing.T) {
	repo, ctx := seedScanStateMovie(t)

	require.NoError(t, repo.MarkRemoved(ctx, "mv-scan"))

	got := assertMovieSubtitleSurvived(t, repo, ctx)
	assert.True(t, got.IsRemoved)
	assert.Equal(t, int64(100), got.FileSize.Int64)
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
}

func TestMovieUpdateParseStatus_WritesOnlyParseStatus(t *testing.T) {
	repo, ctx := seedScanStateMovie(t)

	require.NoError(t, repo.UpdateParseStatus(ctx, "mv-scan", models.ParseStatusPending))

	got := assertMovieSubtitleSurvived(t, repo, ctx)
	assert.Equal(t, models.ParseStatusPending, got.ParseStatus)
	assert.Equal(t, int64(100), got.FileSize.Int64)
	assert.False(t, got.IsRemoved)
}

func TestMovieUpdatePosterPath_WritesOnlyPosterPath(t *testing.T) {
	repo, ctx := seedScanStateMovie(t)

	require.NoError(t, repo.UpdatePosterPath(ctx, "mv-scan", "/posters/new.jpg"))

	got := assertMovieSubtitleSurvived(t, repo, ctx)
	assert.Equal(t, "/posters/new.jpg", got.PosterPath.String)
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
}

// AC #4: a narrow write to a vanished row is an error, never silent.
func TestMovieNarrowWriters_MissingRowIsErrNoRows(t *testing.T) {
	repo, ctx := seedScanStateMovie(t)
	for name, call := range map[string]func() error{
		"UpdateScanFileInfo": func() error { return repo.UpdateScanFileInfo(ctx, "nope", 1, models.ParseStatusPending) },
		"MarkRemoved":        func() error { return repo.MarkRemoved(ctx, "nope") },
		"UpdateParseStatus":  func() error { return repo.UpdateParseStatus(ctx, "nope", models.ParseStatusPending) },
		"UpdatePosterPath":   func() error { return repo.UpdatePosterPath(ctx, "nope", "/x.jpg") },
	} {
		err := call()
		assert.True(t, errors.Is(err, sql.ErrNoRows), "%s: got %v", name, err)
	}
	assert.Error(t, repo.MarkRemoved(ctx, ""), "empty id is refused before touching the DB")
}

// ─── series ────────────────────────────────────────────────────────────────

func seedScanStateSeries(t *testing.T) (*SeriesRepository, context.Context) {
	t.Helper()
	db := setupSeriesTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewSeriesRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &models.Series{
		ID:          "sr-scan",
		Title:       "原標題",
		FileSize:    models.NewNullInt64(100),
		ParseStatus: models.ParseStatusPending,
		PosterPath:  models.NewNullString("/posters/old.jpg"),
	}))
	// The concurrent write every test must preserve: enrichment finishing a
	// parse on the same row (what aggregateSeriesFileSizes used to clobber).
	enriched := &models.Series{ID: "sr-scan", Title: "鬼滅之刃", ParseStatus: models.ParseStatusSuccess,
		PosterPath: models.NewNullString("/posters/enriched.jpg"), MetadataSource: models.NewNullString("tmdb")}
	require.NoError(t, repo.UpdateEnrichedMetadata(ctx, enriched))
	return repo, ctx
}

func assertSeriesEnrichmentSurvived(t *testing.T, repo *SeriesRepository, ctx context.Context) *models.Series {
	t.Helper()
	got, err := repo.FindByID(ctx, "sr-scan")
	require.NoError(t, err)
	assert.Equal(t, "鬼滅之刃", got.Title, "the concurrent enrichment write must survive a narrow write")
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
	return got
}

func TestSeriesUpdateFileSize_WritesOnlyFileSize(t *testing.T) {
	repo, ctx := seedScanStateSeries(t)

	require.NoError(t, repo.UpdateFileSize(ctx, "sr-scan", 9_000_000))

	got := assertSeriesEnrichmentSurvived(t, repo, ctx)
	assert.Equal(t, int64(9_000_000), got.FileSize.Int64)
	assert.Equal(t, "/posters/enriched.jpg", got.PosterPath.String)
}

func TestSeriesUpdateParseStatus_WritesOnlyParseStatus(t *testing.T) {
	repo, ctx := seedScanStateSeries(t)

	require.NoError(t, repo.UpdateParseStatus(ctx, "sr-scan", models.ParseStatusPending))

	got, err := repo.FindByID(ctx, "sr-scan")
	require.NoError(t, err)
	assert.Equal(t, models.ParseStatusPending, got.ParseStatus)
	assert.Equal(t, "鬼滅之刃", got.Title)
	assert.Equal(t, int64(100), got.FileSize.Int64)
}

func TestSeriesUpdatePosterPath_WritesOnlyPosterPath(t *testing.T) {
	repo, ctx := seedScanStateSeries(t)

	require.NoError(t, repo.UpdatePosterPath(ctx, "sr-scan", "/posters/new.jpg"))

	got := assertSeriesEnrichmentSurvived(t, repo, ctx)
	assert.Equal(t, "/posters/new.jpg", got.PosterPath.String)
	assert.Equal(t, int64(100), got.FileSize.Int64)
}

func TestSeriesNarrowWriters_MissingRowIsErrNoRows(t *testing.T) {
	repo, ctx := seedScanStateSeries(t)
	for name, call := range map[string]func() error{
		"UpdateFileSize":    func() error { return repo.UpdateFileSize(ctx, "nope", 1) },
		"UpdateParseStatus": func() error { return repo.UpdateParseStatus(ctx, "nope", models.ParseStatusPending) },
		"UpdatePosterPath":  func() error { return repo.UpdatePosterPath(ctx, "nope", "/x.jpg") },
	} {
		err := call()
		assert.True(t, errors.Is(err, sql.ErrNoRows), "%s: got %v", name, err)
	}
}

// ─── characterization: the scanner's two long-window passes ────────────────
//
// These replay the exact sequence of detectRemovedFiles / aggregateSeriesFileSizes:
// load everything, let another writer land, then write. The first half of each
// shows the wide Update losing the concurrent write (so nobody has to re-derive
// why the narrow writer exists); the second half is the pin.

func TestDetectRemovedFilesShape_WideUpdateLosesConcurrentSubtitleWrite(t *testing.T) {
	repo, ctx := createScanStateMovie(t)
	// detectRemovedFiles loads EVERY movie up front…
	all, err := repo.FindAllWithFilePath(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
	stale := &all[0]
	require.Equal(t, models.SubtitleStatusNotSearched, stale.SubtitleStatus, "fixture: nothing delivered yet at load time")

	// …and while it os.Stat's the library, the free lane delivers a subtitle.
	deliverSubtitle(t, repo, ctx)

	stale.IsRemoved = true
	require.NoError(t, repo.Update(ctx, stale)) // the old code path

	got, err := repo.FindByID(ctx, "mv-scan")
	require.NoError(t, err)
	assert.True(t, got.IsRemoved)
	assert.Equal(t, models.SubtitleStatusNotSearched, got.SubtitleStatus,
		"characterisation: the wide Update reverted the subtitle the free lane delivered — if this starts failing, re-check whether MarkRemoved is still needed")
}

func TestDetectRemovedFilesShape_MarkRemovedKeepsConcurrentSubtitleWrite(t *testing.T) {
	repo, ctx := createScanStateMovie(t)
	all, err := repo.FindAllWithFilePath(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
	deliverSubtitle(t, repo, ctx) // lands after the load, as in production

	require.NoError(t, repo.MarkRemoved(ctx, all[0].ID)) // the new code path

	got := assertMovieSubtitleSurvived(t, repo, ctx)
	assert.True(t, got.IsRemoved)
}

func TestAggregateSeriesFileSizesShape_WideUpdateLosesConcurrentEnrichment(t *testing.T) {
	db := setupSeriesTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewSeriesRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &models.Series{ID: "sr-agg", Title: "未解析", ParseStatus: models.ParseStatusPending}))

	// aggregateSeriesFileSizes loads EVERY series up front…
	all, _, err := repo.List(ctx, ListParams{Page: 1, PageSize: 10000})
	require.NoError(t, err)
	require.Len(t, all, 1)
	stale := &all[0]

	// …and while it stats episode files, enrichment finishes this row.
	require.NoError(t, repo.UpdateEnrichedMetadata(ctx, &models.Series{
		ID: "sr-agg", Title: "鬼滅之刃", ParseStatus: models.ParseStatusSuccess,
		PosterPath: models.NewNullString("/posters/enriched.jpg"), MetadataSource: models.NewNullString("tmdb"),
	}))

	stale.FileSize = models.NewNullInt64(9_000_000)
	require.NoError(t, repo.Update(ctx, stale)) // the old code path

	got, err := repo.FindByID(ctx, "sr-agg")
	require.NoError(t, err)
	assert.Equal(t, int64(9_000_000), got.FileSize.Int64)
	assert.Equal(t, "未解析", got.Title, "characterisation: the wide Update reverted enrichment's title")
	assert.Equal(t, models.ParseStatusPending, got.ParseStatus, "…and its parse_status — the row looks un-enriched again")
}

func TestAggregateSeriesFileSizesShape_UpdateFileSizeKeepsConcurrentEnrichment(t *testing.T) {
	repo, ctx := seedScanStateSeries(t) // seeds, then enrichment lands
	all, _, err := repo.List(ctx, ListParams{Page: 1, PageSize: 10000})
	require.NoError(t, err)
	require.Len(t, all, 1)

	require.NoError(t, repo.UpdateFileSize(ctx, all[0].ID, 9_000_000)) // the new code path

	got := assertSeriesEnrichmentSurvived(t, repo, ctx)
	assert.Equal(t, int64(9_000_000), got.FileSize.Int64)
}
