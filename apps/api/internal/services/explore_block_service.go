// Package services ExploreBlockService — Story 10.3.
//
// CRUD for homepage custom discover blocks (P2-002) and a content-fetch
// endpoint that delegates to TMDbServiceInterface (Story 10-1's DiscoverMovies
// / DiscoverTVShows) with 1-hour per-block caching via CacheRepository.
//
// Content is filtered post-fetch through ContentFilterService (FarFuture +
// LowQuality), matching Story 10-1 semantics.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// Default cache TTL for per-block content payloads. Kept inline of the service
// to avoid a public setter that wouldn't serve another caller.
const exploreBlockContentCacheTTL = time.Hour

// ExploreBlockServiceInterface defines the contract for explore block operations.
type ExploreBlockServiceInterface interface {
	GetAllBlocks(ctx context.Context) ([]models.ExploreBlock, error)
	GetBlock(ctx context.Context, id string) (*models.ExploreBlock, error)
	CreateBlock(ctx context.Context, req CreateExploreBlockRequest) (*models.ExploreBlock, error)
	UpdateBlock(ctx context.Context, id string, req UpdateExploreBlockRequest) (*models.ExploreBlock, error)
	DeleteBlock(ctx context.Context, id string) error
	ReorderBlocks(ctx context.Context, orderedIDs []string) ([]models.ExploreBlock, error)
	SeedDefaultsIfEmpty(ctx context.Context) error
	GetBlockContent(ctx context.Context, id string) (*ExploreBlockContent, error)
}

// CreateExploreBlockRequest is the input for creating a block.
type CreateExploreBlockRequest struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	GenreIDs    string `json:"genre_ids"`
	Language    string `json:"language"`
	Region      string `json:"region"`
	SortBy      string `json:"sort_by"`
	MaxItems    int    `json:"max_items"`
}

// UpdateExploreBlockRequest is the input for updating a block. Any nil field
// is left untouched on the existing record.
type UpdateExploreBlockRequest struct {
	Name        *string `json:"name,omitempty"`
	ContentType *string `json:"content_type,omitempty"`
	GenreIDs    *string `json:"genre_ids,omitempty"`
	Language    *string `json:"language,omitempty"`
	Region      *string `json:"region,omitempty"`
	SortBy      *string `json:"sort_by,omitempty"`
	MaxItems    *int    `json:"max_items,omitempty"`
}

// ExploreBlockContent is the response envelope for GET /explore-blocks/:id/content.
// Content is a union: either Movies is populated (content_type=movie) or
// TVShows is populated (content_type=tv), never both.
type ExploreBlockContent struct {
	BlockID     string            `json:"block_id"`
	ContentType string            `json:"content_type"`
	Movies      []tmdb.Movie      `json:"movies,omitempty"`
	TVShows     []tmdb.TVShow     `json:"tv_shows,omitempty"`
	TotalItems  int               `json:"total_items"`
}

// ExploreBlockService implements ExploreBlockServiceInterface.
type ExploreBlockService struct {
	repo          repository.ExploreBlockRepositoryInterface
	tmdbService   TMDbServiceInterface
	contentFilter *ContentFilterService
	cacheRepo     repository.CacheRepositoryInterface
}

// Compile-time verification.
var _ ExploreBlockServiceInterface = (*ExploreBlockService)(nil)

// NewExploreBlockService builds a new ExploreBlockService.
// cacheRepo is optional — when nil, per-block content is fetched fresh every time.
func NewExploreBlockService(
	repo repository.ExploreBlockRepositoryInterface,
	tmdbService TMDbServiceInterface,
	cacheRepo repository.CacheRepositoryInterface,
) *ExploreBlockService {
	return &ExploreBlockService{
		repo:          repo,
		tmdbService:   tmdbService,
		contentFilter: NewContentFilterService(),
		cacheRepo:     cacheRepo,
	}
}

// SetContentFilter is a test seam for injecting a deterministic clock filter.
func (s *ExploreBlockService) SetContentFilter(cf *ContentFilterService) {
	s.contentFilter = cf
}

// --- CRUD ---

func (s *ExploreBlockService) GetAllBlocks(ctx context.Context) ([]models.ExploreBlock, error) {
	blocks, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all blocks: %w", err)
	}
	return blocks, nil
}

func (s *ExploreBlockService) GetBlock(ctx context.Context, id string) (*models.ExploreBlock, error) {
	block, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get block: %w", err)
	}
	return block, nil
}

func (s *ExploreBlockService) CreateBlock(ctx context.Context, req CreateExploreBlockRequest) (*models.ExploreBlock, error) {
	maxItems := req.MaxItems
	if maxItems == 0 {
		maxItems = models.ExploreBlockDefaultMaxItems
	}

	// Append new blocks at the bottom of the current list.
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count blocks: %w", err)
	}

	block := &models.ExploreBlock{
		Name:        strings.TrimSpace(req.Name),
		ContentType: models.ExploreBlockContentType(req.ContentType),
		GenreIDs:    req.GenreIDs,
		Language:    req.Language,
		Region:      req.Region,
		SortBy:      req.SortBy,
		MaxItems:    maxItems,
		SortOrder:   count,
	}

	if err := block.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	if err := s.repo.Create(ctx, block); err != nil {
		return nil, fmt.Errorf("create block: %w", err)
	}

	slog.Info("Explore block created", "id", block.ID, "name", block.Name, "content_type", block.ContentType)
	return block, nil
}

func (s *ExploreBlockService) UpdateBlock(ctx context.Context, id string, req UpdateExploreBlockRequest) (*models.ExploreBlock, error) {
	block, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get block: %w", err)
	}

	if req.Name != nil {
		block.Name = strings.TrimSpace(*req.Name)
	}
	if req.ContentType != nil {
		block.ContentType = models.ExploreBlockContentType(*req.ContentType)
	}
	if req.GenreIDs != nil {
		block.GenreIDs = *req.GenreIDs
	}
	if req.Language != nil {
		block.Language = *req.Language
	}
	if req.Region != nil {
		block.Region = *req.Region
	}
	if req.SortBy != nil {
		block.SortBy = *req.SortBy
	}
	if req.MaxItems != nil {
		block.MaxItems = *req.MaxItems
	}

	if err := block.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	if err := s.repo.Update(ctx, block); err != nil {
		return nil, fmt.Errorf("update block: %w", err)
	}

	// Invalidate any cached content for this block — config changed.
	s.invalidateContentCache(ctx, id)

	slog.Info("Explore block updated", "id", block.ID)
	return block, nil
}

func (s *ExploreBlockService) DeleteBlock(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete block: %w", err)
	}
	s.invalidateContentCache(ctx, id)
	slog.Info("Explore block deleted", "id", id)
	return nil
}

// ReorderBlocks assigns sort_order = index for each ID in the slice and
// returns the updated list. Missing IDs roll the whole operation back.
func (s *ExploreBlockService) ReorderBlocks(ctx context.Context, orderedIDs []string) ([]models.ExploreBlock, error) {
	if len(orderedIDs) == 0 {
		return s.repo.GetAll(ctx)
	}
	if err := s.repo.Reorder(ctx, orderedIDs); err != nil {
		return nil, fmt.Errorf("reorder blocks: %w", err)
	}
	return s.repo.GetAll(ctx)
}

// SeedDefaultsIfEmpty inserts the AC #5 default blocks when the table is empty.
// Safe to call on every startup — a no-op once at least one block exists.
func (s *ExploreBlockService) SeedDefaultsIfEmpty(ctx context.Context) error {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return fmt.Errorf("count blocks: %w", err)
	}
	if count > 0 {
		return nil
	}

	defaults := []models.ExploreBlock{
		{
			Name:        "熱門電影",
			ContentType: models.ExploreBlockContentMovie,
			SortBy:      "popularity.desc",
			MaxItems:    models.ExploreBlockDefaultMaxItems,
			SortOrder:   0,
		},
		{
			Name:        "熱門影集",
			ContentType: models.ExploreBlockContentTV,
			SortBy:      "popularity.desc",
			MaxItems:    models.ExploreBlockDefaultMaxItems,
			SortOrder:   1,
		},
		{
			Name:        "近期新片",
			ContentType: models.ExploreBlockContentMovie,
			SortBy:      "primary_release_date.desc",
			MaxItems:    models.ExploreBlockDefaultMaxItems,
			SortOrder:   2,
		},
	}

	for i := range defaults {
		block := defaults[i]
		if err := s.repo.Create(ctx, &block); err != nil {
			return fmt.Errorf("seed default block %q: %w", block.Name, err)
		}
	}
	slog.Info("Seeded default explore blocks", "count", len(defaults))
	return nil
}

// --- Content ---

// GetBlockContent fetches TMDb discover results for the block, applies content
// filters (far-future + low-quality), caps at block.MaxItems, and caches for
// one hour using cache_type "explore_block".
func (s *ExploreBlockService) GetBlockContent(ctx context.Context, id string) (*ExploreBlockContent, error) {
	block, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get block: %w", err)
	}

	// Cache lookup
	cacheKey := exploreBlockCacheKey(id)
	if cached, ok := s.readCache(ctx, cacheKey); ok {
		cached.BlockID = id
		cached.ContentType = string(block.ContentType)
		return cached, nil
	}

	content, err := s.fetchBlockContent(ctx, block)
	if err != nil {
		return nil, err
	}

	s.writeCache(ctx, cacheKey, content)
	return content, nil
}

func (s *ExploreBlockService) fetchBlockContent(ctx context.Context, block *models.ExploreBlock) (*ExploreBlockContent, error) {
	params := tmdb.DiscoverParams{
		Genre:    block.GenreIDs,
		Region:   block.Region,
		Language: block.Language,
		SortBy:   block.SortBy,
		Page:     1,
	}

	content := &ExploreBlockContent{
		BlockID:     block.ID,
		ContentType: string(block.ContentType),
	}

	switch block.ContentType {
	case models.ExploreBlockContentMovie:
		result, err := s.tmdbService.DiscoverMovies(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("discover movies: %w", err)
		}
		movies := s.contentFilter.FilterFarFutureMovies(result.Results)
		movies = s.contentFilter.FilterLowQualityMovies(movies)
		content.Movies = capMovies(movies, block.MaxItems)
		content.TotalItems = len(content.Movies)

	case models.ExploreBlockContentTV:
		result, err := s.tmdbService.DiscoverTVShows(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("discover tv shows: %w", err)
		}
		shows := s.contentFilter.FilterFarFutureTVShows(result.Results)
		shows = s.contentFilter.FilterLowQualityTVShows(shows)
		content.TVShows = capTVShows(shows, block.MaxItems)
		content.TotalItems = len(content.TVShows)

	default:
		return nil, fmt.Errorf("unsupported content type %q", block.ContentType)
	}

	return content, nil
}

func capMovies(movies []tmdb.Movie, maxItems int) []tmdb.Movie {
	if maxItems <= 0 || len(movies) <= maxItems {
		return movies
	}
	return movies[:maxItems]
}

func capTVShows(shows []tmdb.TVShow, maxItems int) []tmdb.TVShow {
	if maxItems <= 0 || len(shows) <= maxItems {
		return shows
	}
	return shows[:maxItems]
}

// --- Cache helpers ---

const exploreBlockCacheType = "explore_block"

func exploreBlockCacheKey(id string) string {
	return "explore_block:" + id
}

func (s *ExploreBlockService) readCache(ctx context.Context, key string) (*ExploreBlockContent, bool) {
	if s.cacheRepo == nil {
		return nil, false
	}
	entry, err := s.cacheRepo.Get(ctx, key)
	if err != nil {
		slog.Warn("explore block cache read failed", "key", key, "error", err)
		return nil, false
	}
	if entry == nil {
		return nil, false
	}
	var content ExploreBlockContent
	if err := json.Unmarshal([]byte(entry.Value), &content); err != nil {
		slog.Warn("explore block cache decode failed", "key", key, "error", err)
		return nil, false
	}
	return &content, true
}

func (s *ExploreBlockService) writeCache(ctx context.Context, key string, content *ExploreBlockContent) {
	if s.cacheRepo == nil || content == nil {
		return
	}
	payload, err := json.Marshal(content)
	if err != nil {
		slog.Warn("explore block cache encode failed", "key", key, "error", err)
		return
	}
	if err := s.cacheRepo.Set(ctx, key, string(payload), exploreBlockCacheType, exploreBlockContentCacheTTL); err != nil {
		slog.Warn("explore block cache write failed", "key", key, "error", err)
	}
}

func (s *ExploreBlockService) invalidateContentCache(ctx context.Context, id string) {
	if s.cacheRepo == nil {
		return
	}
	if err := s.cacheRepo.Delete(ctx, exploreBlockCacheKey(id)); err != nil {
		slog.Warn("explore block cache delete failed", "id", id, "error", err)
	}
}
