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
	"time"

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

// dsr-2b-a AC #2: single-item re-match runs enrichment for that item and
// returns the row it left — it used to confirm the item existed and answer
// "reparse_queued" while nothing was queued anywhere.
type fakeItemEnricher struct {
	kind  services.MediaKind
	id    string
	item  *services.EnrichedItem
	err   error
	block bool // wait for the deadline, like a hung TMDb lookup
}

func (f *fakeItemEnricher) EnrichOne(ctx context.Context, kind services.MediaKind, id string) (*services.EnrichedItem, error) {
	f.kind, f.id = kind, id
	if f.block {
		<-ctx.Done()
		return nil, fmt.Errorf("re-match %s: %w", id, ctx.Err())
	}
	return f.item, f.err
}

func reparse(t *testing.T, handler *LibraryHandler, path string) (int, map[string]interface{}) {
	t.Helper()
	router := setupLibraryTestRouter(handler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", path, nil)
	router.ServeHTTP(w, req)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body
}

func TestLibraryHandler_ReparseMovie(t *testing.T) {
	t.Run("success returns the row the re-match left", func(t *testing.T) {
		enricher := &fakeItemEnricher{item: &services.EnrichedItem{ID: "movie-123", ParseStatus: models.ParseStatusSuccess, Title: "鬥陣俱樂部", TMDbID: 550}}
		handler := NewLibraryHandler(new(MockLibraryService))
		handler.SetItemEnricher(enricher)

		status, body := reparse(t, handler, "/api/v1/library/movies/movie-123/reparse")

		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, services.MediaKindMovie, enricher.kind)
		assert.Equal(t, "movie-123", enricher.id)
		data := body["data"].(map[string]interface{})
		assert.Equal(t, "movie-123", data["id"])
		assert.Equal(t, "success", data["parse_status"])
		assert.Equal(t, "鬥陣俱樂部", data["title"])
		assert.Equal(t, float64(550), data["tmdb_id"])
		assert.NotContains(t, data, "status", "the old reparse_queued shape is gone")
	})

	t.Run("still no match is a 200 carrying failed", func(t *testing.T) {
		handler := NewLibraryHandler(new(MockLibraryService))
		handler.SetItemEnricher(&fakeItemEnricher{item: &services.EnrichedItem{ID: "m", ParseStatus: models.ParseStatusFailed, Title: "zzqx"}})

		status, body := reparse(t, handler, "/api/v1/library/movies/m/reparse")

		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "failed", body["data"].(map[string]interface{})["parse_status"])
	})

	errCases := map[string]struct {
		err    error
		status int
		code   string
	}{
		"not found":     {services.ErrEnrichItemNotFound, http.StatusNotFound, "DB_NOT_FOUND"},
		"batch running": {services.ErrEnrichmentAlreadyRunning, http.StatusConflict, "ENRICHMENT_ALREADY_RUNNING"},
		"write failed":  {fmt.Errorf("%w: update movie: database is locked", services.ErrEnrichPersist), http.StatusInternalServerError, "DB_QUERY_FAILED"},
		// Not a storage fault — must not be labelled one (CR #5).
		"client went away": {fmt.Errorf("re-match m: %w", context.Canceled), http.StatusInternalServerError, "INTERNAL_ERROR"},
		"not configured":   {errors.New("series enrichment not configured"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for name, tc := range errCases {
		t.Run(name, func(t *testing.T) {
			handler := NewLibraryHandler(new(MockLibraryService))
			handler.SetItemEnricher(&fakeItemEnricher{err: tc.err})

			status, body := reparse(t, handler, "/api/v1/library/movies/m/reparse")

			assert.Equal(t, tc.status, status)
			assert.Equal(t, tc.code, body["error"].(map[string]interface{})["code"])
		})
	}

	t.Run("deadline is a 504 METADATA_TIMEOUT", func(t *testing.T) {
		handler := NewLibraryHandler(new(MockLibraryService))
		handler.SetItemEnricher(&fakeItemEnricher{block: true})
		handler.reparseTimeout = 20 * time.Millisecond

		status, body := reparse(t, handler, "/api/v1/library/movies/m/reparse")

		assert.Equal(t, http.StatusGatewayTimeout, status)
		assert.Equal(t, "METADATA_TIMEOUT", body["error"].(map[string]interface{})["code"])
	})

	t.Run("no enricher wired is a 500, not a fake success", func(t *testing.T) {
		status, body := reparse(t, NewLibraryHandler(new(MockLibraryService)), "/api/v1/library/movies/m/reparse")

		assert.Equal(t, http.StatusInternalServerError, status)
		assert.False(t, body["success"].(bool))
		assert.Equal(t, "INTERNAL_ERROR", body["error"].(map[string]interface{})["code"])
	})
}

func TestLibraryHandler_ReparseSeries(t *testing.T) {
	enricher := &fakeItemEnricher{item: &services.EnrichedItem{ID: "s-1", ParseStatus: models.ParseStatusSuccess, Title: "絕命毒師", TMDbID: 1396}}
	handler := NewLibraryHandler(new(MockLibraryService))
	handler.SetItemEnricher(enricher)

	status, body := reparse(t, handler, "/api/v1/library/series/s-1/reparse")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, services.MediaKindSeries, enricher.kind)
	assert.Equal(t, "絕命毒師", body["data"].(map[string]interface{})["title"])
}

func TestLibraryHandler_Reparse_DefaultTimeoutIsSixtySeconds(t *testing.T) {
	assert.Equal(t, 60*time.Second, NewLibraryHandler(new(MockLibraryService)).reparseTimeout)
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
		assert.Contains(t, dataMap, "total_items")
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
		assert.Contains(t, dataMap, "total_count")
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

		totalCount, ok := dataMap["total_count"].(float64)
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

	t.Run("unmatched filter passed to service", func(t *testing.T) {
		expectedResult := &services.LibraryListResult{
			Items: []services.LibraryItem{},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 0, TotalPages: 0,
			},
		}

		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			unmatched, ok := p.Filters["unmatched"].(bool)
			return ok && unmatched
		}), "all").Return(expectedResult, nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?unmatched=true", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// dsr-1b-a AC #1 [@contract-v1]: subtitle_status is a comma-separated status list,
	// validated against models.SubtitleStatus.IsValid, stored as []string (same shape
	// as genres — a type mismatch on the repo side would silently drop the filter).
	okResult := func() *services.LibraryListResult {
		return &services.LibraryListResult{
			Items: []services.LibraryItem{},
			Pagination: &repository.PaginationResult{
				Page: 1, PageSize: 20, TotalResults: 0, TotalPages: 0,
			},
		}
	}

	t.Run("subtitle_status single value passed to service as []string", func(t *testing.T) {
		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			statuses, ok := p.Filters["subtitle_status"].([]string)
			return ok && len(statuses) == 1 && statuses[0] == "not_found"
		}), "all").Return(okResult(), nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?subtitle_status=not_found", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("subtitle_status csv is split, trimmed and kept in order", func(t *testing.T) {
		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			statuses, ok := p.Filters["subtitle_status"].([]string)
			return ok && len(statuses) == 2 && statuses[0] == "not_found" && statuses[1] == "not_searched"
		}), "all").Return(okResult(), nil).Once()

		w := httptest.NewRecorder()
		// "%20" = the space a hand-typed deep link tends to carry after the comma.
		req, _ := http.NewRequest("GET", "/api/v1/library?subtitle_status=not_found,%20not_searched", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("subtitle_status unknown value returns 400 and never reaches the service", func(t *testing.T) {
		fresh := new(MockLibraryService)
		freshRouter := setupLibraryTestRouter(NewLibraryHandler(fresh))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?subtitle_status=bogus", nil)
		freshRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_INVALID_FORMAT")
		assert.Contains(t, w.Body.String(), "bogus", "the 400 must name the offending value")
		fresh.AssertNotCalled(t, "ListLibrary", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("subtitle_status one bad value in a csv poisons the whole request", func(t *testing.T) {
		fresh := new(MockLibraryService)
		freshRouter := setupLibraryTestRouter(NewLibraryHandler(fresh))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?subtitle_status=not_found,bogus", nil)
		freshRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		fresh.AssertNotCalled(t, "ListLibrary", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("subtitle_status empty string means no filter", func(t *testing.T) {
		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			_, present := p.Filters["subtitle_status"]
			return !present
		}), "all").Return(okResult(), nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?subtitle_status=", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("subtitle_status stacks with unmatched and genres", func(t *testing.T) {
		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			statuses, ok1 := p.Filters["subtitle_status"].([]string)
			unmatched, ok2 := p.Filters["unmatched"].(bool)
			genres, ok3 := p.Filters["genres"].([]string)
			return ok1 && ok2 && ok3 &&
				len(statuses) == 1 && statuses[0] == "not_found" &&
				unmatched &&
				len(genres) == 1 && genres[0] == "動畫"
		}), "movie").Return(okResult(), nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?type=movie&subtitle_status=not_found&unmatched=true&genres=%E5%8B%95%E7%95%AB", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("subtitle_status duplicates collapse to one IN value", func(t *testing.T) {
		mockService.On("ListLibrary", mock.Anything, mock.MatchedBy(func(p repository.ListParams) bool {
			statuses, ok := p.Filters["subtitle_status"].([]string)
			return ok && len(statuses) == 1 && statuses[0] == "not_found"
		}), "all").Return(okResult(), nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?subtitle_status=not_found,not_found,not_found", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("subtitle_status is case-sensitive: NOT_FOUND is rejected", func(t *testing.T) {
		fresh := new(MockLibraryService)
		freshRouter := setupLibraryTestRouter(NewLibraryHandler(fresh))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?subtitle_status=NOT_FOUND", nil)
		freshRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		fresh.AssertNotCalled(t, "ListLibrary", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("subtitle_status oversized bad value is truncated in the 400 message", func(t *testing.T) {
		fresh := new(MockLibraryService)
		freshRouter := setupLibraryTestRouter(NewLibraryHandler(fresh))
		huge := strings.Repeat("x", 5000)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/library?subtitle_status="+huge, nil)
		freshRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Less(t, w.Body.Len(), 600, "the echoed value must be capped, not reflected whole")
		assert.Contains(t, w.Body.String(), strings.Repeat("x", 64)+"…")
	})

	// Kept at the very end on purpose: every .Once() expectation above is
	// accounted for here, including the subtitle_status ones.
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
		assert.Equal(t, float64(1999), dataMap["year_min"])
		assert.Equal(t, float64(2024), dataMap["year_max"])
		assert.Equal(t, float64(50), dataMap["movie_count"])
		assert.Equal(t, float64(30), dataMap["tv_count"])
		assert.Equal(t, float64(80), dataMap["total_count"])
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
