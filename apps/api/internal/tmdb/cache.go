package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/vido/api/internal/repository"
)

const (
	// CacheTypeTMDb is the cache type identifier for TMDb responses
	CacheTypeTMDb = "tmdb"

	// DefaultCacheTTL is the default cache duration (24 hours as per NFR-I7)
	DefaultCacheTTL = 24 * time.Hour

	// TrendingDiscoverCacheTTL is the cache duration for /trending/* and /discover/*
	// endpoints (Story 10-1 AC #5). Shorter than DefaultCacheTTL because these
	// lists change frequently and we want homepage content to stay fresh while
	// still respecting TMDb's 40-req/10s rate limit.
	TrendingDiscoverCacheTTL = 1 * time.Hour
)

// CacheServiceConfig holds configuration for the cache service
type CacheServiceConfig struct {
	TTL time.Duration // Cache TTL, defaults to DefaultCacheTTL
}

// CacheService wraps a LanguageFallbackClient and adds caching
type CacheService struct {
	client LanguageFallbackClientInterface
	cache  repository.CacheRepositoryInterface
	ttl    time.Duration
}

// CacheServiceInterface defines the contract for cached TMDb operations
type CacheServiceInterface interface {
	// SearchMovies searches for movies with caching
	SearchMovies(ctx context.Context, query string, page int) (*SearchResultMovies, error)
	// SearchTVShows searches for TV shows with caching
	SearchTVShows(ctx context.Context, query string, page int) (*SearchResultTVShows, error)
	// GetMovieDetails gets movie details with caching
	GetMovieDetails(ctx context.Context, movieID int) (*MovieDetails, error)
	// GetTVShowDetails gets TV show details with caching
	GetTVShowDetails(ctx context.Context, tvID int) (*TVShowDetails, error)
	// GetTrendingMovies returns trending movies cached at TrendingDiscoverCacheTTL
	GetTrendingMovies(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, error)
	// GetTrendingTVShows returns trending TV shows cached at TrendingDiscoverCacheTTL
	GetTrendingTVShows(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, error)
	// DiscoverMovies queries /discover/movie with caching at TrendingDiscoverCacheTTL
	DiscoverMovies(ctx context.Context, params DiscoverParams) (*SearchResultMovies, error)
	// DiscoverTVShows queries /discover/tv with caching at TrendingDiscoverCacheTTL
	DiscoverTVShows(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, error)
}

// Compile-time interface verification
var _ CacheServiceInterface = (*CacheService)(nil)

// NewCacheService creates a new CacheService with the given client and cache repository
func NewCacheService(client LanguageFallbackClientInterface, cache repository.CacheRepositoryInterface, cfg CacheServiceConfig) *CacheService {
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = DefaultCacheTTL
	}

	return &CacheService{
		client: client,
		cache:  cache,
		ttl:    ttl,
	}
}

// SearchMovies searches for movies with caching
// Cache key format: tmdb:search/movie:{query}:{page}
func (s *CacheService) SearchMovies(ctx context.Context, query string, page int) (*SearchResultMovies, error) {
	cacheKey := fmt.Sprintf("tmdb:search/movie:%s:%d", query, page)

	// Try cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultMovies
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit",
				"key", cacheKey,
				"type", CacheTypeTMDb,
			)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data",
			"key", cacheKey,
			"error", err,
		)
	}

	// Cache miss - fetch from API using fallback client
	slog.Debug("Cache miss",
		"key", cacheKey,
		"type", CacheTypeTMDb,
	)

	result, lang, err := s.client.SearchMoviesWithFallback(ctx, query, page)
	if err != nil {
		return nil, err
	}

	slog.Debug("TMDb search movies completed",
		"query", query,
		"language", lang,
		"results", len(result.Results),
	)

	// Store in cache
	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache TMDb response",
				"key", cacheKey,
				"error", err,
			)
		}
	}

	return result, nil
}

// SearchTVShows searches for TV shows with caching
// Cache key format: tmdb:search/tv:{query}:{page}
func (s *CacheService) SearchTVShows(ctx context.Context, query string, page int) (*SearchResultTVShows, error) {
	cacheKey := fmt.Sprintf("tmdb:search/tv:%s:%d", query, page)

	// Try cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultTVShows
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit",
				"key", cacheKey,
				"type", CacheTypeTMDb,
			)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data",
			"key", cacheKey,
			"error", err,
		)
	}

	// Cache miss - fetch from API
	slog.Debug("Cache miss",
		"key", cacheKey,
		"type", CacheTypeTMDb,
	)

	result, lang, err := s.client.SearchTVShowsWithFallback(ctx, query, page)
	if err != nil {
		return nil, err
	}

	slog.Debug("TMDb search TV shows completed",
		"query", query,
		"language", lang,
		"results", len(result.Results),
	)

	// Store in cache
	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache TMDb response",
				"key", cacheKey,
				"error", err,
			)
		}
	}

	return result, nil
}

// GetMovieDetails gets movie details with caching
// Cache key format: tmdb:movie/{id}
func (s *CacheService) GetMovieDetails(ctx context.Context, movieID int) (*MovieDetails, error) {
	cacheKey := fmt.Sprintf("tmdb:movie/%d", movieID)

	// Try cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result MovieDetails
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit",
				"key", cacheKey,
				"type", CacheTypeTMDb,
			)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data",
			"key", cacheKey,
			"error", err,
		)
	}

	// Cache miss - fetch from API
	slog.Debug("Cache miss",
		"key", cacheKey,
		"type", CacheTypeTMDb,
	)

	result, lang, err := s.client.GetMovieDetailsWithFallback(ctx, movieID)
	if err != nil {
		return nil, err
	}

	slog.Debug("TMDb get movie details completed",
		"movie_id", movieID,
		"language", lang,
	)

	// Store in cache
	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache TMDb response",
				"key", cacheKey,
				"error", err,
			)
		}
	}

	return result, nil
}

// GetTrendingMovies returns trending movies with caching (1-hour TTL).
// Cache key format: tmdb:trending/movie:{window}:{page}
func (s *CacheService) GetTrendingMovies(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, error) {
	if page < 1 {
		page = 1
	}
	cacheKey := fmt.Sprintf("tmdb:trending/movie:%s:%d", timeWindow, page)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultMovies
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	result, lang, err := s.client.GetTrendingMoviesWithFallback(ctx, timeWindow, page)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb trending movies completed",
		"time_window", timeWindow,
		"language", lang,
		"results", len(result.Results),
	)

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, TrendingDiscoverCacheTTL); err != nil {
			slog.Warn("Failed to cache trending movies", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}

// GetTrendingTVShows returns trending TV shows with caching (1-hour TTL).
// Cache key format: tmdb:trending/tv:{window}:{page}
func (s *CacheService) GetTrendingTVShows(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, error) {
	if page < 1 {
		page = 1
	}
	cacheKey := fmt.Sprintf("tmdb:trending/tv:%s:%d", timeWindow, page)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultTVShows
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	result, lang, err := s.client.GetTrendingTVShowsWithFallback(ctx, timeWindow, page)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb trending TV shows completed",
		"time_window", timeWindow,
		"language", lang,
		"results", len(result.Results),
	)

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, TrendingDiscoverCacheTTL); err != nil {
			slog.Warn("Failed to cache trending TV shows", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}

// DiscoverMovies queries /discover/movie with caching (1-hour TTL).
// Cache key includes all filter params so different queries get distinct entries.
func (s *CacheService) DiscoverMovies(ctx context.Context, params DiscoverParams) (*SearchResultMovies, error) {
	cacheKey := discoverCacheKey("movie", params)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultMovies
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	result, lang, err := s.client.DiscoverMoviesWithFallback(ctx, params)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb discover movies completed",
		"language", lang,
		"results", len(result.Results),
	)

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, TrendingDiscoverCacheTTL); err != nil {
			slog.Warn("Failed to cache discover movies", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}

// DiscoverTVShows queries /discover/tv with caching (1-hour TTL).
func (s *CacheService) DiscoverTVShows(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, error) {
	cacheKey := discoverCacheKey("tv", params)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultTVShows
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	result, lang, err := s.client.DiscoverTVShowsWithFallback(ctx, params)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb discover TV shows completed",
		"language", lang,
		"results", len(result.Results),
	)

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, TrendingDiscoverCacheTTL); err != nil {
			slog.Warn("Failed to cache discover TV shows", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}

// discoverCacheKey builds a deterministic cache key from DiscoverParams.
// Field order is fixed so the same logical query always maps to the same key.
func discoverCacheKey(kind string, p DiscoverParams) string {
	page := p.Page
	if page < 1 {
		page = 1
	}
	return fmt.Sprintf("tmdb:discover/%s:g=%s:yg=%d:yl=%d:r=%s:lang=%s:sort=%s:p=%d",
		kind, p.Genre, p.YearGte, p.YearLte, p.Region, p.Language, p.SortBy, page)
}

// GetTVShowDetails gets TV show details with caching
// Cache key format: tmdb:tv/{id}
func (s *CacheService) GetTVShowDetails(ctx context.Context, tvID int) (*TVShowDetails, error) {
	cacheKey := fmt.Sprintf("tmdb:tv/%d", tvID)

	// Try cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result TVShowDetails
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit",
				"key", cacheKey,
				"type", CacheTypeTMDb,
			)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data",
			"key", cacheKey,
			"error", err,
		)
	}

	// Cache miss - fetch from API
	slog.Debug("Cache miss",
		"key", cacheKey,
		"type", CacheTypeTMDb,
	)

	result, lang, err := s.client.GetTVShowDetailsWithFallback(ctx, tvID)
	if err != nil {
		return nil, err
	}

	slog.Debug("TMDb get TV show details completed",
		"tv_id", tvID,
		"language", lang,
	)

	// Store in cache
	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache TMDb response",
				"key", cacheKey,
				"error", err,
			)
		}
	}

	return result, nil
}
