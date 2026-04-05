package services

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// --- Parse Queue Mocks ---

type mockPQParseJobRepo struct {
	jobs map[string]*models.ParseJob
	err  error
}

func newMockPQParseJobRepo() *mockPQParseJobRepo {
	return &mockPQParseJobRepo{jobs: make(map[string]*models.ParseJob)}
}

func (m *mockPQParseJobRepo) Create(_ context.Context, job *models.ParseJob) error {
	if m.err != nil {
		return m.err
	}
	m.jobs[job.ID] = job
	return nil
}

func (m *mockPQParseJobRepo) GetByID(_ context.Context, id string) (*models.ParseJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	if j, ok := m.jobs[id]; ok {
		return j, nil
	}
	return nil, fmt.Errorf("parse job with id %s not found", id)
}

func (m *mockPQParseJobRepo) GetByTorrentHash(_ context.Context, hash string) (*models.ParseJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, j := range m.jobs {
		if j.TorrentHash == hash {
			return j, nil
		}
	}
	return nil, nil
}

func (m *mockPQParseJobRepo) GetPending(_ context.Context, limit int) ([]*models.ParseJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	var pending []*models.ParseJob
	for _, j := range m.jobs {
		if j.Status == models.ParseJobPending {
			pending = append(pending, j)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (m *mockPQParseJobRepo) UpdateStatus(_ context.Context, id string, status models.ParseJobStatus, errMsg string) error {
	if m.err != nil {
		return m.err
	}
	if j, ok := m.jobs[id]; ok {
		j.Status = status
		if errMsg != "" {
			j.ErrorMessage = &errMsg
		}
		return nil
	}
	return fmt.Errorf("not found")
}

func (m *mockPQParseJobRepo) Update(_ context.Context, job *models.ParseJob) error {
	if m.err != nil {
		return m.err
	}
	m.jobs[job.ID] = job
	return nil
}

func (m *mockPQParseJobRepo) Delete(_ context.Context, id string) error {
	delete(m.jobs, id)
	return nil
}

func (m *mockPQParseJobRepo) ListAll(_ context.Context, limit int) ([]*models.ParseJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	var all []*models.ParseJob
	for _, j := range m.jobs {
		all = append(all, j)
		if len(all) >= limit {
			break
		}
	}
	return all, nil
}

var _ repository.ParseJobRepositoryInterface = (*mockPQParseJobRepo)(nil)

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

func newMockPQMovieRepo() *mockPQMovieRepo {
	return &mockPQMovieRepo{movies: make(map[string]*models.Movie)}
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
func (m *mockPQMovieRepo) Delete(_ context.Context, _ string) error        { return nil }
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
func (m *mockPQMovieRepo) Count(_ context.Context) (int, error)            { return 0, nil }
func (m *mockPQMovieRepo) BulkCreate(_ context.Context, _ []*models.Movie) error {
	return nil
}
func (m *mockPQMovieRepo) FindByParseStatus(_ context.Context, _ models.ParseStatus) ([]models.Movie, error) {
	return nil, nil
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
func (m *mockPQMovieRepo) FindAllWithFilePath(_ context.Context) ([]models.Movie, error) {
	return nil, nil
}
func (m *mockPQMovieRepo) GetStats(_ context.Context) (*repository.MediaStats, error) {
	return &repository.MediaStats{}, nil
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
func (m *mockPQSeriesRepo) Update(_ context.Context, _ *models.Series) error { return nil }
func (m *mockPQSeriesRepo) Delete(_ context.Context, _ string) error         { return nil }
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
func (m *mockPQSeriesRepo) Count(_ context.Context) (int, error)            { return 0, nil }
func (m *mockPQSeriesRepo) BulkCreate(_ context.Context, _ []*models.Series) error {
	return nil
}
func (m *mockPQSeriesRepo) FindByParseStatus(_ context.Context, _ models.ParseStatus) ([]models.Series, error) {
	return nil, nil
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

func (m *mockPQEpisodeRepo) FindBySeriesSeasonEpisode(_ context.Context, seriesID string, season, episode int) (*models.Episode, error) {
	for _, ep := range m.episodes {
		if ep.SeriesID == seriesID && ep.SeasonNumber == season && ep.EpisodeNumber == episode {
			return ep, nil
		}
	}
	return nil, fmt.Errorf("episode S%02dE%02d for series %s: %w", season, episode, seriesID, repository.ErrEpisodeNotFound)
}

func (m *mockPQEpisodeRepo) Update(_ context.Context, _ *models.Episode) error { return nil }
func (m *mockPQEpisodeRepo) Delete(_ context.Context, _ string) error          { return nil }
func (m *mockPQEpisodeRepo) Upsert(_ context.Context, ep *models.Episode) error {
	if m.err != nil {
		return m.err
	}
	m.episodes[ep.ID] = ep
	return nil
}

var _ repository.EpisodeRepositoryInterface = (*mockPQEpisodeRepo)(nil)

func newTestParseQueueService(
	parseJobRepo repository.ParseJobRepositoryInterface,
	parserSvc ParserServiceInterface,
	metaSvc MetadataServiceInterface,
	movieRepo repository.MovieRepositoryInterface,
) *ParseQueueService {
	return NewParseQueueService(
		parseJobRepo, parserSvc, metaSvc, movieRepo,
		newMockPQSeriesRepo(), newMockPQSeasonRepo(), newMockPQEpisodeRepo(),
		slog.Default(),
	)
}

func newTestParseQueueServiceFull(
	parseJobRepo repository.ParseJobRepositoryInterface,
	parserSvc ParserServiceInterface,
	metaSvc MetadataServiceInterface,
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
	seasonRepo repository.SeasonRepositoryInterface,
	episodeRepo repository.EpisodeRepositoryInterface,
) *ParseQueueService {
	return NewParseQueueService(
		parseJobRepo, parserSvc, metaSvc, movieRepo,
		seriesRepo, seasonRepo, episodeRepo,
		slog.Default(),
	)
}

// --- Tests ---

func TestParseQueueService_QueueParseJob(t *testing.T) {
	repo := newMockPQParseJobRepo()
	svc := newTestParseQueueService(repo, nil, nil, nil)

	torrent := &qbittorrent.Torrent{
		Hash:     "abc123",
		Name:     "[SubGroup] Movie (2024).mkv",
		SavePath: "/downloads",
	}

	job, err := svc.QueueParseJob(context.Background(), torrent)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "abc123", job.TorrentHash)
	assert.Equal(t, "[SubGroup] Movie (2024).mkv", job.FileName)
	assert.Equal(t, "/downloads/[SubGroup] Movie (2024).mkv", job.FilePath)
	assert.Equal(t, models.ParseJobPending, job.Status)
	assert.NotEmpty(t, job.ID)
}

func TestParseQueueService_QueueParseJob_NilTorrent(t *testing.T) {
	svc := newTestParseQueueService(newMockPQParseJobRepo(), nil, nil, nil)

	_, err := svc.QueueParseJob(context.Background(), nil)
	assert.Error(t, err)
}

func TestParseQueueService_ProcessNextJob_NoPending(t *testing.T) {
	svc := newTestParseQueueService(newMockPQParseJobRepo(), nil, nil, nil)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err)
}

func TestParseQueueService_ProcessNextJob_Success(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:          "job-1",
		TorrentHash: "hash1",
		FilePath:    "/downloads/movie.mkv",
		FileName:    "[SubGroup] Movie (2024).mkv",
		Status:      models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			Title:        "Movie",
			CleanedTitle: "Movie",
			Year:         2024,
			MediaType:    parser.MediaTypeMovie,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items: []metadata.MetadataItem{
				{
					ID:            "12345",
					Title:         "Movie",
					OriginalTitle: "Movie Original",
					Year:          2024,
					PosterURL:     "https://image.tmdb.org/poster.jpg",
					Overview:      "A great movie",
					Genres:        []string{"Action"},
					Rating:        8.5,
					ReleaseDate:   "2024-01-15",
				},
			},
			Source: models.MetadataSourceTMDb,
		},
	}

	movieRepo := newMockPQMovieRepo()
	svc := newTestParseQueueService(repo, parserSvc, metaSvc, movieRepo)

	err := svc.ProcessNextJob(context.Background())
	require.NoError(t, err)

	// Verify job is completed
	assert.Equal(t, models.ParseJobCompleted, repo.jobs["job-1"].Status)
	assert.NotNil(t, repo.jobs["job-1"].MediaID)

	// Verify movie was created
	assert.Len(t, movieRepo.movies, 1)
}

func TestParseQueueService_ProcessNextJob_ParseFailed(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "unknown.file",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status: parser.ParseStatusFailed,
			ErrorMessage:  "could not parse filename",
		},
	}

	svc := newTestParseQueueService(repo, parserSvc, nil, nil)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err) // Should not propagate error

	assert.Equal(t, models.ParseJobFailed, repo.jobs["job-1"].Status)
	require.NotNil(t, repo.jobs["job-1"].ErrorMessage)
	assert.Contains(t, *repo.jobs["job-1"].ErrorMessage, "could not parse filename")
}

func TestParseQueueService_ProcessNextJob_MetadataSearchFailed(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "Movie.2024.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Movie",
			Year:         2024,
			MediaType:    parser.MediaTypeMovie,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchErr: fmt.Errorf("TMDb API timeout"),
	}

	svc := newTestParseQueueService(repo, parserSvc, metaSvc, nil)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, models.ParseJobFailed, repo.jobs["job-1"].Status)
}

func TestParseQueueService_ProcessNextJob_NoMetadataResults(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "Obscure.Film.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Obscure Film",
			MediaType:    parser.MediaTypeMovie,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{Items: nil},
	}

	svc := newTestParseQueueService(repo, parserSvc, metaSvc, nil)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, models.ParseJobFailed, repo.jobs["job-1"].Status)
}

func TestParseQueueService_RetryJob(t *testing.T) {
	repo := newMockPQParseJobRepo()
	errMsg := "previous error"
	repo.jobs["job-1"] = &models.ParseJob{
		ID:           "job-1",
		TorrentHash:  "hash1",
		Status:       models.ParseJobFailed,
		ErrorMessage: &errMsg,
		RetryCount:   1,
	}

	svc := newTestParseQueueService(repo, nil, nil, nil)

	err := svc.RetryJob(context.Background(), "job-1")
	require.NoError(t, err)

	assert.Equal(t, models.ParseJobPending, repo.jobs["job-1"].Status)
	assert.Equal(t, 2, repo.jobs["job-1"].RetryCount)
	assert.Nil(t, repo.jobs["job-1"].ErrorMessage)
}

func TestParseQueueService_RetryJob_NotFailed(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:     "job-1",
		Status: models.ParseJobProcessing,
	}

	svc := newTestParseQueueService(repo, nil, nil, nil)

	err := svc.RetryJob(context.Background(), "job-1")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrJobNotRetryable)
}

func TestParseQueueService_RetryJob_MaxRetriesReached(t *testing.T) {
	repo := newMockPQParseJobRepo()
	errMsg := "previous error"
	repo.jobs["job-1"] = &models.ParseJob{
		ID:           "job-1",
		TorrentHash:  "hash1",
		Status:       models.ParseJobFailed,
		ErrorMessage: &errMsg,
		RetryCount:   MaxRetryAttempts,
	}

	svc := newTestParseQueueService(repo, nil, nil, nil)

	err := svc.RetryJob(context.Background(), "job-1")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMaxRetriesReached)
}

func TestParseQueueService_GetJobStatus(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:          "job-1",
		TorrentHash: "abc123",
		Status:      models.ParseJobCompleted,
	}

	svc := newTestParseQueueService(repo, nil, nil, nil)

	job, err := svc.GetJobStatus(context.Background(), "abc123")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, models.ParseJobCompleted, job.Status)
}

func TestParseQueueService_GetJobStatus_NotFound(t *testing.T) {
	svc := newTestParseQueueService(newMockPQParseJobRepo(), nil, nil, nil)

	job, err := svc.GetJobStatus(context.Background(), "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, job)
}

// --- Per-method error mock for targeted error injection ---

type mockPQRepoWithMethodErrors struct {
	*mockPQParseJobRepo
	updateStatusErr error
	updateErr       error
}

func (m *mockPQRepoWithMethodErrors) UpdateStatus(ctx context.Context, id string, status models.ParseJobStatus, errMsg string) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	return m.mockPQParseJobRepo.UpdateStatus(ctx, id, status, errMsg)
}

func (m *mockPQRepoWithMethodErrors) Update(ctx context.Context, job *models.ParseJob) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.mockPQParseJobRepo.Update(ctx, job)
}

var _ repository.ParseJobRepositoryInterface = (*mockPQRepoWithMethodErrors)(nil)

// --- Expanded error path tests ---

func TestParseQueueService_QueueParseJob_RepoError(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.err = fmt.Errorf("db connection lost")
	svc := newTestParseQueueService(repo, nil, nil, nil)

	torrent := &qbittorrent.Torrent{Hash: "abc", Name: "test.mkv", SavePath: "/dl"}
	_, err := svc.QueueParseJob(context.Background(), torrent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create parse job")
	assert.Contains(t, err.Error(), "db connection lost")
}

func TestParseQueueService_ProcessNextJob_GetPendingError(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.err = fmt.Errorf("database locked")
	svc := newTestParseQueueService(repo, nil, nil, nil)

	err := svc.ProcessNextJob(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get pending jobs")
	assert.Contains(t, err.Error(), "database locked")
}

func TestParseQueueService_ProcessNextJob_MarkProcessingError(t *testing.T) {
	baseRepo := newMockPQParseJobRepo()
	baseRepo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "test.mkv",
		Status:   models.ParseJobPending,
	}
	repo := &mockPQRepoWithMethodErrors{
		mockPQParseJobRepo: baseRepo,
		updateStatusErr:    fmt.Errorf("disk full"),
	}
	svc := newTestParseQueueService(repo, nil, nil, nil)

	err := svc.ProcessNextJob(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark job processing")
	assert.Contains(t, err.Error(), "disk full")
}

func TestParseQueueService_ProcessNextJob_NilParseResult(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "test.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{result: nil}
	svc := newTestParseQueueService(repo, parserSvc, nil, nil)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err) // Should not propagate

	assert.Equal(t, models.ParseJobFailed, repo.jobs["job-1"].Status)
	require.NotNil(t, repo.jobs["job-1"].ErrorMessage)
	assert.Contains(t, *repo.jobs["job-1"].ErrorMessage, "filename parsing failed")
}

func TestParseQueueService_ProcessNextJob_MovieCreateError(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "Movie.2024.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Movie",
			Year:         2024,
			MediaType:    parser.MediaTypeMovie,
		},
	}
	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items:  []metadata.MetadataItem{{ID: "123", Title: "Movie", Year: 2024}},
			Source: models.MetadataSourceTMDb,
		},
	}
	movieRepo := newMockPQMovieRepo()
	movieRepo.err = fmt.Errorf("unique constraint violated")
	svc := newTestParseQueueService(repo, parserSvc, metaSvc, movieRepo)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err) // Doesn't propagate

	assert.Equal(t, models.ParseJobFailed, repo.jobs["job-1"].Status)
	require.NotNil(t, repo.jobs["job-1"].ErrorMessage)
	assert.Contains(t, *repo.jobs["job-1"].ErrorMessage, "create media entry failed")
}

func TestParseQueueService_ProcessNextJob_FinalUpdateError(t *testing.T) {
	baseRepo := newMockPQParseJobRepo()
	baseRepo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "Movie.2024.mkv",
		Status:   models.ParseJobPending,
	}
	repo := &mockPQRepoWithMethodErrors{
		mockPQParseJobRepo: baseRepo,
		updateErr:          fmt.Errorf("WAL corrupted"),
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Movie",
			Year:         2024,
			MediaType:    parser.MediaTypeMovie,
		},
	}
	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items:  []metadata.MetadataItem{{ID: "123", Title: "Movie", Year: 2024}},
			Source: models.MetadataSourceTMDb,
		},
	}
	movieRepo := newMockPQMovieRepo()
	svc := newTestParseQueueService(repo, parserSvc, metaSvc, movieRepo)

	err := svc.ProcessNextJob(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark job completed")
	assert.Contains(t, err.Error(), "WAL corrupted")
}

func TestParseQueueService_ListJobs(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{ID: "job-1", Status: models.ParseJobCompleted}
	repo.jobs["job-2"] = &models.ParseJob{ID: "job-2", Status: models.ParseJobPending}
	svc := newTestParseQueueService(repo, nil, nil, nil)

	jobs, err := svc.ListJobs(context.Background(), 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestParseQueueService_ListJobs_DefaultLimit(t *testing.T) {
	repo := newMockPQParseJobRepo()
	svc := newTestParseQueueService(repo, nil, nil, nil)

	// Zero limit should default to 50 (no error)
	jobs, err := svc.ListJobs(context.Background(), 0)
	assert.NoError(t, err)
	assert.Empty(t, jobs)

	// Negative limit should also default to 50
	jobs, err = svc.ListJobs(context.Background(), -1)
	assert.NoError(t, err)
	assert.Empty(t, jobs)
}

// --- TV Show Branch Tests ---

func TestParseQueueService_ProcessNextJob_TVShow_Success(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "[SubGroup] Show S01E05.mkv",
		FilePath: "/downloads/[SubGroup] Show S01E05.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			Title:        "Show",
			CleanedTitle: "Show",
			Year:         2024,
			MediaType:    parser.MediaTypeTVShow,
			Season:       1,
			Episode:      5,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items: []metadata.MetadataItem{
				{
					ID:            "99999",
					Title:         "Show",
					OriginalTitle: "Show Original",
					Year:          2024,
					PosterURL:     "https://image.tmdb.org/poster.jpg",
					Overview:      "A great show",
					Genres:        []string{"Drama"},
					Rating:        8.0,
					ReleaseDate:   "2024-01-01",
				},
			},
			Source: models.MetadataSourceTMDb,
		},
	}

	seriesRepo := newMockPQSeriesRepo()
	seasonRepo := newMockPQSeasonRepo()
	episodeRepo := newMockPQEpisodeRepo()

	svc := newTestParseQueueServiceFull(repo, parserSvc, metaSvc, nil, seriesRepo, seasonRepo, episodeRepo)

	err := svc.ProcessNextJob(context.Background())
	require.NoError(t, err)

	// Verify job is completed
	assert.Equal(t, models.ParseJobCompleted, repo.jobs["job-1"].Status)
	assert.NotNil(t, repo.jobs["job-1"].MediaID)

	// Verify series was created
	assert.Len(t, seriesRepo.series, 1)

	// Verify season was created
	assert.Len(t, seasonRepo.seasons, 1)
	for _, s := range seasonRepo.seasons {
		assert.Equal(t, 1, s.SeasonNumber)
	}

	// Verify episode was created
	assert.Len(t, episodeRepo.episodes, 1)
	for _, ep := range episodeRepo.episodes {
		assert.Equal(t, 1, ep.SeasonNumber)
		assert.Equal(t, 5, ep.EpisodeNumber)
		assert.True(t, ep.SeasonID.Valid)
	}
}

func TestParseQueueService_ProcessNextJob_TVShow_ExistingSeries(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "[SubGroup] Show S01E06.mkv",
		FilePath: "/downloads/[SubGroup] Show S01E06.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Show",
			MediaType:    parser.MediaTypeTVShow,
			Season:       1,
			Episode:      6,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items: []metadata.MetadataItem{
				{
					ID:    "99999",
					Title: "Show",
				},
			},
			Source: models.MetadataSourceTMDb,
		},
	}

	// Pre-populate series with TMDb ID 99999
	seriesRepo := newMockPQSeriesRepo()
	seriesRepo.series["existing-series"] = &models.Series{
		ID:     "existing-series",
		Title:  "Show",
		TMDbID: models.NewNullInt64(99999),
	}

	seasonRepo := newMockPQSeasonRepo()
	episodeRepo := newMockPQEpisodeRepo()

	svc := newTestParseQueueServiceFull(repo, parserSvc, metaSvc, nil, seriesRepo, seasonRepo, episodeRepo)

	err := svc.ProcessNextJob(context.Background())
	require.NoError(t, err)

	// Should reuse existing series, not create a new one
	assert.Len(t, seriesRepo.series, 1)
	assert.NotNil(t, repo.jobs["job-1"].MediaID)
	assert.Equal(t, "existing-series", *repo.jobs["job-1"].MediaID)
}

func TestParseQueueService_ProcessNextJob_TVShow_SeasonReuse(t *testing.T) {
	repo := newMockPQParseJobRepo()
	// Two episodes from the same season
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "[SubGroup] Show S01E05.mkv",
		FilePath: "/downloads/S01E05.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Show",
			MediaType:    parser.MediaTypeTVShow,
			Season:       1,
			Episode:      5,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items:  []metadata.MetadataItem{{ID: "99999", Title: "Show"}},
			Source: models.MetadataSourceTMDb,
		},
	}

	seriesRepo := newMockPQSeriesRepo()
	seasonRepo := newMockPQSeasonRepo()
	episodeRepo := newMockPQEpisodeRepo()

	// Pre-populate existing series and season
	seriesRepo.series["existing-series"] = &models.Series{
		ID:     "existing-series",
		Title:  "Show",
		TMDbID: models.NewNullInt64(99999),
	}
	seasonRepo.seasons["existing-season"] = &models.Season{
		ID:           "existing-season",
		SeriesID:     "existing-series",
		SeasonNumber: 1,
	}

	svc := newTestParseQueueServiceFull(repo, parserSvc, metaSvc, nil, seriesRepo, seasonRepo, episodeRepo)

	err := svc.ProcessNextJob(context.Background())
	require.NoError(t, err)

	// Should reuse existing series — no new series created
	assert.Len(t, seriesRepo.series, 1)

	// Should reuse existing season — no new season created
	assert.Len(t, seasonRepo.seasons, 1)

	// Episode should reference the existing season
	assert.Len(t, episodeRepo.episodes, 1)
	for _, ep := range episodeRepo.episodes {
		assert.Equal(t, "existing-series", ep.SeriesID)
		assert.Equal(t, "existing-season", ep.SeasonID.String)
		assert.True(t, ep.SeasonID.Valid)
	}
}

func TestParseQueueService_ProcessNextJob_TVShow_SeriesCreateError(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "[SubGroup] Show S01E01.mkv",
		FilePath: "/downloads/show.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Show",
			MediaType:    parser.MediaTypeTVShow,
			Season:       1,
			Episode:      1,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items:  []metadata.MetadataItem{{ID: "99999", Title: "Show"}},
			Source: models.MetadataSourceTMDb,
		},
	}

	seriesRepo := newMockPQSeriesRepo()
	seriesRepo.err = fmt.Errorf("database connection lost")
	seasonRepo := newMockPQSeasonRepo()
	episodeRepo := newMockPQEpisodeRepo()

	svc := newTestParseQueueServiceFull(repo, parserSvc, metaSvc, nil, seriesRepo, seasonRepo, episodeRepo)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err) // ProcessNextJob returns nil but marks job failed

	// Job should be marked as failed
	assert.Equal(t, models.ParseJobFailed, repo.jobs["job-1"].Status)
	assert.NotNil(t, repo.jobs["job-1"].ErrorMessage)
	assert.Contains(t, *repo.jobs["job-1"].ErrorMessage, "create media entry failed")
}

func TestParseQueueService_ProcessNextJob_TVShow_SeasonCreateError(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "[SubGroup] Show S01E01.mkv",
		FilePath: "/downloads/show.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Show",
			MediaType:    parser.MediaTypeTVShow,
			Season:       1,
			Episode:      1,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items:  []metadata.MetadataItem{{ID: "99999", Title: "Show"}},
			Source: models.MetadataSourceTMDb,
		},
	}

	seriesRepo := newMockPQSeriesRepo()
	seasonRepo := newMockPQSeasonRepo()
	seasonRepo.err = fmt.Errorf("season table locked")
	episodeRepo := newMockPQEpisodeRepo()

	svc := newTestParseQueueServiceFull(repo, parserSvc, metaSvc, nil, seriesRepo, seasonRepo, episodeRepo)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err) // ProcessNextJob returns nil but marks job failed

	// Job should be marked as failed
	assert.Equal(t, models.ParseJobFailed, repo.jobs["job-1"].Status)
	assert.Contains(t, *repo.jobs["job-1"].ErrorMessage, "create media entry failed")
}

func TestParseQueueService_ProcessNextJob_TVShow_EpisodeCreateError(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "[SubGroup] Show S01E01.mkv",
		FilePath: "/downloads/show.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Show",
			MediaType:    parser.MediaTypeTVShow,
			Season:       1,
			Episode:      1,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items:  []metadata.MetadataItem{{ID: "99999", Title: "Show"}},
			Source: models.MetadataSourceTMDb,
		},
	}

	seriesRepo := newMockPQSeriesRepo()
	seasonRepo := newMockPQSeasonRepo()
	episodeRepo := newMockPQEpisodeRepo()
	episodeRepo.err = fmt.Errorf("episode insert failed")

	svc := newTestParseQueueServiceFull(repo, parserSvc, metaSvc, nil, seriesRepo, seasonRepo, episodeRepo)

	err := svc.ProcessNextJob(context.Background())
	assert.NoError(t, err) // ProcessNextJob returns nil but marks job failed

	// Job should be marked as failed
	assert.Equal(t, models.ParseJobFailed, repo.jobs["job-1"].Status)
	assert.Contains(t, *repo.jobs["job-1"].ErrorMessage, "create media entry failed")
}

func TestParseQueueService_ProcessNextJob_TVShow_SeasonMetadataFromSeasonsJSON(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "[SubGroup] Show S02E01.mkv",
		FilePath: "/downloads/show-s02e01.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Show",
			MediaType:    parser.MediaTypeTVShow,
			Season:       2,
			Episode:      1,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items:  []metadata.MetadataItem{{ID: "99999", Title: "Show"}},
			Source: models.MetadataSourceTMDb,
		},
	}

	// Pre-populate series with SeasonsJSON containing season 2 metadata
	seriesRepo := newMockPQSeriesRepo()
	existingSeries := &models.Series{
		ID:     "existing-series",
		Title:  "Show",
		TMDbID: models.NewNullInt64(99999),
	}
	existingSeries.SetSeasons([]models.SeasonSummary{
		{
			ID:           100,
			SeasonNumber: 1,
			Name:         "Season 1",
			PosterPath:   "/posters/s1.jpg",
			AirDate:      "2023-01-01",
			EpisodeCount: 10,
		},
		{
			ID:           200,
			SeasonNumber: 2,
			Name:         "Season 2",
			Overview:     "The second season",
			PosterPath:   "/posters/s2.jpg",
			AirDate:      "2024-01-01",
			EpisodeCount: 8,
		},
	})
	seriesRepo.series["existing-series"] = existingSeries

	seasonRepo := newMockPQSeasonRepo()
	episodeRepo := newMockPQEpisodeRepo()

	svc := newTestParseQueueServiceFull(repo, parserSvc, metaSvc, nil, seriesRepo, seasonRepo, episodeRepo)

	err := svc.ProcessNextJob(context.Background())
	require.NoError(t, err)

	// Verify season was created with metadata from SeasonsJSON
	assert.Len(t, seasonRepo.seasons, 1)
	for _, s := range seasonRepo.seasons {
		assert.Equal(t, 2, s.SeasonNumber)
		assert.Equal(t, "existing-series", s.SeriesID)
		assert.Equal(t, int64(200), s.TMDbID.Int64)
		assert.True(t, s.TMDbID.Valid)
		assert.Equal(t, "Season 2", s.Name.String)
		assert.Equal(t, "The second season", s.Overview.String)
		assert.Equal(t, "/posters/s2.jpg", s.PosterPath.String)
		assert.Equal(t, "2024-01-01", s.AirDate.String)
		assert.Equal(t, int64(8), s.EpisodeCount.Int64)
	}
}

func TestParseQueueService_ProcessNextJob_TVShow_SpecialsSeason0(t *testing.T) {
	repo := newMockPQParseJobRepo()
	repo.jobs["job-1"] = &models.ParseJob{
		ID:       "job-1",
		FileName: "[SubGroup] Show S00E01 Special.mkv",
		FilePath: "/downloads/special.mkv",
		Status:   models.ParseJobPending,
	}

	parserSvc := &mockPQParserService{
		result: &parser.ParseResult{
			Status:       parser.ParseStatusSuccess,
			CleanedTitle: "Show",
			MediaType:    parser.MediaTypeTVShow,
			Season:       0,
			Episode:      1,
		},
	}

	metaSvc := &mockPQMetadataService{
		searchResult: &metadata.SearchResult{
			Items:  []metadata.MetadataItem{{ID: "99999", Title: "Show"}},
			Source: models.MetadataSourceTMDb,
		},
	}

	seriesRepo := newMockPQSeriesRepo()
	seasonRepo := newMockPQSeasonRepo()
	episodeRepo := newMockPQEpisodeRepo()

	svc := newTestParseQueueServiceFull(repo, parserSvc, metaSvc, nil, seriesRepo, seasonRepo, episodeRepo)

	err := svc.ProcessNextJob(context.Background())
	require.NoError(t, err)

	// Verify season 0 (Specials) was created
	for _, s := range seasonRepo.seasons {
		assert.Equal(t, 0, s.SeasonNumber)
	}
}
