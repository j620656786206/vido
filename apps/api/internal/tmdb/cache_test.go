package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/repository"
)

// MockCacheRepository is a mock implementation of CacheRepositoryInterface
type MockCacheRepository struct {
	data       map[string]*repository.CacheEntry
	setError   error
	getError   error
	setCalled  int
	getCalled  int
}

func NewMockCacheRepository() *MockCacheRepository {
	return &MockCacheRepository{
		data: make(map[string]*repository.CacheEntry),
	}
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (*repository.CacheEntry, error) {
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
	m.setCalled++
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
	delete(m.data, key)
	return nil
}

func (m *MockCacheRepository) Clear(ctx context.Context) error {
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
	SearchMoviesResponse    *SearchResultMovies
	SearchMoviesError       error
	SearchMoviesCalled      int
	SearchTVShowsResponse   *SearchResultTVShows
	SearchTVShowsError      error
	SearchTVShowsCalled     int
	GetMovieDetailsResponse *MovieDetails
	GetMovieDetailsError    error
	GetMovieDetailsCalled   int
	GetTVShowDetailsResponse *TVShowDetails
	GetTVShowDetailsError    error
	GetTVShowDetailsCalled   int
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
		name           string
		query          string
		page           int
		cachedData     *SearchResultMovies
		apiResponse    *SearchResultMovies
		apiError       error
		wantFromCache  bool
		wantAPICall    bool
		wantCacheSet   bool
		wantErr        bool
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
