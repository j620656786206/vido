package services

import (
	"context"
	"database/sql"
	"encoding/json"
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
	seriesRepo      repository.SeriesRepositoryInterface
	parserService   ParserServiceInterface
	metadataService MetadataServiceInterface
	nfoReader       *NFOReaderService
	tmdbService     TMDbServiceInterface
	ffprobeService  *FFprobeService
	sseHub          *sse.Hub
	logger          *slog.Logger

	// sub-7-3: once a row is matched to TMDb, its cast is fetched and stored
	// on the row, and the row's glossary scope is resolved once — which is
	// where the show's glossary gets seeded (GlossaryScopeResolver → seeder).
	// Both optional — nil = the pre-sub-7-3 behaviour (no credits, no seeding).
	glossaryCredits GlossaryCreditsFetcher
	glossaryScopes  GlossaryScopeResolverInterface

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
	ffprobeService *FFprobeService,
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
		ffprobeService:  ffprobeService,
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

	// Series pass. The scanner creates a series row per TV folder with the folder name as
	// its title and parse_status=pending; without this it would sit unmatched forever —
	// no poster, no overview, a filename for a title. Skipped when seriesRepo is not wired
	// (older call sites), which keeps the movie-only behaviour intact.
	if s.seriesRepo != nil {
		seriesList, err := s.findUnenrichedSeries(ctx)
		if err != nil {
			s.logger.Warn("failed to find unenriched series", "error", err)
		} else if len(seriesList) > 0 {
			s.mu.Lock()
			s.progress.Total += len(seriesList)
			s.mu.Unlock()

			s.logger.Info("enriching series", "total_series", len(seriesList))

			for i := range seriesList {
				select {
				case <-s.cancelChan:
					s.logger.Info("enrichment cancelled during series pass", "processed", i)
					return s.buildResult(startedAt), nil
				case <-ctx.Done():
					return s.buildResult(startedAt), nil
				default:
				}

				series := &seriesList[i]

				s.mu.Lock()
				s.progress.CurrentTitle = series.Title
				s.mu.Unlock()

				if err := s.enrichSeries(ctx, series); err != nil {
					s.logger.Warn("enrichment failed for series",
						"id", series.ID, "title", series.Title, "error", err)
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

				if (i+1)%5 == 0 || i == len(seriesList)-1 {
					s.broadcastProgress()
				}
			}
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

// findUnenrichedSeries queries for series with empty or pending parse_status.
func (s *EnrichmentService) findUnenrichedSeries(ctx context.Context) ([]models.Series, error) {
	pending, err := s.seriesRepo.FindByParseStatus(ctx, models.ParseStatusPending)
	if err != nil {
		return nil, err
	}
	empty, err := s.seriesRepo.FindByParseStatus(ctx, models.ParseStatus(""))
	if err != nil {
		return nil, err
	}
	return append(pending, empty...), nil
}

// enrichSeries resolves TMDb metadata for a scanner-created series and writes it to the
// series row. Note the contrast with the pre-fix behaviour: enrichMovie ALSO detected TV
// correctly and searched TMDb as TV — and then wrote the series' metadata onto a movie
// row, because EnrichmentService had no seriesRepo. That is why every episode looked like
// a movie wearing its show's poster.
func (s *EnrichmentService) enrichSeries(ctx context.Context, series *models.Series) error {
	title := series.Title
	if parseResult := s.parserService.ParseFilename(title); parseResult != nil && parseResult.CleanedTitle != "" {
		title = parseResult.CleanedTitle
	}

	searchResult, _, err := s.metadataService.SearchMetadata(ctx, &SearchMetadataRequest{
		Query:     title,
		MediaType: "tv",
	})
	if err != nil {
		series.ParseStatus = models.ParseStatusFailed
		series.UpdatedAt = time.Now()
		_ = s.seriesRepo.UpdateEnrichedMetadata(ctx, series)
		return fmt.Errorf("metadata search failed: %w", err)
	}

	if searchResult == nil || !searchResult.HasResults() {
		series.ParseStatus = models.ParseStatusFailed
		series.UpdatedAt = time.Now()
		if updateErr := s.seriesRepo.UpdateEnrichedMetadata(ctx, series); updateErr != nil {
			return fmt.Errorf("update series after no match: %w", updateErr)
		}
		return nil
	}

	previousSource := currentMetadataSource(series.MetadataSource)
	s.applyMetadataToSeries(series, searchResult.Items[0], searchResult.Source)
	series.ParseStatus = models.ParseStatusSuccess
	series.UpdatedAt = time.Now()
	// sub-7-3: cast from TMDb — persisted through its own writer right after
	// the match lands, then planted into the show's glossary.
	var credits *models.Credits
	if searchResult.Source == models.MetadataSourceTMDb {
		credits = s.matchedCredits(ctx, series.ID, "tv", series.TMDbID, previousSource, searchResult.Source)
	}

	if err := s.seriesRepo.UpdateEnrichedMetadata(ctx, series); err != nil {
		return fmt.Errorf("update series: %w", err)
	}
	s.persistCredits(ctx, s.seriesRepo.UpdateCredits, series.ID, credits)
	s.touchGlossaryScope(ctx, series.ID)
	return nil
}

// applyMetadataToSeries copies a metadata match onto the series row.
func (s *EnrichmentService) applyMetadataToSeries(series *models.Series, item metadata.MetadataItem, source models.MetadataSource) {
	// bugfix-d CR M4: prefer the zh-TW title, mirroring applyMetadataToMovie.
	// Without this the Douban/Wikipedia providers' 繁中 titles were discarded for
	// series while movies honored them — the same field, two different rules.
	if item.TitleZhTW != "" {
		series.Title = item.TitleZhTW
	} else if item.Title != "" {
		series.Title = item.Title
	}
	if item.OriginalTitle != "" {
		series.OriginalTitle = models.NewNullString(item.OriginalTitle)
	}
	if id := parseProviderID(item.ID); id > 0 {
		series.TMDbID = models.NewNullInt64(id)
	}
	// bugfix-d CR H1: the D2 format convergence covered movies only — series
	// kept writing the absolute URL. Same rule as applyMetadataToMovie: the TMDb
	// relative path is canonical; providers with no relative path (Douban,
	// Wikipedia) keep their absolute URL and the frontend renders it as-is.
	if item.PosterPath != "" {
		series.PosterPath = models.NewNullString(item.PosterPath)
	} else if item.PosterURL != "" {
		series.PosterPath = models.NewNullString(item.PosterURL)
	}
	if item.BackdropPath != "" {
		series.BackdropPath = models.NewNullString(item.BackdropPath)
	} else if item.BackdropURL != "" {
		series.BackdropPath = models.NewNullString(item.BackdropURL)
	}
	if item.Overview != "" {
		series.Overview = models.NewNullString(item.Overview)
	}
	if item.ReleaseDate != "" {
		series.FirstAirDate = item.ReleaseDate
	}
	if item.Rating > 0 {
		series.VoteAverage = models.NewNullFloat64(item.Rating)
	}
	if len(item.Genres) > 0 {
		series.Genres = item.Genres
	}
	series.MetadataSource = models.NewNullString(string(source))
}

// SetSeriesRepo enables the series enrichment pass (bugfix-b). Without it the service
// keeps its historical movie-only behaviour.
func (s *EnrichmentService) SetSeriesRepo(repo repository.SeriesRepositoryInterface) {
	s.seriesRepo = repo
}

// GlossaryCreditsFetcher is the slice of the seeder enrichment uses: the cast
// to persist on the row. Seeding itself is not enrichment's job any more —
// it happens on the first Resolve of the shared scope, for every entry path.
type GlossaryCreditsFetcher interface {
	FetchCredits(ctx context.Context, mediaType string, tmdbID int64) (*models.Credits, []CastPair, error)
}

// SetGlossarySeeder enables the sub-7-3 cast step: after a TMDb match the
// fetcher's credits are persisted on the row through the credits-only writer
// (so the detail page's cast section and the .nfo get them too), and the
// row's glossary scope is resolved once. That resolve is sub-7-1's
// local→tmdb upgrade AND the seed-on-first-resolve moment — the match just
// landed, so the show's glossary is planted right here at scan time; the
// same seam also covers titles that never pass through enrichment.
func (s *EnrichmentService) SetGlossarySeeder(fetcher GlossaryCreditsFetcher, scopes GlossaryScopeResolverInterface) {
	s.glossaryCredits = fetcher
	s.glossaryScopes = scopes
}

// matchedCredits asks the fetcher for a matched title's credits and returns
// the ones to PERSIST — nil when the row's metadata_source outranks the
// incoming source (a Metadata-Editor cast is the user's word, same rule as
// title/poster: models.ShouldOverwrite). Any fetch failure is logged and
// swallowed: TMDb credits gate the CAST, never the enrichment — the row is
// still written with everything else it matched.
func (s *EnrichmentService) matchedCredits(ctx context.Context, rowID, mediaType string, tmdbID models.NullInt64, previous, incoming models.MetadataSource) *models.Credits {
	if s.glossaryCredits == nil || !tmdbID.Valid || tmdbID.Int64 <= 0 {
		return nil
	}
	credits, _, err := s.glossaryCredits.FetchCredits(ctx, mediaType, tmdbID.Int64)
	if err != nil && credits == nil {
		s.logger.Warn("TMDb credits fetch failed; row enriched without cast",
			"id", rowID, "media_type", mediaType, "tmdb_id", tmdbID.Int64, "error", err)
		return nil
	}
	if credits != nil && !models.ShouldOverwrite(previous, incoming) {
		s.logger.Debug("credits kept: current metadata source outranks the match",
			"id", rowID, "current_source", previous, "incoming_source", incoming)
		return nil
	}
	return credits
}

// persistCredits writes the cast through the credits-only writer AFTER the
// enrichment row write (the narrow-writer discipline of
// repository/enriched_metadata_update.go: one column, one intent). A failure
// here is a warning — the match itself already landed.
func (s *EnrichmentService) persistCredits(ctx context.Context, write func(context.Context, string, *models.Credits) error, rowID string, credits *models.Credits) {
	if credits == nil {
		return
	}
	if err := write(ctx, rowID, credits); err != nil {
		s.logger.Warn("credits write failed", "id", rowID, "error", err)
	}
}

// currentMetadataSource reads the row's provenance before a match overwrites it.
func currentMetadataSource(ns models.NullString) models.MetadataSource {
	if ns.Valid {
		return models.MetadataSource(ns.String)
	}
	return ""
}

// touchGlossaryScope resolves the row's glossary scope once, AFTER the row
// is written so the resolver sees the new tmdb_id. Resolve is where the
// `local:` terms collected before the match move into the shared drawer and
// where the drawer is seeded from TMDb credits on first sight. Fail-soft: the
// next resolve (the first subtitle run, the glossary panel) does the same.
func (s *EnrichmentService) touchGlossaryScope(ctx context.Context, rowID string) {
	if s.glossaryScopes == nil {
		return
	}
	if _, err := s.glossaryScopes.Resolve(ctx, rowID); err != nil {
		s.logger.Warn("glossary scope resolve after match failed", "id", rowID, "error", err)
	}
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
		return s.persistFailedWithLocalAnalysis(ctx, movie)
	}

	// If the parser couldn't extract a meaningful title, mark as failed
	cleanedTitle := parseResult.CleanedTitle
	if cleanedTitle == "" {
		cleanedTitle = parseResult.Title
	}
	if cleanedTitle == "" {
		return s.persistFailedWithLocalAnalysis(ctx, movie)
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
		_ = s.persistFailedWithLocalAnalysis(ctx, movie)
		return fmt.Errorf("metadata search: %w", err)
	}

	if searchResult == nil || !searchResult.HasResults() {
		_ = s.persistFailedWithLocalAnalysis(ctx, movie)
		return fmt.Errorf("no metadata found for: %s", cleanedTitle)
	}

	// Step 4: Apply best match to movie record
	best := searchResult.Items[0]
	previousSource := currentMetadataSource(movie.MetadataSource)
	s.applyMetadataToMovie(movie, best, searchResult.Source)

	// Step 4b (sub-7-3): cast from TMDb — written through its own writer after
	// the row write; the glossary is seeded after that so the scope resolver
	// sees tmdb_id. Only for a TMDb match: applyMetadataToMovie stores ANY
	// numeric provider id in TMDbID (a Douban subject id is numeric too), and
	// that must not turn into a credits call for an unrelated TMDb title.
	var credits *models.Credits
	if searchResult.Source == models.MetadataSourceTMDb {
		credits = s.matchedCredits(ctx, movie.ID, mediaType, movie.TMDbID, previousSource, searchResult.Source)
	}

	// Step 5: FFprobe technical info extraction (AC #7: skip if already set from NFO)
	// Runs BEFORE DB update to consolidate into a single write
	s.applyFFprobeTechInfo(ctx, movie)

	// Step 6: Update DB (single write with metadata + tech info)
	movie.UpdatedAt = time.Now()
	if err := s.movieRepo.UpdateEnrichedMetadata(ctx, movie); err != nil {
		return fmt.Errorf("update movie: %w", err)
	}
	s.persistCredits(ctx, s.movieRepo.UpdateCredits, movie.ID, credits)
	s.touchGlossaryScope(ctx, movie.ID)

	s.logger.Debug("movie enriched",
		"id", movie.ID,
		"old_title", filename,
		"new_title", movie.Title,
		"tmdb_id", movie.TMDbID.Int64,
	)

	return nil
}

// persistFailedWithLocalAnalysis marks the movie parse-failed but FIRST runs
// every analysis that needs NO external service — ffprobe tech info, embedded
// subtitle tracks and sidecar subtitle detection — then persists in one write.
//
// Before this helper, every failure exit returned early and starved the whole
// local line: a NAS with no TMDB key had no tech badges, no subtitle-track
// detection, and an existing .srt sitting right next to the movie was reported
// as 尚無字幕 (2026-08-31, Alexyu 內測實測 — 「TMDB 變相必填」). Metadata keys
// gate METADATA; they must not gate what the box can see with its own eyes.
func (s *EnrichmentService) persistFailedWithLocalAnalysis(ctx context.Context, movie *models.Movie) error {
	s.applyFFprobeTechInfo(ctx, movie)
	movie.ParseStatus = models.ParseStatusFailed
	movie.UpdatedAt = time.Now()
	return s.movieRepo.UpdateEnrichedMetadata(ctx, movie)
}

// applyFFprobeTechInfo extracts technical info via FFprobe and applies it to the movie in-memory.
// Does NOT write to DB — caller is responsible for persisting.
// Skips if VideoCodec is already set (from NFO streamdetails).
// Errors are logged but never propagate — FFprobe failure does not block enrichment.
func (s *EnrichmentService) applyFFprobeTechInfo(ctx context.Context, movie *models.Movie) {
	if !movie.FilePath.Valid || movie.FilePath.String == "" {
		return
	}

	// Sidecar subtitle detection is a pure filesystem walk — it needs neither
	// ffprobe nor any API key, so it must never be starved by either (the
	// first Synology install had a .srt beside the movie and still showed
	// 尚無字幕 because this lived after the probe).
	writeSubs := func(embedded []SubtitleTrack) {
		allSubs := MergeSubtitleTracks(embedded, DetectExternalSubtitles(movie.FilePath.String))
		if len(allSubs) == 0 {
			return
		}
		if subsJSON, err := json.Marshal(allSubs); err == nil {
			movie.SubtitleTracks = models.NewNullString(string(subsJSON))
		} else {
			s.logger.Warn("failed to marshal subtitle tracks", "id", movie.ID, "error", err)
		}
	}

	if s.ffprobeService == nil || !s.ffprobeService.IsAvailable() {
		writeSubs(nil)
		return
	}
	// AC #7: Skip if tech info already set (from NFO)
	if movie.VideoCodec.Valid && movie.VideoCodec.String != "" {
		s.logger.Debug("ffprobe skipped: tech info already set",
			"id", movie.ID, "source", "nfo")
		return
	}

	info, err := s.ffprobeService.Probe(ctx, movie.FilePath.String)
	if err != nil {
		s.logger.Warn("ffprobe failed, skipping tech info",
			"id", movie.ID, "file", movie.FilePath.String, "error", err)
		writeSubs(nil) // sidecar detection still runs — it does not need the probe
		return
	}

	// Apply tech info
	if info.VideoCodec != "" {
		movie.VideoCodec = models.NewNullString(info.VideoCodec)
	}
	if info.VideoResolution != "" {
		movie.VideoResolution = models.NewNullString(info.VideoResolution)
	}
	if info.AudioCodec != "" {
		movie.AudioCodec = models.NewNullString(info.AudioCodec)
	}
	if info.AudioChannels > 0 {
		movie.AudioChannels = models.NewNullInt64(int64(info.AudioChannels))
	}
	if info.HDRFormat != "" {
		movie.HDRFormat = models.NewNullString(info.HDRFormat)
	}
	// Container duration (sub-6-10a AC #1). ffprobe has always measured this;
	// until migration 035 there was nowhere to put it, so the subtitle
	// estimator priced from TMDb `runtime` alone and every unmatched title
	// fell through to the 45-minute assumption.
	//
	// > 0 guards the same way the fields above do: ffprobe reports 0 for a
	// container with no duration header, and storing that would turn "we
	// don't know" into "this film is zero minutes long" — which prices as
	// free. NOTE the early return above: a movie whose tech info came from an
	// NFO never reaches this probe, so it keeps pricing from TMDb runtime.
	// That is the pre-existing NFO fast path, not a regression, and the
	// estimator's fallback chain covers it.
	if info.DurationSeconds > 0 {
		movie.DurationSeconds = models.NewNullInt64(int64(info.DurationSeconds))
	}

	// Merge embedded + external subtitles (AC #9)
	writeSubs(info.SubtitleTracks)
}

// tryFFprobeEnrichment is a convenience wrapper that applies FFprobe tech info
// and persists to DB. Used by the NFO enrichment path which needs its own DB write.
func (s *EnrichmentService) tryFFprobeEnrichment(ctx context.Context, movie *models.Movie) {
	s.applyFFprobeTechInfo(ctx, movie)
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

	// Persist IMDB ID from NFO if available (before TMDB lookup may overwrite)
	if nfoData.IMDbID != "" {
		movie.IMDbID = models.NewNullString(nfoData.IMDbID)
	}

	// Try TMDB direct lookup using NFO uniqueid (AC #2, #3)
	matchedTMDb := false
	if s.tmdbService != nil {
		matched, err := s.enrichFromNFOWithTMDb(ctx, movie, nfoData)
		if err != nil {
			s.logger.Warn("TMDB lookup from NFO failed, applying NFO data only",
				"id", movie.ID, "error", err)
			// Still apply basic NFO data below
		}
		matchedTMDb = matched
	}

	// sub-7-3: an NFO that resolved to a TMDb id IN THIS PASS gets its cast
	// too. A tmdb_id already sitting on the row is not enough — a reparse of
	// a wrong earlier match keeps the stale id, and seeding from it would
	// plant the wrong show's cast. The ShouldOverwrite gate above already
	// settled that NFO may overwrite this row.
	var credits *models.Credits
	if matchedTMDb {
		credits = s.matchedCredits(ctx, movie.ID, "movie", movie.TMDbID, currentSource, models.MetadataSourceNFO)
	}

	// Set metadata source and parse status
	movie.MetadataSource = models.NewNullString(string(models.MetadataSourceNFO))
	movie.ParseStatus = models.ParseStatusSuccess
	movie.UpdatedAt = time.Now()

	if err := s.movieRepo.UpdateEnrichedMetadata(ctx, movie); err != nil {
		return false, fmt.Errorf("update movie after NFO: %w", err)
	}
	s.persistCredits(ctx, s.movieRepo.UpdateCredits, movie.ID, credits)
	if matchedTMDb {
		s.touchGlossaryScope(ctx, movie.ID)
	}

	s.logger.Debug("movie enriched from NFO",
		"id", movie.ID,
		"nfo", nfoPath,
		"tmdb_id", movie.TMDbID.Int64,
	)

	return true, nil
}

// enrichFromNFOWithTMDb uses NFO uniqueid to do a direct TMDB lookup. The
// bool reports whether TMDb details were applied in THIS call (sub-7-3 keys
// the cast fetch on it, not on whatever tmdb_id the row already carried).
func (s *EnrichmentService) enrichFromNFOWithTMDb(ctx context.Context, movie *models.Movie, nfoData *NFOData) (bool, error) {
	// AC #2: Direct TMDB ID lookup
	if nfoData.TMDbID != "" {
		tmdbID, err := strconv.Atoi(nfoData.TMDbID)
		if err != nil {
			return false, fmt.Errorf("invalid tmdb id %q: %w", nfoData.TMDbID, err)
		}
		details, err := s.tmdbService.GetMovieDetails(ctx, tmdbID)
		if err != nil {
			return false, fmt.Errorf("tmdb get movie %d: %w", tmdbID, err)
		}
		s.applyTMDbMovieDetails(movie, details)
		return true, nil
	}

	// AC #3: IMDB ID → TMDB find by external ID
	if nfoData.IMDbID != "" {
		if err := s.enrichFromIMDbID(ctx, movie, nfoData.IMDbID); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
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
	if len(nfoData.Subtitles) > 0 {
		langs := make([]string, len(nfoData.Subtitles))
		for i, sub := range nfoData.Subtitles {
			langs[i] = sub.Language
		}
		movie.SubtitleTracks = models.NewNullString(strings.Join(langs, ","))
	}
}

// applyTMDbMovieDetails applies TMDB movie details to a movie record
func (s *EnrichmentService) applyTMDbMovieDetails(movie *models.Movie, details *tmdb.MovieDetails) {
	if details == nil {
		return
	}

	movie.TMDbID = models.NewNullInt64(int64(details.ID))

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

	// bugfix-d D2: store the TMDb-RELATIVE path, matching the other write path
	// (applyTMDbMovieDetails) and what the frontend assumes — `getImageUrl`
	// prefixes `https://image.tmdb.org/t/p/{size}` unconditionally, so an
	// absolute URL stored here rendered as ".../w342https://image.tmdb.org/..."
	// and the poster broke. (The old comment here claimed TMDb "returns full URL
	// like '/poster.jpg'", which contradicted itself and was the bug's alibi.)
	//
	// Providers whose artwork lives on their own host — Douban, Wikipedia — have
	// no relative path to give; their absolute URL is stored as-is and the
	// frontend passes any absolute value straight through.
	if item.PosterPath != "" {
		movie.PosterPath = models.NewNullString(item.PosterPath)
	} else if item.PosterURL != "" {
		movie.PosterPath = models.NewNullString(item.PosterURL)
	}

	if item.BackdropPath != "" {
		movie.BackdropPath = models.NewNullString(item.BackdropPath)
	} else if item.BackdropURL != "" {
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
