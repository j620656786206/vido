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

// disc-2026-10-movie-subtitle-tracks-unknown-refresh: the backfill lists only
// movies with a file whose subtitle tracks are unknown (NULL) — `[]` is a
// known "no subtitles" and stays out — and its write never overwrites a value.
func TestFindMissingSubtitleTracks_ListsOnlyUnknownRowsWithAFile(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	movies := NewMovieRepository(db)

	path := func(p string) models.NullString { return models.NewNullString(p) }
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-unknown", Title: "A", FilePath: path("/m/a.mkv")}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-none", Title: "B", FilePath: path("/m/b.mkv"), SubtitleTracks: models.NewNullString("[]")}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-has", Title: "C", FilePath: path("/m/c.mkv"), SubtitleTracks: models.NewNullString(`[{"language":"chi"}]`)}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-no-file", Title: "D"}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "mv-removed", Title: "E", FilePath: path("/m/e.mkv"), IsRemoved: true}))

	got, err := movies.FindMissingSubtitleTracks(ctx, "", 50)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "mv-unknown", got[0].ID)

	before, err := movies.FindByID(ctx, "mv-unknown")
	require.NoError(t, err)

	written, err := movies.UpdateSubtitleTracksIfMissing(ctx, "mv-unknown", "[]")
	require.NoError(t, err)
	assert.True(t, written)
	after, err := movies.FindByID(ctx, "mv-unknown")
	require.NoError(t, err)
	assert.Equal(t, "[]", after.SubtitleTracks.String)
	assert.Equal(t, before.UpdatedAt, after.UpdatedAt, "a background fill is not an edit of the movie")

	got, err = movies.FindMissingSubtitleTracks(ctx, "", 50)
	require.NoError(t, err)
	assert.Empty(t, got, "once filled, the row leaves the list")

	// A value that is already there — the scan's, or this fill's — is kept.
	written, err = movies.UpdateSubtitleTracksIfMissing(ctx, "mv-has", "[]")
	require.NoError(t, err)
	assert.False(t, written)
	kept, err := movies.FindByID(ctx, "mv-has")
	require.NoError(t, err)
	assert.Equal(t, `[{"language":"chi"}]`, kept.SubtitleTracks.String)

	written, err = movies.UpdateSubtitleTracksIfMissing(ctx, "no-such-movie", "[]")
	require.NoError(t, err)
	assert.False(t, written)
}

// The backfill pages by id so rows that stay unknown cannot crowd out the
// rest; and enrichment holding a stale NULL cannot erase a filled answer.
func TestFindMissingSubtitleTracks_PagesByIDAndNullNeverErasesAnAnswer(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	movies := NewMovieRepository(db)
	for _, id := range []string{"mv-a", "mv-b", "mv-c"} {
		require.NoError(t, movies.Create(ctx, &models.Movie{ID: id, Title: id, FilePath: models.NewNullString("/m/" + id + ".mkv")}))
	}

	page, err := movies.FindMissingSubtitleTracks(ctx, "", 2)
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, []string{"mv-a", "mv-b"}, []string{page[0].ID, page[1].ID})
	page, err = movies.FindMissingSubtitleTracks(ctx, "mv-b", 2)
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, "mv-c", page[0].ID)

	// Enrichment loaded mv-a while it was unknown, skipped the probe (NFO
	// tech info), and writes back after the backfill filled it.
	stale, err := movies.FindByID(ctx, "mv-a")
	require.NoError(t, err)
	written, err := movies.UpdateSubtitleTracksIfMissing(ctx, "mv-a", "[]")
	require.NoError(t, err)
	require.True(t, written)
	stale.Title = "Matched Title"
	require.NoError(t, movies.UpdateEnrichedMetadata(ctx, stale))
	got, err := movies.FindByID(ctx, "mv-a")
	require.NoError(t, err)
	assert.Equal(t, "Matched Title", got.Title)
	assert.Equal(t, "[]", got.SubtitleTracks.String, "a stale NULL must not turn 「缺中文」 back into 「不知道」")

	// Same for the wide Update; a real value still replaces the old one.
	stale.SubtitleTracks = models.NullString{}
	require.NoError(t, movies.Update(ctx, stale))
	got, err = movies.FindByID(ctx, "mv-a")
	require.NoError(t, err)
	assert.Equal(t, "[]", got.SubtitleTracks.String)
	stale.SubtitleTracks = models.NewNullString(`[{"language":"chi"}]`)
	require.NoError(t, movies.UpdateEnrichedMetadata(ctx, stale))
	got, err = movies.FindByID(ctx, "mv-a")
	require.NoError(t, err)
	assert.Equal(t, `[{"language":"chi"}]`, got.SubtitleTracks.String)
}
