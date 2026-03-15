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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/tmdb"
)

// MockLibraryService is a mock implementation of LibraryServiceInterface
type MockLibraryService struct {
	mock.Mock
}

func (m *MockLibraryService) SaveMovieFromTMDb(ctx context.Context, tmdbMovie *tmdb.MovieDetails, filePath string) (*models.Movie, error) {
	args := m.Called(ctx, tmdbMovie, filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockLibraryService) SaveSeriesFromTMDb(ctx context.Context, tmdbSeries *tmdb.TVShowDetails, filePath string) (*models.Series, error) {
	args := m.Called(ctx, tmdbSeries, filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Series), args.Error(1)
}

func (m *MockLibraryService) SearchLibrary(ctx context.Context, query string, params repository.ListParams) (*services.LibrarySearchResults, error) {
	args := m.Called(ctx, query, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LibrarySearchResults), args.Error(1)
}

func (m *MockLibraryService) GetMovieByID(ctx context.Context, id string) (*models.Movie, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockLibraryService) GetSeriesByID(ctx context.Context, id string) (*models.Series, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Series), args.Error(1)
}

func (m *MockLibraryService) GetMovieByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	args := m.Called(ctx, tmdbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockLibraryService) GetSeriesByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	args := m.Called(ctx, tmdbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Series), args.Error(1)
}

func (m *MockLibraryService) ListLibrary(ctx context.Context, params repository.ListParams, mediaType string) (*services.LibraryListResult, error) {
	args := m.Called(ctx, params, mediaType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LibraryListResult), args.Error(1)
}

func (m *MockLibraryService) GetRecentlyAdded(ctx context.Context, limit int) (*services.LibraryListResult, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LibraryListResult), args.Error(1)
}

func (m *MockLibraryService) DeleteMovie(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockLibraryService) DeleteSeries(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Verify mock implements interface
var _ services.LibraryServiceInterface = (*MockLibraryService)(nil)

func setupLibraryTestRouter(handler *LibraryHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestLibraryHandler_ListLibrary(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success - list all", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{
				{Type: "movie", Movie: &models.Movie{ID: "m1", Title: "Movie 1"}},
				{Type: "series", Series: &models.Series{ID: "s1", Title: "Series 1"}},
			},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 2, TotalPages: 1,
			},
		}

		mockService.On("ListLibrary", mock.Anything, mock.Anything, "all").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("success - filter by movie type", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{
				{Type: "movie", Movie: &models.Movie{ID: "m1", Title: "Movie 1"}},
			},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1,
			},
		}

		mockService.On("ListLibrary", mock.Anything, mock.Anything, "movie").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?type=movie", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success - filter by tv type", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{
				{Type: "series", Series: &models.Series{ID: "s1", Title: "Series 1"}},
			},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1,
			},
		}

		mockService.On("ListLibrary", mock.Anything, mock.Anything, "tv").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?type=tv", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid type returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?type=invalid", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockService.On("ListLibrary", mock.Anything, mock.Anything, "all").Return(nil, errors.New("db error")).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?type=all", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("pagination params passed correctly", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{},
			Pagination: &repository.PaginationResult{
				Page: 2, PageSize: 10, TotalResults: 15, TotalPages: 2,
			},
		}

		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			return p.Page == 2 && p.PageSize == 10
		}), "all").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?page=2&page_size=10", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_DeleteMovie(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success", func(t *testing.T) {
		mockService.On("DeleteMovie", mock.Anything, "movie-123").Return(nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/library/movies/movie-123", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockService.On("DeleteMovie", mock.Anything, "bad-id").Return(errors.New("not found")).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/library/movies/bad-id", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_DeleteSeries(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success", func(t *testing.T) {
		mockService.On("DeleteSeries", mock.Anything, "series-123").Return(nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/library/series/series-123", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_ReparseMovie(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success", func(t *testing.T) {
		mockService.On("GetMovieByID", mock.Anything, "movie-123").
			Return(&models.Movie{ID: "movie-123", Title: "Test"}, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/library/movies/movie-123/reparse", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockService.On("GetMovieByID", mock.Anything, "missing").
			Return(nil, errors.New("not found")).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/library/movies/missing/reparse", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_ExportMovie(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success", func(t *testing.T) {
		mockService.On("GetMovieByID", mock.Anything, "movie-123").
			Return(&models.Movie{ID: "movie-123", Title: "Test Movie"}, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/library/movies/movie-123/export", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_GetRecentlyAdded(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success - default limit", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{
				{Type: "movie", Movie: &models.Movie{ID: "m1", Title: "New Movie"}},
				{Type: "series", Series: &models.Series{ID: "s1", Title: "New Series"}},
			},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 2, TotalPages: 1,
			},
		}

		mockService.On("GetRecentlyAdded", mock.Anything, 20).Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/recent", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		// Verify paginated response structure
		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok, "Response data should be a paginated object")
		assert.Contains(t, dataMap, "items")
		assert.Contains(t, dataMap, "totalItems")
	})

	t.Run("success - custom limit", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{
				{Type: "movie", Movie: &models.Movie{ID: "m1", Title: "New Movie"}},
			},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 10, TotalResults: 1, TotalPages: 1,
			},
		}

		mockService.On("GetRecentlyAdded", mock.Anything, 10).Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/recent?limit=10", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid limit - not a number", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/recent?limit=abc", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid limit - zero", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/recent?limit=0", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid limit - over 100", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/recent?limit=101", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockService.On("GetRecentlyAdded", mock.Anything, 20).Return(nil, errors.New("db error")).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/recent", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_SearchLibrary(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success - search with results", func(t *testing.T) {
		expectedResult := &services.LibrarySearchResults{
			Results: []services.SearchResult{
				{Type: "movie", Movie: &models.Movie{ID: "m1", Title: "駭客任務"}},
				{Type: "series", Series: &models.Series{ID: "s1", Title: "進擊的巨人"}},
			},
			Movies: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1,
			},
			Series: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1,
			},
			TotalCount: 2,
		}

		mockService.On("SearchLibrary", mock.Anything, "駭客", mock.Anything).Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=%E9%A7%AD%E5%AE%A2", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, dataMap, "results")
		assert.Contains(t, dataMap, "totalCount")
	})

	t.Run("missing query returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty query returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("query too short returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=a", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success - with type filter", func(t *testing.T) {
		expectedResult := &services.LibrarySearchResults{
			Results:    []services.SearchResult{},
			TotalCount: 0,
		}

		mockService.On("SearchLibrary", mock.Anything, "test", mock.Anything).Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=test&type=movie", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success - with pagination", func(t *testing.T) {
		expectedResult := &services.LibrarySearchResults{
			Results:    []services.SearchResult{},
			TotalCount: 0,
		}

		mockService.On("SearchLibrary", mock.Anything, "movie", mock.MatchedBy(func(p repository.ListParams) bool {
			return p.Page == 2 && p.PageSize == 10
		})).Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=movie&page=2&page_size=10", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid type returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=test&type=invalid", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockService.On("SearchLibrary", mock.Anything, "fail", mock.Anything).Return(nil, errors.New("db error")).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=fail", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success - English query", func(t *testing.T) {
		expectedResult := &services.LibrarySearchResults{
			Results: []services.SearchResult{
				{Type: "movie", Movie: &models.Movie{ID: "m2", Title: "The Matrix"}},
			},
			Movies: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1,
			},
			TotalCount: 1,
		}

		mockService.On("SearchLibrary", mock.Anything, "matrix", mock.Anything).Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=matrix", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("success - empty results returns valid structure", func(t *testing.T) {
		expectedResult := &services.LibrarySearchResults{
			Results:    []services.SearchResult{},
			TotalCount: 0,
		}

		mockService.On("SearchLibrary", mock.Anything, "nonexistent", mock.Anything).Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=nonexistent", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)

		totalCount, ok := dataMap["totalCount"].(float64)
		require.True(t, ok)
		assert.Equal(t, float64(0), totalCount)
	})

	t.Run("success - type=tv filter passes to service", func(t *testing.T) {
		expectedResult := &services.LibrarySearchResults{
			Results:    []services.SearchResult{},
			TotalCount: 0,
		}

		mockService.On("SearchLibrary", mock.Anything, "drama", mock.Anything).Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=drama&type=tv", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockService.AssertExpectations(t)
}
