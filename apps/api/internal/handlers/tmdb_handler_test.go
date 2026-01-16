package handlers

import (
	"context"
	"encoding/json"
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
	SearchMoviesResponse      *tmdb.SearchResultMovies
	SearchMoviesError         error
	SearchTVShowsResponse     *tmdb.SearchResultTVShows
	SearchTVShowsError        error
	GetMovieDetailsResponse   *tmdb.MovieDetails
	GetMovieDetailsError      error
	GetTVShowDetailsResponse  *tmdb.TVShowDetails
	GetTVShowDetailsError     error
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
		"/api/v1/tmdb/search/movies": http.MethodGet,
		"/api/v1/tmdb/search/tv":     http.MethodGet,
		"/api/v1/tmdb/movies/:id":    http.MethodGet,
		"/api/v1/tmdb/tv/:id":        http.MethodGet,
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
