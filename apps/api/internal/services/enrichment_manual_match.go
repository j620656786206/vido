package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// dsr-2b-a: the ways back for a library item the scanner could not match.
//
// Before this file the app offered three rescue paths and none of them worked:
// POST /metadata/apply answered success without writing anything, the single
// re-parse endpoint was a TODO stub, and a Metadata-Editor save left the row
// reading 失敗. On top of that, auto-enrichment's success path never consulted
// the row's provenance, so the next scan-triggered pass overwrote whatever the
// user had typed or picked.

// MediaKind names which library table an operation targets.
type MediaKind string

const (
	MediaKindMovie  MediaKind = "movie"
	MediaKindSeries MediaKind = "series"
)

var (
	// ErrEnrichItemNotFound: no library row with that id.
	ErrEnrichItemNotFound = errors.New("media item not found")
	// ErrEnrichmentAlreadyRunning: a batch enrichment pass is active. Single-item
	// operations refuse rather than race it — the batch loads its rows up front
	// and would write its stale copy back over the user's action.
	ErrEnrichmentAlreadyRunning = errors.New("ENRICHMENT_ALREADY_RUNNING: an enrichment is already in progress")
	// ErrEnrichPersist: the library row could not be read or written. A caller
	// must never report success (or an untouched 'pending') after this.
	ErrEnrichPersist = errors.New("library row could not be read or written")
)

// EnrichedItem is what a single-item match or re-match leaves on the row, read
// back from the store after the write.
//
// [@contract-v1] (story dsr-2b-a AC #2) — consumed by dsr-2b-b's detail page.
// A 200 always carries parse_status "success" or "failed".
type EnrichedItem struct {
	ID          string             `json:"id"`
	ParseStatus models.ParseStatus `json:"parse_status"`
	Title       string             `json:"title"`
	TMDbID      int64              `json:"tmdb_id"`
}

// ItemEnricherInterface re-matches one library item on demand (dsr-2b-a AC #2).
type ItemEnricherInterface interface {
	EnrichOne(ctx context.Context, kind MediaKind, id string) (*EnrichedItem, error)
}

// MatchApplier writes a user-picked TMDb match onto one library item
// (dsr-2b-a AC #1).
type MatchApplier interface {
	ApplyTMDbMatch(ctx context.Context, kind MediaKind, id string, tmdbID int) (*EnrichedItem, error)
}

// ApplyTMDbMatch fetches the TMDb details for a known id and writes them onto
// the row as the user's own choice: metadata_source=manual (nothing automatic
// outranks it) and parse_status=success. The cast is stored on the row too —
// the detail page reads local credits for manual rows.
func (s *EnrichmentService) ApplyTMDbMatch(ctx context.Context, kind MediaKind, id string, tmdbID int) (*EnrichedItem, error) {
	if s.IsEnrichmentActive() {
		return nil, ErrEnrichmentAlreadyRunning
	}
	if s.tmdbService == nil {
		return nil, errors.New("TMDb service not configured")
	}

	switch kind {
	case MediaKindMovie:
		movie, err := s.loadMovie(ctx, id)
		if err != nil {
			return nil, err
		}
		details, err := s.tmdbService.GetMovieDetails(ctx, tmdbID)
		if err != nil {
			return nil, err
		}
		if details == nil {
			return nil, fmt.Errorf("TMDb returned no details for movie %d", tmdbID)
		}
		previous := currentMetadataSource(movie.MetadataSource)
		s.applyTMDbMovieDetails(movie, details)
		movie.MetadataSource = models.NewNullString(string(models.MetadataSourceManual))
		movie.ParseStatus = models.ParseStatusSuccess
		credits := s.matchedCredits(ctx, movie.ID, "movie", movie.TMDbID, previous, models.MetadataSourceManual)
		if credits == nil {
			// No cast for the new match (no credits client, or the fetch
			// failed): clear the old one. The detail page reads local credits
			// for manual rows, so a leftover cast would be the WRONG film's;
			// empty makes it fall back to live TMDb credits for the new id.
			credits = &models.Credits{}
		}
		if err := s.persistMovieMatch(ctx, movie, credits, true); err != nil {
			return nil, err
		}
	case MediaKindSeries:
		series, err := s.loadSeries(ctx, id)
		if err != nil {
			return nil, err
		}
		details, err := s.tmdbService.GetTVShowDetails(ctx, tmdbID)
		if err != nil {
			return nil, err
		}
		if details == nil {
			return nil, fmt.Errorf("TMDb returned no details for series %d", tmdbID)
		}
		previous := currentMetadataSource(series.MetadataSource)
		applyTMDbSeriesDetails(series, details)
		series.MetadataSource = models.NewNullString(string(models.MetadataSourceManual))
		series.ParseStatus = models.ParseStatusSuccess
		credits := s.matchedCredits(ctx, series.ID, "tv", series.TMDbID, previous, models.MetadataSourceManual)
		if credits == nil {
			credits = &models.Credits{} // same as movies: never keep the old match's cast
		}
		if err := s.persistSeriesMatch(ctx, series, credits); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown media kind %q", kind)
	}

	slog.Info("user-picked TMDb match applied", "id", id, "kind", kind, "tmdb_id", tmdbID)
	return s.readItem(ctx, kind, id)
}

// EnrichOne re-runs enrichment for one item and reads the result back.
//
// Movies are re-parsed from the FILE NAME, not the current title: after a wrong
// match or a manual edit the title is no longer the filename, and parsing it
// again would only find the same wrong film. Series keep their title — a series
// title is parsed from an episode file at ingest and only falls back to the
// folder name, and a flat library layout's "series dir" is the library root.
//
// It deliberately does NOT take the batch flag: CancelEnrichment closes the
// batch's channel (nil during a single run → panic), and a post-scan batch
// refused while a single re-match runs would silently drop that scan's work.
// The race with a batch is closed by the write-time provenance re-read instead.
//
// A deadline is detected through ctx.Err(): on cancel the metadata
// orchestrator returns (nil, status) with no error, so the run itself only
// ever looks like "no match found".
func (s *EnrichmentService) EnrichOne(ctx context.Context, kind MediaKind, id string) (*EnrichedItem, error) {
	if s.IsEnrichmentActive() {
		return nil, ErrEnrichmentAlreadyRunning
	}

	var runErr error
	var before time.Time
	switch kind {
	case MediaKindMovie:
		movie, err := s.loadMovie(ctx, id)
		if err != nil {
			return nil, err
		}
		before = movie.UpdatedAt
		input := movie.Title
		if movie.FilePath.Valid && movie.FilePath.String != "" {
			input = filepath.Base(movie.FilePath.String)
		}
		runErr = s.enrichMovieFrom(ctx, movie, input)
	case MediaKindSeries:
		series, err := s.loadSeries(ctx, id)
		if err != nil {
			return nil, err
		}
		before = series.UpdatedAt
		runErr = s.enrichSeries(ctx, series)
	default:
		return nil, fmt.Errorf("unknown media kind %q", kind)
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		// The deadline can land AFTER the row write — during the cast fetch or
		// the glossary resolve that follow it. That run succeeded; report it.
		if item := s.writtenDespiteDeadline(ctx, kind, id, before); item != nil {
			return item, nil
		}
		return nil, fmt.Errorf("re-match %s: %w", id, ctxErr)
	}
	if errors.Is(runErr, ErrEnrichPersist) {
		return nil, runErr
	}
	if runErr != nil {
		s.logger.Info("re-match finished without a match", "id", id, "kind", kind, "reason", runErr)
	}

	item, err := s.readItem(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	if item.ParseStatus != models.ParseStatusSuccess && item.ParseStatus != models.ParseStatusFailed {
		return nil, fmt.Errorf("%w: row still %q after re-match", ErrEnrichPersist, item.ParseStatus)
	}
	return item, nil
}

// writtenDespiteDeadline reads the row back on a short context detached from
// the expired one: if this run wrote a final status, it is the answer.
func (s *EnrichmentService) writtenDespiteDeadline(ctx context.Context, kind MediaKind, id string, before time.Time) *EnrichedItem {
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var updated time.Time
	var item *EnrichedItem
	if kind == MediaKindSeries {
		series, err := s.loadSeries(rctx, id)
		if err != nil {
			return nil
		}
		updated = series.UpdatedAt
		item = &EnrichedItem{ID: series.ID, ParseStatus: series.ParseStatus, Title: series.Title, TMDbID: nullableID(series.TMDbID)}
	} else {
		movie, err := s.loadMovie(rctx, id)
		if err != nil {
			return nil
		}
		updated = movie.UpdatedAt
		item = &EnrichedItem{ID: movie.ID, ParseStatus: movie.ParseStatus, Title: movie.Title, TMDbID: nullableID(movie.TMDbID)}
	}
	if !updated.After(before) {
		return nil
	}
	if item.ParseStatus != models.ParseStatusSuccess && item.ParseStatus != models.ParseStatusFailed {
		return nil
	}
	return item
}

func (s *EnrichmentService) loadMovie(ctx context.Context, id string) (*models.Movie, error) {
	movie, err := s.movieRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEnrichItemNotFound
		}
		return nil, fmt.Errorf("%w: load movie %s: %w", ErrEnrichPersist, id, err)
	}
	if movie == nil {
		return nil, ErrEnrichItemNotFound
	}
	return movie, nil
}

func (s *EnrichmentService) loadSeries(ctx context.Context, id string) (*models.Series, error) {
	if s.seriesRepo == nil {
		return nil, errors.New("series enrichment not configured")
	}
	series, err := s.seriesRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEnrichItemNotFound
		}
		return nil, fmt.Errorf("%w: load series %s: %w", ErrEnrichPersist, id, err)
	}
	if series == nil {
		return nil, ErrEnrichItemNotFound
	}
	return series, nil
}

func (s *EnrichmentService) readItem(ctx context.Context, kind MediaKind, id string) (*EnrichedItem, error) {
	if kind == MediaKindSeries {
		series, err := s.loadSeries(ctx, id)
		if err != nil {
			return nil, err
		}
		return &EnrichedItem{ID: series.ID, ParseStatus: series.ParseStatus, Title: series.Title, TMDbID: nullableID(series.TMDbID)}, nil
	}
	movie, err := s.loadMovie(ctx, id)
	if err != nil {
		return nil, err
	}
	return &EnrichedItem{ID: movie.ID, ParseStatus: movie.ParseStatus, Title: movie.Title, TMDbID: nullableID(movie.TMDbID)}, nil
}

func nullableID(n models.NullInt64) int64 {
	if !n.Valid {
		return 0
	}
	return n.Int64
}

// isUnenriched is the batch's own selection rule (findUnenriched*).
func isUnenriched(status models.ParseStatus) bool {
	return status == models.ParseStatusPending || status == ""
}

func isManualSource(ns models.NullString) bool {
	return ns.Valid && models.MetadataSource(ns.String) == models.MetadataSourceManual
}

// freshMovie re-reads the row. nil when it cannot be read — callers then fall
// back to the copy they hold (the pre-dsr-2b-a behaviour), never fail on it.
func (s *EnrichmentService) freshMovie(ctx context.Context, id string) *models.Movie {
	if id == "" {
		return nil
	}
	movie, err := s.movieRepo.FindByID(ctx, id)
	if err != nil {
		return nil
	}
	return movie
}

func (s *EnrichmentService) freshSeries(ctx context.Context, id string) *models.Series {
	if id == "" || s.seriesRepo == nil {
		return nil
	}
	series, err := s.seriesRepo.FindByID(ctx, id)
	if err != nil {
		return nil
	}
	return series
}

// userTookOverMovie is the write-time gate (AC #4): a batch holds rows it
// loaded minutes ago, so a user save or a user-picked match that landed while
// this row was being searched must win over the stale copy.
func (s *EnrichmentService) userTookOverMovie(ctx context.Context, id string) bool {
	fresh := s.freshMovie(ctx, id)
	if fresh == nil || !isManualSource(fresh.MetadataSource) {
		return false
	}
	s.logger.Info("enrichment result dropped: the user set this movie's metadata mid-run", "id", id)
	return true
}

func (s *EnrichmentService) userTookOverSeries(ctx context.Context, id string) bool {
	fresh := s.freshSeries(ctx, id)
	if fresh == nil || !isManualSource(fresh.MetadataSource) {
		return false
	}
	s.logger.Info("enrichment result dropped: the user set this series' metadata mid-run", "id", id)
	return true
}

// refreshManualMovie is what enrichment does for a row the user owns: no
// search, no metadata overwrite — but the local facts are re-read, because the
// scanner only knocks a row back to pending when its file changed.
func (s *EnrichmentService) refreshManualMovie(ctx context.Context, movie *models.Movie) error {
	// applyFFprobeTechInfo skips the probe (and the subtitle re-detection)
	// once a codec is stored. Here the stored facts are exactly what may be
	// stale — the file changed — so clear that sentinel and probe again,
	// keeping the old codec if the probe cannot produce one.
	if s.ffprobeService != nil && s.ffprobeService.IsAvailable() {
		prevCodec := movie.VideoCodec
		movie.VideoCodec = models.NullString{}
		s.applyFFprobeTechInfo(ctx, movie)
		if !movie.VideoCodec.Valid {
			movie.VideoCodec = prevCodec
		}
	} else {
		s.applyFFprobeTechInfo(ctx, movie)
	}
	movie.ParseStatus = models.ParseStatusSuccess
	movie.UpdatedAt = time.Now()
	if err := s.movieRepo.UpdateEnrichedMetadata(ctx, movie); err != nil {
		return fmt.Errorf("%w: refresh manual movie: %w", ErrEnrichPersist, err)
	}
	// Glossary terms carry their own provenance and never overwrite, so the
	// scope is still resolved (sub-7-3) even though no match was made.
	s.touchGlossaryScope(ctx, movie.ID)
	s.logger.Info("manual movie kept; local analysis refreshed", "id", movie.ID)
	return nil
}

// refreshManualSeries: a series row carries no file tech info, so only the
// status is restored.
func (s *EnrichmentService) refreshManualSeries(ctx context.Context, series *models.Series) error {
	series.ParseStatus = models.ParseStatusSuccess
	series.UpdatedAt = time.Now()
	if err := s.seriesRepo.UpdateEnrichedMetadata(ctx, series); err != nil {
		return fmt.Errorf("%w: refresh manual series: %w", ErrEnrichPersist, err)
	}
	s.touchGlossaryScope(ctx, series.ID)
	s.logger.Info("manual series kept", "id", series.ID)
	return nil
}

// persistMovieMatch is the one tail every movie match goes through — NFO,
// search, and a user-picked TMDb id: row write, then cast, then the glossary
// scope (which must see the new tmdb_id, so it runs after the write).
func (s *EnrichmentService) persistMovieMatch(ctx context.Context, movie *models.Movie, credits *models.Credits, touchScope bool) error {
	movie.UpdatedAt = time.Now()
	if err := s.movieRepo.UpdateEnrichedMetadata(ctx, movie); err != nil {
		return fmt.Errorf("%w: update movie: %w", ErrEnrichPersist, err)
	}
	s.persistCredits(ctx, s.movieRepo.UpdateCredits, movie.ID, credits)
	if touchScope {
		s.touchGlossaryScope(ctx, movie.ID)
	}
	return nil
}

func (s *EnrichmentService) persistSeriesMatch(ctx context.Context, series *models.Series, credits *models.Credits) error {
	series.UpdatedAt = time.Now()
	if err := s.seriesRepo.UpdateEnrichedMetadata(ctx, series); err != nil {
		return fmt.Errorf("%w: update series: %w", ErrEnrichPersist, err)
	}
	s.persistCredits(ctx, s.seriesRepo.UpdateCredits, series.ID, credits)
	s.touchGlossaryScope(ctx, series.ID)
	return nil
}

// applyTMDbSeriesDetails is applyTMDbMovieDetails' series counterpart, with the
// same field set applyMetadataToSeries writes (the series writer has no
// runtime, tech info, imdb id or popularity columns).
func applyTMDbSeriesDetails(series *models.Series, details *tmdb.TVShowDetails) {
	series.TMDbID = models.NewNullInt64(int64(details.ID))
	if details.Name != "" {
		series.Title = details.Name
	}
	if details.OriginalName != "" {
		series.OriginalTitle = models.NewNullString(details.OriginalName)
	}
	if details.PosterPath != nil && *details.PosterPath != "" {
		series.PosterPath = models.NewNullString(*details.PosterPath)
	}
	if details.BackdropPath != nil && *details.BackdropPath != "" {
		series.BackdropPath = models.NewNullString(*details.BackdropPath)
	}
	if details.Overview != "" {
		series.Overview = models.NewNullString(details.Overview)
	}
	if details.FirstAirDate != "" {
		series.FirstAirDate = details.FirstAirDate
	}
	if details.VoteAverage > 0 {
		series.VoteAverage = models.NewNullFloat64(details.VoteAverage)
	}
	if len(details.Genres) > 0 {
		genres := make([]string, len(details.Genres))
		for i, g := range details.Genres {
			genres[i] = g.Name
		}
		series.Genres = genres
	}
}
