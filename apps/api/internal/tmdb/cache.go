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
