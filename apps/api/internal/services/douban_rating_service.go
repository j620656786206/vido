package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/vido/api/internal/douban"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/repository"
)

// ErrMediaNotFound is returned when the media record being enriched does not
// exist. The handler maps it to a 404; any OTHER error is a genuine
// infrastructure failure and maps to a 500 (so a transient DB error is not
// silently masked as "not found").
var ErrMediaNotFound = errors.New("media record not found")

// doubanLookupTimeout caps a cold (cache-miss) Douban scrape so a slow or hung
// scrape cannot block the detail-page request indefinitely. On timeout the
// lookup degrades to a nil rating (TMDb-only) exactly like any other Douban
// failure (AC #4). The bound is generous enough to allow a first-page scrape
// under the provider's 0.5 req/s rate limit.
const doubanLookupTimeout = 10 * time.Second

// DoubanRatingResult is the enrichment payload returned to the detail page.
// A nil *DoubanRatingResult means "no Douban rating available" — the handler
// serializes it as `data: null` and the UI falls back to TMDb-only (AC #4).
type DoubanRatingResult struct {
	DoubanID        string  `json:"douban_id"`
	DoubanRating    float64 `json:"douban_rating"`
	DoubanVoteCount int     `json:"douban_vote_count"`
}

// DoubanSearcher is the subset of metadata.DoubanProvider that the rating
// enrichment needs. Defining it here keeps the service unit-testable with a
// fake searcher and lets main.go inject the SAME provider instance that the
// MetadataService already owns (single rate limiter — project-context Rule 27 ①
// / Rule 14). *metadata.DoubanProvider satisfies this interface.
type DoubanSearcher interface {
	Search(ctx context.Context, req *metadata.SearchRequest) (*metadata.SearchResult, error)
	IsAvailable() bool
}

// DoubanReviewScraper is the subset of the Douban provider the review-summary
// enrichment needs (Story 12-6). It is injected (mirroring DoubanSearcher) so the
// service depends on a behavior contract rather than the concrete internal/douban
// scraper, and so main.go can hand it the SAME *metadata.DoubanProvider instance —
// keeping the single rate limiter / cache (Rule 27 ①/②, Rule 14). The
// implementation is cache-aware (a warm subject serves without a new scrape, AC #6).
type DoubanReviewScraper interface {
	ScrapeReviewSummary(ctx context.Context, id string) (*douban.ReviewSummaryResult, error)
}

// DoubanRatingService enriches movies/series with Douban ratings on demand.
// Flow (Story 12-1 AC #2–#5): check the denormalized douban_rating column →
// if present, return it (cached); otherwise search Douban by title+year, take
// the best match, persist the rating, and return it. Any Douban failure
// degrades gracefully to a nil result with a slog.Warn (AC #4).
type DoubanRatingService struct {
	searcher      DoubanSearcher
	reviewScraper DoubanReviewScraper
	movieRepo     repository.MovieRepositoryInterface
	seriesRepo    repository.SeriesRepositoryInterface
}

// NewDoubanRatingService creates the service. searcher and reviewScraper may each be
// nil (Douban disabled), in which case the corresponding enrichment degrades
// gracefully to nil. reviewScraper powers EnrichDoubanReviewSummary (Story 12-6).
func NewDoubanRatingService(
	searcher DoubanSearcher,
	reviewScraper DoubanReviewScraper,
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
) *DoubanRatingService {
	return &DoubanRatingService{
		searcher:      searcher,
		reviewScraper: reviewScraper,
		movieRepo:     movieRepo,
		seriesRepo:    seriesRepo,
	}
}

// Media types accepted by EnrichDoubanRating.
const (
	doubanMediaMovie  = "movie"
	doubanMediaSeries = "series"
)

// EnrichDoubanRating returns the Douban rating for the given media record,
// fetching and persisting it on a cache miss. Returns (nil, nil) whenever no
// rating is available — this is the graceful-degradation path, NOT an error.
func (s *DoubanRatingService) EnrichDoubanRating(ctx context.Context, mediaID, mediaType string) (*DoubanRatingResult, error) {
	switch mediaType {
	case doubanMediaMovie:
		return s.enrichMovie(ctx, mediaID)
	case doubanMediaSeries:
		return s.enrichSeries(ctx, mediaID)
	default:
		// Caller passes a fixed literal; an unknown value is a programming error.
		slog.Warn("EnrichDoubanRating called with invalid media type", "media_type", mediaType, "media_id", mediaID)
		return nil, nil
	}
}

// EnrichDoubanReviewSummary returns the Douban short-comment summary (短評) for the
// given media record (Story 12-6). It resolves the douban_id (stored by Story 12-1,
// or via a fresh title+year lookup), then scrapes the review summary by that id
// through the injected cache-aware review scraper. Returns (nil, nil) whenever no
// summary is available — the graceful-degradation path, NOT an error (AC #4/#5).
func (s *DoubanRatingService) EnrichDoubanReviewSummary(ctx context.Context, mediaID, mediaType string) (*douban.ReviewSummaryResult, error) {
	if s.reviewScraper == nil {
		return nil, nil
	}

	doubanID, err := s.resolveDoubanID(ctx, mediaID, mediaType)
	if err != nil {
		// Genuine infrastructure error (e.g. media record not found) — let the
		// handler map it (404 vs 500). NOT a Douban-scrape failure.
		return nil, err
	}
	if doubanID == "" {
		// No Douban match — omit the review section (AC #4).
		return nil, nil
	}

	// Bound the (cache-miss) scrape so a slow/hung Douban cannot block the detail
	// page; a timeout degrades to an omitted review block (AC #5).
	ctx, cancel := context.WithTimeout(ctx, doubanLookupTimeout)
	defer cancel()

	summary, err := s.reviewScraper.ScrapeReviewSummary(ctx, doubanID)
	if err != nil {
		// Block / parse / timeout — degrade to an omitted review block (AC #5).
		slog.Warn("Douban review summary enrichment failed", "error", err, "media_id", mediaID, "douban_id", doubanID)
		return nil, nil
	}
	return summary, nil
}

// resolveDoubanID returns the Douban subject id for a media record: the stored id
// (Story 12-1) when present, otherwise a fresh title+year lookup reusing the rating
// path's lookup/pickBestMatch. The resolved rating/id is persisted best-effort so
// later loads skip the lookup. Returns "" when unresolved (no Douban match); a
// genuine repository error (other than not-found) is propagated.
func (s *DoubanRatingService) resolveDoubanID(ctx context.Context, mediaID, mediaType string) (string, error) {
	switch mediaType {
	case doubanMediaMovie:
		movie, err := s.movieRepo.FindByID(ctx, mediaID)
		if err != nil {
			return "", classifyFindErr(err)
		}
		if movie.DoubanID.Valid && movie.DoubanID.String != "" {
			return movie.DoubanID.String, nil
		}
		result := s.lookup(ctx, movie.Title, yearFromDate(movie.ReleaseDate), metadata.MediaTypeMovie, mediaID)
		if result == nil {
			return "", nil
		}
		if err := s.movieRepo.UpdateDoubanRating(ctx, mediaID, result.DoubanID, result.DoubanRating, result.DoubanVoteCount); err != nil {
			slog.Warn("Failed to persist resolved Douban id for movie", "error", err, "movie_id", mediaID)
		}
		return result.DoubanID, nil

	case doubanMediaSeries:
		series, err := s.seriesRepo.FindByID(ctx, mediaID)
		if err != nil {
			return "", classifyFindErr(err)
		}
		if series.DoubanID.Valid && series.DoubanID.String != "" {
			return series.DoubanID.String, nil
		}
		result := s.lookup(ctx, series.Title, yearFromDate(series.FirstAirDate), metadata.MediaTypeTV, mediaID)
		if result == nil {
			return "", nil
		}
		if err := s.seriesRepo.UpdateDoubanRating(ctx, mediaID, result.DoubanID, result.DoubanRating, result.DoubanVoteCount); err != nil {
			slog.Warn("Failed to persist resolved Douban id for series", "error", err, "series_id", mediaID)
		}
		return result.DoubanID, nil

	default:
		slog.Warn("EnrichDoubanReviewSummary called with invalid media type", "media_type", mediaType, "media_id", mediaID)
		return "", nil
	}
}

func (s *DoubanRatingService) enrichMovie(ctx context.Context, id string) (*DoubanRatingResult, error) {
	movie, err := s.movieRepo.FindByID(ctx, id)
	if err != nil {
		return nil, classifyFindErr(err)
	}

	// Cache hit: rating already persisted on the record.
	if movie.DoubanRating.Valid {
		return &DoubanRatingResult{
			DoubanID:        movie.DoubanID.String,
			DoubanRating:    movie.DoubanRating.Float64,
			DoubanVoteCount: int(movie.DoubanVoteCount.Int64),
		}, nil
	}

	result := s.lookup(ctx, movie.Title, yearFromDate(movie.ReleaseDate), metadata.MediaTypeMovie, id)
	if result == nil {
		return nil, nil
	}

	if err := s.movieRepo.UpdateDoubanRating(ctx, id, result.DoubanID, result.DoubanRating, result.DoubanVoteCount); err != nil {
		// Persistence failure should not break the page — log and still return the rating.
		slog.Warn("Failed to persist Douban rating for movie", "error", err, "movie_id", id)
	}
	return result, nil
}

func (s *DoubanRatingService) enrichSeries(ctx context.Context, id string) (*DoubanRatingResult, error) {
	series, err := s.seriesRepo.FindByID(ctx, id)
	if err != nil {
		return nil, classifyFindErr(err)
	}

	if series.DoubanRating.Valid {
		return &DoubanRatingResult{
			DoubanID:        series.DoubanID.String,
			DoubanRating:    series.DoubanRating.Float64,
			DoubanVoteCount: int(series.DoubanVoteCount.Int64),
		}, nil
	}

	result := s.lookup(ctx, series.Title, yearFromDate(series.FirstAirDate), metadata.MediaTypeTV, id)
	if result == nil {
		return nil, nil
	}

	if err := s.seriesRepo.UpdateDoubanRating(ctx, id, result.DoubanID, result.DoubanRating, result.DoubanVoteCount); err != nil {
		slog.Warn("Failed to persist Douban rating for series", "error", err, "series_id", id)
	}
	return result, nil
}

// lookup searches Douban for the best match and returns its rating, or nil on
// any failure / no result (graceful degradation, AC #4).
func (s *DoubanRatingService) lookup(ctx context.Context, title string, year int, mediaType metadata.MediaType, mediaID string) *DoubanRatingResult {
	if s.searcher == nil || !s.searcher.IsAvailable() {
		slog.Warn("Douban searcher unavailable, skipping rating enrichment", "media_id", mediaID)
		return nil
	}
	if title == "" {
		return nil
	}

	// Bound the (potentially rate-limited) scrape so it cannot hang the
	// detail-page request; a timeout degrades to TMDb-only (M1 / AC #4).
	ctx, cancel := context.WithTimeout(ctx, doubanLookupTimeout)
	defer cancel()

	res, err := s.searcher.Search(ctx, &metadata.SearchRequest{
		Query:     title,
		Year:      year,
		MediaType: mediaType,
		Language:  "zh-TW",
		Page:      1,
	})
	if err != nil {
		// Douban blocked / timeout / parse error — degrade to TMDb-only (AC #4).
		slog.Warn("Douban rating enrichment failed", "error", err, "media_id", mediaID, "title", title)
		return nil
	}
	if res == nil || len(res.Items) == 0 {
		slog.Warn("Douban rating enrichment found no results", "media_id", mediaID, "title", title)
		return nil
	}

	best := pickBestMatch(res.Items, year)
	if best == nil {
		// No matched subject carried a usable rating — treat as "no rating".
		slog.Warn("Douban rating enrichment: no rated match", "media_id", mediaID, "title", title, "year", year)
		return nil
	}

	return &DoubanRatingResult{
		DoubanID:        best.ID,
		DoubanRating:    best.Rating,
		DoubanVoteCount: best.VoteCount,
	}
}

// pickBestMatch selects the best Douban subject from a search result (H1 —
// AC #3 "by title+year"). Douban search ordering alone is unreliable for
// same-title works, so when the record's year is known an EXACT year match is
// preferred over Douban's first hit; otherwise the first subject carrying a
// usable rating is used. Subjects with no rating (Rating <= 0) are skipped.
func pickBestMatch(items []metadata.MetadataItem, year int) *metadata.MetadataItem {
	var fallback *metadata.MetadataItem
	for i := range items {
		it := &items[i]
		if it.Rating <= 0 {
			continue
		}
		if year > 0 && it.Year == year {
			return it
		}
		if fallback == nil {
			fallback = it
		}
	}
	return fallback
}

// classifyFindErr maps a repository FindByID error to a 404-eligible
// ErrMediaNotFound when (and only when) the record was absent, leaving genuine
// infrastructure errors intact so the handler can return a 500 (M2).
func classifyFindErr(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMediaNotFound
	}
	return err
}

// yearFromDate extracts the 4-digit year from an ISO-ish date string
// ("1993-01-01" → 1993). Returns 0 when no year can be parsed.
func yearFromDate(date string) int {
	if len(date) < 4 {
		return 0
	}
	year, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return year
}
