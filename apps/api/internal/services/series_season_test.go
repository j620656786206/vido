package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/testutil"
	"github.com/vido/api/internal/tmdb"
)

// stubEpisodeRepo implements repository.EpisodeRepositoryInterface; only
// FindBySeasonNumber is exercised by the season-merge logic.
type stubEpisodeRepo struct {
	episodes []models.Episode
	err      error
}

func (s *stubEpisodeRepo) FindBySeasonNumber(_ context.Context, _ string, _ int) ([]models.Episode, error) {
	return s.episodes, s.err
}
func (s *stubEpisodeRepo) Create(context.Context, *models.Episode) error             { return nil }
func (s *stubEpisodeRepo) FindByID(context.Context, string) (*models.Episode, error) { return nil, nil }
func (s *stubEpisodeRepo) FindBySeriesID(context.Context, string) ([]models.Episode, error) {
	return nil, nil
}
func (s *stubEpisodeRepo) FindBySeasonID(context.Context, string) ([]models.Episode, error) {
	return nil, nil
}
func (s *stubEpisodeRepo) FindBySeriesSeasonEpisode(context.Context, string, int, int) (*models.Episode, error) {
	return nil, nil
}
func (s *stubEpisodeRepo) Update(context.Context, *models.Episode) error { return nil }
func (s *stubEpisodeRepo) UpdateEpisodeSubtitleStatus(context.Context, string, models.SubtitleStatus, string, string) error {
	return nil
}
func (s *stubEpisodeRepo) Delete(context.Context, string) error          { return nil }
func (s *stubEpisodeRepo) Upsert(context.Context, *models.Episode) error { return nil }

// stubSeasonProvider implements SeasonDetailsProvider.
type stubSeasonProvider struct {
	details *tmdb.SeasonDetails
	err     error
}

func (s *stubSeasonProvider) GetSeasonDetails(_ context.Context, _ int, _ int) (*tmdb.SeasonDetails, error) {
	return s.details, s.err
}

// seriesRepoReturning builds a testutil.MockSeriesRepository whose FindByID
// returns the given series.
func seriesRepoReturning(series *models.Series) *testutil.MockSeriesRepository {
	repo := new(testutil.MockSeriesRepository)
	repo.On("FindByID", mock.Anything, mock.Anything).Return(series, nil)
	return repo
}

func newSeasonSeries(t *testing.T, tmdbID int64) *models.Series {
	t.Helper()
	series := &models.Series{ID: "series-1", Title: "Test"}
	if tmdbID > 0 {
		series.TMDbID = models.NewNullInt64(tmdbID)
	}
	require.NoError(t, series.SetSeasons([]models.SeasonSummary{
		{ID: 1, SeasonNumber: 1, Name: "第 1 季", EpisodeCount: 2},
	}))
	return series
}

func newSeasonServiceUnderTest(series *models.Series, episodes []models.Episode, episodeErr error, details *tmdb.SeasonDetails, tmdbErr error) *SeriesService {
	svc := NewSeriesService(seriesRepoReturning(series))
	svc.SetEpisodeDeps(
		&stubEpisodeRepo{episodes: episodes, err: episodeErr},
		&stubSeasonProvider{details: details, err: tmdbErr},
	)
	return svc
}

func TestSeriesService_GetSeasonEpisodes_Merge(t *testing.T) {
	series := newSeasonSeries(t, 1396)
	details := &tmdb.SeasonDetails{
		ID: 1, Name: "第 1 季", SeasonNumber: 1,
		Episodes: []tmdb.EpisodeInfo{
			{EpisodeNumber: 1, Name: "第一集"},
			{EpisodeNumber: 2, Name: "第二集"},
			{EpisodeNumber: 3, Name: "第三集"},
		},
	}
	// Local records: ep1 has a file + found subtitle; ep2 has NO file (so no indicator).
	local := []models.Episode{
		{ID: "e1", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 1,
			FilePath: models.NewNullString("/m/S01E01.mkv"), SubtitleStatus: models.SubtitleStatusFound,
			SubtitleLanguage: models.NewNullString("zh-Hant")},
		{ID: "e2", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 2,
			SubtitleStatus: models.SubtitleStatusNotSearched},
	}

	svc := newSeasonServiceUnderTest(series, local, nil, details, nil)
	resp, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, 1, resp.Season.SeasonNumber)
	assert.Equal(t, "第 1 季", resp.Season.Name)
	require.Len(t, resp.Episodes, 3)

	// Episode 1: local file present → enriched.
	assert.True(t, resp.Episodes[0].HasLocalFile)
	assert.Equal(t, "found", resp.Episodes[0].SubtitleStatus)
	assert.Equal(t, "zh-Hant", resp.Episodes[0].SubtitleLanguage)
	assert.Equal(t, "/m/S01E01.mkv", resp.Episodes[0].FilePath)

	// Episode 2: local record exists but NO file → no indicator (AC #6).
	assert.False(t, resp.Episodes[1].HasLocalFile)
	assert.Empty(t, resp.Episodes[1].SubtitleStatus)

	// Episode 3: no local record at all → no indicator.
	assert.False(t, resp.Episodes[2].HasLocalFile)
}

func TestSeriesService_GetSeasonEpisodes_TMDbError(t *testing.T) {
	series := newSeasonSeries(t, 1396)
	svc := newSeasonServiceUnderTest(series, nil, nil, nil, errors.New("tmdb down"))

	_, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch season episodes")
}

func TestSeriesService_GetSeasonEpisodes_NotLinkedToTMDb(t *testing.T) {
	series := newSeasonSeries(t, 0) // no tmdb_id
	svc := newSeasonServiceUnderTest(series, nil, nil, &tmdb.SeasonDetails{}, nil)

	_, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	assert.ErrorIs(t, err, ErrSeriesNotLinkedToTMDb)
}

func TestSeriesService_GetSeasonEpisodes_DepsNotConfigured(t *testing.T) {
	svc := NewSeriesService(seriesRepoReturning(newSeasonSeries(t, 1396)))
	_, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	assert.ErrorIs(t, err, ErrSeasonDepsNotConfigured)
}

func TestSeriesService_GetSeasonEpisodes_LocalRepoErrorDegrades(t *testing.T) {
	series := newSeasonSeries(t, 1396)
	details := &tmdb.SeasonDetails{Episodes: []tmdb.EpisodeInfo{{EpisodeNumber: 1, Name: "第一集"}}}
	// Local repo errors → degrade to TMDb-only, no overall failure.
	svc := newSeasonServiceUnderTest(series, nil, errors.New("db error"), details, nil)

	resp, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	require.NoError(t, err)
	require.Len(t, resp.Episodes, 1)
	assert.False(t, resp.Episodes[0].HasLocalFile)
}

func TestSeriesService_GetSeasons(t *testing.T) {
	series := newSeasonSeries(t, 1396)
	svc := NewSeriesService(seriesRepoReturning(series))

	seasons, err := svc.GetSeasons(context.Background(), "series-1")
	require.NoError(t, err)
	require.Len(t, seasons, 1)
	assert.Equal(t, 1, seasons[0].SeasonNumber)
}
