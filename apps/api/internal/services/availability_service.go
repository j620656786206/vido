package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// AvailabilityServiceInterface defines the contract for media availability lookups
// used by the homepage to render 已有 / 已請求 badges on poster cards.
//
// Story 10-4 (P2-006). The "requested" state is stubbed to false until the
// request system lands in Phase 3 (Epic 13). The interface is defined now so
// the frontend hook and PosterCard integration can be built against a stable
// contract.
type AvailabilityServiceInterface interface {
	// CheckOwned returns the deduplicated union of TMDb IDs from the input that
	// exist as non-removed records in either the movies or series table.
	CheckOwned(ctx context.Context, tmdbIDs []int64) ([]int64, error)
	// CheckOwnedByType returns owned TMDb IDs for ONE media type
	// ('movie'|'tv'), routing to the matching table only — the type-aware
	// sibling of CheckOwned. TMDb assigns ids independently per type, so the
	// merged check can cross-match a movie request against an owned series
	// sharing the same numeric id (Story 13-3a CR M1; mirrors the 13-1a
	// type-aware create guard).
	CheckOwnedByType(ctx context.Context, mediaType string, tmdbIDs []int64) ([]int64, error)
	// LibraryID returns the library id of the owned title with this TMDb id
	// and media type ('movie'|'tv'), or "" when the library has none or the
	// lookup is not wired (disc-2026-11-tmdb-detail-owned-no-link-to-library).
	LibraryID(ctx context.Context, mediaType string, tmdbID int64) (string, error)
}

// libraryIDFinder is the one repository method LibraryID needs. It is kept off
// the big repository interfaces so their many test doubles stay untouched;
// both concrete repositories satisfy it.
type libraryIDFinder interface {
	FindLibraryIDByTMDbID(ctx context.Context, tmdbID int64) (string, error)
}

// The wiring in NewAvailabilityService relies on these; break them and the
// build fails instead of the TMDb page silently never finding a library row.
var (
	_ libraryIDFinder = (*repository.MovieRepository)(nil)
	_ libraryIDFinder = (*repository.SeriesRepository)(nil)
)

// AvailabilityService wraps movie + series repositories to answer "do I already
// own this?" queries with a single DB round trip per table. Handlers use this
// instead of reaching into two separate services (Rule 4 — one service per
// concern) because the ownership concept is cross-type.
type AvailabilityService struct {
	movies repository.MovieRepositoryInterface
	series repository.SeriesRepositoryInterface

	movieIDs  libraryIDFinder
	seriesIDs libraryIDFinder
}

// NewAvailabilityService wires the two repository dependencies.
func NewAvailabilityService(
	movies repository.MovieRepositoryInterface,
	series repository.SeriesRepositoryInterface,
) *AvailabilityService {
	svc := &AvailabilityService{movies: movies, series: series}
	// The production repositories carry the id lookup; test doubles that do
	// not simply answer "not in the library".
	svc.movieIDs, _ = movies.(libraryIDFinder)
	svc.seriesIDs, _ = series.(libraryIDFinder)
	return svc
}

// LibraryID answers "which library row is this TMDb title?" for one media
// type — 'tv' reads the series table, anything else movies, matching
// CheckOwnedByType.
func (s *AvailabilityService) LibraryID(ctx context.Context, mediaType string, tmdbID int64) (string, error) {
	finder := s.movieIDs
	if mediaType == models.RequestMediaTypeTV {
		finder = s.seriesIDs
	}
	if finder == nil || tmdbID <= 0 {
		return "", nil
	}
	return finder.FindLibraryIDByTMDbID(ctx, tmdbID)
}

// CheckOwned merges ownership hits from the movies and series tables. Empty
// input returns an empty slice (not nil) so JSON encodes as [], not null.
func (s *AvailabilityService) CheckOwned(ctx context.Context, tmdbIDs []int64) ([]int64, error) {
	if len(tmdbIDs) == 0 {
		return []int64{}, nil
	}

	movieIDs, err := s.movies.FindOwnedTMDbIDs(ctx, tmdbIDs)
	if err != nil {
		slog.Error("Failed to check owned movies", "error", err, "id_count", len(tmdbIDs))
		return nil, fmt.Errorf("check owned movies: %w", err)
	}

	// Skip the series query if the client has already disconnected — no point
	// doing more DB work for a response the caller will never read.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	seriesIDs, err := s.series.FindOwnedTMDbIDs(ctx, tmdbIDs)
	if err != nil {
		slog.Error("Failed to check owned series", "error", err, "id_count", len(tmdbIDs))
		return nil, fmt.Errorf("check owned series: %w", err)
	}

	// TMDb assigns ids independently per type, so tmdb_id 1 can legitimately
	// identify both a movie and a TV show in the same library. Dedupe here so
	// the contract returns each matching id exactly once regardless of which
	// table(s) own it.
	seen := make(map[int64]struct{}, len(movieIDs)+len(seriesIDs))
	merged := make([]int64, 0, len(movieIDs)+len(seriesIDs))
	for _, id := range movieIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, id)
	}
	for _, id := range seriesIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, id)
	}

	return merged, nil
}

// CheckOwnedByType answers ownership for a single media type: 'tv' consults
// the series table, anything else the movies table (the request domain's
// two-value vocabulary — validation upstream guarantees movie|tv).
func (s *AvailabilityService) CheckOwnedByType(ctx context.Context, mediaType string, tmdbIDs []int64) ([]int64, error) {
	if len(tmdbIDs) == 0 {
		return []int64{}, nil
	}
	if mediaType == models.RequestMediaTypeTV {
		owned, err := s.series.FindOwnedTMDbIDs(ctx, tmdbIDs)
		if err != nil {
			slog.Error("Failed to check owned series by type", "error", err, "id_count", len(tmdbIDs))
			return nil, fmt.Errorf("check owned series: %w", err)
		}
		return owned, nil
	}
	owned, err := s.movies.FindOwnedTMDbIDs(ctx, tmdbIDs)
	if err != nil {
		slog.Error("Failed to check owned movies by type", "error", err, "id_count", len(tmdbIDs))
		return nil, fmt.Errorf("check owned movies: %w", err)
	}
	return owned, nil
}
