package repository

// bugfix-upsert-zeroes-unloaded-columns — Upsert's ownership contract.
//
// Upsert receives a FRESH TMDb conversion. Before this story, a re-match wrote
// that model through the wide Update, zeroing every column the converter never
// produces: the delivered subtitle, library assignment, removed flag, file
// size, and the ffprobe technical columns. Dormant (no production caller), but
// the tests below make the helper safe to wire.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func seedMatchedMovie(t *testing.T) (*MovieRepository, context.Context) {
	t.Helper()
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewMovieRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &models.Movie{
		ID:              "mv-up",
		Title:           "舊標題",
		TMDbID:          models.NewNullInt64(603),
		FilePath:        models.NewNullString("/media/matrix.mkv"),
		FileSize:        models.NewNullInt64(4096),
		LibraryID:       models.NewNullString("lib-a"),
		IsRemoved:       true,
		VideoCodec:      models.NewNullString("hevc"),
		VideoResolution: models.NewNullString("2160p"),
		AudioCodec:      models.NewNullString("truehd"),
		AudioChannels:   models.NewNullInt64(8),
		SubtitleTracks:  models.NewNullString(`[{"lang":"zh"}]`),
		HDRFormat:       models.NewNullString("dolby-vision"),
	}))
	// The subtitle pipeline delivered meanwhile. UpdateSubtitleStatus (the
	// search path's writer) also stamps last_searched + search_score, so every
	// preserved subtitle column is non-zero in the fixture (review F2).
	require.NoError(t, repo.UpdateSubtitleStatus(ctx,
		"mv-up", models.SubtitleStatusFound, "/media/matrix.zh-Hant.srt", "zh-Hant", 0.87))
	return repo, ctx
}

// freshTMDbMovie is what ConvertTMDbMovieToModel produces on a re-match: TMDb
// metadata only; none of the other writers' columns.
func freshTMDbMovie(filePath string) *models.Movie {
	m := &models.Movie{
		Title:       "駭客任務",
		TMDbID:      models.NewNullInt64(603),
		ParseStatus: models.ParseStatusSuccess,
		Overview:    models.NewNullString("What is the Matrix?"),
		VoteAverage: models.NewNullFloat64(8.2),
	}
	if filePath != "" {
		m.FilePath = models.NewNullString(filePath)
	}
	return m
}

// AC #1: a re-match refreshes TMDb metadata and preserves everyone else's columns.
func TestMovieUpsert_RematchPreservesColumnsTMDbDoesNotOwn(t *testing.T) {
	repo, ctx := seedMatchedMovie(t)

	require.NoError(t, repo.Upsert(ctx, freshTMDbMovie("")))

	got, err := repo.FindByID(ctx, "mv-up")
	require.NoError(t, err)
	// TMDb surface refreshed.
	assert.Equal(t, "駭客任務", got.Title)
	assert.Equal(t, "What is the Matrix?", got.Overview.String)
	// Everyone else's columns preserved.
	assert.Equal(t, models.SubtitleStatusFound, got.SubtitleStatus, "the delivered subtitle must survive a re-match")
	assert.Equal(t, "/media/matrix.zh-Hant.srt", got.SubtitlePath.String)
	assert.Equal(t, "zh-Hant", got.SubtitleLanguage.String)
	assert.True(t, got.SubtitleLastSearched.Valid, "last_searched must survive")
	assert.Equal(t, 0.87, got.SubtitleSearchScore.Float64)
	assert.Equal(t, "lib-a", got.LibraryID.String)
	assert.True(t, got.IsRemoved)
	assert.Equal(t, int64(4096), got.FileSize.Int64)
	assert.Equal(t, "hevc", got.VideoCodec.String)
	assert.Equal(t, "2160p", got.VideoResolution.String)
	assert.Equal(t, "truehd", got.AudioCodec.String)
	assert.Equal(t, int64(8), got.AudioChannels.Int64)
	assert.Equal(t, `[{"lang":"zh"}]`, got.SubtitleTracks.String)
	assert.Equal(t, "dolby-vision", got.HDRFormat.String)
	assert.Equal(t, "/media/matrix.mkv", got.FilePath.String, "empty incoming file_path keeps the stored one")
}

// AC #2: a provided file path wins (re-match after a file move).
func TestMovieUpsert_IncomingFilePathWins(t *testing.T) {
	repo, ctx := seedMatchedMovie(t)

	require.NoError(t, repo.Upsert(ctx, freshTMDbMovie("/media/new/matrix-remux.mkv")))

	got, err := repo.FindByID(ctx, "mv-up")
	require.NoError(t, err)
	assert.Equal(t, "/media/new/matrix-remux.mkv", got.FilePath.String)
	assert.Equal(t, models.SubtitleStatusFound, got.SubtitleStatus)
}

// AC #4: the create branches are untouched.
func TestMovieUpsert_CreatePathsUnchanged(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Unknown TMDb id → create. (ID assignment is the service's job —
	// SaveMovieFromTMDb uuids it — so the repo-level test supplies one.)
	fresh := freshTMDbMovie("/media/matrix.mkv")
	fresh.ID = "mv-new"
	require.NoError(t, repo.Upsert(ctx, fresh))
	created, err := repo.FindByTMDbID(ctx, 603)
	require.NoError(t, err)
	assert.Equal(t, "駭客任務", created.Title)

	// No TMDb id at all → create.
	require.NoError(t, repo.Upsert(ctx, &models.Movie{ID: "mv-no-tmdb", Title: "無 TMDb"}))
}

// AC #5: the pre-existing CreditsJSON rule still holds both ways.
func TestMovieUpsert_CreditsPreservationStillHolds(t *testing.T) {
	repo, ctx := seedMatchedMovie(t)
	manual := &models.Movie{ID: "mv-up"}
	require.NoError(t, manual.SetCredits(&models.Credits{Crew: []models.CrewMember{{Name: "Wachowski", Job: "Director"}}}))
	seeded, err := repo.FindByID(ctx, "mv-up")
	require.NoError(t, err)
	seeded.CreditsJSON = manual.CreditsJSON
	require.NoError(t, repo.Update(ctx, seeded))

	require.NoError(t, repo.Upsert(ctx, freshTMDbMovie("")))

	got, err := repo.FindByID(ctx, "mv-up")
	require.NoError(t, err)
	credits, err := got.GetCredits()
	require.NoError(t, err)
	require.NotNil(t, credits, "manual credits must survive a re-match (pre-existing rule)")
	require.Len(t, credits.Crew, 1)
	assert.Equal(t, "Wachowski", credits.Crew[0].Name)
}

// AC #3: the series flavour.
func TestSeriesUpsert_RematchPreservesColumnsTMDbDoesNotOwn(t *testing.T) {
	db := setupSeriesTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewSeriesRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &models.Series{
		ID:              "sr-up",
		Title:           "舊標題",
		TMDbID:          models.NewNullInt64(1396),
		IMDbID:          models.NewNullString("tt0903747"),
		FilePath:        models.NewNullString("/media/bb"),
		FileSize:        models.NewNullInt64(9000),
		LibraryID:       models.NewNullString("lib-b"),
		IsRemoved:       true,
		VideoCodec:      models.NewNullString("hevc"),
		VideoResolution: models.NewNullString("1080p"),
		AudioCodec:      models.NewNullString("eac3"),
		AudioChannels:   models.NewNullInt64(6),
		SubtitleTracks:  models.NewNullString(`[{"lang":"en"}]`),
		HDRFormat:       models.NewNullString("hdr10"),
	}))

	require.NoError(t, repo.Upsert(ctx, &models.Series{
		Title:       "絕命毒師",
		TMDbID:      models.NewNullInt64(1396),
		ParseStatus: models.ParseStatusSuccess,
	}))

	got, err := repo.FindByID(ctx, "sr-up")
	require.NoError(t, err)
	assert.Equal(t, "絕命毒師", got.Title)
	// Review F1: TMDb's TV payload has no imdb field — the series create
	// handler owns imdb_id and a re-match must not NULL it.
	assert.Equal(t, "tt0903747", got.IMDbID.String)
	assert.Equal(t, "lib-b", got.LibraryID.String)
	assert.True(t, got.IsRemoved)
	assert.Equal(t, int64(9000), got.FileSize.Int64)
	assert.Equal(t, "hevc", got.VideoCodec.String)
	assert.Equal(t, "1080p", got.VideoResolution.String)
	assert.Equal(t, "eac3", got.AudioCodec.String)
	assert.Equal(t, int64(6), got.AudioChannels.Int64)
	assert.Equal(t, `[{"lang":"en"}]`, got.SubtitleTracks.String)
	assert.Equal(t, "hdr10", got.HDRFormat.String)
	assert.Equal(t, "/media/bb", got.FilePath.String)
}
