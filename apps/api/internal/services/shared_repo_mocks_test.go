package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/repository"
)

// Shared service-package test doubles. They used to live in
// parse_queue_service_test.go; that service was deleted (bugfix-g — it was
// never constructed), but the enrichment, scanner and recommendation tests use
// them, so they stay here under their old names.
type mockPQParserService struct {
	result *parser.ParseResult
}

func (m *mockPQParserService) ParseFilename(filename string) *parser.ParseResult {
	return m.result
}

func (m *mockPQParserService) ParseBatch(filenames []string) []*parser.ParseResult {
	return nil
}

func (m *mockPQParserService) ParseFilenameWithContext(_ context.Context, filename string) *parser.ParseResult {
	return m.result
}

var _ ParserServiceInterface = (*mockPQParserService)(nil)

type mockPQMetadataService struct {
	searchResult *metadata.SearchResult
	searchErr    error
}

func (m *mockPQMetadataService) SearchMetadata(_ context.Context, _ *SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
	return m.searchResult, nil, m.searchErr
}

func (m *mockPQMetadataService) GetProviders() []ProviderInfo { return nil }

func (m *mockPQMetadataService) ManualSearch(_ context.Context, _ *ManualSearchRequest) (*ManualSearchResponse, error) {
	return nil, nil
}

func (m *mockPQMetadataService) ApplyMetadata(_ context.Context, _ *ApplyMetadataRequest) (*ApplyMetadataResponse, error) {
	return nil, nil
}

func (m *mockPQMetadataService) UpdateMetadata(_ context.Context, _ *UpdateMetadataRequest) (*UpdateMetadataResponse, error) {
	return nil, nil
}

func (m *mockPQMetadataService) UploadPoster(_ context.Context, _ *UploadPosterRequest) (*UploadPosterResponse, error) {
	return nil, nil
}

var _ MetadataServiceInterface = (*mockPQMetadataService)(nil)

type mockPQMovieRepo struct {
	movies map[string]*models.Movie
	err    error
}

func (m *mockPQMovieRepo) Create(_ context.Context, movie *models.Movie) error {
	if m.err != nil {
		return m.err
	}
	m.movies[movie.ID] = movie
	return nil
}

func (m *mockPQMovieRepo) FindByID(_ context.Context, id string) (*models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) FindByTMDbID(_ context.Context, _ int64) (*models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) FindByIMDbID(_ context.Context, _ string) (*models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) FindByFilePath(_ context.Context, _ string) (*models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) Update(_ context.Context, _ *models.Movie) error { return nil }
func (m *mockPQMovieRepo) UpdateCredits(_ context.Context, _ string, _ *models.Credits) error {
	return nil
}
func (m *mockPQMovieRepo) UpdateEnrichedMetadata(_ context.Context, _ *models.Movie) error {
	return nil
}
func (m *mockPQMovieRepo) UpdateScanFileInfo(context.Context, string, int64, models.ParseStatus) error {
	return nil
}
func (m *mockPQMovieRepo) MarkRemoved(context.Context, string) error    { return nil }
func (m *mockPQMovieRepo) RestoreRemoved(context.Context, string) error { return nil }
func (m *mockPQMovieRepo) UpdateParseStatus(context.Context, string, models.ParseStatus) error {
	return nil
}
func (m *mockPQMovieRepo) UpdatePosterPath(context.Context, string, string) error { return nil }
func (m *mockPQMovieRepo) Delete(_ context.Context, _ string) error               { return nil }
func (m *mockPQMovieRepo) List(_ context.Context, _ repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockPQMovieRepo) SearchByTitle(_ context.Context, _ string, _ repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockPQMovieRepo) FullTextSearch(_ context.Context, _ string, _ repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockPQMovieRepo) Upsert(_ context.Context, _ *models.Movie) error { return nil }
func (m *mockPQMovieRepo) GetDistinctGenres(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) GetYearRange(_ context.Context) (int, int, error) { return 0, 0, nil }
func (m *mockPQMovieRepo) Count(_ context.Context) (int, error)             { return 0, nil }
func (m *mockPQMovieRepo) BulkCreate(_ context.Context, _ []*models.Movie) error {
	return nil
}
func (m *mockPQMovieRepo) FindByParseStatus(_ context.Context, _ models.ParseStatus) ([]models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) UpdateSubtitleGenerationStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string) error {
	return nil
}

func (m *mockPQMovieRepo) UpdateSubtitleStatus(_ context.Context, _ string, _ models.SubtitleStatus, _, _ string, _ float64) error {
	return nil
}
func (m *mockPQMovieRepo) FindBySubtitleStatus(_ context.Context, _ models.SubtitleStatus) ([]models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) FindNeedingSubtitleSearch(_ context.Context, _ time.Time) ([]models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) FindMissingZhHantSubtitle(_ context.Context) ([]models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) CountMissingZhHantSubtitle(_ context.Context) (int, error) { return 0, nil }
func (m *mockPQMovieRepo) FindAllWithFilePath(_ context.Context) ([]models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) GetStats(_ context.Context) (*repository.MediaStats, error) {
	return &repository.MediaStats{}, nil
}
func (m *mockPQMovieRepo) FindOwnedTMDbIDs(_ context.Context, _ []int64) ([]int64, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) UpdateDoubanRating(_ context.Context, _, _ string, _ float64, _ int) error {
	return nil
}

func (m *mockPQMovieRepo) CountZhHantSubtitle(_ context.Context) (int, error) {
	return 0, nil
}

var _ repository.MovieRepositoryInterface = (*mockPQMovieRepo)(nil)

type mockPQSeriesRepo struct {
	series map[string]*models.Series
	err    error
}

func newMockPQSeriesRepo() *mockPQSeriesRepo {
	return &mockPQSeriesRepo{series: make(map[string]*models.Series)}
}

func (m *mockPQSeriesRepo) Create(_ context.Context, s *models.Series) error {
	if m.err != nil {
		return m.err
	}
	m.series[s.ID] = s
	return nil
}

func (m *mockPQSeriesRepo) FindByID(_ context.Context, id string) (*models.Series, error) {
	if s, ok := m.series[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("series with id %s not found", id)
}

func (m *mockPQSeriesRepo) FindByTMDbID(_ context.Context, tmdbID int64) (*models.Series, error) {
	for _, s := range m.series {
		if s.TMDbID.Valid && s.TMDbID.Int64 == tmdbID {
			return s, nil
		}
	}
	return nil, fmt.Errorf("series with tmdb_id %d not found", tmdbID)
}

func (m *mockPQSeriesRepo) FindByIMDbID(_ context.Context, _ string) (*models.Series, error) {
	return nil, nil
}
func (m *mockPQSeriesRepo) FindByFilePath(_ context.Context, filePath string) (*models.Series, error) {
	for _, s := range m.series {
		if s.FilePath.Valid && s.FilePath.String == filePath {
			return s, nil
		}
	}
	return nil, nil // (nil, nil) on miss, matching the real repository
}
func (m *mockPQSeriesRepo) Update(_ context.Context, _ *models.Series) error { return nil }
func (m *mockPQSeriesRepo) UpdateCredits(_ context.Context, _ string, _ *models.Credits) error {
	return nil
}
func (m *mockPQSeriesRepo) UpdateEnrichedMetadata(_ context.Context, _ *models.Series) error {
	return nil
}
func (m *mockPQSeriesRepo) UpdateFileSize(context.Context, string, int64) error { return nil }
func (m *mockPQSeriesRepo) UpdateParseStatus(context.Context, string, models.ParseStatus) error {
	return nil
}
func (m *mockPQSeriesRepo) UpdatePosterPath(context.Context, string, string) error { return nil }
func (m *mockPQSeriesRepo) Delete(_ context.Context, _ string) error               { return nil }
func (m *mockPQSeriesRepo) List(_ context.Context, _ repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockPQSeriesRepo) SearchByTitle(_ context.Context, _ string, _ repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockPQSeriesRepo) FullTextSearch(_ context.Context, _ string, _ repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockPQSeriesRepo) Upsert(_ context.Context, _ *models.Series) error { return nil }
func (m *mockPQSeriesRepo) GetDistinctGenres(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockPQSeriesRepo) GetYearRange(_ context.Context) (int, int, error) { return 0, 0, nil }
func (m *mockPQSeriesRepo) Count(_ context.Context) (int, error)             { return 0, nil }
func (m *mockPQSeriesRepo) BulkCreate(_ context.Context, _ []*models.Series) error {
	return nil
}
func (m *mockPQSeriesRepo) FindByParseStatus(_ context.Context, _ models.ParseStatus) ([]models.Series, error) {
	return nil, nil
}
func (m *mockPQSeriesRepo) UpdateSubtitleGenerationStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string) error {
	return nil
}

func (m *mockPQSeriesRepo) UpdateSubtitleStatus(_ context.Context, _ string, _ models.SubtitleStatus, _, _ string, _ float64) error {
	return nil
}
func (m *mockPQSeriesRepo) FindBySubtitleStatus(_ context.Context, _ models.SubtitleStatus) ([]models.Series, error) {
	return nil, nil
}
func (m *mockPQSeriesRepo) FindNeedingSubtitleSearch(_ context.Context, _ time.Time) ([]models.Series, error) {
	return nil, nil
}
func (m *mockPQSeriesRepo) GetStats(_ context.Context) (*repository.MediaStats, error) {
	return &repository.MediaStats{}, nil
}
func (m *mockPQSeriesRepo) FindOwnedTMDbIDs(_ context.Context, _ []int64) ([]int64, error) {
	return nil, nil
}
func (m *mockPQSeriesRepo) UpdateDoubanRating(_ context.Context, _, _ string, _ float64, _ int) error {
	return nil
}

func (m *mockPQSeriesRepo) CountZhHantCovered(_ context.Context) (int, error) {
	return 0, nil
}

var _ repository.SeriesRepositoryInterface = (*mockPQSeriesRepo)(nil)

type mockPQSeasonRepo struct {
	seasons map[string]*models.Season
	err     error
}

func newMockPQSeasonRepo() *mockPQSeasonRepo {
	return &mockPQSeasonRepo{seasons: make(map[string]*models.Season)}
}

func (m *mockPQSeasonRepo) Create(_ context.Context, s *models.Season) error {
	if m.err != nil {
		return m.err
	}
	m.seasons[s.ID] = s
	return nil
}

func (m *mockPQSeasonRepo) FindByID(_ context.Context, id string) (*models.Season, error) {
	if s, ok := m.seasons[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("season with id %s not found", id)
}

func (m *mockPQSeasonRepo) FindBySeriesID(_ context.Context, seriesID string) ([]models.Season, error) {
	var result []models.Season
	for _, s := range m.seasons {
		if s.SeriesID == seriesID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockPQSeasonRepo) FindBySeriesAndNumber(_ context.Context, seriesID string, seasonNumber int) (*models.Season, error) {
	for _, s := range m.seasons {
		if s.SeriesID == seriesID && s.SeasonNumber == seasonNumber {
			return s, nil
		}
	}
	return nil, fmt.Errorf("season %d for series %s: %w", seasonNumber, seriesID, repository.ErrSeasonNotFound)
}

func (m *mockPQSeasonRepo) Update(_ context.Context, _ *models.Season) error { return nil }
func (m *mockPQSeasonRepo) Delete(_ context.Context, _ string) error         { return nil }
func (m *mockPQSeasonRepo) Upsert(_ context.Context, s *models.Season) error {
	if m.err != nil {
		return m.err
	}
	m.seasons[s.ID] = s
	return nil
}

var _ repository.SeasonRepositoryInterface = (*mockPQSeasonRepo)(nil)

type mockPQEpisodeRepo struct {
	episodes map[string]*models.Episode
	err      error
}

func newMockPQEpisodeRepo() *mockPQEpisodeRepo {
	return &mockPQEpisodeRepo{episodes: make(map[string]*models.Episode)}
}

func (m *mockPQEpisodeRepo) Create(_ context.Context, ep *models.Episode) error {
	if m.err != nil {
		return m.err
	}
	m.episodes[ep.ID] = ep
	return nil
}

func (m *mockPQEpisodeRepo) FindByID(_ context.Context, id string) (*models.Episode, error) {
	if ep, ok := m.episodes[id]; ok {
		return ep, nil
	}
	return nil, fmt.Errorf("episode with id %s not found", id)
}

func (m *mockPQEpisodeRepo) FindBySeriesID(_ context.Context, _ string) ([]models.Episode, error) {
	return nil, nil
}

func (m *mockPQEpisodeRepo) FindBySeasonID(_ context.Context, _ string) ([]models.Episode, error) {
	return nil, nil
}

func (m *mockPQEpisodeRepo) FindBySeasonNumber(_ context.Context, _ string, _ int) ([]models.Episode, error) {
	return nil, nil
}

func (m *mockPQEpisodeRepo) CountMissingZhHantSubtitle(context.Context) (int, error) {
	return 0, nil
}

func (m *mockPQEpisodeRepo) FindMissingZhHantSubtitle(context.Context) ([]models.Episode, error) {
	return nil, nil
}

// UpdateDurationSeconds satisfies the sub-6-10a addition to the interface.
// The parse queue never measures durations, so this stub records nothing.
func (m *mockPQEpisodeRepo) UpdateDurationSeconds(context.Context, string, int64) error {
	return nil
}

func (m *mockPQEpisodeRepo) FindBySeriesSeasonEpisode(_ context.Context, seriesID string, season, episode int) (*models.Episode, error) {
	for _, ep := range m.episodes {
		if ep.SeriesID == seriesID && ep.SeasonNumber == season && ep.EpisodeNumber == episode {
			return ep, nil
		}
	}
	return nil, fmt.Errorf("episode S%02dE%02d for series %s: %w", season, episode, seriesID, repository.ErrEpisodeNotFound)
}

func (m *mockPQEpisodeRepo) Update(_ context.Context, _ *models.Episode) error { return nil }
func (m *mockPQEpisodeRepo) UpdateEpisodeSubtitleStatus(_ context.Context, _ string, _ models.SubtitleStatus, _, _ string) error {
	return nil
}
func (m *mockPQEpisodeRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *mockPQEpisodeRepo) Upsert(_ context.Context, ep *models.Episode) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	_, existed := m.episodes[ep.ID]
	m.episodes[ep.ID] = ep
	return !existed, nil
}

var _ repository.EpisodeRepositoryInterface = (*mockPQEpisodeRepo)(nil)

func (*mockPQMovieRepo) FindWithFileByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	return nil, nil
}

// FindActiveByTMDbID — dl-import-1 interface addition; not exercised by these tests.
func (*mockPQSeriesRepo) FindActiveByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	return nil, nil
}
