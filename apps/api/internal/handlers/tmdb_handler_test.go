package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/tmdb"
)

// MockTMDbService is a mock implementation of TMDbServiceInterface
type MockTMDbService struct {
	SearchMoviesResponse     *tmdb.SearchResultMovies
	SearchMoviesError        error
	SearchTVShowsResponse    *tmdb.SearchResultTVShows
	SearchTVShowsError       error
	GetMovieDetailsResponse  *tmdb.MovieDetails
	GetMovieDetailsError     error
	GetTVShowDetailsResponse *tmdb.TVShowDetails
	GetTVShowDetailsError    error
	// Story 10-1
	GetTrendingMoviesResponse  *tmdb.SearchResultMovies
	GetTrendingMoviesError     error
	GetTrendingMoviesCalls     []string // captured time_window values
	GetTrendingTVShowsResponse *tmdb.SearchResultTVShows
	GetTrendingTVShowsError    error
	GetTrendingTVShowsCalls    []string
	DiscoverMoviesResponse     *tmdb.SearchResultMovies
	DiscoverMoviesError        error
	DiscoverMoviesCalls        []tmdb.DiscoverParams
	DiscoverTVShowsResponse    *tmdb.SearchResultTVShows
	DiscoverTVShowsError       error
	DiscoverTVShowsCalls       []tmdb.DiscoverParams
	// Story 10-2
	GetMovieVideosResponse  *tmdb.VideosResponse
	GetMovieVideosError     error
	GetMovieVideosCalls     []int
	GetTVShowVideosResponse *tmdb.VideosResponse
	GetTVShowVideosError    error
	GetTVShowVideosCalls    []int
}

func (m *MockTMDbService) SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
	if m.SearchMoviesError != nil {
		return nil, m.SearchMoviesError
	}
	return m.SearchMoviesResponse, nil
}

func (m *MockTMDbService) SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
	if m.SearchTVShowsError != nil {
		return nil, m.SearchTVShowsError
	}
	return m.SearchTVShowsResponse, nil
}

func (m *MockTMDbService) GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error) {
	if m.GetMovieDetailsError != nil {
		return nil, m.GetMovieDetailsError
	}
	return m.GetMovieDetailsResponse, nil
}

func (m *MockTMDbService) GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error) {
	if m.GetTVShowDetailsError != nil {
		return nil, m.GetTVShowDetailsError
	}
	return m.GetTVShowDetailsResponse, nil
}

func (m *MockTMDbService) FindByExternalID(ctx context.Context, externalID string, externalSource string) (*tmdb.FindByExternalIDResponse, error) {
	return &tmdb.FindByExternalIDResponse{}, nil
}

// Story 10-1 additions

func (m *MockTMDbService) GetTrendingMovies(ctx context.Context, timeWindow string, page int) (*tmdb.SearchResultMovies, error) {
	m.GetTrendingMoviesCalls = append(m.GetTrendingMoviesCalls, timeWindow)
	if m.GetTrendingMoviesError != nil {
		return nil, m.GetTrendingMoviesError
	}
	return m.GetTrendingMoviesResponse, nil
}

func (m *MockTMDbService) GetTrendingTVShows(ctx context.Context, timeWindow string, page int) (*tmdb.SearchResultTVShows, error) {
	m.GetTrendingTVShowsCalls = append(m.GetTrendingTVShowsCalls, timeWindow)
	if m.GetTrendingTVShowsError != nil {
		return nil, m.GetTrendingTVShowsError
	}
	return m.GetTrendingTVShowsResponse, nil
}

func (m *MockTMDbService) DiscoverMovies(ctx context.Context, params tmdb.DiscoverParams) (*tmdb.SearchResultMovies, error) {
	m.DiscoverMoviesCalls = append(m.DiscoverMoviesCalls, params)
	if m.DiscoverMoviesError != nil {
		return nil, m.DiscoverMoviesError
	}
	return m.DiscoverMoviesResponse, nil
}

func (m *MockTMDbService) DiscoverTVShows(ctx context.Context, params tmdb.DiscoverParams) (*tmdb.SearchResultTVShows, error) {
	m.DiscoverTVShowsCalls = append(m.DiscoverTVShowsCalls, params)
	if m.DiscoverTVShowsError != nil {
		return nil, m.DiscoverTVShowsError
	}
	return m.DiscoverTVShowsResponse, nil
}

// Story 10-2 additions

func (m *MockTMDbService) GetMovieVideos(ctx context.Context, movieID int) (*tmdb.VideosResponse, error) {
	m.GetMovieVideosCalls = append(m.GetMovieVideosCalls, movieID)
	if m.GetMovieVideosError != nil {
		return nil, m.GetMovieVideosError
	}
	return m.GetMovieVideosResponse, nil
}

func (m *MockTMDbService) GetTVShowVideos(ctx context.Context, tvID int) (*tmdb.VideosResponse, error) {
	m.GetTVShowVideosCalls = append(m.GetTVShowVideosCalls, tvID)
	if m.GetTVShowVideosError != nil {
		return nil, m.GetTVShowVideosError
	}
	return m.GetTVShowVideosResponse, nil
}

func setupTMDbRouter(handler *TMDbHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)
	return router
}

func TestTMDbHandler_SearchMovies(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockResponse   *tmdb.SearchResultMovies
		mockError      error
		wantStatus     int
		wantSuccess    bool
		wantResultsLen int
	}{
		{
			name:        "successful search",
			queryParams: "query=%E9%AC%BC%E6%BB%85%E4%B9%8B%E5%88%83&page=1",
			mockResponse: &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{ID: 1, Title: "鬼滅之刃劇場版"},
				},
				TotalResults: 1,
			},
			wantStatus:     http.StatusOK,
			wantSuccess:    true,
			wantResultsLen: 1,
		},
		{
			name:        "missing query",
			queryParams: "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "service error - not found",
			queryParams: "query=test",
			mockError:   tmdb.NewNotFoundErrorWithResource("movie"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "service error - rate limit",
			queryParams: "query=test",
			mockError:   tmdb.NewRateLimitError(),
			wantStatus:  http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTMDbService{
				SearchMoviesResponse: tt.mockResponse,
				SearchMoviesError:    tt.mockError,
			}
			handler := NewTMDbHandler(mock)
			router := setupTMDbRouter(handler)

			url := "/api/v1/tmdb/search/movies"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantSuccess {
				var response APIResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
			}
		})
	}
}

func TestTMDbHandler_SearchTVShows(t *testing.T) {
	tests := []struct {
		name         string
		queryParam   string
		mockResponse *tmdb.SearchResultTVShows
		mockError    error
		wantStatus   int
		wantSuccess  bool
	}{
		{
			name:       "successful search",
			queryParam: "query=Breaking+Bad",
			mockResponse: &tmdb.SearchResultTVShows{
				Page: 1,
				Results: []tmdb.TVShow{
					{ID: 1396, Name: "Breaking Bad"},
				},
				TotalResults: 1,
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:       "missing query",
			queryParam: "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTMDbService{
				SearchTVShowsResponse: tt.mockResponse,
				SearchTVShowsError:    tt.mockError,
			}
			handler := NewTMDbHandler(mock)
			router := setupTMDbRouter(handler)

			url := "/api/v1/tmdb/search/tv"
			if tt.queryParam != "" {
				url += "?" + tt.queryParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestTMDbHandler_GetMovieDetails(t *testing.T) {
	tests := []struct {
		name         string
		movieID      string
		mockResponse *tmdb.MovieDetails
		mockError    error
		wantStatus   int
		wantSuccess  bool
	}{
		{
			name:    "successful get",
			movieID: "550",
			mockResponse: &tmdb.MovieDetails{
				Movie: tmdb.Movie{
					ID:    550,
					Title: "Fight Club",
				},
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:       "invalid ID - not a number",
			movieID:    "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid ID - zero",
			movieID:    "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid ID - negative",
			movieID:    "-1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			movieID:    "999999",
			mockError:  tmdb.NewNotFoundError(999999),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTMDbService{
				GetMovieDetailsResponse: tt.mockResponse,
				GetMovieDetailsError:    tt.mockError,
			}
			handler := NewTMDbHandler(mock)
			router := setupTMDbRouter(handler)

			url := "/api/v1/tmdb/movies/" + tt.movieID

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantSuccess {
				var response APIResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
			}
		})
	}
}

func TestTMDbHandler_GetTVShowDetails(t *testing.T) {
	tests := []struct {
		name         string
		tvID         string
		mockResponse *tmdb.TVShowDetails
		mockError    error
		wantStatus   int
		wantSuccess  bool
	}{
		{
			name: "successful get",
			tvID: "1396",
			mockResponse: &tmdb.TVShowDetails{
				TVShow: tmdb.TVShow{
					ID:   1396,
					Name: "Breaking Bad",
				},
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:       "invalid ID - not a number",
			tvID:       "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid ID - zero",
			tvID:       "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "not found",
			tvID:      "999999",
			mockError: tmdb.NewNotFoundError(999999),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTMDbService{
				GetTVShowDetailsResponse: tt.mockResponse,
				GetTVShowDetailsError:    tt.mockError,
			}
			handler := NewTMDbHandler(mock)
			router := setupTMDbRouter(handler)

			url := "/api/v1/tmdb/tv/" + tt.tvID

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestTMDbHandler_RegisterRoutes(t *testing.T) {
	mock := &MockTMDbService{}
	handler := NewTMDbHandler(mock)
	router := setupTMDbRouter(handler)

	// Verify routes are registered
	routes := router.Routes()

	expectedRoutes := map[string]string{
		"/api/v1/tmdb/search/movies":   http.MethodGet,
		"/api/v1/tmdb/search/tv":       http.MethodGet,
		"/api/v1/tmdb/movies/:id":      http.MethodGet,
		"/api/v1/tmdb/tv/:id":          http.MethodGet,
		"/api/v1/tmdb/trending/movies": http.MethodGet, // Story 10-1
		"/api/v1/tmdb/trending/tv":     http.MethodGet, // Story 10-1
		"/api/v1/tmdb/discover/movies": http.MethodGet, // Story 10-1
		"/api/v1/tmdb/discover/tv":     http.MethodGet, // Story 10-1
		"/api/v1/tmdb/movies/:id/videos": http.MethodGet, // Story 10-2
		"/api/v1/tmdb/tv/:id/videos":     http.MethodGet, // Story 10-2
	}

	for path, method := range expectedRoutes {
		found := false
		for _, route := range routes {
			if route.Path == path && route.Method == method {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s %s should be registered", method, path)
	}
}

// --- Story 10-1 handler tests ---

func TestTMDbHandler_GetTrendingMovies(t *testing.T) {
	tests := []struct {
		name             string
		queryParams      string
		mockResp         *tmdb.SearchResultMovies
		mockErr          error
		wantStatus       int
		wantSuccess      bool
		wantResultsLen   int
		wantTimeWindow   string
	}{
		{
			name:        "default time_window is week",
			queryParams: "",
			mockResp: &tmdb.SearchResultMovies{
				Page: 1, Results: []tmdb.Movie{{ID: 1, Title: "Hot"}},
			},
			wantStatus:     http.StatusOK,
			wantSuccess:    true,
			wantResultsLen: 1,
			wantTimeWindow: "week",
		},
		{
			name:           "explicit day",
			queryParams:    "time_window=day&page=2",
			mockResp:       &tmdb.SearchResultMovies{Page: 2, Results: []tmdb.Movie{}},
			wantStatus:     http.StatusOK,
			wantSuccess:    true,
			wantTimeWindow: "day",
		},
		{
			name:           "unknown time_window falls back to week",
			queryParams:    "time_window=year",
			mockResp:       &tmdb.SearchResultMovies{Page: 1, Results: []tmdb.Movie{}},
			wantStatus:     http.StatusOK,
			wantSuccess:    true,
			wantTimeWindow: "week",
		},
		{
			name:        "upstream error surfaces as 500",
			queryParams: "",
			mockErr:     tmdb.NewServerError(errors.New("upstream down")),
			wantStatus:  http.StatusBadGateway,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockTMDbService{
				GetTrendingMoviesResponse: tt.mockResp,
				GetTrendingMoviesError:    tt.mockErr,
			}
			handler := NewTMDbHandler(mockSvc)
			router := setupTMDbRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/trending/movies?"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var body APIResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tt.wantSuccess, body.Success)
			if tt.wantTimeWindow != "" {
				assert.Contains(t, mockSvc.GetTrendingMoviesCalls, tt.wantTimeWindow)
			}
		})
	}
}

func TestTMDbHandler_GetTrendingTVShows_RoutesCorrectly(t *testing.T) {
	mockSvc := &MockTMDbService{
		GetTrendingTVShowsResponse: &tmdb.SearchResultTVShows{Page: 1},
	}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/trending/tv?time_window=day", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"day"}, mockSvc.GetTrendingTVShowsCalls)
}

func TestTMDbHandler_DiscoverMovies_QueryParamMapping(t *testing.T) {
	mockSvc := &MockTMDbService{
		DiscoverMoviesResponse: &tmdb.SearchResultMovies{Page: 1},
	}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	// Exercise all 7 query params — matches the story's example request
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/tmdb/discover/movies?genre=28,12&year_gte=2024&year_lte=2026&region=TW&language=zh&sort=popularity.desc&page=3",
		nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, mockSvc.DiscoverMoviesCalls, 1)
	got := mockSvc.DiscoverMoviesCalls[0]
	assert.Equal(t, "28,12", got.Genre)
	assert.Equal(t, 2024, got.YearGte)
	assert.Equal(t, 2026, got.YearLte)
	assert.Equal(t, "TW", got.Region)
	assert.Equal(t, "zh", got.Language)
	assert.Equal(t, "popularity.desc", got.SortBy)
	assert.Equal(t, 3, got.Page)
}

func TestTMDbHandler_DiscoverMovies_DefaultsWhenEmpty(t *testing.T) {
	mockSvc := &MockTMDbService{DiscoverMoviesResponse: &tmdb.SearchResultMovies{}}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/discover/movies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, mockSvc.DiscoverMoviesCalls, 1)
	got := mockSvc.DiscoverMoviesCalls[0]
	assert.Equal(t, "", got.Genre)
	assert.Equal(t, 0, got.YearGte)
	assert.Equal(t, 1, got.Page, "empty page query defaults to 1")
}

func TestTMDbHandler_DiscoverTVShows_RoutesCorrectly(t *testing.T) {
	mockSvc := &MockTMDbService{DiscoverTVShowsResponse: &tmdb.SearchResultTVShows{}}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/discover/tv?genre=18&language=zh&sort=popularity.desc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, mockSvc.DiscoverTVShowsCalls, 1)
	got := mockSvc.DiscoverTVShowsCalls[0]
	assert.Equal(t, "18", got.Genre)
	assert.Equal(t, "zh", got.Language)
	assert.Equal(t, "popularity.desc", got.SortBy)
}

func TestTMDbHandler_DiscoverMovies_ErrorPropagates(t *testing.T) {
	mockSvc := &MockTMDbService{
		DiscoverMoviesError: tmdb.NewServerError(errors.New("down")),
	}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/discover/movies?genre=28", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code, "TMDb server errors surface as 502 via handleTMDbError")

	var body APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Success)
	assert.NotNil(t, body.Error)
}

// TestTMDbHandler_GenericError_Returns500 verifies that when a non-*TMDbError
// bubbles up from the service (defensive path — no current code produces this,
// but handleTMDbError has a generic fallback branch), the handler emits
// HTTP 500 with TMDB_INTERNAL_ERROR code. This keeps the generic branch at
// tmdb_handler.go:349-355 covered.
func TestTMDbHandler_GenericError_Returns500(t *testing.T) {
	mockSvc := &MockTMDbService{
		GetTrendingMoviesError: errors.New("unexpected non-TMDb error"),
	}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/trending/movies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code,
		"non-TMDbError must surface as 500 via handleTMDbError generic branch")

	var body APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Success)
	require.NotNil(t, body.Error)
	assert.Equal(t, "TMDB_INTERNAL_ERROR", body.Error.Code)
}

// TestTMDbHandler_DiscoverMovies_YearRangeValidation covers Story 10-1a
// ACs #1, #3, #4, #5, #6 against the movies endpoint. AC #2 (TV endpoint)
// lives in TestTMDbHandler_DiscoverTVShows_YearRangeValidation_Reversed
// below — intentionally separate so a regression where only one handler
// wires parseDiscoverParams' error return cannot silently pass.
func TestTMDbHandler_DiscoverMovies_YearRangeValidation(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		wantStatus     int
		wantErrorCode  string // empty for 200 cases
		wantSvcCalled  bool
		wantYearGte    int
		wantYearLte    int
	}{
		{
			name:          "reversed range rejects with 400 TMDB_INVALID_YEAR_RANGE (AC #1)",
			query:         "year_gte=2030&year_lte=2020",
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: tmdb.ErrCodeInvalidYearRange,
			wantSvcCalled: false,
		},
		{
			name:          "same-year range is valid (AC #4)",
			query:         "year_gte=2024&year_lte=2024",
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantYearGte:   2024,
			wantYearLte:   2024,
		},
		{
			name:          "zero-gte keeps unlimited lower bound (AC #3)",
			query:         "year_gte=0&year_lte=2024",
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantYearGte:   0,
			wantYearLte:   2024,
		},
		{
			name:          "zero-lte keeps unlimited upper bound (AC #3)",
			query:         "year_gte=2024&year_lte=0",
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantYearGte:   2024,
			wantYearLte:   0,
		},
		{
			name:          "normal ascending range proceeds (sanity baseline)",
			query:         "year_gte=2024&year_lte=2025",
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantYearGte:   2024,
			wantYearLte:   2025,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockTMDbService{DiscoverMoviesResponse: &tmdb.SearchResultMovies{}}
			handler := NewTMDbHandler(mockSvc)
			router := setupTMDbRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/discover/movies?"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, tt.wantStatus, w.Code)

			if tt.wantErrorCode != "" {
				// AC #5: error body follows ApiResponse envelope with code+message.
				var body APIResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				assert.False(t, body.Success)
				require.NotNil(t, body.Error)
				assert.Equal(t, tt.wantErrorCode, body.Error.Code)
				assert.Contains(t, body.Error.Message, "year_gte")
			}

			if tt.wantSvcCalled {
				require.Len(t, mockSvc.DiscoverMoviesCalls, 1, "service must be invoked for valid ranges")
				got := mockSvc.DiscoverMoviesCalls[0]
				assert.Equal(t, tt.wantYearGte, got.YearGte)
				assert.Equal(t, tt.wantYearLte, got.YearLte)
			} else {
				// AC #6: validation lives in the handler layer — no service/cache/client call on rejection.
				assert.Empty(t, mockSvc.DiscoverMoviesCalls, "service must NOT be called when handler-layer validation fails")
			}
		})
	}
}

// TestTMDbHandler_DiscoverTVShows_YearRangeValidation_Reversed covers AC #2.
// Kept separate from the movies table so a future regression where only one
// handler threads parseDiscoverParams' error cannot silently pass.
func TestTMDbHandler_DiscoverTVShows_YearRangeValidation_Reversed(t *testing.T) {
	mockSvc := &MockTMDbService{DiscoverTVShowsResponse: &tmdb.SearchResultTVShows{}}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/discover/tv?year_gte=2030&year_lte=2020", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.DiscoverTVShowsCalls, "TV service must NOT be called when handler-layer validation fails")

	var body APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Success)
	require.NotNil(t, body.Error)
	assert.Equal(t, tmdb.ErrCodeInvalidYearRange, body.Error.Code)
	assert.Contains(t, body.Error.Message, "year_gte")
}

func TestTMDbHandler_ResponseIsApiResponseWrapped(t *testing.T) {
	// AC #6: responses follow the existing ApiResponse<T> wrapper format with
	// snake_case fields (success, data, error).
	mockSvc := &MockTMDbService{
		GetTrendingMoviesResponse: &tmdb.SearchResultMovies{Page: 1, TotalResults: 0},
	}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/trending/movies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	// Rule 18: snake_case at JSON boundary — here we just verify the top-level
	// wrapper keys exist and no PascalCase leakage occurred.
	_, ok := raw["success"]
	assert.True(t, ok, "response must include `success` field")
	_, ok = raw["data"]
	assert.True(t, ok, "response must include `data` field")
}

// --- Story 10-2 handler tests ---

func TestTMDbHandler_GetMovieVideos(t *testing.T) {
	tests := []struct {
		name         string
		movieID      string
		mockResponse *tmdb.VideosResponse
		mockError    error
		wantStatus   int
		wantCallID   int
	}{
		{
			name:    "successful fetch returns trailers",
			movieID: "550",
			mockResponse: &tmdb.VideosResponse{
				ID: 550,
				Results: []tmdb.Video{
					{Key: "SUXWAEX2jlg", Name: "Official Trailer", Site: "YouTube", Type: "Trailer", Official: true},
				},
			},
			wantStatus: http.StatusOK,
			wantCallID: 550,
		},
		{
			name:       "invalid ID rejected as 400",
			movieID:    "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero ID rejected as 400",
			movieID:    "0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative ID rejected as 400",
			movieID:    "-5",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "upstream not found propagates 404",
			movieID:    "999999",
			mockError:  tmdb.NewNotFoundError(999999),
			wantStatus: http.StatusNotFound,
			wantCallID: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockTMDbService{
				GetMovieVideosResponse: tt.mockResponse,
				GetMovieVideosError:    tt.mockError,
			}
			handler := NewTMDbHandler(mockSvc)
			router := setupTMDbRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/movies/"+tt.movieID+"/videos", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var response APIResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				assert.True(t, response.Success)
				assert.Equal(t, []int{tt.wantCallID}, mockSvc.GetMovieVideosCalls)
			}
			// Validation failures short-circuit before reaching the service.
			if tt.wantStatus == http.StatusBadRequest && tt.mockError == nil {
				assert.Empty(t, mockSvc.GetMovieVideosCalls, "validation failures must not call service")
			}
		})
	}
}

func TestTMDbHandler_GetTVShowVideos(t *testing.T) {
	// Smoke test: parallel shape to GetMovieVideos. Deep branches covered there.
	mockSvc := &MockTMDbService{
		GetTVShowVideosResponse: &tmdb.VideosResponse{
			ID: 1396,
			Results: []tmdb.Video{
				{Key: "HhesaQXLuRY", Name: "Breaking Bad Trailer", Site: "YouTube", Type: "Trailer", Official: true},
			},
		},
	}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/tv/1396/videos", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []int{1396}, mockSvc.GetTVShowVideosCalls)

	var response APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, response.Success)
}

func TestTMDbHandler_GetTVShowVideos_InvalidID(t *testing.T) {
	mockSvc := &MockTMDbService{}
	handler := NewTMDbHandler(mockSvc)
	router := setupTMDbRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tmdb/tv/0/videos", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.GetTVShowVideosCalls)
}
