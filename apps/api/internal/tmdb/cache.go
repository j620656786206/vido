package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/vido/api/internal/repository"
)

const (
	// CacheTypeTMDb is the cache type identifier for TMDb responses
	CacheTypeTMDb = "tmdb"

	// CacheTypeTMDbFacet tags facet-count cache entries (Story
	// ux3-facet-count-cache-refinement, AR-F4), distinct from CacheTypeTMDb so the
	// count workload can be evicted on its own via
	// CacheRepository.ClearByType(ctx, "tmdb_facet") without touching the
	// result-grid entries.
	// NOTE (AR-F4 boundary / AC7): this gives TARGETED eviction only. The manual
	// full purge still runs a wholesale clearTable("cache_entries"); true
	// purge-isolation needs a SEPARATE table (Hard, deferred — see story AC7 /
	// tech-spec AR-F4 note).
	CacheTypeTMDbFacet = "tmdb_facet"

	// DefaultCacheTTL is the default cache duration (24 hours as per NFR-I7)
	DefaultCacheTTL = 24 * time.Hour

	// TrendingDiscoverCacheTTL is the cache duration for /trending/* and /discover/*
	// endpoints (Story 10-1 AC #5). Shorter than DefaultCacheTTL because these
	// lists change frequently and we want homepage content to stay fresh while
	// still respecting TMDb's 40-req/10s rate limit.
	TrendingDiscoverCacheTTL = 1 * time.Hour
)

// Facet-count dimension keys (Story ux3-discover-facet-aggregation-be).
// Exported so the HTTP handler builds the candidates map with the SAME keys the
// fan-out switches on — a single source of truth prevents BE-internal key drift
// (the curated facet inventory itself stays FE-owned per Q1=A / AC8). These
// strings are also the outer keys of the FacetCounts.Counts response (AC1).
const (
	DimGenre    = "genre"
	DimRegion   = "region"
	DimRating   = "rating"
	DimPlatform = "platform"
)

const (
	// facetCountBudget bounds the facet-count fan-out wall-clock (AC5). Cached
	// facets (1h) resolve instantly; cold sub-queries that exceed this budget are
	// omitted and Partial is set, so the endpoint never blocks the rail.
	facetCountBudget = 800 * time.Millisecond

	// facetCountConcurrency caps concurrent facet sub-queries (AC4) via errgroup
	// SetLimit, so the count fan-out cannot starve interactive TMDb calls
	// (detail / search / homepage) of the shared 40-req/10s rate budget.
	facetCountConcurrency = 4

	// maxFacetProbes defensively bounds the fan-out: at most this many distinct
	// (dim,value) sub-queries are issued per request, regardless of how many
	// candidates the caller supplies. The FE's curated inventory is ~30 values
	// (18 genres + 5 regions + 4 ratings + 3 platforms), so 64 leaves generous
	// headroom for growth while capping cache-entry write-amplification from a
	// pathological *_values list. Excess candidates are dropped and Partial is set.
	maxFacetProbes = 64

	// defaultFacetCountLanguage pins the count-probe language when the caller did
	// not supply one (AC9 / AR-F3). A non-empty Language makes the
	// LanguageFallbackClient issue exactly ONE /discover call per probe instead of
	// fanning the zh-TW→zh-CN→en chain, and collapses all locales onto one cache
	// entry. Counts are therefore documented as per-locale.
	// NOTE: this intentionally mirrors the TMDb Client's own default language
	// (client.go NewClient — `language = "zh-TW"`); keep the two in sync so facet
	// counts pin the SAME locale as the grid when neither side is given a language.
	defaultFacetCountLanguage = "zh-TW"

	// FacetCountCacheTTL governs NON-ZERO facet-count cache entries (Story
	// ux3-facet-count-cache-refinement, AR-F8). Kept SEPARATE from
	// TrendingDiscoverCacheTTL (the result-grid TTL, above) so count
	// freshness/eviction is tunable INDEPENDENTLY of the grid — "trending
	// freshness" ≠ "count freshness". May start equal to 1h; either can change
	// without affecting the other.
	FacetCountCacheTTL = 1 * time.Hour

	// FacetCountZeroTTL caches a total_results==0 count for a SHORT window
	// (AR-F5 + CR M1). A 0 is NOT pinned for the full FacetCountCacheTTL — a
	// transient/wrong-locale 0 must self-correct quickly (AR-F5) — but it IS cached
	// briefly so a GENUINE dead-end facet is not re-fetched (2 TMDb calls) on every
	// debounced facet-counts request, which would otherwise pressure both the
	// 40-req/10s rate budget and the 800ms fan-out budget (and could force Partial).
	// Tunable independently; kept well below FacetCountCacheTTL by construction.
	FacetCountZeroTTL = 1 * time.Minute
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
	// providersClient is the raw (non-language-fallback) client used for the
	// language-neutral watch-providers call (Story 12-4 Task 1.4). Injected via
	// SetProvidersClient — mirrors the SetContentFilter injection pattern so the
	// existing NewCacheService callers (16 test sites) need no signature change.
	providersClient ClientInterface
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
	// GetSeasonDetails gets a season's episode list with caching (24h TTL)
	GetSeasonDetails(ctx context.Context, tvID int, seasonNumber int) (*SeasonDetails, error)
	// GetTrendingMovies returns trending movies cached at TrendingDiscoverCacheTTL
	GetTrendingMovies(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, error)
	// GetTrendingTVShows returns trending TV shows cached at TrendingDiscoverCacheTTL
	GetTrendingTVShows(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, error)
	// DiscoverMovies queries /discover/movie with caching at TrendingDiscoverCacheTTL
	DiscoverMovies(ctx context.Context, params DiscoverParams) (*SearchResultMovies, error)
	// DiscoverTVShows queries /discover/tv with caching at TrendingDiscoverCacheTTL
	DiscoverTVShows(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, error)
	// DiscoverFacetCounts returns, for the given base filter, the contextual
	// movie+tv result count for every supplied candidate facet value
	// (Story ux3-discover-facet-aggregation-be, AC1 [@contract-v1]).
	DiscoverFacetCounts(ctx context.Context, base DiscoverParams, candidates map[string][]string) (*FacetCounts, error)
	// GetMovieRecommendations returns recommended movies with caching (24h TTL)
	GetMovieRecommendations(ctx context.Context, movieID int) (*SearchResultMovies, error)
	// GetMovieSimilar returns similar movies with caching (24h TTL)
	GetMovieSimilar(ctx context.Context, movieID int) (*SearchResultMovies, error)
	// GetTVRecommendations returns recommended TV shows with caching (24h TTL)
	GetTVRecommendations(ctx context.Context, tvID int) (*SearchResultTVShows, error)
	// GetTVSimilar returns similar TV shows with caching (24h TTL)
	GetTVSimilar(ctx context.Context, tvID int) (*SearchResultTVShows, error)
	// GetWatchProviders returns streaming/rent/buy providers for a title, filtered
	// to a single region and cached 24h (Story 12-4). Bypasses the language-fallback
	// layer — watch-provider data is language-neutral.
	GetWatchProviders(ctx context.Context, mediaType string, id int, region string) (*WatchProvidersResponse, error)
	// GetTVExternalIDs returns a TV show's external ids cached at the default
	// TTL (Story 13-4b — ids are immutable-ish). Language-neutral: rides the
	// raw providersClient like GetWatchProviders.
	GetTVExternalIDs(ctx context.Context, tvID int) (*TVExternalIDs, error)
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

// SetProvidersClient injects the raw TMDb client used by GetWatchProviders.
// Watch-provider data is language-neutral, so it bypasses the language-fallback
// layer and talks to the raw client directly (Story 12-4 Task 1.4). Wired by
// NewTMDbService in production; tests that exercise GetWatchProviders set a
// mock ClientInterface here.
func (s *CacheService) SetProvidersClient(c ClientInterface) {
	s.providersClient = c
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

// GetSeasonDetails gets a season's full episode list with caching.
// Cache key format: tmdb:tv/{id}/season/{n} — uses the default 24h TTL since
// episode metadata changes infrequently (Story 12-2 Task 3.4).
func (s *CacheService) GetSeasonDetails(ctx context.Context, tvID int, seasonNumber int) (*SeasonDetails, error) {
	cacheKey := fmt.Sprintf("tmdb:tv/%d/season/%d", tvID, seasonNumber)

	// Try cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SeasonDetails
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data", "key", cacheKey, "error", err)
	}

	// Cache miss - fetch from API
	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)

	result, lang, err := s.client.GetSeasonDetailsWithFallback(ctx, tvID, seasonNumber)
	if err != nil {
		return nil, err
	}

	slog.Debug("TMDb get season details completed",
		"tv_id", tvID,
		"season_number", seasonNumber,
		"language", lang,
		"episodes", len(result.Episodes),
	)

	// Store in cache
	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache TMDb response", "key", cacheKey, "error", err)
		}
	}

	return result, nil
}

// GetTVExternalIDs returns a TV show's external-service ids with caching.
// Cache key: tmdb:tv/{id}/external_ids — default TTL; external ids are
// immutable-ish (Story 13-4b AC #1). Language-neutral → providersClient.
func (s *CacheService) GetTVExternalIDs(ctx context.Context, tvID int) (*TVExternalIDs, error) {
	cacheKey := fmt.Sprintf("tmdb:tv/%d/external_ids", tvID)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result TVExternalIDs
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data", "key", cacheKey, "error", err)
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	if s.providersClient == nil {
		return nil, fmt.Errorf("external-ids client not initialized")
	}
	result, err := s.providersClient.GetTVExternalIDs(ctx, tvID)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache TMDb response", "key", cacheKey, "error", err)
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
// Every filter dimension is included so two queries differing in any single
// dimension (genre, year, region, rating, watch provider, sort, page) map to
// distinct cache entries.
func discoverCacheKey(kind string, p DiscoverParams) string {
	page := p.Page
	if page < 1 {
		page = 1
	}
	return fmt.Sprintf("tmdb:discover/%s:g=%s:yg=%d:yl=%d:r=%s:vg=%s:vl=%s:wp=%s:wr=%s:lang=%s:sort=%s:p=%d",
		kind, joinInts(p.GenreIDs, ","), p.YearGte, p.YearLte, p.Region,
		strconv.FormatFloat(p.VoteAverageGte, 'f', -1, 64),
		strconv.FormatFloat(p.VoteAverageLte, 'f', -1, 64),
		joinInts(p.WatchProviders, ","), p.WatchRegion,
		p.Language, p.SortBy, page)
}

// facetProbe is one (dimension, value) we can actually count: the cloned+normalized
// DiscoverParams for (base + that facet added). Built synchronously so unparseable
// candidates are dropped BEFORE the fan-out and never inflate the partial denominator.
type facetProbe struct {
	dim   string
	value string
	param DiscoverParams
}

// DiscoverFacetCounts computes, for the given base filter, the contextual movie+tv
// result count for every supplied candidate facet value (Story
// ux3-discover-facet-aggregation-be). For each (dim, value) it clones base, ADDS the
// facet per AC2 semantics (multi-select genre/platform append; single-select
// region/rating replace), normalizes sort/page/language (AC9), and sums the
// movie + tv total_results via the DEDICATED facet-count cache (discoverCountCached
// — Story ux3-facet-count-cache-refinement), so a warm facet costs ZERO TMDb calls
// (AC3). The dedicated path tags entries type="tmdb_facet", expires non-zero counts
// on FacetCountCacheTTL and 0-counts on the short FacetCountZeroTTL, diverging from
// the grid's DiscoverMovies/DiscoverTVShows.
//
// The fan-out is bounded two ways: errgroup SetLimit(facetCountConcurrency) caps
// concurrent sub-queries so interactive TMDb calls are not starved (AC4), and a
// facetCountBudget deadline caps wall-clock — facets unresolved within budget are
// omitted with Partial=true (AC5). Per-facet errors are swallowed+logged at the
// goroutine boundary (each g.Go always returns nil), so one bad sub-query omits only
// that facet and never fails the page (AC6 / Rule 13). A resolved count of 0 is a
// real dead-end and is KEPT in the response (the FE dims-but-keeps-selectable);
// 0 is never treated as "missing".
//
// AC8: only the FE-supplied candidate values are counted — a dimension absent from
// candidates is absent from the response; the BE holds no facet inventory of its own.
func (s *CacheService) DiscoverFacetCounts(ctx context.Context, base DiscoverParams, candidates map[string][]string) (*FacetCounts, error) {
	// Build the probe list synchronously. Unparseable values (e.g. genre="abc")
	// can never resolve, so they are dropped here and excluded from the partial
	// denominator rather than perpetually reported as "still computing".
	// Duplicate (dim,value) pairs are de-duplicated so a repeated candidate does
	// not inflate the denominator (which would wrongly mark a fully-resolved
	// response Partial). The total is capped at maxFacetProbes; excess candidates
	// are dropped and capped=true forces Partial.
	var probes []facetProbe
	seen := make(map[string]struct{})
	capped := false
	for dim, values := range candidates {
		for _, value := range values {
			dedupKey := dim + "\x00" + value
			if _, dup := seen[dedupKey]; dup {
				continue
			}
			seen[dedupKey] = struct{}{}
			p, ok := applyFacet(base, dim, value)
			if !ok {
				slog.Warn("facet-count: skipping unknown/unparseable candidate",
					"dim", dim, "value", value)
				continue
			}
			if len(probes) >= maxFacetProbes {
				capped = true
				continue
			}
			normalizeFacetParams(&p)
			probes = append(probes, facetProbe{dim: dim, value: value, param: p})
		}
	}
	if capped {
		slog.Warn("facet-count: candidate count exceeded cap; excess dropped",
			"cap", maxFacetProbes)
	}

	counts := make(map[string]map[string]int)
	var mu sync.Mutex

	// The budget deadline is layered on the caller's context, so whichever fires
	// first wins (a caller with a tighter deadline shortens the budget).
	bctx, cancel := context.WithTimeout(ctx, facetCountBudget)
	defer cancel()

	g, gctx := errgroup.WithContext(bctx)
	g.SetLimit(facetCountConcurrency)

	for _, probe := range probes {
		probe := probe
		g.Go(func() error {
			// Budget already spent before this slot opened → omit (→ Partial).
			if gctx.Err() != nil {
				return nil
			}
			mc, err := s.discoverCountCached(gctx, "movie", probe.param)
			if err != nil {
				slog.Warn("facet-count movie sub-query failed (omitted, fail-soft)",
					"dim", probe.dim, "value", probe.value, "error", err)
				return nil // AC6 — swallow at the per-facet boundary, never fail the page
			}
			tc, err := s.discoverCountCached(gctx, "tv", probe.param)
			if err != nil {
				slog.Warn("facet-count tv sub-query failed (omitted, fail-soft)",
					"dim", probe.dim, "value", probe.value, "error", err)
				return nil
			}
			mu.Lock()
			if counts[probe.dim] == nil {
				counts[probe.dim] = make(map[string]int)
			}
			// A 0-side is cached only briefly (FacetCountZeroTTL) inside
			// discoverCountCached; the SUMMED count is still KEPT in the response —
			// e.g. movie=5 tv=0 → 5 returned. A resolved dead-end 0 stays
			// selectable-but-dimmed on the FE (AC6 parity with the BE story).
			counts[probe.dim][probe.value] = mc + tc
			mu.Unlock()
			return nil
		})
	}
	// Every g.Go returns nil (errors are swallowed per-facet), so Wait never
	// reports a real error; the return is ignored deliberately.
	_ = g.Wait()

	resolved := 0
	for _, m := range counts {
		resolved += len(m)
	}

	return &FacetCounts{Counts: counts, Partial: capped || resolved < len(probes)}, nil
}

// applyFacet clones base and ADDS a single candidate facet value per AC2
// add-semantics, returning false when the dimension is unknown or the value cannot
// be parsed for that dimension (caller drops it). Multi-select dimensions
// (genre→GenreIDs, platform→WatchProviders) APPEND the value; single-select
// dimensions (region→Region, rating→VoteAverageGte) REPLACE/SET it. The genre and
// watch-provider slices are copied before appending so the caller's base slices are
// never mutated by the fan-out.
func applyFacet(base DiscoverParams, dim, value string) (DiscoverParams, bool) {
	p := base
	p.GenreIDs = append([]int(nil), base.GenreIDs...)
	p.WatchProviders = append([]int(nil), base.WatchProviders...)

	switch dim {
	case DimGenre:
		id, err := strconv.Atoi(value)
		if err != nil {
			return p, false
		}
		p.GenreIDs = append(p.GenreIDs, id)
	case DimPlatform:
		id, err := strconv.Atoi(value)
		if err != nil {
			return p, false
		}
		p.WatchProviders = append(p.WatchProviders, id)
		// TMDb requires watch_region whenever with_watch_providers is set; default
		// to the base region, then TW, so a platform probe is never region-less.
		if p.WatchRegion == "" {
			if base.Region != "" {
				p.WatchRegion = base.Region
			} else {
				p.WatchRegion = "TW"
			}
		}
	case DimRegion:
		p.Region = value
	case DimRating:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return p, false
		}
		p.VoteAverageGte = f
	default:
		return p, false
	}
	return p, true
}

// normalizeFacetParams pins SortBy="" / Page=1 / Language before a count sub-query
// is issued (AC9, AR-F2/F3): clearing sort+page collapses every sort order and page
// onto ONE cache entry (so counts reuse each other across the grid's many sorts),
// and a non-empty Language makes the LanguageFallbackClient short-circuit to a single
// /discover call per probe instead of fanning the fallback chain. The caller's pinned
// Language is honored if set, else defaultFacetCountLanguage.
func normalizeFacetParams(p *DiscoverParams) {
	p.SortBy = ""
	p.Page = 1
	if p.Language == "" {
		p.Language = defaultFacetCountLanguage
	}
}

// facetCountCacheKey builds the dedicated cache key for a facet-count sub-query.
// It mirrors discoverCacheKey's field order but uses a DISTINCT namespace
// (tmdb:facetcount/… vs the grid's tmdb:discover/…) so count entries can never
// collide with result-grid entries even though both derive from DiscoverParams
// (Story ux3-facet-count-cache-refinement, AR-F4). Inputs are already normalized
// upstream by normalizeFacetParams (sort=""/page=1/pinned Language), so sort+page
// are constant here — the full serialization is kept for parity with the grid key.
func facetCountCacheKey(kind string, p DiscoverParams) string {
	page := p.Page
	if page < 1 {
		page = 1
	}
	return fmt.Sprintf("tmdb:facetcount/%s:g=%s:yg=%d:yl=%d:r=%s:vg=%s:vl=%s:wp=%s:wr=%s:lang=%s:sort=%s:p=%d",
		kind, joinInts(p.GenreIDs, ","), p.YearGte, p.YearLte, p.Region,
		strconv.FormatFloat(p.VoteAverageGte, 'f', -1, 64),
		strconv.FormatFloat(p.VoteAverageLte, 'f', -1, 64),
		joinInts(p.WatchProviders, ","), p.WatchRegion,
		p.Language, p.SortBy, page)
}

// discoverCountCached returns the movie- OR tv-side total_results for one
// already-normalized facet probe, backed by a DEDICATED count cache (Story
// ux3-facet-count-cache-refinement). It diverges from the grid path
// (DiscoverMovies/DiscoverTVShows) in three ways the grid MUST NOT adopt:
//   - entries are tagged CacheTypeTMDbFacet ("tmdb_facet"), enabling targeted
//     ClearByType eviction of counts alone (AR-F4 / AC2);
//   - entries expire on FacetCountCacheTTL, tunable independently of the grid's
//     TrendingDiscoverCacheTTL (AR-F8 / AC1);
//   - a total_results==0 is cached only briefly on FacetCountZeroTTL, NOT the full
//     FacetCountCacheTTL (AR-F5 / AC3 / CR M1), so a transient or wrong-locale 0
//     self-corrects within ~1min instead of dimming a facet chip for an hour —
//     while a GENUINE dead-end 0 is still spared a re-fetch on every debounced probe.
//
// Only the compact integer count is stored, not the full SearchResult* JSON blob
// (AR-F6 / AC5). It calls the SAME fallback-client methods the grid path uses
// (DiscoverMoviesWithFallback / DiscoverTVShowsWithFallback), so count-to-count
// reuse and the pinned-Language single-call short-circuit are preserved (AC4).
//
// INTENTIONAL grid-cache bypass: going through s.client.*WithFallback rather than
// s.DiscoverMovies/DiscoverTVShows is REQUIRED precisely because those grid methods
// always Set at type="tmdb"/1h including zeros — the behavior AC2/AC3 must diverge
// from. Consequence: a facet probe no longer warms the grid blob cache. The cost is
// small and bounded: the key-spaces overlap ONLY at the default-sort landing (grid
// SortBy=""/page=1, which facet probes also normalize to); any explicit sort or
// page>1 never shared a key, and even the overlapping case loses just a one-time
// grid cache warm.
func (s *CacheService) discoverCountCached(ctx context.Context, kind string, p DiscoverParams) (int, error) {
	cacheKey := facetCountCacheKey(kind, p)

	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		// Stored as the integer count string; a parse failure (corrupt/legacy
		// entry) falls through to a re-fetch rather than surfacing an error.
		if n, convErr := strconv.Atoi(cached.Value); convErr == nil {
			slog.Debug("Facet-count cache hit", "key", cacheKey, "type", CacheTypeTMDbFacet)
			return n, nil
		}
	}

	slog.Debug("Facet-count cache miss", "key", cacheKey, "type", CacheTypeTMDbFacet)

	var count int
	switch kind {
	case "movie":
		result, _, err := s.client.DiscoverMoviesWithFallback(ctx, p)
		if err != nil {
			return 0, err
		}
		count = result.TotalResults
	case "tv":
		result, _, err := s.client.DiscoverTVShowsWithFallback(ctx, p)
		if err != nil {
			return 0, err
		}
		count = result.TotalResults
	default:
		return 0, fmt.Errorf("discoverCountCached: unknown kind %q", kind)
	}

	// AR-F5 / AC3 / CR M1: a 0 is cached only briefly (FacetCountZeroTTL), NOT the
	// full FacetCountCacheTTL — a transient/wrong-locale 0 self-corrects within ~1min
	// while a genuine dead-end 0 is still spared a re-fetch on every debounced probe.
	ttl := FacetCountCacheTTL
	if count == 0 {
		ttl = FacetCountZeroTTL
	}

	// AR-F6 / AC5: store the compact int string, not the SearchResult* blob.
	if err := s.cache.Set(ctx, cacheKey, strconv.Itoa(count), CacheTypeTMDbFacet, ttl); err != nil {
		slog.Warn("Failed to cache facet count", "key", cacheKey, "error", err)
	}
	return count, nil
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

// GetMovieRecommendations returns recommended movies with caching (24h TTL).
// Cache key format: tmdb:recommendations:movie:{id}:v1 (Rule 27 Pillar 2 —
// {source}:{type}:{id}:{version}; checked before the rate limiter).
func (s *CacheService) GetMovieRecommendations(ctx context.Context, movieID int) (*SearchResultMovies, error) {
	cacheKey := fmt.Sprintf("tmdb:recommendations:movie:%d:v1", movieID)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultMovies
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data", "key", cacheKey, "error", err)
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	result, lang, err := s.client.GetMovieRecommendationsWithFallback(ctx, movieID)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb movie recommendations completed",
		"movie_id", movieID, "language", lang, "results", len(result.Results))

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache movie recommendations", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}

// GetMovieSimilar returns similar movies with caching (24h TTL).
// Cache key format: tmdb:similar:movie:{id}:v1
func (s *CacheService) GetMovieSimilar(ctx context.Context, movieID int) (*SearchResultMovies, error) {
	cacheKey := fmt.Sprintf("tmdb:similar:movie:%d:v1", movieID)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultMovies
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data", "key", cacheKey, "error", err)
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	result, lang, err := s.client.GetMovieSimilarWithFallback(ctx, movieID)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb similar movies completed",
		"movie_id", movieID, "language", lang, "results", len(result.Results))

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache similar movies", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}

// GetTVRecommendations returns recommended TV shows with caching (24h TTL).
// Cache key format: tmdb:recommendations:tv:{id}:v1
func (s *CacheService) GetTVRecommendations(ctx context.Context, tvID int) (*SearchResultTVShows, error) {
	cacheKey := fmt.Sprintf("tmdb:recommendations:tv:%d:v1", tvID)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultTVShows
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data", "key", cacheKey, "error", err)
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	result, lang, err := s.client.GetTVRecommendationsWithFallback(ctx, tvID)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb TV recommendations completed",
		"tv_id", tvID, "language", lang, "results", len(result.Results))

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache TV recommendations", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}

// GetTVSimilar returns similar TV shows with caching (24h TTL).
// Cache key format: tmdb:similar:tv:{id}:v1
func (s *CacheService) GetTVSimilar(ctx context.Context, tvID int) (*SearchResultTVShows, error) {
	cacheKey := fmt.Sprintf("tmdb:similar:tv:%d:v1", tvID)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result SearchResultTVShows
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data", "key", cacheKey, "error", err)
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	result, lang, err := s.client.GetTVSimilarWithFallback(ctx, tvID)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb similar TV shows completed",
		"tv_id", tvID, "language", lang, "results", len(result.Results))

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache similar TV shows", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}

// GetWatchProviders returns streaming/rent/buy providers for a movie or TV show,
// filtered to a single region and cached at 24h (Rule 27 Pillar 2 — the ADR
// Pillar 2 table mandates 24h for watch providers; catalogs change daily at most).
// Cache key format: tmdb:watchproviders:{movie|tv}:{id}:{region}:v1 — region is
// part of the key because availability is region-specific. The cache is checked
// BEFORE the client (which fronts the rate limiter), per Rule 27 Pillar 2.
// Bypasses the language-fallback layer (watch-provider data is language-neutral).
func (s *CacheService) GetWatchProviders(ctx context.Context, mediaType string, id int, region string) (*WatchProvidersResponse, error) {
	cacheKey := fmt.Sprintf("tmdb:watchproviders:%s:%d:%s:v1", mediaType, id, region)

	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		var result WatchProvidersResponse
		if err := json.Unmarshal([]byte(cached.Value), &result); err == nil {
			slog.Debug("Cache hit", "key", cacheKey, "type", CacheTypeTMDb)
			return &result, nil
		}
		slog.Warn("Failed to unmarshal cached data", "key", cacheKey, "error", err)
	}

	slog.Debug("Cache miss", "key", cacheKey, "type", CacheTypeTMDb)
	if s.providersClient == nil {
		return nil, fmt.Errorf("watch-providers client not initialized")
	}
	result, err := s.providersClient.GetWatchProviders(ctx, mediaType, id, region)
	if err != nil {
		return nil, err
	}
	slog.Debug("TMDb watch providers completed", "media_type", mediaType, "id", id, "region", region)

	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data), CacheTypeTMDb, s.ttl); err != nil {
			slog.Warn("Failed to cache watch providers", "key", cacheKey, "error", err)
		}
	}
	return result, nil
}
