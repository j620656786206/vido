package services

import (
	"context"
	"fmt"
	"log/slog"

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
}

// AvailabilityService wraps movie + series repositories to answer "do I already
// own this?" queries with a single DB round trip per table. Handlers use this
// instead of reaching into two separate services (Rule 4 — one service per
// concern) because the ownership concept is cross-type.
type AvailabilityService struct {
	movies repository.MovieRepositoryInterface
	series repository.SeriesRepositoryInterface
}

// NewAvailabilityService wires the two repository dependencies.
func NewAvailabilityService(
	movies repository.MovieRepositoryInterface,
	series repository.SeriesRepositoryInterface,
) *AvailabilityService {
	return &AvailabilityService{movies: movies, series: series}
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
