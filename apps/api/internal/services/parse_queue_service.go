package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// MaxRetryAttempts is the maximum number of times a parse job can be retried.
const MaxRetryAttempts = 4

// Sentinel errors for parse queue operations.
var (
	ErrJobNotRetryable = errors.New("can only retry failed jobs")
	ErrMaxRetriesReached = errors.New("maximum retry attempts reached")
)

// ParseQueueServiceInterface defines the contract for parse queue operations.
type ParseQueueServiceInterface interface {
	QueueParseJob(ctx context.Context, torrent *qbittorrent.Torrent) (*models.ParseJob, error)
	ProcessNextJob(ctx context.Context) error
	GetJobStatus(ctx context.Context, torrentHash string) (*models.ParseJob, error)
	RetryJob(ctx context.Context, jobID string) error
	ListJobs(ctx context.Context, limit int) ([]*models.ParseJob, error)
}

// ParseQueueService manages the queue of parse jobs for completed downloads.
type ParseQueueService struct {
	parseJobRepo    repository.ParseJobRepositoryInterface
	parserService   ParserServiceInterface
	metadataService MetadataServiceInterface
	movieRepo       repository.MovieRepositoryInterface
	logger          *slog.Logger
}

// NewParseQueueService creates a new ParseQueueService.
func NewParseQueueService(
	parseJobRepo repository.ParseJobRepositoryInterface,
	parserService ParserServiceInterface,
	metadataService MetadataServiceInterface,
	movieRepo repository.MovieRepositoryInterface,
	logger *slog.Logger,
) *ParseQueueService {
	return &ParseQueueService{
		parseJobRepo:    parseJobRepo,
		parserService:   parserService,
		metadataService: metadataService,
		movieRepo:       movieRepo,
		logger:          logger,
	}
}

// QueueParseJob creates a new parse job for a completed torrent.
func (s *ParseQueueService) QueueParseJob(ctx context.Context, torrent *qbittorrent.Torrent) (*models.ParseJob, error) {
	if torrent == nil {
		return nil, fmt.Errorf("torrent cannot be nil")
	}

	job := &models.ParseJob{
		ID:          uuid.New().String(),
		TorrentHash: torrent.Hash,
		FilePath:    filepath.Join(torrent.SavePath, torrent.Name),
		FileName:    torrent.Name,
		Status:      models.ParseJobPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.parseJobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create parse job: %w", err)
	}

	s.logger.Info("Parse job queued",
		"job_id", job.ID,
		"torrent_hash", torrent.Hash,
		"filename", torrent.Name,
	)

	return job, nil
}

// ProcessNextJob fetches the next pending job and runs the parse pipeline.
func (s *ParseQueueService) ProcessNextJob(ctx context.Context) error {
	jobs, err := s.parseJobRepo.GetPending(ctx, 1)
	if err != nil {
		return fmt.Errorf("get pending jobs: %w", err)
	}
	if len(jobs) == 0 {
		return nil
	}

	job := jobs[0]

	if err := s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobProcessing, ""); err != nil {
		return fmt.Errorf("mark job processing: %w", err)
	}

	s.logger.Info("Processing parse job",
		"job_id", job.ID,
		"filename", job.FileName,
	)

	// Step 1: Parse filename
	parseResult := s.parserService.ParseFilenameWithContext(ctx, job.FileName)
	if parseResult == nil || parseResult.Status == parser.ParseStatusFailed {
		errMsg := "filename parsing failed"
		if parseResult != nil && parseResult.ErrorMessage != "" {
			errMsg = parseResult.ErrorMessage
		}
		s.logger.Error("Parsing failed", "job_id", job.ID, "error", errMsg)
		if statusErr := s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobFailed, errMsg); statusErr != nil {
			s.logger.Error("Failed to update job status to failed", "job_id", job.ID, "error", statusErr)
		}
		return nil
	}

	// Step 2: Search metadata
	mediaType := "movie"
	if parseResult.MediaType == parser.MediaTypeTVShow {
		mediaType = "tv"
	}

	searchReq := &SearchMetadataRequest{
		Query:     parseResult.CleanedTitle,
		MediaType: mediaType,
		Year:      parseResult.Year,
	}

	searchResult, _, err := s.metadataService.SearchMetadata(ctx, searchReq)
	if err != nil {
		errMsg := fmt.Sprintf("metadata search failed: %s", err.Error())
		s.logger.Error("Metadata search failed", "job_id", job.ID, "error", err)
		if statusErr := s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobFailed, errMsg); statusErr != nil {
			s.logger.Error("Failed to update job status to failed", "job_id", job.ID, "error", statusErr)
		}
		return nil
	}

	if searchResult == nil || !searchResult.HasResults() {
		errMsg := "no metadata found"
		s.logger.Warn("No metadata found", "job_id", job.ID, "query", parseResult.CleanedTitle)
		if statusErr := s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobFailed, errMsg); statusErr != nil {
			s.logger.Error("Failed to update job status to failed", "job_id", job.ID, "error", statusErr)
		}
		return nil
	}

	// Step 3: Create media entry from best match
	bestMatch := searchResult.Items[0]
	mediaID := uuid.New().String()

	movie := &models.Movie{
		ID:         mediaID,
		Title:      bestMatch.Title,
		TMDbID:     sql.NullInt64{Int64: parseProviderID(bestMatch.ID), Valid: bestMatch.ID != ""},
		PosterPath: sql.NullString{String: bestMatch.PosterURL, Valid: bestMatch.PosterURL != ""},
		Overview:   sql.NullString{String: bestMatch.Overview, Valid: bestMatch.Overview != ""},
		Genres:     bestMatch.Genres,
		FilePath:   sql.NullString{String: job.FilePath, Valid: true},
		ParseStatus:    models.ParseStatusSuccess,
		MetadataSource: sql.NullString{String: string(searchResult.Source), Valid: true},
	}

	if bestMatch.ReleaseDate != "" {
		movie.ReleaseDate = bestMatch.ReleaseDate
	}
	if bestMatch.Rating > 0 {
		movie.VoteAverage = sql.NullFloat64{Float64: bestMatch.Rating, Valid: true}
	}
	if bestMatch.OriginalTitle != "" {
		movie.OriginalTitle = sql.NullString{String: bestMatch.OriginalTitle, Valid: true}
	}

	if err := s.movieRepo.Create(ctx, movie); err != nil {
		errMsg := fmt.Sprintf("create media entry failed: %s", err.Error())
		s.logger.Error("Media creation failed", "job_id", job.ID, "error", err)
		if statusErr := s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobFailed, errMsg); statusErr != nil {
			s.logger.Error("Failed to update job status to failed", "job_id", job.ID, "error", statusErr)
		}
		return nil
	}

	// Step 4: Mark job as completed
	job.MediaID = &mediaID
	job.Status = models.ParseJobCompleted
	now := time.Now()
	job.CompletedAt = &now

	if err := s.parseJobRepo.Update(ctx, job); err != nil {
		return fmt.Errorf("mark job completed: %w", err)
	}

	s.logger.Info("Parse job completed",
		"job_id", job.ID,
		"media_id", mediaID,
		"title", searchResult.Items[0].Title,
	)

	return nil
}

// GetJobStatus retrieves the current status of a parse job by torrent hash.
func (s *ParseQueueService) GetJobStatus(ctx context.Context, torrentHash string) (*models.ParseJob, error) {
	return s.parseJobRepo.GetByTorrentHash(ctx, torrentHash)
}

// RetryJob resets a failed job to pending for re-processing.
func (s *ParseQueueService) RetryJob(ctx context.Context, jobID string) error {
	job, err := s.parseJobRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	if job.Status != models.ParseJobFailed {
		return fmt.Errorf("%w: current status: %s", ErrJobNotRetryable, job.Status)
	}

	if job.RetryCount >= MaxRetryAttempts {
		return fmt.Errorf("%w: %d/%d attempts used", ErrMaxRetriesReached, job.RetryCount, MaxRetryAttempts)
	}

	job.Status = models.ParseJobPending
	job.RetryCount++
	job.ErrorMessage = nil
	job.CompletedAt = nil
	job.UpdatedAt = time.Now()

	return s.parseJobRepo.Update(ctx, job)
}

// ListJobs retrieves all parse jobs with a limit.
func (s *ParseQueueService) ListJobs(ctx context.Context, limit int) ([]*models.ParseJob, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.parseJobRepo.ListAll(ctx, limit)
}

// parseProviderID extracts an int64 from a provider ID string.
func parseProviderID(id string) int64 {
	// TMDb IDs are numeric strings
	var n int64
	fmt.Sscanf(strings.TrimSpace(id), "%d", &n)
	return n
}

// Compile-time interface verification
var _ ParseQueueServiceInterface = (*ParseQueueService)(nil)
