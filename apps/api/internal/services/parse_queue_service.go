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
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// MaxRetryAttempts is the maximum number of times a parse job can be retried.
const MaxRetryAttempts = 4

// Sentinel errors for parse queue operations.
var (
	ErrJobNotRetryable   = errors.New("can only retry failed jobs")
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
	seriesRepo      repository.SeriesRepositoryInterface
	seasonRepo      repository.SeasonRepositoryInterface
	episodeRepo     repository.EpisodeRepositoryInterface
	logger          *slog.Logger
}

// NewParseQueueService creates a new ParseQueueService.
func NewParseQueueService(
	parseJobRepo repository.ParseJobRepositoryInterface,
	parserService ParserServiceInterface,
	metadataService MetadataServiceInterface,
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
	seasonRepo repository.SeasonRepositoryInterface,
	episodeRepo repository.EpisodeRepositoryInterface,
	logger *slog.Logger,
) *ParseQueueService {
	return &ParseQueueService{
		parseJobRepo:    parseJobRepo,
		parserService:   parserService,
		metadataService: metadataService,
		movieRepo:       movieRepo,
		seriesRepo:      seriesRepo,
		seasonRepo:      seasonRepo,
		episodeRepo:     episodeRepo,
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

	// Step 3: Create media entry from best match — branch on mediaType
	bestMatch := searchResult.Items[0]
	var mediaID string
	var createErr error

	if mediaType == "tv" {
		mediaID, createErr = s.createTVEntryFromMatch(ctx, bestMatch, searchResult, job, parseResult)
	} else {
		mediaID, createErr = s.createMovieFromMatch(ctx, bestMatch, searchResult, job)
	}

	if createErr != nil {
		errMsg := fmt.Sprintf("create media entry failed: %s", createErr.Error())
		s.logger.Error("Media creation failed", "job_id", job.ID, "media_type", mediaType, "error", createErr)
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
		"media_type", mediaType,
		"title", bestMatch.Title,
	)

	return nil
}

// createMovieFromMatch creates a Movie record from the best metadata match.
func (s *ParseQueueService) createMovieFromMatch(
	ctx context.Context,
	bestMatch metadata.MetadataItem,
	searchResult *metadata.SearchResult,
	job *models.ParseJob,
) (string, error) {
	mediaID := uuid.New().String()

	movie := &models.Movie{
		ID:         mediaID,
		Title:      bestMatch.Title,
		TMDbID:     models.NullInt64{sql.NullInt64{Int64: parseProviderID(bestMatch.ID), Valid: bestMatch.ID != ""}},
		PosterPath: models.NullString{sql.NullString{String: bestMatch.PosterURL, Valid: bestMatch.PosterURL != ""}},
		Overview:   models.NullString{sql.NullString{String: bestMatch.Overview, Valid: bestMatch.Overview != ""}},
		Genres:     bestMatch.Genres,
		FilePath:   models.NewNullString(job.FilePath),
		ParseStatus:    models.ParseStatusSuccess,
		MetadataSource: models.NewNullString(string(searchResult.Source)),
	}

	if bestMatch.ReleaseDate != "" {
		movie.ReleaseDate = bestMatch.ReleaseDate
	}
	if bestMatch.Rating > 0 {
		movie.VoteAverage = models.NewNullFloat64(bestMatch.Rating)
	}
	if bestMatch.OriginalTitle != "" {
		movie.OriginalTitle = models.NewNullString(bestMatch.OriginalTitle)
	}

	if err := s.movieRepo.Create(ctx, movie); err != nil {
		return "", fmt.Errorf("create movie: %w", err)
	}

	return mediaID, nil
}

// createTVEntryFromMatch creates Series, Season, and Episode records from the best metadata match.
func (s *ParseQueueService) createTVEntryFromMatch(
	ctx context.Context,
	bestMatch metadata.MetadataItem,
	searchResult *metadata.SearchResult,
	job *models.ParseJob,
	parseResult *parser.ParseResult,
) (string, error) {
	tmdbID := parseProviderID(bestMatch.ID)

	// 1. Upsert Series
	seriesID, err := s.upsertSeries(ctx, bestMatch, searchResult, tmdbID)
	if err != nil {
		return "", fmt.Errorf("upsert series: %w", err)
	}

	// 2. Upsert Season
	seasonNumber := parseResult.Season
	seasonID, err := s.upsertSeason(ctx, seriesID, seasonNumber)
	if err != nil {
		return "", fmt.Errorf("upsert season: %w", err)
	}

	// 3. Upsert Episode
	episodeNumber := parseResult.Episode
	episode := &models.Episode{
		ID:            uuid.New().String(),
		SeriesID:      seriesID,
		SeasonID:      models.NewNullString(seasonID),
		SeasonNumber:  seasonNumber,
		EpisodeNumber: episodeNumber,
		Title:         models.NullString{sql.NullString{String: bestMatch.Title, Valid: bestMatch.Title != ""}},
		FilePath:      models.NewNullString(job.FilePath),
	}

	if err := s.episodeRepo.Upsert(ctx, episode); err != nil {
		return "", fmt.Errorf("upsert episode: %w", err)
	}

	s.logger.Info("TV entry created",
		"series_id", seriesID,
		"season_id", seasonID,
		"season", seasonNumber,
		"episode", episodeNumber,
	)

	return seriesID, nil
}

// upsertSeries finds an existing series by TMDb ID or creates a new one.
func (s *ParseQueueService) upsertSeries(
	ctx context.Context,
	bestMatch metadata.MetadataItem,
	searchResult *metadata.SearchResult,
	tmdbID int64,
) (string, error) {
	// Check if series already exists by TMDb ID
	if tmdbID > 0 {
		existing, err := s.seriesRepo.FindByTMDbID(ctx, tmdbID)
		if err == nil && existing != nil {
			return existing.ID, nil
		}
	}

	// Create new series
	seriesID := uuid.New().String()
	series := &models.Series{
		ID:         seriesID,
		Title:      bestMatch.Title,
		TMDbID:     models.NullInt64{sql.NullInt64{Int64: tmdbID, Valid: tmdbID > 0}},
		PosterPath: models.NullString{sql.NullString{String: bestMatch.PosterURL, Valid: bestMatch.PosterURL != ""}},
		Overview:   models.NullString{sql.NullString{String: bestMatch.Overview, Valid: bestMatch.Overview != ""}},
		Genres:     bestMatch.Genres,
		ParseStatus:    models.ParseStatusSuccess,
		MetadataSource: models.NewNullString(string(searchResult.Source)),
	}

	if bestMatch.ReleaseDate != "" {
		series.FirstAirDate = bestMatch.ReleaseDate
	}
	if bestMatch.Rating > 0 {
		series.VoteAverage = models.NewNullFloat64(bestMatch.Rating)
	}
	if bestMatch.OriginalTitle != "" {
		series.OriginalTitle = models.NewNullString(bestMatch.OriginalTitle)
	}

	if err := s.seriesRepo.Create(ctx, series); err != nil {
		return "", fmt.Errorf("create series: %w", err)
	}

	return seriesID, nil
}

// upsertSeason finds an existing season or creates a new one.
// Season data is sourced from the Series' SeasonsJSON (TMDb summary) when available.
func (s *ParseQueueService) upsertSeason(
	ctx context.Context,
	seriesID string,
	seasonNumber int,
) (string, error) {
	// Check if season already exists
	existing, err := s.seasonRepo.FindBySeriesAndNumber(ctx, seriesID, seasonNumber)
	if err == nil && existing != nil {
		return existing.ID, nil
	}

	// Try to get season data from the Series' SeasonsJSON
	var seasonSummary *models.SeasonSummary
	series, seriesErr := s.seriesRepo.FindByID(ctx, seriesID)
	if seriesErr == nil && series != nil {
		seasons, parseErr := series.GetSeasons()
		if parseErr == nil {
			for i := range seasons {
				if seasons[i].SeasonNumber == seasonNumber {
					seasonSummary = &seasons[i]
					break
				}
			}
		}
	}

	// Create season
	season := &models.Season{
		ID:           uuid.New().String(),
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
	}

	if seasonSummary != nil {
		season.TMDbID = models.NullInt64{sql.NullInt64{Int64: int64(seasonSummary.ID), Valid: seasonSummary.ID > 0}}
		season.Name = models.NullString{sql.NullString{String: seasonSummary.Name, Valid: seasonSummary.Name != ""}}
		season.Overview = models.NullString{sql.NullString{String: seasonSummary.Overview, Valid: seasonSummary.Overview != ""}}
		season.PosterPath = models.NullString{sql.NullString{String: seasonSummary.PosterPath, Valid: seasonSummary.PosterPath != ""}}
		season.AirDate = models.NullString{sql.NullString{String: seasonSummary.AirDate, Valid: seasonSummary.AirDate != ""}}
		season.EpisodeCount = models.NullInt64{sql.NullInt64{Int64: int64(seasonSummary.EpisodeCount), Valid: seasonSummary.EpisodeCount > 0}}
	}

	if err := s.seasonRepo.Create(ctx, season); err != nil {
		return "", fmt.Errorf("create season: %w", err)
	}

	return season.ID, nil
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
