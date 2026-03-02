package services

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

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

var _ repository.MovieRepositoryInterface = (*mockPQMovieRepo)(nil)

func newTestParseQueueService(
	parseJobRepo repository.ParseJobRepositoryInterface,
	parserSvc ParserServiceInterface,
	metaSvc MetadataServiceInterface,
	movieRepo repository.MovieRepositoryInterface,
) *ParseQueueService {
	return NewParseQueueService(parseJobRepo, parserSvc, metaSvc, movieRepo, slog.Default())
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
	assert.Contains(t, err.Error(), "can only retry failed jobs")
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
