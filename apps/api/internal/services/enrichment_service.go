package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/tmdb"
)

// EnrichmentProgress represents the current state of an active enrichment
type EnrichmentProgress struct {
	Total        int    `json:"total"`
	Processed    int    `json:"processed"`
	Succeeded    int    `json:"succeeded"`
	Failed       int    `json:"failed"`
	Skipped      int    `json:"skipped"`
	CurrentTitle string `json:"current_title"`
	IsActive     bool   `json:"is_active"`
}

// EnrichmentResult contains the outcome of a completed enrichment
type EnrichmentResult struct {
	Total     int    `json:"total"`
	Succeeded int    `json:"succeeded"`
	Failed    int    `json:"failed"`
	Skipped   int    `json:"skipped"`
	Duration  string `json:"duration"`
}

// EnrichmentService processes unenriched movies by parsing filenames
// and searching TMDB for metadata.
type EnrichmentService struct {
	movieRepo       repository.MovieRepositoryInterface
	parserService   ParserServiceInterface
	metadataService MetadataServiceInterface
	nfoReader       *NFOReaderService
	tmdbService     TMDbServiceInterface
	sseHub          *sse.Hub
	logger          *slog.Logger

	mu          sync.Mutex
	isEnriching bool
	cancelChan  chan struct{}
	progress    EnrichmentProgress
}

// NewEnrichmentService creates a new EnrichmentService.
func NewEnrichmentService(
	movieRepo repository.MovieRepositoryInterface,
	parserService ParserServiceInterface,
	metadataService MetadataServiceInterface,
	nfoReader *NFOReaderService,
	tmdbService TMDbServiceInterface,
	sseHub *sse.Hub,
	logger *slog.Logger,
) *EnrichmentService {
	if logger == nil {
		logger = slog.Default()
	}
	return &EnrichmentService{
		movieRepo:       movieRepo,
		parserService:   parserService,
		metadataService: metadataService,
		nfoReader:       nfoReader,
		tmdbService:     tmdbService,
		sseHub:          sseHub,
		logger:          logger.With("service", "enrichment"),
	}
}

// IsEnrichmentActive returns whether enrichment is currently running.
func (s *EnrichmentService) IsEnrichmentActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isEnriching
}

// GetProgress returns the current enrichment progress.
func (s *EnrichmentService) GetProgress() EnrichmentProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

// CancelEnrichment cancels a running enrichment.
func (s *EnrichmentService) CancelEnrichment() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isEnriching {
		return fmt.Errorf("ENRICHMENT_NOT_ACTIVE: no enrichment is currently active")
	}
	close(s.cancelChan)
	return nil
}

// StartEnrichment finds all unenriched movies and processes them.
// Thread-safe: only one enrichment can run at a time.
func (s *EnrichmentService) StartEnrichment(ctx context.Context) (*EnrichmentResult, error) {
	s.mu.Lock()
	if s.isEnriching {
		s.mu.Unlock()
		return nil, fmt.Errorf("ENRICHMENT_ALREADY_RUNNING: an enrichment is already in progress")
	}
	s.isEnriching = true
	s.cancelChan = make(chan struct{})
	s.progress = EnrichmentProgress{IsActive: true}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isEnriching = false
		s.progress.IsActive = false
		s.mu.Unlock()
	}()

	startedAt := time.Now()

	// Find all unenriched movies (parse_status="" or "pending")
	movies, err := s.findUnenrichedMovies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find unenriched movies: %w", err)
	}

	s.mu.Lock()
	s.progress.Total = len(movies)
	s.mu.Unlock()

	s.logger.Info("enrichment started", "total_movies", len(movies))

	for i := range movies {
		// Check cancellation
		select {
		case <-s.cancelChan:
			s.logger.Info("enrichment cancelled", "processed", i)
			return s.buildResult(startedAt), nil
		case <-ctx.Done():
			s.logger.Info("enrichment context cancelled", "processed", i)
			return s.buildResult(startedAt), nil
		default:
		}

		movie := &movies[i]

		s.mu.Lock()
		s.progress.CurrentTitle = movie.Title
		s.mu.Unlock()

		if err := s.enrichMovie(ctx, movie); err != nil {
			s.logger.Warn("enrichment failed for movie",
				"id", movie.ID,
				"title", movie.Title,
				"error", err,
			)
			s.mu.Lock()
			s.progress.Failed++
			s.progress.Processed++
			s.mu.Unlock()
		} else {
			s.mu.Lock()
			s.progress.Succeeded++
			s.progress.Processed++
			s.mu.Unlock()
		}

		// Broadcast progress every 5 movies
		if (i+1)%5 == 0 || i == len(movies)-1 {
			s.broadcastProgress()
		}
	}

	result := s.buildResult(startedAt)
	s.broadcastComplete(result)

	s.logger.Info("enrichment completed",
		"total", result.Total,
		"succeeded", result.Succeeded,
		"failed", result.Failed,
		"duration", result.Duration,
	)

	return result, nil
}

// findUnenrichedMovies queries for movies with empty or pending parse_status.
func (s *EnrichmentService) findUnenrichedMovies(ctx context.Context) ([]models.Movie, error) {
	pending, err := s.movieRepo.FindByParseStatus(ctx, models.ParseStatusPending)
	if err != nil {
		return nil, fmt.Errorf("query pending: %w", err)
	}

	empty, err := s.movieRepo.FindByParseStatus(ctx, models.ParseStatus(""))
	if err != nil {
		return nil, fmt.Errorf("query empty: %w", err)
	}

	return append(pending, empty...), nil
}

// enrichMovie processes a single movie: NFO sidecar → parse filename → search TMDB → update record.
func (s *EnrichmentService) enrichMovie(ctx context.Context, movie *models.Movie) error {
	filename := movie.Title

	// Step 0: NFO sidecar detection — runs BEFORE filename parsing
	if s.nfoReader != nil && movie.FilePath.Valid && movie.FilePath.String != "" {
		if nfoEnriched, err := s.tryNFOEnrichment(ctx, movie); err != nil {
			s.logger.Warn("NFO parse failed, falling back to AI parse",
				"file", movie.FilePath.String, "error", err)
		} else if nfoEnriched {
			return nil // NFO enrichment succeeded, skip AI parse
		}
	}

	// Step 1: Parse filename
	parseResult := s.parserService.ParseFilenameWithContext(ctx, filename)
	if parseResult == nil || parseResult.Status == parser.ParseStatusFailed {
		movie.ParseStatus = models.ParseStatusFailed
		movie.UpdatedAt = time.Now()
		return s.movieRepo.Update(ctx, movie)
	}

	// If the parser couldn't extract a meaningful title, mark as failed
	cleanedTitle := parseResult.CleanedTitle
	if cleanedTitle == "" {
		cleanedTitle = parseResult.Title
	}
	if cleanedTitle == "" {
		movie.ParseStatus = models.ParseStatusFailed
		movie.UpdatedAt = time.Now()
		return s.movieRepo.Update(ctx, movie)
	}

	// Step 2: Determine media type
	mediaType := "movie"
	if parseResult.MediaType == parser.MediaTypeTVShow {
		mediaType = "tv"
	}

	// Step 3: Search metadata via TMDB fallback chain
	searchReq := &SearchMetadataRequest{
		Query:     cleanedTitle,
		MediaType: mediaType,
		Year:      parseResult.Year,
	}

	searchResult, _, err := s.metadataService.SearchMetadata(ctx, searchReq)
	if err != nil {
		movie.ParseStatus = models.ParseStatusFailed
		movie.UpdatedAt = time.Now()
		_ = s.movieRepo.Update(ctx, movie)
		return fmt.Errorf("metadata search: %w", err)
	}

	if searchResult == nil || !searchResult.HasResults() {
		movie.ParseStatus = models.ParseStatusFailed
		movie.UpdatedAt = time.Now()
		_ = s.movieRepo.Update(ctx, movie)
		return fmt.Errorf("no metadata found for: %s", cleanedTitle)
	}

	// Step 4: Apply best match to movie record
	best := searchResult.Items[0]
	s.applyMetadataToMovie(movie, best, searchResult.Source)

	// Step 5: Update DB
	movie.UpdatedAt = time.Now()
	if err := s.movieRepo.Update(ctx, movie); err != nil {
		return fmt.Errorf("update movie: %w", err)
	}

	s.logger.Debug("movie enriched",
		"id", movie.ID,
		"old_title", filename,
		"new_title", movie.Title,
		"tmdb_id", movie.TMDbID.Int64,
	)

	return nil
}

// tryNFOEnrichment attempts to enrich a movie from its NFO sidecar file.
// Returns (true, nil) if NFO was found, accepted, and enrichment succeeded.
// Returns (false, nil) if no NFO found or ShouldOverwrite rejected it.
// Returns (false, err) if NFO was found but parsing failed.
func (s *EnrichmentService) tryNFOEnrichment(ctx context.Context, movie *models.Movie) (bool, error) {
	nfoPath := s.nfoReader.FindNFOSidecar(movie.FilePath.String)
	if nfoPath == "" {
		return false, nil
	}

	// Check ShouldOverwrite gate before applying NFO data
	currentSource := models.MetadataSource("")
	if movie.MetadataSource.Valid {
		currentSource = models.MetadataSource(movie.MetadataSource.String)
	}
	if !models.ShouldOverwrite(currentSource, models.MetadataSourceNFO) {
		s.logger.Debug("NFO skipped: current source has higher priority",
			"id", movie.ID, "current_source", currentSource)
		return false, nil
	}

	nfoData, err := s.nfoReader.Parse(nfoPath)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", nfoPath, err)
	}

	// Apply technical info from NFO streamdetails (AC #5)
	s.applyNFOTechInfo(movie, nfoData)

	// Try TMDB direct lookup using NFO uniqueid (AC #2, #3)
	if s.tmdbService != nil {
		if err := s.enrichFromNFOWithTMDb(ctx, movie, nfoData); err != nil {
			s.logger.Warn("TMDB lookup from NFO failed, applying NFO data only",
				"id", movie.ID, "error", err)
			// Still apply basic NFO data below
		}
	}

	// Set metadata source and parse status
	movie.MetadataSource = models.NewNullString(string(models.MetadataSourceNFO))
	movie.ParseStatus = models.ParseStatusSuccess
	movie.UpdatedAt = time.Now()

	if err := s.movieRepo.Update(ctx, movie); err != nil {
		return false, fmt.Errorf("update movie after NFO: %w", err)
	}

	s.logger.Debug("movie enriched from NFO",
		"id", movie.ID,
		"nfo", nfoPath,
		"tmdb_id", movie.TMDbID.Int64,
	)

	return true, nil
}

// enrichFromNFOWithTMDb uses NFO uniqueid to do a direct TMDB lookup
func (s *EnrichmentService) enrichFromNFOWithTMDb(ctx context.Context, movie *models.Movie, nfoData *NFOData) error {
	// AC #2: Direct TMDB ID lookup
	if nfoData.TMDbID != "" {
		tmdbID, err := strconv.Atoi(nfoData.TMDbID)
		if err != nil {
			return fmt.Errorf("invalid tmdb id %q: %w", nfoData.TMDbID, err)
		}
		details, err := s.tmdbService.GetMovieDetails(ctx, tmdbID)
		if err != nil {
			return fmt.Errorf("tmdb get movie %d: %w", tmdbID, err)
		}
		s.applyTMDbMovieDetails(movie, details)
		return nil
	}

	// AC #3: IMDB ID → TMDB find by external ID
	if nfoData.IMDbID != "" {
		return s.enrichFromIMDbID(ctx, movie, nfoData.IMDbID)
	}

	return nil
}

// applyNFOTechInfo applies streamdetails from NFO to movie tech info fields
func (s *EnrichmentService) applyNFOTechInfo(movie *models.Movie, nfoData *NFOData) {
	if nfoData.VideoCodec != "" {
		movie.VideoCodec = models.NewNullString(nfoData.VideoCodec)
	}
	if nfoData.VideoResolution != "" {
		movie.VideoResolution = models.NewNullString(nfoData.VideoResolution)
	}
	if nfoData.AudioCodec != "" {
		movie.AudioCodec = models.NewNullString(nfoData.AudioCodec)
	}
	if nfoData.AudioChannels > 0 {
		movie.AudioChannels = models.NewNullInt64(int64(nfoData.AudioChannels))
	}
}

// applyTMDbMovieDetails applies TMDB movie details to a movie record
func (s *EnrichmentService) applyTMDbMovieDetails(movie *models.Movie, details *tmdb.MovieDetails) {
	if details == nil {
		return
	}

	movie.TMDbID = models.NullInt64{NullInt64: sql.NullInt64{Int64: int64(details.ID), Valid: true}}

	if details.Title != "" {
		movie.Title = details.Title
	}
	if details.OriginalTitle != "" {
		movie.OriginalTitle = models.NewNullString(details.OriginalTitle)
	}
	if details.Overview != "" {
		movie.Overview = models.NewNullString(details.Overview)
	}
	if details.ReleaseDate != "" {
		movie.ReleaseDate = details.ReleaseDate
	}
	if details.PosterPath != nil && *details.PosterPath != "" {
		movie.PosterPath = models.NewNullString(*details.PosterPath)
	}
	if details.BackdropPath != nil && *details.BackdropPath != "" {
		movie.BackdropPath = models.NewNullString(*details.BackdropPath)
	}
	if details.VoteAverage > 0 {
		movie.VoteAverage = models.NewNullFloat64(details.VoteAverage)
	}
	if details.VoteCount > 0 {
		movie.VoteCount = models.NewNullInt64(int64(details.VoteCount))
	}
	if details.Popularity > 0 {
		movie.Popularity = models.NewNullFloat64(details.Popularity)
	}
	if details.Runtime > 0 {
		movie.Runtime = models.NewNullInt64(int64(details.Runtime))
	}
	if details.ImdbID != "" {
		movie.IMDbID = models.NewNullString(details.ImdbID)
	}
	if len(details.Genres) > 0 {
		genres := make([]string, len(details.Genres))
		for i, g := range details.Genres {
			genres[i] = g.Name
		}
		movie.Genres = genres
	}
}

// enrichFromIMDbID uses IMDB ID to find the movie on TMDB via /find endpoint
func (s *EnrichmentService) enrichFromIMDbID(ctx context.Context, movie *models.Movie, imdbID string) error {
	findResult, err := s.tmdbService.FindByExternalID(ctx, imdbID, "imdb_id")
	if err != nil {
		return fmt.Errorf("tmdb find by imdb %s: %w", imdbID, err)
	}

	if len(findResult.MovieResults) > 0 {
		// Use the first movie result's ID to get full details
		tmdbID := findResult.MovieResults[0].ID
		details, err := s.tmdbService.GetMovieDetails(ctx, tmdbID)
		if err != nil {
			return fmt.Errorf("tmdb get movie %d (from imdb %s): %w", tmdbID, imdbID, err)
		}
		s.applyTMDbMovieDetails(movie, details)
		return nil
	}

	// No movie results — might be a TV show, but we're enriching a movie record
	return fmt.Errorf("no TMDB movie found for IMDB ID %s", imdbID)
}

// applyMetadataToMovie updates movie fields from a MetadataItem.
func (s *EnrichmentService) applyMetadataToMovie(movie *models.Movie, item metadata.MetadataItem, source models.MetadataSource) {
	// Use zh-TW title if available, fallback to default title
	if item.TitleZhTW != "" {
		movie.Title = item.TitleZhTW
	} else {
		movie.Title = item.Title
	}

	if item.OriginalTitle != "" {
		movie.OriginalTitle = models.NewNullString(item.OriginalTitle)
	}

	// TMDb ID
	tmdbID := parseProviderIDFromString(item.ID)
	if tmdbID > 0 {
		movie.TMDbID = models.NullInt64{NullInt64: sql.NullInt64{Int64: tmdbID, Valid: true}}
	}

	// Poster path (TMDB returns full URL like "/poster.jpg")
	if item.PosterURL != "" {
		movie.PosterPath = models.NewNullString(item.PosterURL)
	}

	if item.BackdropURL != "" {
		movie.BackdropPath = models.NewNullString(item.BackdropURL)
	}

	if item.Overview != "" {
		movie.Overview = models.NewNullString(item.Overview)
	} else if item.OverviewZhTW != "" {
		movie.Overview = models.NewNullString(item.OverviewZhTW)
	}

	if item.ReleaseDate != "" {
		movie.ReleaseDate = item.ReleaseDate
	}

	if item.Rating > 0 {
		movie.VoteAverage = models.NewNullFloat64(item.Rating)
	}

	if item.VoteCount > 0 {
		movie.VoteCount = models.NewNullInt64(int64(item.VoteCount))
	}

	if item.Popularity > 0 {
		movie.Popularity = models.NewNullFloat64(item.Popularity)
	}

	if len(item.Genres) > 0 {
		movie.Genres = item.Genres
	}

	movie.ParseStatus = models.ParseStatusSuccess
	movie.MetadataSource = models.NewNullString(string(source))
}

func (s *EnrichmentService) buildResult(startedAt time.Time) *EnrichmentResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &EnrichmentResult{
		Total:     s.progress.Total,
		Succeeded: s.progress.Succeeded,
		Failed:    s.progress.Failed,
		Skipped:   s.progress.Skipped,
		Duration:  time.Since(startedAt).Round(time.Second).String(),
	}
}

func (s *EnrichmentService) broadcastProgress() {
	if s.sseHub == nil {
		return
	}
	s.mu.Lock()
	progress := s.progress
	s.mu.Unlock()

	s.sseHub.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: sse.EventEnrichProgress,
		Data: progress,
	})
}

func (s *EnrichmentService) broadcastComplete(result *EnrichmentResult) {
	if s.sseHub == nil {
		return
	}
	s.sseHub.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: sse.EventEnrichComplete,
		Data: result,
	})
}

// parseProviderIDFromString extracts numeric ID from a provider ID string.
func parseProviderIDFromString(id string) int64 {
	var n int64
	fmt.Sscanf(strings.TrimSpace(id), "%d", &n)
	return n
}
