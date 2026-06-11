package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/tmdb"
)

// MockCacheService is a mock implementation of tmdb.CacheServiceInterface
type MockCacheService struct {
	SearchMoviesResponse     *tmdb.SearchResultMovies
	SearchMoviesError        error
	SearchTVShowsResponse    *tmdb.SearchResultTVShows
	SearchTVShowsError       error
	GetMovieDetailsResponse  *tmdb.MovieDetails
	GetMovieDetailsError     error
	GetTVShowDetailsResponse *tmdb.TVShowDetails
	GetTVShowDetailsError    error
	GetSeasonDetailsResponse *tmdb.SeasonDetails
	GetSeasonDetailsError    error
	// Story 10-1
	GetTrendingMoviesResponse  *tmdb.SearchResultMovies
	GetTrendingMoviesError     error
	GetTrendingTVShowsResponse *tmdb.SearchResultTVShows
	GetTrendingTVShowsError    error
	DiscoverMoviesResponse     *tmdb.SearchResultMovies
	DiscoverMoviesError        error
	DiscoverTVShowsResponse    *tmdb.SearchResultTVShows
	DiscoverTVShowsError       error
	// Story 12-3
	MovieRecommendationsResponse *tmdb.SearchResultMovies
	MovieRecommendationsError    error
	MovieSimilarResponse         *tmdb.SearchResultMovies
	MovieSimilarError            error
	TVRecommendationsResponse    *tmdb.SearchResultTVShows
	TVRecommendationsError       error
	TVSimilarResponse            *tmdb.SearchResultTVShows
	TVSimilarError               error
	// Story 12-4
	WatchProvidersResponse *tmdb.WatchProvidersResponse
	WatchProvidersError    error
	WatchProvidersCalls    []string // "{mediaType}:{id}:{region}" per call
}

func (m *MockCacheService) GetMovieRecommendations(ctx context.Context, movieID int) (*tmdb.SearchResultMovies, error) {
	if m.MovieRecommendationsError != nil {
		return nil, m.MovieRecommendationsError
	}
	return m.MovieRecommendationsResponse, nil
}

func (m *MockCacheService) GetMovieSimilar(ctx context.Context, movieID int) (*tmdb.SearchResultMovies, error) {
	if m.MovieSimilarError != nil {
		return nil, m.MovieSimilarError
	}
	return m.MovieSimilarResponse, nil
}

func (m *MockCacheService) GetTVRecommendations(ctx context.Context, tvID int) (*tmdb.SearchResultTVShows, error) {
	if m.TVRecommendationsError != nil {
		return nil, m.TVRecommendationsError
	}
	return m.TVRecommendationsResponse, nil
}

func (m *MockCacheService) GetTVSimilar(ctx context.Context, tvID int) (*tmdb.SearchResultTVShows, error) {
	if m.TVSimilarError != nil {
		return nil, m.TVSimilarError
	}
	return m.TVSimilarResponse, nil
}

func (m *MockCacheService) GetWatchProviders(ctx context.Context, mediaType string, id int, region string) (*tmdb.WatchProvidersResponse, error) {
	m.WatchProvidersCalls = append(m.WatchProvidersCalls, fmt.Sprintf("%s:%d:%s", mediaType, id, region))
	if m.WatchProvidersError != nil {
		return nil, m.WatchProvidersError
	}
	return m.WatchProvidersResponse, nil
}

func (m *MockCacheService) SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
	if m.SearchMoviesError != nil {
		return nil, m.SearchMoviesError
	}
	return m.SearchMoviesResponse, nil
}

func (m *MockCacheService) SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
	if m.SearchTVShowsError != nil {
		return nil, m.SearchTVShowsError
	}
	return m.SearchTVShowsResponse, nil
}

func (m *MockCacheService) GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error) {
	if m.GetMovieDetailsError != nil {
		return nil, m.GetMovieDetailsError
	}
	return m.GetMovieDetailsResponse, nil
}

func (m *MockCacheService) GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error) {
	if m.GetTVShowDetailsError != nil {
		return nil, m.GetTVShowDetailsError
	}
	return m.GetTVShowDetailsResponse, nil
}

func (m *MockCacheService) GetSeasonDetails(ctx context.Context, tvID int, seasonNumber int) (*tmdb.SeasonDetails, error) {
	if m.GetSeasonDetailsError != nil {
		return nil, m.GetSeasonDetailsError
	}
	return m.GetSeasonDetailsResponse, nil
}

// Story 10-1 additions

func (m *MockCacheService) GetTrendingMovies(ctx context.Context, timeWindow string, page int) (*tmdb.SearchResultMovies, error) {
	if m.GetTrendingMoviesError != nil {
		return nil, m.GetTrendingMoviesError
	}
	return m.GetTrendingMoviesResponse, nil
}

func (m *MockCacheService) GetTrendingTVShows(ctx context.Context, timeWindow string, page int) (*tmdb.SearchResultTVShows, error) {
	if m.GetTrendingTVShowsError != nil {
		return nil, m.GetTrendingTVShowsError
	}
	return m.GetTrendingTVShowsResponse, nil
}

func (m *MockCacheService) DiscoverMovies(ctx context.Context, params tmdb.DiscoverParams) (*tmdb.SearchResultMovies, error) {
	if m.DiscoverMoviesError != nil {
		return nil, m.DiscoverMoviesError
	}
	return m.DiscoverMoviesResponse, nil
}

func (m *MockCacheService) DiscoverTVShows(ctx context.Context, params tmdb.DiscoverParams) (*tmdb.SearchResultTVShows, error) {
	if m.DiscoverTVShowsError != nil {
		return nil, m.DiscoverTVShowsError
	}
	return m.DiscoverTVShowsResponse, nil
}

func TestTMDbService_SearchMovies(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		page        int
		mockResp    *tmdb.SearchResultMovies
		mockErr     error
		wantErr     bool
		wantErrCode string
	}{
		{
			name:  "successful search",
			query: "鬼滅之刃",
			page:  1,
			mockResp: &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{ID: 1, Title: "鬼滅之刃"},
				},
				TotalResults: 1,
			},
		},
		{
			name:        "empty query",
			query:       "",
			page:        1,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
		{
			name:  "negative page defaults to 1",
			query: "test",
			page:  -1,
			mockResp: &tmdb.SearchResultMovies{
				Page:    1,
				Results: []tmdb.Movie{},
			},
		},
		{
			name:    "API error",
			query:   "test",
			page:    1,
			mockErr: errors.New("API error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCacheService{
				SearchMoviesResponse: tt.mockResp,
				SearchMoviesError:    tt.mockErr,
			}

			service := NewTMDbServiceWithCacheService(mock)
			result, err := service.SearchMovies(context.Background(), tt.query, tt.page)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					tmdbErr, ok := err.(*tmdb.TMDbError)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, tmdbErr.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestTMDbService_SearchTVShows(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		page        int
		mockResp    *tmdb.SearchResultTVShows
		mockErr     error
		wantErr     bool
		wantErrCode string
	}{
		{
			name:  "successful search",
			query: "Breaking Bad",
			page:  1,
			mockResp: &tmdb.SearchResultTVShows{
				Page: 1,
				Results: []tmdb.TVShow{
					{ID: 1396, Name: "Breaking Bad"},
				},
				TotalResults: 1,
			},
		},
		{
			name:        "empty query",
			query:       "",
			page:        1,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCacheService{
				SearchTVShowsResponse: tt.mockResp,
				SearchTVShowsError:    tt.mockErr,
			}

			service := NewTMDbServiceWithCacheService(mock)
			result, err := service.SearchTVShows(context.Background(), tt.query, tt.page)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					tmdbErr, ok := err.(*tmdb.TMDbError)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, tmdbErr.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestTMDbService_GetMovieDetails(t *testing.T) {
	tests := []struct {
		name        string
		movieID     int
		mockResp    *tmdb.MovieDetails
		mockErr     error
		wantErr     bool
		wantErrCode string
	}{
		{
			name:    "successful get",
			movieID: 550,
			mockResp: &tmdb.MovieDetails{
				Movie: tmdb.Movie{
					ID:    550,
					Title: "Fight Club",
				},
			},
		},
		{
			name:        "invalid ID - zero",
			movieID:     0,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
		{
			name:        "invalid ID - negative",
			movieID:     -1,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
		{
			name:    "API error",
			movieID: 550,
			mockErr: tmdb.NewNotFoundError(550),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCacheService{
				GetMovieDetailsResponse: tt.mockResp,
				GetMovieDetailsError:    tt.mockErr,
			}

			service := NewTMDbServiceWithCacheService(mock)
			result, err := service.GetMovieDetails(context.Background(), tt.movieID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					tmdbErr, ok := err.(*tmdb.TMDbError)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, tmdbErr.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.mockResp.Title, result.Title)
		})
	}
}

func TestTMDbService_GetTVShowDetails(t *testing.T) {
	tests := []struct {
		name        string
		tvID        int
		mockResp    *tmdb.TVShowDetails
		mockErr     error
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "successful get",
			tvID: 1396,
			mockResp: &tmdb.TVShowDetails{
				TVShow: tmdb.TVShow{
					ID:   1396,
					Name: "Breaking Bad",
				},
			},
		},
		{
			name:        "invalid ID - zero",
			tvID:        0,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
		{
			name:        "invalid ID - negative",
			tvID:        -1,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCacheService{
				GetTVShowDetailsResponse: tt.mockResp,
				GetTVShowDetailsError:    tt.mockErr,
			}

			service := NewTMDbServiceWithCacheService(mock)
			result, err := service.GetTVShowDetails(context.Background(), tt.tvID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					tmdbErr, ok := err.(*tmdb.TMDbError)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, tmdbErr.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.mockResp.Name, result.Name)
		})
	}
}

func TestTMDbService_InterfaceCompliance(t *testing.T) {
	var _ TMDbServiceInterface = (*TMDbService)(nil)
}

// --- Story 10-1 service-layer tests ---
// These verify that the TMDbService pipelines content-filter AFTER cache fetch
// (AC #3, #4 must hold end-to-end, not just inside ContentFilterService).

func TestTMDbService_GetTrendingMovies_AppliesContentFilters(t *testing.T) {
	now := mustParseDate(t, "2026-04-14")
	horizonCrosser := "2028-01-01" // > 6 months after 2026-04-14

	mockCache := &MockCacheService{
		GetTrendingMoviesResponse: &tmdb.SearchResultMovies{
			Page: 1,
			Results: []tmdb.Movie{
				{ID: 1, Title: "Good", VoteAverage: 8.0, VoteCount: 1000, ReleaseDate: "2026-01-01"},
				{ID: 2, Title: "BadObscure", VoteAverage: 2.0, VoteCount: 10, ReleaseDate: "2026-01-01"}, // low quality → drop
				{ID: 3, Title: "Future", VoteAverage: 7.5, VoteCount: 500, ReleaseDate: horizonCrosser},  // far future → drop
				{ID: 4, Title: "Kept", VoteAverage: 6.0, VoteCount: 200, ReleaseDate: "2026-10-01"},      // within 6mo → keep
			},
			TotalResults: 4,
		},
	}

	svc := NewTMDbServiceWithCacheService(mockCache)
	svc.SetContentFilter(NewContentFilterServiceWithClock(func() time.Time { return now }))

	result, err := svc.GetTrendingMovies(context.Background(), "week", 1)

	require.NoError(t, err)
	require.NotNil(t, result)
	var ids []int
	for _, m := range result.Results {
		ids = append(ids, m.ID)
	}
	assert.Equal(t, []int{1, 4}, ids, "filters must drop low-quality AND far-future items")
}

func TestTMDbService_DiscoverTVShows_AppliesContentFilters(t *testing.T) {
	now := mustParseDate(t, "2026-04-14")

	mockCache := &MockCacheService{
		DiscoverTVShowsResponse: &tmdb.SearchResultTVShows{
			Page: 1,
			Results: []tmdb.TVShow{
				{ID: 1, Name: "ok", VoteAverage: 7.0, VoteCount: 300, FirstAirDate: "2025-03-01"},
				{ID: 2, Name: "unwatchable", VoteAverage: 1.0, VoteCount: 5, FirstAirDate: "2025-01-01"},      // low quality
				{ID: 3, Name: "far future show", VoteAverage: 8.0, VoteCount: 50, FirstAirDate: "2028-01-01"}, // far future
			},
			TotalResults: 3,
		},
	}

	svc := NewTMDbServiceWithCacheService(mockCache)
	svc.SetContentFilter(NewContentFilterServiceWithClock(func() time.Time { return now }))

	result, err := svc.DiscoverTVShows(context.Background(), tmdb.DiscoverParams{GenreIDs: []int{18}})

	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	assert.Equal(t, 1, result.Results[0].ID)
}

func TestTMDbService_Trending_ErrorPropagatesFromCacheLayer(t *testing.T) {
	mockCache := &MockCacheService{GetTrendingMoviesError: errors.New("cache layer boom")}
	svc := NewTMDbServiceWithCacheService(mockCache)

	_, err := svc.GetTrendingMovies(context.Background(), "week", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache layer boom")
}

// TestTMDbService_GetTrendingMovies_ItemFailingBothFiltersDroppedOnce verifies
// that a single item violating BOTH FarFuture AND LowQuality is dropped and
// does not appear in the output (i.e., the FarFuture → LowQuality pipeline
// composes correctly without the same item being skipped, double-processed, or
// resurrected). Also asserts that items violating only one predicate are still
// dropped, and that clean items survive both stages.
func TestTMDbService_GetTrendingMovies_ItemFailingBothFiltersDroppedOnce(t *testing.T) {
	now := mustParseDate(t, "2026-04-14")

	mockCache := &MockCacheService{
		GetTrendingMoviesResponse: &tmdb.SearchResultMovies{
			Page: 1,
			Results: []tmdb.Movie{
				{ID: 1, Title: "clean", VoteAverage: 7.5, VoteCount: 500, ReleaseDate: "2026-03-01"},
				{ID: 2, Title: "low quality only", VoteAverage: 1.5, VoteCount: 5, ReleaseDate: "2025-12-01"},
				{ID: 3, Title: "far future only", VoteAverage: 9.0, VoteCount: 2000, ReleaseDate: "2028-06-01"},
				{ID: 4, Title: "fails both", VoteAverage: 1.0, VoteCount: 3, ReleaseDate: "2028-06-01"},
				{ID: 5, Title: "also clean", VoteAverage: 6.0, VoteCount: 100, ReleaseDate: "2026-09-01"},
			},
			TotalResults: 5,
		},
	}

	svc := NewTMDbServiceWithCacheService(mockCache)
	svc.SetContentFilter(NewContentFilterServiceWithClock(func() time.Time { return now }))

	result, err := svc.GetTrendingMovies(context.Background(), "week", 1)
	require.NoError(t, err)
	require.NotNil(t, result)

	var ids []int
	for _, m := range result.Results {
		ids = append(ids, m.ID)
	}

	assert.Equal(t, []int{1, 5}, ids,
		"only items passing BOTH filters survive; ID 4 fails both and must appear zero times")
	assert.NotContains(t, ids, 4, "item failing both predicates must be dropped, not resurrected by second filter")
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return parsed
}

// --- Story 10-2 service-layer tests ---

func TestTMDbService_GetMovieVideos_RejectsInvalidID(t *testing.T) {
	svc := NewTMDbServiceWithCacheService(&MockCacheService{})

	for _, id := range []int{0, -1, -100} {
		_, err := svc.GetMovieVideos(context.Background(), id)
		require.Error(t, err, "id %d must be rejected", id)
		var tmdbErr *tmdb.TMDbError
		require.ErrorAs(t, err, &tmdbErr, "rejection must be a TMDbError so handlers map to 400")
		assert.Equal(t, tmdb.ErrCodeBadRequest, tmdbErr.Code)
	}
}

func TestTMDbService_GetMovieVideos_NilClientReturnsError(t *testing.T) {
	// NewTMDbServiceWithCacheService leaves client=nil; GetMovieVideos must not panic.
	svc := NewTMDbServiceWithCacheService(&MockCacheService{})

	_, err := svc.GetMovieVideos(context.Background(), 550)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TMDb client not initialized")
}

func TestTMDbService_GetTVShowVideos_RejectsInvalidID(t *testing.T) {
	svc := NewTMDbServiceWithCacheService(&MockCacheService{})

	_, err := svc.GetTVShowVideos(context.Background(), 0)
	require.Error(t, err)
	var tmdbErr *tmdb.TMDbError
	require.ErrorAs(t, err, &tmdbErr)
	assert.Equal(t, tmdb.ErrCodeBadRequest, tmdbErr.Code)
}

func TestTMDbService_GetTVShowVideos_NilClientReturnsError(t *testing.T) {
	svc := NewTMDbServiceWithCacheService(&MockCacheService{})

	_, err := svc.GetTVShowVideos(context.Background(), 1396)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TMDb client not initialized")
}

// --- Story 12-4 service-layer tests ---

func TestTMDbService_GetWatchProviders_DefaultsRegionToTW(t *testing.T) {
	mockCache := &MockCacheService{WatchProvidersResponse: &tmdb.WatchProvidersResponse{ID: 550}}
	svc := NewTMDbServiceWithCacheService(mockCache)

	_, err := svc.GetWatchProviders(context.Background(), "movie", 550, "")
	require.NoError(t, err)
	require.Len(t, mockCache.WatchProvidersCalls, 1)
	assert.Equal(t, "movie:550:TW", mockCache.WatchProvidersCalls[0], "empty region must default to TW")
}

func TestTMDbService_GetWatchProviders_PassesThroughExplicitRegion(t *testing.T) {
	mockCache := &MockCacheService{WatchProvidersResponse: &tmdb.WatchProvidersResponse{ID: 1396}}
	svc := NewTMDbServiceWithCacheService(mockCache)

	_, err := svc.GetWatchProviders(context.Background(), "tv", 1396, "US")
	require.NoError(t, err)
	require.Len(t, mockCache.WatchProvidersCalls, 1)
	assert.Equal(t, "tv:1396:US", mockCache.WatchProvidersCalls[0])
}

// CR 12-4 MEDIUM #1 — a lowercase region must be normalized to uppercase so the
// TMDb region filter (uppercase ISO keys) hits and the cache key is canonical.
func TestTMDbService_GetWatchProviders_NormalizesRegionToUpper(t *testing.T) {
	mockCache := &MockCacheService{WatchProvidersResponse: &tmdb.WatchProvidersResponse{ID: 550}}
	svc := NewTMDbServiceWithCacheService(mockCache)

	_, err := svc.GetWatchProviders(context.Background(), "movie", 550, "tw")
	require.NoError(t, err)
	require.Len(t, mockCache.WatchProvidersCalls, 1)
	assert.Equal(t, "movie:550:TW", mockCache.WatchProvidersCalls[0], "lowercase region must be normalized to uppercase")
}

func TestTMDbService_GetWatchProviders_RejectsInvalidArgs(t *testing.T) {
	svc := NewTMDbServiceWithCacheService(&MockCacheService{})

	cases := []struct {
		mediaType string
		id        int
	}{
		{"person", 550}, // bad media type
		{"movie", 0},    // non-positive id
		{"movie", -5},
	}
	for _, c := range cases {
		_, err := svc.GetWatchProviders(context.Background(), c.mediaType, c.id, "TW")
		require.Error(t, err, "mediaType=%q id=%d must be rejected", c.mediaType, c.id)
		var tmdbErr *tmdb.TMDbError
		require.ErrorAs(t, err, &tmdbErr, "rejection must be a TMDbError so handlers map to 400")
		assert.Equal(t, tmdb.ErrCodeBadRequest, tmdbErr.Code)
	}
}

func TestTMDbService_GetWatchProviders_ErrorPropagatesFromCacheLayer(t *testing.T) {
	mockCache := &MockCacheService{WatchProvidersError: errors.New("cache layer boom")}
	svc := NewTMDbServiceWithCacheService(mockCache)

	_, err := svc.GetWatchProviders(context.Background(), "movie", 550, "TW")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache layer boom")
}
