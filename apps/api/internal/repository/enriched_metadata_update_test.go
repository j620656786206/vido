package repository

// 9R-10b CR-249 finding B — the lost-update regression.
//
// EnrichmentService loads a media row, then spends seconds to tens of seconds on
// it (NFO read, filename parse which may call an LLM, TMDB search) before writing
// it back. It used the WIDE Update, which persists all 38 mutable columns —
// including the five subtitle-delivery columns it never assigns. Anything that
// wrote those columns during that window was silently reverted.
//
// The two workers collide on exactly the case story 9R-10b exists for: enrichment
// enumerates rows with parse_status pending/empty (newly scanned files) and the
// auto-trigger runs the free subtitle lane over rows missing zh-Hant subtitles
// (also newly scanned files), CONCURRENTLY, off the same scan-complete callback.
// The symptom is the worst kind: the .srt is written to disk, and then the
// database is told it does not exist.
//
// The first test characterises the bug so nobody has to re-derive it. The second
// is the pin.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// seedMovieMidEnrichment inserts a row and hands back the copy enrichment would
// be holding, plus the repo.
func seedMovieMidEnrichment(t *testing.T) (*MovieRepository, *models.Movie) {
	t.Helper()
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewMovieRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Movie{
		ID:          "mv-race",
		Title:       "未解析的檔名",
		ParseStatus: models.ParseStatusPending,
	}))

	// This is enrichment's in-memory copy: read BEFORE the subtitle pipeline runs.
	stale, err := repo.FindByID(ctx, "mv-race")
	require.NoError(t, err)
	require.NotNil(t, stale)

	// Meanwhile the free lane delivers a subtitle and records it through the
	// NARROW writer the pipeline has always used.
	require.NoError(t, repo.UpdateSubtitleGenerationStatus(ctx,
		"mv-race", models.SubtitleStatusFound, "/media/mv-race.zh-Hant.srt", "zh-Hant"))

	// Enrichment finishes its slow work and mutates only what it computes.
	stale.Title = "鬼滅之刃"
	stale.ParseStatus = models.ParseStatusSuccess

	return repo, stale
}

// TestWideUpdate_LosesConcurrentSubtitleWrite characterises the defect. If this
// test ever starts FAILING, the wide Update stopped clobbering — check whether
// the narrow writer is still needed before deleting anything.
func TestWideUpdate_LosesConcurrentSubtitleWrite(t *testing.T) {
	repo, stale := seedMovieMidEnrichment(t)
	ctx := context.Background()

	require.NoError(t, repo.Update(ctx, stale))

	got, err := repo.FindByID(ctx, "mv-race")
	require.NoError(t, err)
	assert.Equal(t, models.SubtitleStatusNotSearched, got.SubtitleStatus,
		"the wide writer reverts subtitle_status from a stale copy — this is the defect, documented on purpose")
	assert.Empty(t, got.SubtitlePath.String,
		"and the delivered sidecar's path is erased while the file sits on disk")
}

// TestUpdateEnrichedMetadata_PreservesConcurrentSubtitleWrite is the pin.
func TestUpdateEnrichedMetadata_PreservesConcurrentSubtitleWrite(t *testing.T) {
	repo, stale := seedMovieMidEnrichment(t)
	ctx := context.Background()

	require.NoError(t, repo.UpdateEnrichedMetadata(ctx, stale))

	got, err := repo.FindByID(ctx, "mv-race")
	require.NoError(t, err)

	t.Run("enrichment's own columns are persisted", func(t *testing.T) {
		assert.Equal(t, "鬼滅之刃", got.Title)
		assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
	})

	t.Run("the concurrent subtitle delivery survives", func(t *testing.T) {
		assert.Equal(t, models.SubtitleStatusFound, got.SubtitleStatus,
			"enrichment must not revert a subtitle status it never computed")
		assert.Equal(t, "/media/mv-race.zh-Hant.srt", got.SubtitlePath.String,
			"the database must not deny a sidecar that is on disk")
		assert.Equal(t, "zh-Hant", got.SubtitleLanguage.String)
	})
}

func TestUpdateEnrichedMetadata_RejectsMissingRow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewMovieRepository(db)

	err := repo.UpdateEnrichedMetadata(context.Background(), &models.Movie{ID: "ghost", Title: "x"})
	assert.Error(t, err, "a write that hits zero rows means the media vanished underneath us — the caller deserves to know")
}

func TestUpdateEnrichedMetadata_RejectsNilAndEmptyID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewMovieRepository(db)
	ctx := context.Background()

	assert.Error(t, repo.UpdateEnrichedMetadata(ctx, nil))
	assert.Error(t, repo.UpdateEnrichedMetadata(ctx, &models.Movie{}))
}

// disc-2026-10-series-credits-empty: the backfill's query lists matched rows
// with no cast and nothing else — not unmatched rows, not rows that already
// have a cast (TMDb's or the user's), not removed rows.
func TestFindMissingCredits_ListsOnlyMatchedRowsWithoutCast(t *testing.T) {
	db := setupTestDB(t) // movies table
	t.Cleanup(func() { _ = db.Close() })
	sdb := setupSeriesTestDB(t) // series table lives in its own fixture
	t.Cleanup(func() { _ = sdb.Close() })
	ctx := context.Background()
	movies := NewMovieRepository(db)
	series := NewSeriesRepository(sdb)

	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-matched", Title: "A", TMDbID: models.NewNullInt64(1)}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-unmatched", Title: "B"}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-has-cast", Title: "C", TMDbID: models.NewNullInt64(3)}))
	require.NoError(t, movies.UpdateCredits(ctx, "mv-has-cast", &models.Credits{Cast: []models.CastMember{{Name: "Tom"}}}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-removed", Title: "D", TMDbID: models.NewNullInt64(4), IsRemoved: true}))

	require.NoError(t, series.Create(ctx, &models.Series{ID: "see", Title: "末日光明", TMDbID: models.NewNullInt64(80752)}))
	require.NoError(t, series.Create(ctx, &models.Series{ID: "sr-has-cast", Title: "E", TMDbID: models.NewNullInt64(5)}))
	require.NoError(t, series.UpdateCredits(ctx, "sr-has-cast", &models.Credits{Cast: []models.CastMember{{Name: "Ann"}}}))

	gotMovies, err := movies.FindMissingCredits(ctx, 50)
	require.NoError(t, err)
	require.Len(t, gotMovies, 1)
	assert.Equal(t, "mv-matched", gotMovies[0].ID)

	gotSeries, err := series.FindMissingCredits(ctx, 50)
	require.NoError(t, err)
	require.Len(t, gotSeries, 1)
	assert.Equal(t, "see", gotSeries[0].ID)

	// Once filled, the row leaves the list.
	require.NoError(t, series.UpdateCredits(ctx, "see", &models.Credits{Cast: []models.CastMember{{Name: "Jason Momoa", Character: "Baba Voss"}}}))
	gotSeries, err = series.FindMissingCredits(ctx, 50)
	require.NoError(t, err)
	assert.Empty(t, gotSeries)
}
