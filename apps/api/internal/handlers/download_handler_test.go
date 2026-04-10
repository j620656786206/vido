package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
)

// MockDownloadService mocks DownloadServiceInterface for handler tests.
type MockDownloadService struct {
	mock.Mock
}

func (m *MockDownloadService) GetAllDownloads(ctx context.Context, filter string, sort string, order string) ([]qbittorrent.Torrent, error) {
	args := m.Called(ctx, filter, sort, order)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]qbittorrent.Torrent), args.Error(1)
}

func (m *MockDownloadService) GetDownloadDetails(ctx context.Context, hash string) (*qbittorrent.TorrentDetails, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.TorrentDetails), args.Error(1)
}

func (m *MockDownloadService) GetDownloadCounts(ctx context.Context) (*qbittorrent.DownloadCounts, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.DownloadCounts), args.Error(1)
}

func setupDownloadRouter(handler *DownloadHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestDownloadHandler_ListDownloads_Success(t *testing.T) {
	mockService := new(MockDownloadService)
	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	torrents := []qbittorrent.Torrent{
		{
			Hash:          "abc123",
			Name:          "Test Movie [1080p]",
			Size:          4294967296,
			Progress:      0.85,
			DownloadSpeed: 10485760,
			UploadSpeed:   524288,
			ETA:           600,
			Status:        qbittorrent.StatusDownloading,
			AddedOn:       addedOn,
			Seeds:         10,
			Peers:         5,
		},
	}
	mockService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_WithSortParams(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetAllDownloads", mock.Anything, "all", "name", "asc").Return([]qbittorrent.Torrent{}, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads?sort=name&order=asc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_WithFilterParam(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetAllDownloads", mock.Anything, "downloading", "added_on", "desc").Return([]qbittorrent.Torrent{}, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads?filter=downloading", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_WithFilterAndSort(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetAllDownloads", mock.Anything, "paused", "name", "asc").Return([]qbittorrent.Torrent{}, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads?filter=paused&sort=name&order=asc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_NotConfigured(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeNotConfigured,
			Message: "qBittorrent not configured",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_AuthFailure(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeAuthFailed,
			Message: "auth failed",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, qbittorrent.ErrCodeAuthFailed, response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadDetails_Success(t *testing.T) {
	mockService := new(MockDownloadService)
	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	details := &qbittorrent.TorrentDetails{
		Torrent: qbittorrent.Torrent{
			Hash:    "abc123",
			Name:    "Test Movie",
			Status:  qbittorrent.StatusDownloading,
			AddedOn: addedOn,
		},
		PieceSize:    4194304,
		Comment:      "Test comment",
		CreationDate: addedOn,
		TimeElapsed:  3600,
	}
	mockService.On("GetDownloadDetails", mock.Anything, "abc123").Return(details, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/abc123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadDetails_NotFound(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadDetails", mock.Anything, "nonexistent").Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeTorrentNotFound,
			Message: "torrent not found: nonexistent",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "DB_NOT_FOUND", response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadCounts_Success(t *testing.T) {
	mockService := new(MockDownloadService)
	counts := &qbittorrent.DownloadCounts{
		All:         10,
		Downloading: 3,
		Paused:      2,
		Completed:   4,
		Seeding:     1,
		Error:       0,
	}
	mockService.On("GetDownloadCounts", mock.Anything).Return(counts, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/counts", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(10), dataMap["all"])
	assert.Equal(t, float64(3), dataMap["downloading"])
	assert.Equal(t, float64(2), dataMap["paused"])
	assert.Equal(t, float64(4), dataMap["completed"])
	assert.Equal(t, float64(1), dataMap["seeding"])
	assert.Equal(t, float64(0), dataMap["error"])
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadCounts_NotConfigured(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadCounts", mock.Anything).Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeNotConfigured,
			Message: "qBittorrent not configured",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/counts", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadDetails_NotConfigured(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadDetails", mock.Anything, "abc123").Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeNotConfigured,
			Message: "qBittorrent not configured",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/abc123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_AllFilterValues(t *testing.T) {
	// GIVEN: all valid filter values
	filters := []string{"all", "downloading", "paused", "completed", "seeding", "error"}

	for _, filter := range filters {
		t.Run(filter, func(t *testing.T) {
			mockService := new(MockDownloadService)
			mockService.On("GetAllDownloads", mock.Anything, filter, "added_on", "desc").Return([]qbittorrent.Torrent{}, nil)

			handler := NewDownloadHandler(mockService)
			router := setupDownloadRouter(handler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/downloads?filter="+filter, nil)
			router.ServeHTTP(w, req)

			// THEN: each filter value is accepted and passed through
			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDownloadHandler_ListDownloads_InternalServerError(t *testing.T) {
	// GIVEN: service returns a non-ConnectionError
	mockService := new(MockDownloadService)
	mockService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(
		nil,
		fmt.Errorf("unexpected internal error"),
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 500 internal server error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadCounts_AuthFailure(t *testing.T) {
	// GIVEN: counts endpoint returns auth failure
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadCounts", mock.Anything).Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeAuthFailed,
			Message: "auth failed",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/counts", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 400 with auth error code
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, qbittorrent.ErrCodeAuthFailed, response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadCounts_InternalServerError(t *testing.T) {
	// GIVEN: counts service returns a non-ConnectionError
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadCounts", mock.Anything).Return(
		nil,
		fmt.Errorf("unexpected error"),
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/counts", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_GenericConnectionError(t *testing.T) {
	// GIVEN: service returns a generic connection error (not NotConfigured or AuthFailed)
	mockService := new(MockDownloadService)
	mockService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeConnectionFailed,
			Message: "connection refused",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 400 with connection error
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, qbittorrent.ErrCodeConnectionFailed, response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadDetails_EmptyHash(t *testing.T) {
	// GIVEN: request with empty hash (router sends empty string for missing param)
	mockService := new(MockDownloadService)

	handler := NewDownloadHandler(mockService)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Register route that allows empty hash to reach handler
	router.GET("/api/v1/downloads/:hash", handler.GetDownloadDetails)

	w := httptest.NewRecorder()
	// Gin path params are always non-empty when matched, so we test via direct handler call
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/downloads/", nil)
	c.Params = gin.Params{{Key: "hash", Value: ""}}

	handler.GetDownloadDetails(c)

	// THEN: returns 400 validation error
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	// Service should not be called
	mockService.AssertNotCalled(t, "GetDownloadDetails")
}

func TestDownloadHandler_GetDownloadDetails_AuthFailure(t *testing.T) {
	// GIVEN: details endpoint returns auth failure
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadDetails", mock.Anything, "abc123").Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeAuthFailed,
			Message: "auth failed",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/abc123", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 400 with auth error code
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadDetails_InternalServerError(t *testing.T) {
	// GIVEN: details service returns a non-ConnectionError
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadDetails", mock.Anything, "abc123").Return(
		nil,
		fmt.Errorf("unexpected internal error"),
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/abc123", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadDetails_GenericConnectionError(t *testing.T) {
	// GIVEN: details service returns a generic connection error (not NotConfigured/Auth/NotFound)
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadDetails", mock.Anything, "abc123").Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeConnectionFailed,
			Message: "connection refused",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/abc123", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 400 with connection error
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, qbittorrent.ErrCodeConnectionFailed, response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadCounts_GenericConnectionError(t *testing.T) {
	// GIVEN: counts service returns a generic connection error (not NotConfigured/AuthFailed)
	mockService := new(MockDownloadService)
	mockService.On("GetDownloadCounts", mock.Anything).Return(
		nil,
		&qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeConnectionFailed,
			Message: "connection refused",
		},
	)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/counts", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 400 with connection error
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, qbittorrent.ErrCodeConnectionFailed, response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_InvalidFilterDefaultsToAll(t *testing.T) {
	// GIVEN: request with invalid filter value
	mockService := new(MockDownloadService)
	// Handler passes raw filter to service; service normalizes to "all"
	mockService.On("GetAllDownloads", mock.Anything, "bogus", "added_on", "desc").Return([]qbittorrent.Torrent{}, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads?filter=bogus", nil)
	router.ServeHTTP(w, req)

	// THEN: returns 200 (service handles fallback)
	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_GetDownloadCounts_VerifyResponseStructure(t *testing.T) {
	// GIVEN: counts with all statuses including error > 0
	mockService := new(MockDownloadService)
	counts := &qbittorrent.DownloadCounts{
		All:         15,
		Downloading: 5,
		Paused:      3,
		Completed:   4,
		Seeding:     2,
		Error:       1,
	}
	mockService.On("GetDownloadCounts", mock.Anything).Return(counts, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/counts", nil)
	router.ServeHTTP(w, req)

	// THEN: response has all 6 count fields with correct values
	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok)

	// Verify all 6 fields present and correct
	assert.Equal(t, float64(15), dataMap["all"])
	assert.Equal(t, float64(5), dataMap["downloading"])
	assert.Equal(t, float64(3), dataMap["paused"])
	assert.Equal(t, float64(4), dataMap["completed"])
	assert.Equal(t, float64(2), dataMap["seeding"])
	assert.Equal(t, float64(1), dataMap["error"])

	// Verify exactly 6 count fields
	countFields := []string{"all", "downloading", "paused", "completed", "seeding", "error"}
	for _, field := range countFields {
		assert.Contains(t, dataMap, field)
	}
	mockService.AssertExpectations(t)
}

// --- Parse Status Enrichment Tests ---

func setupDownloadRouterWithParseQueue(handler *DownloadHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestDownloadHandler_ListDownloads_WithParseStatus(t *testing.T) {
	mockDLService := new(MockDownloadService)
	mockPQService := new(MockParseQueueService)

	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	torrents := []qbittorrent.Torrent{
		{
			Hash:    "completed-hash",
			Name:    "Movie.mkv",
			Status:  qbittorrent.StatusCompleted,
			AddedOn: addedOn,
		},
		{
			Hash:    "downloading-hash",
			Name:    "Other.mkv",
			Status:  qbittorrent.StatusDownloading,
			AddedOn: addedOn,
		},
	}
	mockDLService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)

	mediaID := "media-123"
	mockPQService.On("GetJobStatus", mock.Anything, "completed-hash").Return(&models.ParseJob{
		ID:          "job-1",
		TorrentHash: "completed-hash",
		Status:      models.ParseJobCompleted,
		MediaID:     &mediaID,
	}, nil)
	// downloading-hash should NOT be looked up (not completed/seeding)

	handler := NewDownloadHandler(mockDLService, mockPQService)
	router := setupDownloadRouterWithParseQueue(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "response.Data should be a PaginatedResponse map")
	dataSlice, ok := dataMap["items"].([]interface{})
	require.True(t, ok, "items should be a slice")
	require.Len(t, dataSlice, 2)

	// Verify pagination metadata
	assert.Equal(t, float64(1), dataMap["page"])
	assert.Equal(t, float64(2), dataMap["total_items"])
	assert.Equal(t, float64(1), dataMap["total_pages"])

	// First item (completed) should have parseStatus
	item0, ok := dataSlice[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "completed-hash", item0["hash"])
	parseStatus, ok := item0["parse_status"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "completed", parseStatus["status"])
	assert.Equal(t, "media-123", parseStatus["media_id"])

	// Second item (downloading) should NOT have parseStatus
	item1, ok := dataSlice[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "downloading-hash", item1["hash"])
	assert.Nil(t, item1["parse_status"])

	mockDLService.AssertExpectations(t)
	mockPQService.AssertExpectations(t)
	// Verify downloading hash was never looked up
	mockPQService.AssertNotCalled(t, "GetJobStatus", mock.Anything, "downloading-hash")
}

func TestDownloadHandler_ListDownloads_WithParseStatus_NoJob(t *testing.T) {
	mockDLService := new(MockDownloadService)
	mockPQService := new(MockParseQueueService)

	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	torrents := []qbittorrent.Torrent{
		{
			Hash:    "completed-hash",
			Name:    "Movie.mkv",
			Status:  qbittorrent.StatusCompleted,
			AddedOn: addedOn,
		},
	}
	mockDLService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)
	mockPQService.On("GetJobStatus", mock.Anything, "completed-hash").Return(nil, nil)

	handler := NewDownloadHandler(mockDLService, mockPQService)
	router := setupDownloadRouterWithParseQueue(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "response.Data should be a PaginatedResponse map")
	dataSlice, ok := dataMap["items"].([]interface{})
	require.True(t, ok, "items should be a slice")
	require.Len(t, dataSlice, 1)

	// Verify pagination metadata
	assert.Equal(t, float64(1), dataMap["page"])
	assert.Equal(t, float64(1), dataMap["total_items"])
	assert.Equal(t, float64(1), dataMap["total_pages"])

	item, ok := dataSlice[0].(map[string]interface{})
	require.True(t, ok)
	// No parse job exists, so parseStatus should be nil/absent
	assert.Nil(t, item["parse_status"])

	mockDLService.AssertExpectations(t)
	mockPQService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_WithParseStatus_Failed(t *testing.T) {
	mockDLService := new(MockDownloadService)
	mockPQService := new(MockParseQueueService)

	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	torrents := []qbittorrent.Torrent{
		{
			Hash:    "failed-hash",
			Name:    "Unknown.mkv",
			Status:  qbittorrent.StatusCompleted,
			AddedOn: addedOn,
		},
	}
	mockDLService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)

	errMsg := "could not parse filename"
	mockPQService.On("GetJobStatus", mock.Anything, "failed-hash").Return(&models.ParseJob{
		ID:           "job-2",
		TorrentHash:  "failed-hash",
		Status:       models.ParseJobFailed,
		ErrorMessage: &errMsg,
	}, nil)

	handler := NewDownloadHandler(mockDLService, mockPQService)
	router := setupDownloadRouterWithParseQueue(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "response.Data should be a PaginatedResponse map")
	dataSlice, ok := dataMap["items"].([]interface{})
	require.True(t, ok, "items should be a slice")

	// Verify pagination metadata
	assert.Equal(t, float64(1), dataMap["page"])
	assert.Equal(t, float64(1), dataMap["total_items"])
	assert.Equal(t, float64(1), dataMap["total_pages"])

	item, ok := dataSlice[0].(map[string]interface{})
	require.True(t, ok)

	parseStatus, ok := item["parse_status"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "failed", parseStatus["status"])
	assert.Equal(t, "could not parse filename", parseStatus["error_message"])

	mockDLService.AssertExpectations(t)
	mockPQService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_WithoutParseQueueService(t *testing.T) {
	// When no parse queue service is provided, response should be plain torrents
	mockDLService := new(MockDownloadService)

	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	torrents := []qbittorrent.Torrent{
		{
			Hash:    "abc123",
			Name:    "Movie.mkv",
			Status:  qbittorrent.StatusCompleted,
			AddedOn: addedOn,
		},
	}
	mockDLService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)

	handler := NewDownloadHandler(mockDLService) // No parse queue service
	router := setupDownloadRouterWithParseQueue(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)

	// Should still work, returning plain torrent data in paginated format
	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "response.Data should be a PaginatedResponse map")
	dataSlice, ok := dataMap["items"].([]interface{})
	require.True(t, ok, "items should be a slice")
	require.Len(t, dataSlice, 1)

	// Verify pagination metadata
	assert.Equal(t, float64(1), dataMap["page"])
	assert.Equal(t, float64(1), dataMap["total_items"])
	assert.Equal(t, float64(1), dataMap["total_pages"])

	mockDLService.AssertExpectations(t)
}

// --- Pagination Boundary Tests ---

func TestDownloadHandler_ListDownloads_PaginationBoundary_PageSizeClamping(t *testing.T) {
	mockService := new(MockDownloadService)
	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	torrents := []qbittorrent.Torrent{
		{Hash: "t1", Name: "Movie1.mkv", Status: qbittorrent.StatusCompleted, AddedOn: addedOn},
	}
	mockService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	tests := []struct {
		name             string
		query            string
		expectedPage     float64
		expectedPageSize float64
	}{
		{"pageSize below 1 clamps to 100", "?pageSize=0", 1, 100},
		{"pageSize above 500 clamps to 500", "?pageSize=999", 1, 500},
		{"page below 1 clamps to 1", "?page=0", 1, 100},
		{"negative page clamps to 1", "?page=-5", 1, 100},
		{"negative pageSize clamps to 100", "?pageSize=-10", 1, 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/downloads"+tc.query, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			dataMap, ok := response.Data.(map[string]interface{})
			require.True(t, ok, "response.Data should be a PaginatedResponse map")

			assert.Equal(t, tc.expectedPage, dataMap["page"])
			assert.Equal(t, tc.expectedPageSize, dataMap["page_size"])
		})
	}
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_PaginationBoundary_PageBeyondTotal(t *testing.T) {
	mockService := new(MockDownloadService)
	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	torrents := []qbittorrent.Torrent{
		{Hash: "t1", Name: "Movie1.mkv", Status: qbittorrent.StatusCompleted, AddedOn: addedOn},
	}
	mockService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads?page=999", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	items, ok := dataMap["items"].([]interface{})
	require.True(t, ok)

	// Page beyond total returns empty items but valid metadata
	assert.Empty(t, items)
	assert.Equal(t, float64(999), dataMap["page"])
	assert.Equal(t, float64(1), dataMap["total_items"])
	assert.Equal(t, float64(1), dataMap["total_pages"])

	mockService.AssertExpectations(t)
}

// --- Parse Status: Seeding Status Enrichment ---

func TestDownloadHandler_ListDownloads_WithParseStatus_SeedingAlsoEnriched(t *testing.T) {
	mockDLService := new(MockDownloadService)
	mockPQService := new(MockParseQueueService)

	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	torrents := []qbittorrent.Torrent{
		{
			Hash:    "seeding-hash",
			Name:    "Movie.mkv",
			Status:  qbittorrent.StatusSeeding,
			AddedOn: addedOn,
		},
	}
	mockDLService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)

	mediaID := "media-456"
	mockPQService.On("GetJobStatus", mock.Anything, "seeding-hash").Return(&models.ParseJob{
		ID:          "job-3",
		TorrentHash: "seeding-hash",
		Status:      models.ParseJobCompleted,
		MediaID:     &mediaID,
	}, nil)

	handler := NewDownloadHandler(mockDLService, mockPQService)
	router := setupDownloadRouterWithParseQueue(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "response.Data should be a PaginatedResponse map")
	dataSlice, ok := dataMap["items"].([]interface{})
	require.True(t, ok, "items should be a slice")
	require.Len(t, dataSlice, 1)

	// Seeding torrent should also get parseStatus enrichment
	item, ok := dataSlice[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "seeding-hash", item["hash"])
	parseStatus, ok := item["parse_status"].(map[string]interface{})
	require.True(t, ok, "seeding torrent should have parse_status")
	assert.Equal(t, "completed", parseStatus["status"])
	assert.Equal(t, "media-456", parseStatus["media_id"])

	mockDLService.AssertExpectations(t)
	mockPQService.AssertExpectations(t)
}

// --- Parse Status: GetJobStatus Error Path ---

func TestDownloadHandler_ListDownloads_WithParseStatus_GetJobStatusError(t *testing.T) {
	mockDLService := new(MockDownloadService)
	mockPQService := new(MockParseQueueService)

	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	torrents := []qbittorrent.Torrent{
		{
			Hash:    "error-hash",
			Name:    "Movie.mkv",
			Status:  qbittorrent.StatusCompleted,
			AddedOn: addedOn,
		},
	}
	mockDLService.On("GetAllDownloads", mock.Anything, "all", "added_on", "desc").Return(torrents, nil)

	// GetJobStatus returns an error — handler should gracefully skip enrichment
	mockPQService.On("GetJobStatus", mock.Anything, "error-hash").Return(nil, errors.New("database connection failed"))

	handler := NewDownloadHandler(mockDLService, mockPQService)
	router := setupDownloadRouterWithParseQueue(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "response.Data should be a PaginatedResponse map")
	dataSlice, ok := dataMap["items"].([]interface{})
	require.True(t, ok, "items should be a slice")
	require.Len(t, dataSlice, 1)

	// GetJobStatus error → parse_status should be nil (graceful degradation)
	item, ok := dataSlice[0].(map[string]interface{})
	require.True(t, ok)
	assert.Nil(t, item["parse_status"], "parse_status should be nil when GetJobStatus errors")

	mockDLService.AssertExpectations(t)
	mockPQService.AssertExpectations(t)
}
