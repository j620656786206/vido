package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/repository"
)

// MockCacheRepository is a mock implementation of CacheRepositoryInterface.
// mu guards all shared state so the mock is safe under the concurrent
// DiscoverFacetCounts fan-out (Story ux3-discover-facet-aggregation-be) — the
// real SQLite-backed CacheRepository is database/sql-pool-safe; this mock needs an
// explicit lock to match. Sequential callers read the *Called / lastSetTTL fields
// after the call returns, so those plain-field reads remain race-free.
type MockCacheRepository struct {
	mu        sync.Mutex
	data      map[string]*repository.CacheEntry
	setError  error
	getError  error
	setCalled int
	getCalled int
	// lastSetTTL captures the TTL passed to the most recent Set() call,
	// used by Story 10-1 tests to verify 1-hour trending/discover TTL.
	lastSetTTL time.Duration
	// lastSetType / lastSetValue capture the cacheType and value of the most
	// recent Set() call, used by Story ux3-facet-count-cache-refinement to verify
	// the dedicated facet-count path tags entries "tmdb_facet" (AC2) and stores the
	// compact integer count string, not a SearchResult* JSON blob (AC5).
	lastSetType  string
	lastSetValue string
}

func NewMockCacheRepository() *MockCacheRepository {
	return &MockCacheRepository{
		data: make(map[string]*repository.CacheEntry),
	}
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (*repository.CacheEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalled++
	if m.getError != nil {
		return nil, m.getError
	}
	if entry, ok := m.data[key]; ok {
		return entry, nil
	}
	return nil, nil
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value string, cacheType string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalled++
	m.lastSetTTL = ttl
	m.lastSetType = cacheType
	m.lastSetValue = value
	if m.setError != nil {
		return m.setError
	}
	m.data[key] = &repository.CacheEntry{
		Key:       key,
		Value:     value,
		Type:      cacheType,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MockCacheRepository) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]*repository.CacheEntry)
	return nil
}

func (m *MockCacheRepository) ClearExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockCacheRepository) ClearByType(ctx context.Context, cacheType string) (int64, error) {
	return 0, nil
}

// MockFallbackClient is a mock implementation of LanguageFallbackClientInterface
type MockFallbackClient struct {
	SearchMoviesResponse     *SearchResultMovies
	SearchMoviesError        error
	SearchMoviesCalled       int
	SearchTVShowsResponse    *SearchResultTVShows
	SearchTVShowsError       error
	SearchTVShowsCalled      int
	GetMovieDetailsResponse  *MovieDetails
	GetMovieDetailsError     error
	GetMovieDetailsCalled    int
	GetTVShowDetailsResponse *TVShowDetails
	GetTVShowDetailsError    error
	GetTVShowDetailsCalled   int
	// Story 12-2 season details
	GetSeasonDetailsResponse *SeasonDetails
	GetSeasonDetailsError    error
	GetSeasonDetailsCalled   int
	// Story 10-1 trending/discover fields — embedded directly on the mock
	// (previously stored in a package-global map keyed by pointer; moved here
	// to remove the global-state anti-pattern and make t.Parallel() safe).
	TrendingMoviesResponse  *SearchResultMovies
	TrendingMoviesError     error
	TrendingMoviesCalled    int
	TrendingTVShowsResponse *SearchResultTVShows
	TrendingTVShowsError    error
	TrendingTVShowsCalled   int
	DiscoverMoviesResponse  *SearchResultMovies
	DiscoverMoviesError     error
	DiscoverMoviesCalled    int
	DiscoverTVShowsResponse *SearchResultTVShows
	DiscoverTVShowsError    error
	DiscoverTVShowsCalled   int
	// Story ux3-discover-facet-aggregation-be: optional per-call hooks for the
	// facet-count fan-out tests. When set, they OVERRIDE the static
	// Discover*Response/Error above so a test can return per-params counts, inject
	// per-value errors/delays, capture the normalized params (AC9), and observe
	// concurrency (AC4). discoverMu guards the *Called counters so the concurrent
	// fan-out increments them without a data race / lost update.
	DiscoverMoviesFunc  func(ctx context.Context, params DiscoverParams) (*SearchResultMovies, string, error)
	DiscoverTVShowsFunc func(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, string, error)
	discoverMu          sync.Mutex
	// Story 12-3 recommendations/similar
	MovieRecommendationsResponse *SearchResultMovies
	MovieRecommendationsError    error
	MovieRecommendationsCalled   int
	MovieSimilarResponse         *SearchResultMovies
	MovieSimilarError            error
	MovieSimilarCalled           int
	TVRecommendationsResponse    *SearchResultTVShows
	TVRecommendationsError       error
	TVRecommendationsCalled      int
	TVSimilarResponse            *SearchResultTVShows
	TVSimilarError               error
	TVSimilarCalled              int
}

func (m *MockFallbackClient) GetMovieRecommendationsWithFallback(ctx context.Context, movieID int) (*SearchResultMovies, string, error) {
	m.MovieRecommendationsCalled++
	if m.MovieRecommendationsError != nil {
		return nil, "", m.MovieRecommendationsError
	}
	if m.MovieRecommendationsResponse != nil {
		return m.MovieRecommendationsResponse, "zh-TW", nil
	}
	return &SearchResultMovies{Results: []Movie{}}, "zh-TW", nil
}

func (m *MockFallbackClient) GetMovieSimilarWithFallback(ctx context.Context, movieID int) (*SearchResultMovies, string, error) {
	m.MovieSimilarCalled++
	if m.MovieSimilarError != nil {
		return nil, "", m.MovieSimilarError
	}
	if m.MovieSimilarResponse != nil {
		return m.MovieSimilarResponse, "zh-TW", nil
	}
	return &SearchResultMovies{Results: []Movie{}}, "zh-TW", nil
}

func (m *MockFallbackClient) GetTVRecommendationsWithFallback(ctx context.Context, tvID int) (*SearchResultTVShows, string, error) {
	m.TVRecommendationsCalled++
	if m.TVRecommendationsError != nil {
		return nil, "", m.TVRecommendationsError
	}
	if m.TVRecommendationsResponse != nil {
		return m.TVRecommendationsResponse, "zh-TW", nil
	}
	return &SearchResultTVShows{Results: []TVShow{}}, "zh-TW", nil
}

func (m *MockFallbackClient) GetTVSimilarWithFallback(ctx context.Context, tvID int) (*SearchResultTVShows, string, error) {
	m.TVSimilarCalled++
	if m.TVSimilarError != nil {
		return nil, "", m.TVSimilarError
	}
	if m.TVSimilarResponse != nil {
		return m.TVSimilarResponse, "zh-TW", nil
	}
	return &SearchResultTVShows{Results: []TVShow{}}, "zh-TW", nil
}

func (m *MockFallbackClient) SearchMoviesWithFallback(ctx context.Context, query string, page int) (*SearchResultMovies, string, error) {
	m.SearchMoviesCalled++
	if m.SearchMoviesError != nil {
		return nil, "", m.SearchMoviesError
	}
	return m.SearchMoviesResponse, "zh-TW", nil
}

func (m *MockFallbackClient) SearchTVShowsWithFallback(ctx context.Context, query string, page int) (*SearchResultTVShows, string, error) {
	m.SearchTVShowsCalled++
	if m.SearchTVShowsError != nil {
		return nil, "", m.SearchTVShowsError
	}
	return m.SearchTVShowsResponse, "zh-TW", nil
}

func (m *MockFallbackClient) GetMovieDetailsWithFallback(ctx context.Context, movieID int) (*MovieDetails, string, error) {
	m.GetMovieDetailsCalled++
	if m.GetMovieDetailsError != nil {
		return nil, "", m.GetMovieDetailsError
	}
	return m.GetMovieDetailsResponse, "zh-TW", nil
}

func (m *MockFallbackClient) GetTVShowDetailsWithFallback(ctx context.Context, tvID int) (*TVShowDetails, string, error) {
	m.GetTVShowDetailsCalled++
	if m.GetTVShowDetailsError != nil {
		return nil, "", m.GetTVShowDetailsError
	}
	return m.GetTVShowDetailsResponse, "zh-TW", nil
}

func (m *MockFallbackClient) GetSeasonDetailsWithFallback(ctx context.Context, tvID int, seasonNumber int) (*SeasonDetails, string, error) {
	m.GetSeasonDetailsCalled++
	if m.GetSeasonDetailsError != nil {
		return nil, "", m.GetSeasonDetailsError
	}
	return m.GetSeasonDetailsResponse, "zh-TW", nil
}

// Story 10-1: LanguageFallbackClientInterface additions — configurable stubs
// with call counters so Story 10-1 cache tests can assert cache-hit vs cache-miss.
// Fields live directly on MockFallbackClient (see struct above) so each test
// owns its own state — no package-global map, safe for t.Parallel().

func (m *MockFallbackClient) GetTrendingMoviesWithFallback(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, string, error) {
	m.TrendingMoviesCalled++
	if m.TrendingMoviesError != nil {
		return nil, "", m.TrendingMoviesError
	}
	if m.TrendingMoviesResponse != nil {
		return m.TrendingMoviesResponse, "zh-TW", nil
	}
	return &SearchResultMovies{Page: page, Results: []Movie{}}, "zh-TW", nil
}

func (m *MockFallbackClient) GetTrendingTVShowsWithFallback(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, string, error) {
	m.TrendingTVShowsCalled++
	if m.TrendingTVShowsError != nil {
		return nil, "", m.TrendingTVShowsError
	}
	if m.TrendingTVShowsResponse != nil {
		return m.TrendingTVShowsResponse, "zh-TW", nil
	}
	return &SearchResultTVShows{Page: page, Results: []TVShow{}}, "zh-TW", nil
}

func (m *MockFallbackClient) DiscoverMoviesWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultMovies, string, error) {
	m.discoverMu.Lock()
	m.DiscoverMoviesCalled++
	m.discoverMu.Unlock()
	if m.DiscoverMoviesFunc != nil {
		return m.DiscoverMoviesFunc(ctx, params)
	}
	if m.DiscoverMoviesError != nil {
		return nil, "", m.DiscoverMoviesError
	}
	if m.DiscoverMoviesResponse != nil {
		return m.DiscoverMoviesResponse, "zh-TW", nil
	}
	return &SearchResultMovies{Page: 1, Results: []Movie{}}, "zh-TW", nil
}

func (m *MockFallbackClient) DiscoverTVShowsWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, string, error) {
	m.discoverMu.Lock()
	m.DiscoverTVShowsCalled++
	m.discoverMu.Unlock()
	if m.DiscoverTVShowsFunc != nil {
		return m.DiscoverTVShowsFunc(ctx, params)
	}
	if m.DiscoverTVShowsError != nil {
		return nil, "", m.DiscoverTVShowsError
	}
	if m.DiscoverTVShowsResponse != nil {
		return m.DiscoverTVShowsResponse, "zh-TW", nil
	}
	return &SearchResultTVShows{Page: 1, Results: []TVShow{}}, "zh-TW", nil
}

func TestNewCacheService(t *testing.T) {
	tests := []struct {
		name    string
		ttl     time.Duration
		wantTTL time.Duration
	}{
		{
			name:    "default TTL",
			ttl:     0,
			wantTTL: DefaultCacheTTL,
		},
		{
			name:    "custom TTL",
			ttl:     1 * time.Hour,
			wantTTL: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockFallbackClient{}
			cache := NewMockCacheRepository()

			service := NewCacheService(client, cache, CacheServiceConfig{TTL: tt.ttl})

			assert.NotNil(t, service)
			assert.Equal(t, tt.wantTTL, service.ttl)
		})
	}
}

func TestCacheService_SearchMovies(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		page          int
		cachedData    *SearchResultMovies
		apiResponse   *SearchResultMovies
		apiError      error
		wantFromCache bool
		wantAPICall   bool
		wantCacheSet  bool
		wantErr       bool
	}{
		{
			name:  "cache hit",
			query: "test",
			page:  1,
			cachedData: &SearchResultMovies{
				Page:         1,
				Results:      []Movie{{ID: 1, Title: "Cached Movie"}},
				TotalResults: 1,
			},
			wantFromCache: true,
			wantAPICall:   false,
			wantCacheSet:  false,
		},
		{
			name:  "cache miss, API success",
			query: "test",
			page:  1,
			apiResponse: &SearchResultMovies{
				Page:         1,
				Results:      []Movie{{ID: 2, Title: "API Movie"}},
				TotalResults: 1,
			},
			wantFromCache: false,
			wantAPICall:   true,
			wantCacheSet:  true,
		},
		{
			name:        "cache miss, API error",
			query:       "test",
			page:        1,
			apiError:    errors.New("API error"),
			wantAPICall: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewMockCacheRepository()
			client := &MockFallbackClient{
				SearchMoviesResponse: tt.apiResponse,
				SearchMoviesError:    tt.apiError,
			}

			// Pre-populate cache if needed
			if tt.cachedData != nil {
				data, _ := json.Marshal(tt.cachedData)
				cacheKey := "tmdb:search/movie:test:1"
				cache.Set(context.Background(), cacheKey, string(data), CacheTypeTMDb, DefaultCacheTTL)
				cache.setCalled = 0 // Reset counter
			}

			service := NewCacheService(client, cache, CacheServiceConfig{})
			result, err := service.SearchMovies(context.Background(), tt.query, tt.page)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.wantFromCache {
				assert.Equal(t, 0, client.SearchMoviesCalled)
				assert.Equal(t, tt.cachedData.Results[0].Title, result.Results[0].Title)
			}

			if tt.wantAPICall {
				assert.Equal(t, 1, client.SearchMoviesCalled)
			}

			if tt.wantCacheSet {
				assert.Equal(t, 1, cache.setCalled)
			}
		})
	}
}

func TestCacheService_SearchTVShows(t *testing.T) {
	tests := []struct {
		name          string
		cachedData    *SearchResultTVShows
		apiResponse   *SearchResultTVShows
		wantFromCache bool
		wantAPICall   bool
	}{
		{
			name: "cache hit",
			cachedData: &SearchResultTVShows{
				Page:    1,
				Results: []TVShow{{ID: 1, Name: "Cached Show"}},
			},
			wantFromCache: true,
			wantAPICall:   false,
		},
		{
			name: "cache miss",
			apiResponse: &SearchResultTVShows{
				Page:    1,
				Results: []TVShow{{ID: 2, Name: "API Show"}},
			},
			wantFromCache: false,
			wantAPICall:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewMockCacheRepository()
			client := &MockFallbackClient{
				SearchTVShowsResponse: tt.apiResponse,
			}

			if tt.cachedData != nil {
				data, _ := json.Marshal(tt.cachedData)
				cache.Set(context.Background(), "tmdb:search/tv:test:1", string(data), CacheTypeTMDb, DefaultCacheTTL)
			}

			service := NewCacheService(client, cache, CacheServiceConfig{})
			result, err := service.SearchTVShows(context.Background(), "test", 1)

			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.wantFromCache {
				assert.Equal(t, 0, client.SearchTVShowsCalled)
			}
			if tt.wantAPICall {
				assert.Equal(t, 1, client.SearchTVShowsCalled)
			}
		})
	}
}

func TestCacheService_GetMovieDetails(t *testing.T) {
	tests := []struct {
		name          string
		movieID       int
		cachedData    *MovieDetails
		apiResponse   *MovieDetails
		wantFromCache bool
		wantAPICall   bool
	}{
		{
			name:    "cache hit",
			movieID: 550,
			cachedData: &MovieDetails{
				Movie: Movie{ID: 550, Title: "Cached Movie"},
			},
			wantFromCache: true,
			wantAPICall:   false,
		},
		{
			name:    "cache miss",
			movieID: 550,
			apiResponse: &MovieDetails{
				Movie: Movie{ID: 550, Title: "API Movie"},
			},
			wantFromCache: false,
			wantAPICall:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewMockCacheRepository()
			client := &MockFallbackClient{
				GetMovieDetailsResponse: tt.apiResponse,
			}

			if tt.cachedData != nil {
				data, _ := json.Marshal(tt.cachedData)
				cache.Set(context.Background(), "tmdb:movie/550", string(data), CacheTypeTMDb, DefaultCacheTTL)
			}

			service := NewCacheService(client, cache, CacheServiceConfig{})
			result, err := service.GetMovieDetails(context.Background(), tt.movieID)

			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.wantFromCache {
				assert.Equal(t, 0, client.GetMovieDetailsCalled)
			}
			if tt.wantAPICall {
				assert.Equal(t, 1, client.GetMovieDetailsCalled)
			}
		})
	}
}

func TestCacheService_GetTVShowDetails(t *testing.T) {
	tests := []struct {
		name          string
		tvID          int
		cachedData    *TVShowDetails
		apiResponse   *TVShowDetails
		wantFromCache bool
		wantAPICall   bool
	}{
		{
			name: "cache hit",
			tvID: 1396,
			cachedData: &TVShowDetails{
				TVShow: TVShow{ID: 1396, Name: "Cached Show"},
			},
			wantFromCache: true,
			wantAPICall:   false,
		},
		{
			name: "cache miss",
			tvID: 1396,
			apiResponse: &TVShowDetails{
				TVShow: TVShow{ID: 1396, Name: "API Show"},
			},
			wantFromCache: false,
			wantAPICall:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewMockCacheRepository()
			client := &MockFallbackClient{
				GetTVShowDetailsResponse: tt.apiResponse,
			}

			if tt.cachedData != nil {
				data, _ := json.Marshal(tt.cachedData)
				cache.Set(context.Background(), "tmdb:tv/1396", string(data), CacheTypeTMDb, DefaultCacheTTL)
			}

			service := NewCacheService(client, cache, CacheServiceConfig{})
			result, err := service.GetTVShowDetails(context.Background(), tt.tvID)

			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.wantFromCache {
				assert.Equal(t, 0, client.GetTVShowDetailsCalled)
			}
			if tt.wantAPICall {
				assert.Equal(t, 1, client.GetTVShowDetailsCalled)
			}
		})
	}
}

func TestCacheService_GetSeasonDetails(t *testing.T) {
	tests := []struct {
		name          string
		tvID          int
		seasonNumber  int
		cachedData    *SeasonDetails
		apiResponse   *SeasonDetails
		wantFromCache bool
		wantAPICall   bool
	}{
		{
			name:         "cache hit",
			tvID:         1396,
			seasonNumber: 1,
			cachedData: &SeasonDetails{
				ID: 3572, Name: "Cached Season", SeasonNumber: 1,
				Episodes: []EpisodeInfo{{ID: 1, EpisodeNumber: 1, Name: "Cached Ep"}},
			},
			wantFromCache: true,
			wantAPICall:   false,
		},
		{
			name:         "cache miss",
			tvID:         1396,
			seasonNumber: 1,
			apiResponse: &SeasonDetails{
				ID: 3572, Name: "API Season", SeasonNumber: 1,
				Episodes: []EpisodeInfo{{ID: 1, EpisodeNumber: 1, Name: "API Ep"}},
			},
			wantFromCache: false,
			wantAPICall:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewMockCacheRepository()
			client := &MockFallbackClient{
				GetSeasonDetailsResponse: tt.apiResponse,
			}

			if tt.cachedData != nil {
				data, _ := json.Marshal(tt.cachedData)
				cache.Set(context.Background(), "tmdb:tv/1396/season/1", string(data), CacheTypeTMDb, DefaultCacheTTL)
			}

			service := NewCacheService(client, cache, CacheServiceConfig{})
			result, err := service.GetSeasonDetails(context.Background(), tt.tvID, tt.seasonNumber)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Episodes, 1)

			if tt.wantFromCache {
				assert.Equal(t, 0, client.GetSeasonDetailsCalled)
				assert.Equal(t, "Cached Ep", result.Episodes[0].Name)
			}
			if tt.wantAPICall {
				assert.Equal(t, 1, client.GetSeasonDetailsCalled)
				assert.Equal(t, "API Ep", result.Episodes[0].Name)
			}
		})
	}
}

// --- Story 12-3 recommendations/similar cache tests ---

func TestCacheService_GetMovieRecommendations_CacheMissThenHit(t *testing.T) {
	repo := NewMockCacheRepository()
	fbClient := &MockFallbackClient{
		MovieRecommendationsResponse: &SearchResultMovies{
			Results: []Movie{{ID: 1, Title: "Rec", Overview: "ov"}},
		},
	}
	service := NewCacheService(fbClient, repo, CacheServiceConfig{})

	// Cache miss → fetch from fallback client + store under the v1 key.
	r1, err := service.GetMovieRecommendations(context.Background(), 603)
	require.NoError(t, err)
	assert.Len(t, r1.Results, 1)
	assert.Equal(t, 1, fbClient.MovieRecommendationsCalled)

	cached, _ := repo.Get(context.Background(), "tmdb:recommendations:movie:603:v1")
	require.NotNil(t, cached, "expected entry under tmdb:recommendations:movie:603:v1")

	// Cache hit → no second fallback call.
	r2, err := service.GetMovieRecommendations(context.Background(), 603)
	require.NoError(t, err)
	assert.Len(t, r2.Results, 1)
	assert.Equal(t, 1, fbClient.MovieRecommendationsCalled)
}

func TestCacheService_GetTVSimilar_CacheKeyAndMissThenHit(t *testing.T) {
	repo := NewMockCacheRepository()
	fbClient := &MockFallbackClient{
		TVSimilarResponse: &SearchResultTVShows{
			Results: []TVShow{{ID: 7, Name: "Sim", Overview: "ov"}},
		},
	}
	service := NewCacheService(fbClient, repo, CacheServiceConfig{})

	_, err := service.GetTVSimilar(context.Background(), 1396)
	require.NoError(t, err)
	assert.Equal(t, 1, fbClient.TVSimilarCalled)

	cached, _ := repo.Get(context.Background(), "tmdb:similar:tv:1396:v1")
	require.NotNil(t, cached, "expected entry under tmdb:similar:tv:1396:v1")

	_, err = service.GetTVSimilar(context.Background(), 1396)
	require.NoError(t, err)
	assert.Equal(t, 1, fbClient.TVSimilarCalled) // served from cache
}

// --- Story 12-4 watch-providers cache tests ---

func TestCacheService_GetWatchProviders_CacheMissThenHit_24hTTL(t *testing.T) {
	repo := NewMockCacheRepository()
	rawClient := &MockClient{
		WatchProvidersResponse: &WatchProvidersResponse{
			ID: 550,
			Results: map[string]WatchProviderRegion{
				"TW": {Link: "https://example/tw", Flatrate: []WatchProvider{{ProviderID: 8, ProviderName: "Netflix"}}},
			},
		},
	}
	svc := NewCacheService(&MockFallbackClient{}, repo, CacheServiceConfig{TTL: DefaultCacheTTL})
	svc.SetProvidersClient(rawClient)

	// Miss → fetch from raw client (bypassing language fallback) + store under the
	// region-keyed v1 key at 24h TTL.
	r1, err := svc.GetWatchProviders(context.Background(), "movie", 550, "TW")
	require.NoError(t, err)
	require.NotNil(t, r1)
	require.Contains(t, r1.Results, "TW")
	assert.Equal(t, 1, rawClient.WatchProvidersCalled)
	assert.Equal(t, DefaultCacheTTL, repo.lastSetTTL, "watch providers must cache at 24h (ADR Pillar 2)")

	cached, _ := repo.Get(context.Background(), "tmdb:watchproviders:movie:550:TW:v1")
	require.NotNil(t, cached, "expected entry under tmdb:watchproviders:movie:550:TW:v1 (region in key)")

	// Hit → no second upstream call.
	r2, err := svc.GetWatchProviders(context.Background(), "movie", 550, "TW")
	require.NoError(t, err)
	require.Contains(t, r2.Results, "TW")
	assert.Equal(t, 1, rawClient.WatchProvidersCalled, "cache hit must not call upstream again")
}

func TestCacheService_GetWatchProviders_RegionInKey(t *testing.T) {
	repo := NewMockCacheRepository()
	rawClient := &MockClient{WatchProvidersResponse: &WatchProvidersResponse{ID: 550, Results: map[string]WatchProviderRegion{}}}
	svc := NewCacheService(&MockFallbackClient{}, repo, CacheServiceConfig{})
	svc.SetProvidersClient(rawClient)

	_, err := svc.GetWatchProviders(context.Background(), "tv", 1396, "TW")
	require.NoError(t, err)
	_, err = svc.GetWatchProviders(context.Background(), "tv", 1396, "US")
	require.NoError(t, err)

	// Distinct regions → distinct cache entries → two upstream calls.
	assert.Equal(t, 2, rawClient.WatchProvidersCalled)
	tw, _ := repo.Get(context.Background(), "tmdb:watchproviders:tv:1396:TW:v1")
	us, _ := repo.Get(context.Background(), "tmdb:watchproviders:tv:1396:US:v1")
	require.NotNil(t, tw)
	require.NotNil(t, us)
}

func TestCacheService_GetWatchProviders_NilClient(t *testing.T) {
	repo := NewMockCacheRepository()
	svc := NewCacheService(&MockFallbackClient{}, repo, CacheServiceConfig{})
	// SetProvidersClient deliberately NOT called.
	_, err := svc.GetWatchProviders(context.Background(), "movie", 550, "TW")
	require.Error(t, err, "must error (not panic) when the providers client is unset")
}

func TestCacheService_CacheSetError(t *testing.T) {
	// Test that cache set errors don't fail the request
	cache := NewMockCacheRepository()
	cache.setError = errors.New("cache set error")

	client := &MockFallbackClient{
		SearchMoviesResponse: &SearchResultMovies{
			Page:    1,
			Results: []Movie{{ID: 1, Title: "Movie"}},
		},
	}

	service := NewCacheService(client, cache, CacheServiceConfig{})
	result, err := service.SearchMovies(context.Background(), "test", 1)

	// Should still succeed even though cache set failed
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Results, 1)
}

func TestCacheService_InterfaceCompliance(t *testing.T) {
	var _ CacheServiceInterface = (*CacheService)(nil)
}

// --- Story 10-1 cache tests ---

func TestCacheService_GetTrendingMovies_CacheMissThenHit(t *testing.T) {
	repo := NewMockCacheRepository()
	fbClient := &MockFallbackClient{
		TrendingMoviesResponse: &SearchResultMovies{
			Page:         1,
			Results:      []Movie{{ID: 42, Title: "Hot"}},
			TotalPages:   1,
			TotalResults: 1,
		},
	}
	svc := NewCacheService(fbClient, repo, CacheServiceConfig{TTL: 24 * time.Hour})

	// Miss → upstream call
	r1, err := svc.GetTrendingMovies(context.Background(), "week", 1)
	require.NoError(t, err)
	require.NotNil(t, r1)
	assert.Equal(t, 42, r1.Results[0].ID)
	assert.Equal(t, 1, fbClient.TrendingMoviesCalled)
	assert.Equal(t, 1, repo.setCalled)
	assert.Equal(t, TrendingDiscoverCacheTTL, repo.lastSetTTL, "cache TTL must be 1 hour for trending (AC #5)")

	// Hit → no upstream call
	r2, err := svc.GetTrendingMovies(context.Background(), "week", 1)
	require.NoError(t, err)
	assert.Equal(t, 42, r2.Results[0].ID)
	assert.Equal(t, 1, fbClient.TrendingMoviesCalled, "cache hit should not call upstream again")
}

// TestCacheService_DiscoverMovies_CacheMissThenHit is the discover-side mirror
// of TestCacheService_GetTrendingMovies_CacheMissThenHit. It directly exercises
// the caching mechanism that delivers AC #4 (<500ms for any filter combination):
// an identical second query for the same DiscoverParams must be served from
// cache WITHOUT a second upstream call. [P1]
func TestCacheService_DiscoverMovies_CacheMissThenHit(t *testing.T) {
	repo := NewMockCacheRepository()
	fbClient := &MockFallbackClient{
		DiscoverMoviesResponse: &SearchResultMovies{
			Page:         1,
			Results:      []Movie{{ID: 99, Title: "Filtered"}},
			TotalPages:   1,
			TotalResults: 1,
		},
	}
	svc := NewCacheService(fbClient, repo, CacheServiceConfig{TTL: 24 * time.Hour})

	params := DiscoverParams{
		GenreIDs: []int{28, 18}, YearGte: 2024, Region: "TW",
		VoteAverageGte: 7, WatchProviders: []int{8}, WatchRegion: "TW",
		SortBy: "popularity.desc", Page: 1,
	}

	// Miss → upstream call, cached at the 1-hour discover TTL.
	r1, err := svc.DiscoverMovies(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, r1)
	assert.Equal(t, 99, r1.Results[0].ID)
	assert.Equal(t, 1, fbClient.DiscoverMoviesCalled)
	assert.Equal(t, TrendingDiscoverCacheTTL, repo.lastSetTTL, "discover cache TTL must be 1 hour (AC #4)")

	// Hit → identical params served from cache, NO second upstream call.
	r2, err := svc.DiscoverMovies(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, 99, r2.Results[0].ID)
	assert.Equal(t, 1, fbClient.DiscoverMoviesCalled, "identical filter query must hit cache, not re-call upstream")
}

func TestCacheService_DiscoverMovies_DifferentParamsDifferentKeys(t *testing.T) {
	repo := NewMockCacheRepository()
	fbClient := &MockFallbackClient{
		DiscoverMoviesResponse: &SearchResultMovies{Page: 1, Results: []Movie{{ID: 1}}},
	}
	svc := NewCacheService(fbClient, repo, CacheServiceConfig{TTL: 24 * time.Hour})

	_, err := svc.DiscoverMovies(context.Background(), DiscoverParams{GenreIDs: []int{28}, YearGte: 2024})
	require.NoError(t, err)
	_, err = svc.DiscoverMovies(context.Background(), DiscoverParams{GenreIDs: []int{28}, YearGte: 2025})
	require.NoError(t, err)

	// Two distinct param sets → two upstream calls (different cache keys)
	assert.Equal(t, 2, fbClient.DiscoverMoviesCalled)
	assert.Equal(t, 2, repo.setCalled)
	assert.Equal(t, TrendingDiscoverCacheTTL, repo.lastSetTTL)
}

// TestDiscoverCacheKey_AllDimensionsDistinct verifies the cache key includes
// every filter dimension, so two queries differing in a single dimension never
// collide (Story 11-1 — Murat's cache-key concern). Each entry below differs
// from the base in exactly one field.
func TestDiscoverCacheKey_AllDimensionsDistinct(t *testing.T) {
	base := DiscoverParams{
		GenreIDs: []int{28}, YearGte: 2024, YearLte: 2025, Region: "TW",
		VoteAverageGte: 7, VoteAverageLte: 9, WatchProviders: []int{8},
		WatchRegion: "TW", Language: "zh-TW", SortBy: "popularity.desc", Page: 1,
	}
	variants := map[string]DiscoverParams{
		"base":            base,
		"genre":           withParam(base, func(p *DiscoverParams) { p.GenreIDs = []int{18} }),
		"year_gte":        withParam(base, func(p *DiscoverParams) { p.YearGte = 2020 }),
		"vote_gte":        withParam(base, func(p *DiscoverParams) { p.VoteAverageGte = 8 }),
		"vote_lte":        withParam(base, func(p *DiscoverParams) { p.VoteAverageLte = 10 }),
		"watch_providers": withParam(base, func(p *DiscoverParams) { p.WatchProviders = []int{337} }),
		"watch_region":    withParam(base, func(p *DiscoverParams) { p.WatchRegion = "US" }),
		"sort":            withParam(base, func(p *DiscoverParams) { p.SortBy = "vote_average.desc" }),
		"page":            withParam(base, func(p *DiscoverParams) { p.Page = 2 }),
	}

	seen := make(map[string]string, len(variants))
	for name, p := range variants {
		key := discoverCacheKey("movie", p)
		if other, dup := seen[key]; dup {
			t.Fatalf("cache key collision: %q and %q produced the same key %q", name, other, key)
		}
		seen[key] = name
	}

	// Identical params must produce identical keys (determinism).
	assert.Equal(t, discoverCacheKey("movie", base), discoverCacheKey("movie", base))
}

// withParam returns a copy of p mutated by fn — a tiny helper so each variant
// above differs from base in exactly one dimension.
func withParam(p DiscoverParams, fn func(*DiscoverParams)) DiscoverParams {
	fn(&p)
	return p
}

func TestCacheService_GetTrendingTVShows_TTLIsOneHour(t *testing.T) {
	repo := NewMockCacheRepository()
	fbClient := &MockFallbackClient{}
	svc := NewCacheService(fbClient, repo, CacheServiceConfig{TTL: DefaultCacheTTL})

	_, err := svc.GetTrendingTVShows(context.Background(), "day", 1)
	require.NoError(t, err)
	// TTL for trending/discover is hardcoded to 1h, NOT the service's DefaultCacheTTL
	assert.Equal(t, 1*time.Hour, repo.lastSetTTL)
}

func TestCacheService_DiscoverTVShows_Error_Propagates(t *testing.T) {
	repo := NewMockCacheRepository()
	fbClient := &MockFallbackClient{
		DiscoverTVShowsError: errors.New("upstream boom"),
	}
	svc := NewCacheService(fbClient, repo, CacheServiceConfig{})

	_, err := svc.DiscoverTVShows(context.Background(), DiscoverParams{GenreIDs: []int{18}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upstream boom")
	assert.Equal(t, 0, repo.setCalled, "error must NOT write to cache")
}

// TestCacheService_TrendingDiscover_RateLimitNotCached verifies that when
// upstream returns a TMDb 429 (rate-limit) or 5xx server error, none of the
// four trending/discover methods write a poisoned cache entry. This protects
// against accidental "negative caching" — the next request must be free to
// retry upstream once the rate-limit window passes.
func TestCacheService_TrendingDiscover_RateLimitNotCached(t *testing.T) {
	cases := []struct {
		name string
		call func(svc *CacheService) error
		set  func(m *MockFallbackClient)
	}{
		{
			name: "GetTrendingMovies/rate-limit",
			call: func(s *CacheService) error {
				_, err := s.GetTrendingMovies(context.Background(), "week", 1)
				return err
			},
			set: func(m *MockFallbackClient) { m.TrendingMoviesError = NewRateLimitError() },
		},
		{
			name: "GetTrendingTVShows/server-5xx",
			call: func(s *CacheService) error {
				_, err := s.GetTrendingTVShows(context.Background(), "day", 1)
				return err
			},
			set: func(m *MockFallbackClient) { m.TrendingTVShowsError = NewServerError(errors.New("HTTP 503")) },
		},
		{
			name: "DiscoverMovies/rate-limit",
			call: func(s *CacheService) error {
				_, err := s.DiscoverMovies(context.Background(), DiscoverParams{GenreIDs: []int{28}})
				return err
			},
			set: func(m *MockFallbackClient) { m.DiscoverMoviesError = NewRateLimitError() },
		},
		{
			name: "DiscoverTVShows/server-5xx",
			call: func(s *CacheService) error {
				_, err := s.DiscoverTVShows(context.Background(), DiscoverParams{Region: "TW"})
				return err
			},
			set: func(m *MockFallbackClient) { m.DiscoverTVShowsError = NewServerError(errors.New("HTTP 502")) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewMockCacheRepository()
			fbClient := &MockFallbackClient{}
			tc.set(fbClient)
			svc := NewCacheService(fbClient, repo, CacheServiceConfig{})

			err := tc.call(svc)
			require.Error(t, err, "upstream error must surface")

			var tmdbErr *TMDbError
			require.ErrorAs(t, err, &tmdbErr, "error must remain a *TMDbError so handler can map to correct HTTP status")

			assert.Equal(t, 0, repo.setCalled, "rate-limit / server error MUST NOT poison cache")
		})
	}
}

// ---------------------------------------------------------------------------
// Story ux3-discover-facet-aggregation-be — DiscoverFacetCounts (AC1–AC10)
// ---------------------------------------------------------------------------

// paramCapture records the (normalized) DiscoverParams handed to the fan-out
// sub-queries so tests can assert add-semantics (AC2) and normalization (AC9).
type paramCapture struct {
	mu     sync.Mutex
	params []DiscoverParams
}

func (c *paramCapture) record(p DiscoverParams) {
	c.mu.Lock()
	c.params = append(c.params, p)
	c.mu.Unlock()
}

func (c *paramCapture) snapshot() []DiscoverParams {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]DiscoverParams, len(c.params))
	copy(out, c.params)
	return out
}

// TestCacheService_DiscoverFacetCounts_MovieTVSumPerValue covers AC1: the count
// for each candidate value is the SUMMED movie + tv total_results, mapped under
// counts[dim][value as supplied]. The hooks return per-genre totals so the test
// proves the per-value mapping, not just a constant.
func TestCacheService_DiscoverFacetCounts_MovieTVSumPerValue(t *testing.T) {
	repo := NewMockCacheRepository()
	fb := &MockFallbackClient{
		DiscoverMoviesFunc: func(_ context.Context, p DiscoverParams) (*SearchResultMovies, string, error) {
			id := p.GenreIDs[len(p.GenreIDs)-1]
			return &SearchResultMovies{TotalResults: id * 10}, "zh-TW", nil
		},
		DiscoverTVShowsFunc: func(_ context.Context, p DiscoverParams) (*SearchResultTVShows, string, error) {
			id := p.GenreIDs[len(p.GenreIDs)-1]
			return &SearchResultTVShows{TotalResults: id}, "zh-TW", nil
		},
	}
	svc := NewCacheService(fb, repo, CacheServiceConfig{})

	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{
		DimGenre: {"28", "12"},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Partial, "all facets resolved → not partial")
	assert.Equal(t, 28*10+28, res.Counts[DimGenre]["28"], "movie(280)+tv(28)")
	assert.Equal(t, 12*10+12, res.Counts[DimGenre]["12"], "movie(120)+tv(12)")
}

// TestCacheService_DiscoverFacetCounts_AddSemantics covers AC2: multi-select
// dims (genre, platform) APPEND the candidate onto the base selection;
// single-select dims (region, rating) REPLACE/SET it. Each row supplies a single
// candidate so the one captured param set is asserted directly.
func TestCacheService_DiscoverFacetCounts_AddSemantics(t *testing.T) {
	tests := []struct {
		name   string
		base   DiscoverParams
		dim    string
		value  string
		assert func(t *testing.T, p DiscoverParams)
	}{
		{
			name:  "genre appends onto existing genre selection",
			base:  DiscoverParams{GenreIDs: []int{28}},
			dim:   DimGenre,
			value: "18",
			assert: func(t *testing.T, p DiscoverParams) {
				assert.Equal(t, []int{28, 18}, p.GenreIDs, "動作 ∩ candidate, not candidate-alone")
			},
		},
		{
			name:  "region replaces (single-select) while keeping base genre",
			base:  DiscoverParams{GenreIDs: []int{28}, Region: "US"},
			dim:   DimRegion,
			value: "KR",
			assert: func(t *testing.T, p DiscoverParams) {
				assert.Equal(t, "KR", p.Region, "region SET to candidate")
				assert.Equal(t, []int{28}, p.GenreIDs, "base genre preserved → 動作 ∩ 韓國")
			},
		},
		{
			name:  "platform appends + sets watch_region from base region",
			base:  DiscoverParams{Region: "US", WatchProviders: []int{8}},
			dim:   DimPlatform,
			value: "337",
			assert: func(t *testing.T, p DiscoverParams) {
				assert.Equal(t, []int{8, 337}, p.WatchProviders, "platform appends")
				assert.Equal(t, "US", p.WatchRegion, "watch_region defaults to base region")
			},
		},
		{
			name:  "platform watch_region falls back to TW when base region empty",
			base:  DiscoverParams{},
			dim:   DimPlatform,
			value: "8",
			assert: func(t *testing.T, p DiscoverParams) {
				assert.Equal(t, []int{8}, p.WatchProviders)
				assert.Equal(t, "TW", p.WatchRegion, "TMDb requires watch_region with with_watch_providers")
			},
		},
		{
			name:  "rating sets VoteAverageGte (single-select)",
			base:  DiscoverParams{GenreIDs: []int{28}},
			dim:   DimRating,
			value: "8",
			assert: func(t *testing.T, p DiscoverParams) {
				assert.InEpsilon(t, 8.0, p.VoteAverageGte, 1e-9)
				assert.Equal(t, []int{28}, p.GenreIDs, "base genre preserved")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capt := &paramCapture{}
			fb := &MockFallbackClient{
				DiscoverMoviesFunc: func(_ context.Context, p DiscoverParams) (*SearchResultMovies, string, error) {
					capt.record(p)
					return &SearchResultMovies{TotalResults: 1}, "zh-TW", nil
				},
				DiscoverTVShowsFunc: func(_ context.Context, _ DiscoverParams) (*SearchResultTVShows, string, error) {
					return &SearchResultTVShows{TotalResults: 0}, "zh-TW", nil
				},
			}
			svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

			_, err := svc.DiscoverFacetCounts(context.Background(), tc.base, map[string][]string{tc.dim: {tc.value}})
			require.NoError(t, err)

			snap := capt.snapshot()
			require.Len(t, snap, 1, "exactly one probe issued")
			// Base slices must never be mutated by the fan-out.
			tc.assert(t, snap[0])
		})
	}
}

// TestCacheService_DiscoverFacetCounts_Normalization covers AC9 (AR-F2/F3): every
// sub-query is issued with SortBy="" and Page=1 (so sorts/pages share one cache
// entry) and an explicit non-empty Language (so the fallback chain is bypassed).
// The caller's pinned Language wins when set, else defaultFacetCountLanguage.
func TestCacheService_DiscoverFacetCounts_Normalization(t *testing.T) {
	tests := []struct {
		name         string
		baseLanguage string
		wantLanguage string
	}{
		{name: "unset language pins default zh-TW", baseLanguage: "", wantLanguage: defaultFacetCountLanguage},
		{name: "caller language is honored", baseLanguage: "en", wantLanguage: "en"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capt := &paramCapture{}
			fb := &MockFallbackClient{
				DiscoverMoviesFunc: func(_ context.Context, p DiscoverParams) (*SearchResultMovies, string, error) {
					capt.record(p)
					return &SearchResultMovies{TotalResults: 1}, "zh-TW", nil
				},
				DiscoverTVShowsFunc: func(_ context.Context, _ DiscoverParams) (*SearchResultTVShows, string, error) {
					return &SearchResultTVShows{TotalResults: 1}, "zh-TW", nil
				},
			}
			svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

			base := DiscoverParams{SortBy: "popularity.desc", Page: 5, Language: tc.baseLanguage}
			_, err := svc.DiscoverFacetCounts(context.Background(), base, map[string][]string{DimGenre: {"28"}})
			require.NoError(t, err)

			snap := capt.snapshot()
			require.Len(t, snap, 1)
			assert.Equal(t, "", snap[0].SortBy, "sort cleared (AR-F2)")
			assert.Equal(t, 1, snap[0].Page, "page pinned to 1 (AR-F2)")
			assert.Equal(t, tc.wantLanguage, snap[0].Language, "language pinned non-empty (AR-F3)")
		})
	}
}

// TestCacheService_DiscoverFacetCounts_CacheReuse covers AC4 (count-to-count reuse):
// an identical normalized, NON-ZERO facet sub-query within FacetCountCacheTTL is
// served from the DEDICATED facet-count cache — NO new upstream (fallback-client)
// call. It also proves the dedicated-path tagging: entries ride FacetCountCacheTTL
// (AC1), are tagged type="tmdb_facet" (AC2), and store the compact integer count
// string, not a SearchResult* JSON blob (AC5).
func TestCacheService_DiscoverFacetCounts_CacheReuse(t *testing.T) {
	repo := NewMockCacheRepository()
	fb := &MockFallbackClient{
		DiscoverMoviesResponse:  &SearchResultMovies{TotalResults: 340},
		DiscoverTVShowsResponse: &SearchResultTVShows{TotalResults: 60},
	}
	svc := NewCacheService(fb, repo, CacheServiceConfig{})

	candidates := map[string][]string{DimGenre: {"28"}}

	// First call → cache miss → one movie + one tv upstream call.
	r1, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, candidates)
	require.NoError(t, err)
	assert.Equal(t, 400, r1.Counts[DimGenre]["28"])
	assert.Equal(t, 1, fb.DiscoverMoviesCalled)
	assert.Equal(t, 1, fb.DiscoverTVShowsCalled)
	// AC1 (AR-F8): counts ride the DEDICATED facet-count TTL, not the trending path.
	assert.Equal(t, FacetCountCacheTTL, repo.lastSetTTL, "counts ride the dedicated facet-count cache TTL (AR-F8)")
	// AC2 (AR-F4): count entries are tagged the distinct "tmdb_facet" type.
	assert.Equal(t, CacheTypeTMDbFacet, repo.lastSetType, "count entries must be tagged tmdb_facet, not the grid's tmdb (AR-F4)")
	// AC5 (AR-F6): the cached value is the compact int string, not a JSON blob — and
	// it is the exact count. Single-probe fan-out sets movie ("340") then tv ("60")
	// sequentially, so the LAST Set captured is tv's 60.
	n, convErr := strconv.Atoi(repo.lastSetValue)
	require.NoError(t, convErr, "cached facet-count value must parse as an int, not a SearchResult JSON blob")
	assert.Equal(t, 60, n, "stored value is the exact tv count (60), not a blob — movie-then-tv Set order")

	// Second identical call → served from the dedicated count cache → upstream unchanged.
	r2, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, candidates)
	require.NoError(t, err)
	assert.Equal(t, 400, r2.Counts[DimGenre]["28"])
	assert.Equal(t, 1, fb.DiscoverMoviesCalled, "warm non-zero facet must NOT re-call upstream (AC4)")
	assert.Equal(t, 1, fb.DiscoverTVShowsCalled)
}

// TestCacheService_DiscoverFacetCounts_ConcurrencyBound covers AC4: the fan-out
// never runs more than facetCountConcurrency sub-queries at once, so interactive
// TMDb calls are not starved. The hooks track max observed in-flight.
func TestCacheService_DiscoverFacetCounts_ConcurrencyBound(t *testing.T) {
	var mu sync.Mutex
	inFlight, maxInFlight := 0, 0
	track := func() func() {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()
		return func() {
			mu.Lock()
			inFlight--
			mu.Unlock()
		}
	}

	fb := &MockFallbackClient{
		DiscoverMoviesFunc: func(_ context.Context, _ DiscoverParams) (*SearchResultMovies, string, error) {
			done := track()
			time.Sleep(20 * time.Millisecond) // hold the slot so overlap is observable
			done()
			return &SearchResultMovies{TotalResults: 1}, "zh-TW", nil
		},
		DiscoverTVShowsFunc: func(_ context.Context, _ DiscoverParams) (*SearchResultTVShows, string, error) {
			done := track()
			time.Sleep(20 * time.Millisecond)
			done()
			return &SearchResultTVShows{TotalResults: 1}, "zh-TW", nil
		},
	}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	// 10 candidate genres → 20 sub-queries through a SetLimit(4) gate.
	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{
		DimGenre: {"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
	})
	require.NoError(t, err)
	assert.Len(t, res.Counts[DimGenre], 10, "all 10 facets resolve within budget")

	mu.Lock()
	got := maxInFlight
	mu.Unlock()
	assert.LessOrEqual(t, got, facetCountConcurrency, "concurrent sub-queries must not exceed SetLimit(N)")
	assert.Greater(t, got, 1, "sanity: the fan-out actually ran concurrently")
}

// TestCacheService_DiscoverFacetCounts_PartialOnTimeout covers AC5: facets not
// resolved within the time budget are omitted and Partial is set; resolved facets
// are still returned. A tight caller deadline shortens the budget for the test.
func TestCacheService_DiscoverFacetCounts_PartialOnTimeout(t *testing.T) {
	fb := &MockFallbackClient{
		DiscoverMoviesFunc: func(ctx context.Context, p DiscoverParams) (*SearchResultMovies, string, error) {
			if p.GenreIDs[len(p.GenreIDs)-1] == 99 { // the cold/slow facet
				select {
				case <-ctx.Done():
					return nil, "", ctx.Err()
				case <-time.After(2 * time.Second):
					return &SearchResultMovies{TotalResults: 5}, "zh-TW", nil
				}
			}
			return &SearchResultMovies{TotalResults: 100}, "zh-TW", nil
		},
		DiscoverTVShowsFunc: func(_ context.Context, _ DiscoverParams) (*SearchResultTVShows, string, error) {
			return &SearchResultTVShows{TotalResults: 0}, "zh-TW", nil
		},
	}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	res, err := svc.DiscoverFacetCounts(ctx, DiscoverParams{}, map[string][]string{
		DimGenre: {"28", "12", "99"},
	})
	require.NoError(t, err, "budget exhaustion degrades, never errors")
	assert.True(t, res.Partial, "the slow facet was omitted → partial")
	assert.Equal(t, 100, res.Counts[DimGenre]["28"], "fast facet resolved")
	assert.Equal(t, 100, res.Counts[DimGenre]["12"], "fast facet resolved")
	_, ok := res.Counts[DimGenre]["99"]
	assert.False(t, ok, "slow facet omitted, not zero-filled")
}

// TestCacheService_DiscoverFacetCounts_FailSoft covers AC6: a single sub-query
// error omits ONLY that facet (logged at the boundary) — the rest still return and
// the page never fails.
func TestCacheService_DiscoverFacetCounts_FailSoft(t *testing.T) {
	fb := &MockFallbackClient{
		DiscoverMoviesFunc: func(_ context.Context, p DiscoverParams) (*SearchResultMovies, string, error) {
			if p.GenreIDs[len(p.GenreIDs)-1] == 99 {
				return nil, "", NewTimeoutError(errors.New("boom"))
			}
			return &SearchResultMovies{TotalResults: 50}, "zh-TW", nil
		},
		DiscoverTVShowsFunc: func(_ context.Context, _ DiscoverParams) (*SearchResultTVShows, string, error) {
			return &SearchResultTVShows{TotalResults: 0}, "zh-TW", nil
		},
	}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{
		DimGenre: {"28", "99", "12"},
	})
	require.NoError(t, err, "a per-facet error must never fail the endpoint (Rule 13)")
	assert.True(t, res.Partial)
	assert.Equal(t, 50, res.Counts[DimGenre]["28"])
	assert.Equal(t, 50, res.Counts[DimGenre]["12"])
	_, ok := res.Counts[DimGenre]["99"]
	assert.False(t, ok, "the failed facet is omitted")
}

// TestCacheService_DiscoverFacetCounts_CandidateOnly covers AC8: only the supplied
// dimensions appear in the response; the BE never enumerates a dimension the FE did
// not ask about.
func TestCacheService_DiscoverFacetCounts_CandidateOnly(t *testing.T) {
	fb := &MockFallbackClient{
		DiscoverMoviesResponse:  &SearchResultMovies{TotalResults: 10},
		DiscoverTVShowsResponse: &SearchResultTVShows{TotalResults: 5},
	}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{
		DimGenre: {"28"},
	})
	require.NoError(t, err)
	require.Len(t, res.Counts, 1, "only the genre dimension present")
	_, hasGenre := res.Counts[DimGenre]
	assert.True(t, hasGenre)
	for _, dim := range []string{DimRegion, DimRating, DimPlatform} {
		_, ok := res.Counts[dim]
		assert.False(t, ok, "unsupplied dimension %q must be absent", dim)
	}
}

// TestCacheService_DiscoverFacetCounts_ZeroKept verifies a resolved count of 0 is a
// real dead-end and is KEPT (not treated as missing) — the FE dims-but-keeps it
// selectable.
func TestCacheService_DiscoverFacetCounts_ZeroKept(t *testing.T) {
	fb := &MockFallbackClient{
		DiscoverMoviesResponse:  &SearchResultMovies{TotalResults: 0},
		DiscoverTVShowsResponse: &SearchResultTVShows{TotalResults: 0},
	}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{
		DimRegion: {"TW"},
	})
	require.NoError(t, err)
	assert.False(t, res.Partial, "a resolved 0 is NOT a missing facet")
	v, ok := res.Counts[DimRegion]["TW"]
	require.True(t, ok, "dead-end 0 facet is present in the response")
	assert.Equal(t, 0, v)
}

// TestCacheService_DiscoverFacetCounts_ZeroShortTTL covers AC3 (AR-F5) as refined by
// CR M1: a total_results==0 sub-query IS cached, but only on the SHORT
// FacetCountZeroTTL (not the full FacetCountCacheTTL) — so a transient/wrong-locale 0
// self-corrects fast (AR-F5) while a genuine dead-end 0 is not re-fetched (2 TMDb
// calls) on every debounced probe (CR M1). It proves, distinct from _ZeroKept (which
// asserts only the response):
//
//	(a) the response KEEPS the 0 (parity — a resolved dead-end, not "missing");
//	(b) the 0 is cached with the SHORT FacetCountZeroTTL, tagged "tmdb_facet",
//	    stored as the compact "0" string (not a blob);
//	(c) a SECOND identical probe WITHIN the TTL is served from cache (no re-fetch) —
//	    genuine-0 facets stop hammering TMDb. (Expiry after FacetCountZeroTTL is
//	    repository-enforced — covered by the CacheRepository TTL tests.)
func TestCacheService_DiscoverFacetCounts_ZeroShortTTL(t *testing.T) {
	repo := NewMockCacheRepository()
	fb := &MockFallbackClient{
		DiscoverMoviesResponse:  &SearchResultMovies{TotalResults: 0},
		DiscoverTVShowsResponse: &SearchResultTVShows{TotalResults: 0},
	}
	svc := NewCacheService(fb, repo, CacheServiceConfig{})

	candidates := map[string][]string{DimRegion: {"TW"}}

	// First probe → both sides 0.
	r1, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, candidates)
	require.NoError(t, err)
	// (a) the 0 is kept in the response (parity with _ZeroKept).
	v, ok := r1.Counts[DimRegion]["TW"]
	require.True(t, ok, "dead-end 0 facet stays present in the response (AC6 parity)")
	assert.Equal(t, 0, v)
	// (b) the 0 IS cached, but on the SHORT TTL (CR M1) — tagged tmdb_facet, stored "0".
	assert.Equal(t, 2, fb.DiscoverMoviesCalled+fb.DiscoverTVShowsCalled, "one movie + one tv fetch")
	assert.Equal(t, FacetCountZeroTTL, repo.lastSetTTL, "a 0 must be cached on the SHORT FacetCountZeroTTL, not FacetCountCacheTTL (AR-F5/CR M1)")
	assert.NotEqual(t, FacetCountCacheTTL, repo.lastSetTTL, "a 0 must NOT be pinned for the full non-zero TTL")
	assert.Equal(t, CacheTypeTMDbFacet, repo.lastSetType, "0-count still tagged tmdb_facet (AC2)")
	assert.Equal(t, "0", repo.lastSetValue, "the compact int string '0' is stored, not a blob (AC5)")

	// (c) a second identical probe WITHIN the short TTL is served from cache — a
	// genuine-0 facet no longer re-hits TMDb on every debounced request (CR M1).
	r2, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, candidates)
	require.NoError(t, err)
	assert.Equal(t, 0, r2.Counts[DimRegion]["TW"])
	assert.Equal(t, 1, fb.DiscoverMoviesCalled, "within FacetCountZeroTTL the 0 is served from cache, NOT re-fetched (CR M1)")
	assert.Equal(t, 1, fb.DiscoverTVShowsCalled)
}

// TestFacetCountCacheKey_DistinctNamespace guards the AR-F4 no-collision guarantee
// (CR M2): the dedicated count key MUST live in a different namespace from the grid
// key, so a count entry (int string) and a grid entry (SearchResult* JSON blob) can
// never clobber each other on the same key even for identical params. AC2 asserts
// the distinct cache TYPE; this asserts the distinct KEY — the actual collision
// surface. Without it, a future "simplification" that reused discoverCacheKey would
// silently degrade both caches (Atoi/Unmarshal cross-failures) with no test to catch it.
func TestFacetCountCacheKey_DistinctNamespace(t *testing.T) {
	p := DiscoverParams{GenreIDs: []int{28}, Region: "TW", Page: 1, Language: "zh-TW"}
	for _, kind := range []string{"movie", "tv"} {
		gridKey := discoverCacheKey(kind, p)
		countKey := facetCountCacheKey(kind, p)
		assert.NotEqual(t, gridKey, countKey,
			"facet-count key must NOT collide with the grid key for identical params (AR-F4), kind=%s", kind)
		assert.True(t, strings.HasPrefix(countKey, "tmdb:facetcount/"),
			"count key uses the dedicated facetcount namespace, got %q", countKey)
		assert.True(t, strings.HasPrefix(gridKey, "tmdb:discover/"),
			"grid key uses the discover namespace, got %q", gridKey)
	}
}

// TestCacheService_DiscoverFacetCounts_UnparseableSkipped verifies an unparseable
// candidate (e.g. genre="abc") is silently dropped and does NOT inflate the partial
// denominator (it can never resolve, so it must not read as "still computing").
func TestCacheService_DiscoverFacetCounts_UnparseableSkipped(t *testing.T) {
	fb := &MockFallbackClient{
		DiscoverMoviesResponse:  &SearchResultMovies{TotalResults: 7},
		DiscoverTVShowsResponse: &SearchResultTVShows{TotalResults: 3},
	}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{
		DimGenre: {"28", "abc"},
	})
	require.NoError(t, err)
	assert.False(t, res.Partial, "the unparseable value is dropped, not pending")
	assert.Equal(t, 10, res.Counts[DimGenre]["28"])
	_, ok := res.Counts[DimGenre]["abc"]
	assert.False(t, ok, "unparseable candidate omitted")
}

// TestCacheService_DiscoverFacetCounts_EmptyCandidates verifies an empty candidate
// map yields an empty (non-nil) counts map and Partial=false — a no-op, no upstream
// calls.
func TestCacheService_DiscoverFacetCounts_EmptyCandidates(t *testing.T) {
	fb := &MockFallbackClient{}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.NotNil(t, res.Counts, "counts is an empty object, not null (Rule 18 wire shape)")
	assert.Empty(t, res.Counts)
	assert.False(t, res.Partial)
	assert.Equal(t, 0, fb.DiscoverMoviesCalled, "no candidates → no upstream calls")
}

// TestCacheService_DiscoverFacetCounts_DuplicateCandidateDeduped covers CR M1: a
// repeated (dim,value) must be counted ONCE and must NOT inflate the partial
// denominator (a fully-resolved response was wrongly marked Partial before the fix).
func TestCacheService_DiscoverFacetCounts_DuplicateCandidateDeduped(t *testing.T) {
	fb := &MockFallbackClient{
		DiscoverMoviesResponse:  &SearchResultMovies{TotalResults: 10},
		DiscoverTVShowsResponse: &SearchResultTVShows{TotalResults: 0},
	}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{
		DimGenre: {"28", "28", "28"}, // same value three times
	})
	require.NoError(t, err)
	assert.False(t, res.Partial, "the single distinct facet resolved → NOT partial")
	assert.Equal(t, 10, res.Counts[DimGenre]["28"])
	assert.Len(t, res.Counts[DimGenre], 1)
	assert.Equal(t, 1, fb.DiscoverMoviesCalled, "duplicates collapse to ONE probe (no wasted upstream calls)")
}

// TestCacheService_DiscoverFacetCounts_CapsExcessCandidates covers CR L2: a
// candidate list beyond maxFacetProbes is bounded — at most maxFacetProbes
// sub-queries run, the excess is dropped, and Partial is set.
func TestCacheService_DiscoverFacetCounts_CapsExcessCandidates(t *testing.T) {
	fb := &MockFallbackClient{
		DiscoverMoviesResponse:  &SearchResultMovies{TotalResults: 1},
		DiscoverTVShowsResponse: &SearchResultTVShows{TotalResults: 0},
	}
	svc := NewCacheService(fb, NewMockCacheRepository(), CacheServiceConfig{})

	values := make([]string, 0, maxFacetProbes+10)
	for i := 0; i < maxFacetProbes+10; i++ {
		values = append(values, strconv.Itoa(1000+i)) // distinct genre IDs
	}

	res, err := svc.DiscoverFacetCounts(context.Background(), DiscoverParams{}, map[string][]string{
		DimGenre: values,
	})
	require.NoError(t, err)
	assert.True(t, res.Partial, "dropping excess candidates forces partial")
	assert.Len(t, res.Counts[DimGenre], maxFacetProbes, "fan-out capped at maxFacetProbes")
	assert.Equal(t, maxFacetProbes, fb.DiscoverMoviesCalled, "no more than maxFacetProbes upstream probes")
}
