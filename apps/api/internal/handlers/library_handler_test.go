package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (m *MockLibraryService) SearchLibrary(ctx context.Context, query string, params repository.ListParams, mediaType string) (*services.LibrarySearchResults, error) {
	args := m.Called(ctx, query, params, mediaType)
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

func (m *MockLibraryService) GetDistinctGenres(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockLibraryService) GetLibraryStats(ctx context.Context) (*services.LibraryStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LibraryStats), args.Error(1)
}

func (m *MockLibraryService) GetMovieVideos(ctx context.Context, id string) (*tmdb.VideosResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tmdb.VideosResponse), args.Error(1)
}

func (m *MockLibraryService) GetSeriesVideos(ctx context.Context, id string) (*tmdb.VideosResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tmdb.VideosResponse), args.Error(1)
}

func (m *MockLibraryService) BatchDelete(ctx context.Context, ids []string, mediaType string) (*services.BatchResult, error) {
	args := m.Called(ctx, ids, mediaType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.BatchResult), args.Error(1)
}

func (m *MockLibraryService) BatchReparse(ctx context.Context, ids []string, mediaType string) (*services.BatchResult, error) {
	args := m.Called(ctx, ids, mediaType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.BatchResult), args.Error(1)
}

func (m *MockLibraryService) BatchExport(ctx context.Context, ids []string, mediaType string) ([]interface{}, error) {
	args := m.Called(ctx, ids, mediaType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
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

		mockService.On("SearchLibrary", mock.Anything, "駭客", mock.Anything, "all").Return(expectedResult, nil).Once()

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

		mockService.On("SearchLibrary", mock.Anything, "test", mock.Anything, "movie").Return(expectedResult, nil).Once()

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
		}), "all").Return(expectedResult, nil).Once()

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
		mockService.On("SearchLibrary", mock.Anything, "fail", mock.Anything, "all").Return(nil, errors.New("db error")).Once()

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

		mockService.On("SearchLibrary", mock.Anything, "matrix", mock.Anything, "all").Return(expectedResult, nil).Once()

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

		mockService.On("SearchLibrary", mock.Anything, "nonexistent", mock.Anything, "all").Return(expectedResult, nil).Once()

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

		mockService.On("SearchLibrary", mock.Anything, "drama", mock.Anything, "tv").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/search?q=drama&type=tv", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_ListLibrary_WithFilters(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("genre filter passed to service", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 0, TotalPages: 0,
			},
		}

		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			genres, ok := p.Filters["genres"].([]string)
			return ok && len(genres) == 1 && genres[0] == "科幻"
		}), "all").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?genres=%E7%A7%91%E5%B9%BB", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("multiple genres filter passed to service", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 0, TotalPages: 0,
			},
		}

		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			genres, ok := p.Filters["genres"].([]string)
			return ok && len(genres) == 2 && genres[0] == "Action" && genres[1] == "Drama"
		}), "all").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?genres=Action,Drama", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("year range filters passed to service", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 0, TotalPages: 0,
			},
		}

		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			yearMin, ok1 := p.Filters["year_min"].(string)
			yearMax, ok2 := p.Filters["year_max"].(string)
			return ok1 && ok2 && yearMin == "2000" && yearMax == "2020"
		}), "all").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?year_min=2000&year_max=2020", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid year_min returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?year_min=abc", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid year_max returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?year_max=xyz", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("year_min out of range returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?year_min=-1", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("year_max out of range returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?year_max=99999", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("inverted year range returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?year_min=2020&year_max=2000", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_GetGenres(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success - returns genres", func(t *testing.T) {
		expectedGenres := []string{"Action", "Drama", "科幻"}

		mockService.On("GetDistinctGenres", mock.Anything).Return(expectedGenres, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/genres", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		dataSlice, ok := resp.Data.([]interface{})
		require.True(t, ok)
		assert.Equal(t, 3, len(dataSlice))
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockService.On("GetDistinctGenres", mock.Anything).Return(nil, errors.New("db error")).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/genres", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_GetStats(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success - returns stats", func(t *testing.T) {
		expectedStats := &services.LibraryStats{
			YearMin:    1999,
			YearMax:    2024,
			MovieCount: 50,
			TvCount:    30,
			TotalCount: 80,
		}

		mockService.On("GetLibraryStats", mock.Anything).Return(expectedStats, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/stats", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1999), dataMap["yearMin"])
		assert.Equal(t, float64(2024), dataMap["yearMax"])
		assert.Equal(t, float64(50), dataMap["movieCount"])
		assert.Equal(t, float64(30), dataMap["tvCount"])
		assert.Equal(t, float64(80), dataMap["totalCount"])
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockService.On("GetLibraryStats", mock.Anything).Return(nil, errors.New("db error")).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/stats", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_GetMovieVideos(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success - returns videos", func(t *testing.T) {
		expectedVideos := &tmdb.VideosResponse{
			ID: 550,
			Results: []tmdb.Video{
				{ID: "v1", Key: "BdJKm16Co6M", Name: "Official Trailer", Site: "YouTube", Type: "Trailer", Official: true},
			},
		}

		mockService.On("GetMovieVideos", mock.Anything, "movie-123").Return(expectedVideos, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/movies/movie-123/videos", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, dataMap, "results")
	})

	t.Run("not found returns 404", func(t *testing.T) {
		mockService.On("GetMovieVideos", mock.Anything, "missing").
			Return(nil, fmt.Errorf("%w: movie missing", services.ErrNotFound)).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/movies/missing/videos", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockService.On("GetMovieVideos", mock.Anything, "err-id").
			Return(nil, errors.New("tmdb api error")).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/movies/err-id/videos", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_GetSeriesVideos(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success - returns videos", func(t *testing.T) {
		expectedVideos := &tmdb.VideosResponse{
			ID: 1396,
			Results: []tmdb.Video{
				{ID: "v2", Key: "HhesaQXLuRY", Name: "Season 1 Trailer", Site: "YouTube", Type: "Trailer", Official: true},
			},
		}

		mockService.On("GetSeriesVideos", mock.Anything, "series-123").Return(expectedVideos, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/series/series-123/videos", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		mockService.On("GetSeriesVideos", mock.Anything, "missing").
			Return(nil, fmt.Errorf("%w: series missing", services.ErrNotFound)).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library/series/missing/videos", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_BatchDelete(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success - all deleted", func(t *testing.T) {
		ids := []string{"m1", "m2", "m3"}
		expectedResult := &services.BatchResult{
			SuccessCount: 3,
			FailedCount:  0,
		}

		mockService.On("BatchDelete", mock.Anything, ids, "movie").Return(expectedResult, nil).Once()

		body := `{"ids":["m1","m2","m3"],"type":"movie"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/library/batch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(3), dataMap["success_count"])
		assert.Equal(t, float64(0), dataMap["failed_count"])
	})

	t.Run("success - partial failure", func(t *testing.T) {
		ids := []string{"m1", "m2"}
		expectedResult := &services.BatchResult{
			SuccessCount: 1,
			FailedCount:  1,
			Errors:       []services.BatchError{{ID: "m2", Message: "not found"}},
		}

		mockService.On("BatchDelete", mock.Anything, ids, "series").Return(expectedResult, nil).Once()

		body := `{"ids":["m1","m2"],"type":"series"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/library/batch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		dataMap, ok := resp.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1), dataMap["success_count"])
		assert.Equal(t, float64(1), dataMap["failed_count"])
	})

	t.Run("missing ids returns 400", func(t *testing.T) {
		body := `{"type":"movie"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/library/batch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid type returns 400", func(t *testing.T) {
		body := `{"ids":["m1"],"type":"invalid"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/library/batch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty body returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/library/batch", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_BatchReparse(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success", func(t *testing.T) {
		ids := []string{"m1", "m2"}
		expectedResult := &services.BatchResult{
			SuccessCount: 2,
			FailedCount:  0,
		}

		mockService.On("BatchReparse", mock.Anything, ids, "movie").Return(expectedResult, nil).Once()

		body := `{"ids":["m1","m2"],"type":"movie"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/library/batch/reparse", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("missing type returns 400", func(t *testing.T) {
		body := `{"ids":["m1"]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/library/batch/reparse", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		ids := []string{"m1"}
		mockService.On("BatchReparse", mock.Anything, ids, "movie").Return(nil, errors.New("db error")).Once()

		body := `{"ids":["m1"],"type":"movie"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/library/batch/reparse", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	mockService.AssertExpectations(t)
}

func TestLibraryHandler_BatchExport(t *testing.T) {
	mockService := new(MockLibraryService)
	handler := NewLibraryHandler(mockService)
	router := setupLibraryTestRouter(handler)

	t.Run("success", func(t *testing.T) {
		ids := []string{"m1", "m2"}
		expectedItems := []interface{}{
			&models.Movie{ID: "m1", Title: "Movie 1"},
			&models.Movie{ID: "m2", Title: "Movie 2"},
		}

		mockService.On("BatchExport", mock.Anything, ids, "movie").Return(expectedItems, nil).Once()

		body := `{"ids":["m1","m2"],"format":"json"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/library/batch/export?type=movie", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("missing format returns 400", func(t *testing.T) {
		body := `{"ids":["m1"]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/library/batch/export", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	mockService.AssertExpectations(t)
}
