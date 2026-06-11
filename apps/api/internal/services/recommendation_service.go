package services

import (
	"context"
	"log/slog"

	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// maxRecommendationItems caps the rendered set so the payload and the ownership
// lookup placeholder count stay bounded (Story 12-3 Task 2.5).
const maxRecommendationItems = 18

// RecommendationItem is the normalized tile shape shared across movie/TV
// recommendations. The frontend receives a single uniform shape regardless of
// the underlying TMDb media type. JSON is snake_case per Rule 6/18; the frontend
// transforms to camelCase via snakeToCamel.
type RecommendationItem struct {
	ID          int     `json:"id"`
	MediaType   string  `json:"media_type"` // "movie" | "tv"
	Title       string  `json:"title"`
	PosterPath  *string `json:"poster_path"`
	ReleaseDate string  `json:"release_date"`
	VoteAverage float64 `json:"vote_average"`
	IsOwned     bool    `json:"is_owned"`
}

// RecommendationResult is the service-level result for a recommendations lookup.
// Source records which TMDb endpoint filled the list: "recommendations" (primary),
// "similar" (fallback), or "" (both empty). Items marshals to the wire field
// "results" (Story 12-3 AC #4, handler returns { results, source }).
type RecommendationResult struct {
	Items  []RecommendationItem `json:"results"`
	Source string               `json:"source"`
}

// Source constants for RecommendationResult.Source.
const (
	recommendationSourceRecommendations = "recommendations"
	recommendationSourceSimilar         = "similar"
	recommendationSourceEmpty           = ""
)

// RecommendationServiceInterface defines the contract for related-content lookups.
type RecommendationServiceInterface interface {
	// GetMovieRecommendations returns related movies for a TMDb movie id with the
	// "已有" ownership flag stamped per tile.
	GetMovieRecommendations(ctx context.Context, tmdbID int) (*RecommendationResult, error)
	// GetTVRecommendations returns related TV shows for a TMDb TV id with ownership.
	GetTVRecommendations(ctx context.Context, tmdbID int) (*RecommendationResult, error)
}

// RecommendationService resolves TMDb recommendations/similar and annotates each
// tile with local-library ownership. It owns the cross-domain ownership join
// (Rule 4 — the handler never touches a repo directly).
type RecommendationService struct {
	tmdbService TMDbServiceInterface
	movieRepo   repository.MovieRepositoryInterface
	seriesRepo  repository.SeriesRepositoryInterface
}

// Compile-time interface verification.
var _ RecommendationServiceInterface = (*RecommendationService)(nil)

// NewRecommendationService wires the TMDb service and ownership repositories.
func NewRecommendationService(
	tmdbService TMDbServiceInterface,
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
) *RecommendationService {
	return &RecommendationService{
		tmdbService: tmdbService,
		movieRepo:   movieRepo,
		seriesRepo:  seriesRepo,
	}
}

// GetMovieRecommendations calls /recommendations first and falls back to /similar
// when recommendations is empty (AC #4). The TMDb error is propagated as-is so the
// handler can map it via handleTMDbError (Rule 7 — reuse TMDB_* codes, AC #6).
func (s *RecommendationService) GetMovieRecommendations(ctx context.Context, tmdbID int) (*RecommendationResult, error) {
	if tmdbID <= 0 {
		return nil, tmdb.NewBadRequestError("movie ID must be greater than 0")
	}

	recs, err := s.tmdbService.GetMovieRecommendations(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	movies := recs.Results
	source := recommendationSourceRecommendations

	if len(movies) == 0 {
		sim, err := s.tmdbService.GetMovieSimilar(ctx, tmdbID)
		if err != nil {
			return nil, err
		}
		movies = sim.Results
		source = recommendationSourceSimilar
	}

	if len(movies) == 0 {
		source = recommendationSourceEmpty
	}

	movies = capMovies(movies, maxRecommendationItems) // package helper (explore_block_service.go)
	items := make([]RecommendationItem, 0, len(movies))
	for _, m := range movies {
		items = append(items, RecommendationItem{
			ID:          m.ID,
			MediaType:   "movie",
			Title:       m.Title,
			PosterPath:  m.PosterPath,
			ReleaseDate: m.ReleaseDate,
			VoteAverage: m.VoteAverage,
		})
	}

	s.annotateOwnership(ctx, items, s.movieRepo.FindOwnedTMDbIDs)

	return &RecommendationResult{Items: items, Source: source}, nil
}

// GetTVRecommendations is the TV counterpart of GetMovieRecommendations.
func (s *RecommendationService) GetTVRecommendations(ctx context.Context, tmdbID int) (*RecommendationResult, error) {
	if tmdbID <= 0 {
		return nil, tmdb.NewBadRequestError("TV show ID must be greater than 0")
	}

	recs, err := s.tmdbService.GetTVRecommendations(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	shows := recs.Results
	source := recommendationSourceRecommendations

	if len(shows) == 0 {
		sim, err := s.tmdbService.GetTVSimilar(ctx, tmdbID)
		if err != nil {
			return nil, err
		}
		shows = sim.Results
		source = recommendationSourceSimilar
	}

	if len(shows) == 0 {
		source = recommendationSourceEmpty
	}

	shows = capTVShows(shows, maxRecommendationItems) // package helper (explore_block_service.go)
	items := make([]RecommendationItem, 0, len(shows))
	for _, sh := range shows {
		items = append(items, RecommendationItem{
			ID:          sh.ID,
			MediaType:   "tv",
			Title:       sh.Name,
			PosterPath:  sh.PosterPath,
			ReleaseDate: sh.FirstAirDate,
			VoteAverage: sh.VoteAverage,
		})
	}

	s.annotateOwnership(ctx, items, s.seriesRepo.FindOwnedTMDbIDs)

	return &RecommendationResult{Items: items, Source: source}, nil
}

// annotateOwnership stamps IsOwned per item via a single batched lookup
// (Story 10-4 FindOwnedTMDbIDs — no N+1). An ownership-lookup error MUST NOT
// fail the whole call (Rule 27 Pillar 3 / Rule 13 intentional discard): it is
// logged and every tile degrades to IsOwned=false so recommendations still render.
func (s *RecommendationService) annotateOwnership(
	ctx context.Context,
	items []RecommendationItem,
	lookup func(context.Context, []int64) ([]int64, error),
) {
	if len(items) == 0 {
		return
	}

	ids := make([]int64, 0, len(items))
	for _, it := range items {
		ids = append(ids, int64(it.ID))
	}

	owned, err := lookup(ctx, ids)
	if err != nil {
		// Degrade, don't fail — ownership is enrichment, not core content.
		slog.Warn("Recommendation ownership lookup failed; rendering without badges",
			"error", err, "count", len(ids))
		return
	}

	ownedSet := make(map[int64]struct{}, len(owned))
	for _, id := range owned {
		ownedSet[id] = struct{}{}
	}
	for i := range items {
		if _, ok := ownedSet[int64(items[i].ID)]; ok {
			items[i].IsOwned = true
		}
	}
}
