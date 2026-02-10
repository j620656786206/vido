package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/qbittorrent"
)

// MockDownloadService mocks DownloadServiceInterface for handler tests.
type MockDownloadService struct {
	mock.Mock
}

func (m *MockDownloadService) GetAllDownloads(ctx context.Context, sort string, order string) ([]qbittorrent.Torrent, error) {
	args := m.Called(ctx, sort, order)
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
	mockService.On("GetAllDownloads", mock.Anything, "added_on", "desc").Return(torrents, nil)

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
	mockService.On("GetAllDownloads", mock.Anything, "name", "asc").Return([]qbittorrent.Torrent{}, nil)

	handler := NewDownloadHandler(mockService)
	router := setupDownloadRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads?sort=name&order=asc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestDownloadHandler_ListDownloads_NotConfigured(t *testing.T) {
	mockService := new(MockDownloadService)
	mockService.On("GetAllDownloads", mock.Anything, "added_on", "desc").Return(
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
	mockService.On("GetAllDownloads", mock.Anything, "added_on", "desc").Return(
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
