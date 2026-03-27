package handlers

import (
	"context"
	"encoding/json"
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
	"github.com/vido/api/internal/services"
)

// MockParseQueueService mocks ParseQueueServiceInterface for handler tests.
type MockParseQueueService struct {
	mock.Mock
}

func (m *MockParseQueueService) QueueParseJob(ctx context.Context, torrent *qbittorrent.Torrent) (*models.ParseJob, error) {
	args := m.Called(ctx, torrent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ParseJob), args.Error(1)
}

func (m *MockParseQueueService) ProcessNextJob(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockParseQueueService) GetJobStatus(ctx context.Context, torrentHash string) (*models.ParseJob, error) {
	args := m.Called(ctx, torrentHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ParseJob), args.Error(1)
}

func (m *MockParseQueueService) RetryJob(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockParseQueueService) ListJobs(ctx context.Context, limit int) ([]*models.ParseJob, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ParseJob), args.Error(1)
}

func setupParseJobRouter(handler *ParseJobHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

// --- GetParseStatus Tests ---

func TestParseJobHandler_GetParseStatus_Success(t *testing.T) {
	mockService := new(MockParseQueueService)
	now := time.Now()
	job := &models.ParseJob{
		ID:          "job-1",
		TorrentHash: "abc123",
		FileName:    "[SubGroup] Movie (2024).mkv",
		FilePath:    "/downloads",
		Status:      models.ParseJobCompleted,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	mockService.On("GetJobStatus", mock.Anything, "abc123").Return(job, nil)

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/abc123/parse-status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "abc123", dataMap["torrent_hash"])
	assert.Equal(t, "completed", dataMap["status"])
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_GetParseStatus_NotFound(t *testing.T) {
	mockService := new(MockParseQueueService)
	mockService.On("GetJobStatus", mock.Anything, "nonexistent").Return(nil, nil)

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/nonexistent/parse-status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_GetParseStatus_ServiceError(t *testing.T) {
	mockService := new(MockParseQueueService)
	mockService.On("GetJobStatus", mock.Anything, "abc123").Return(nil, fmt.Errorf("database error"))

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/downloads/abc123/parse-status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_GetParseStatus_EmptyHash(t *testing.T) {
	mockService := new(MockParseQueueService)

	handler := NewParseJobHandler(mockService)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/downloads//parse-status", nil)
	c.Params = gin.Params{{Key: "hash", Value: ""}}

	handler.GetParseStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	mockService.AssertNotCalled(t, "GetJobStatus")
}

// --- ListJobs Tests ---

func TestParseJobHandler_ListJobs_Success(t *testing.T) {
	mockService := new(MockParseQueueService)
	now := time.Now()
	jobs := []*models.ParseJob{
		{
			ID:          "job-1",
			TorrentHash: "hash1",
			FileName:    "Movie1.mkv",
			Status:      models.ParseJobCompleted,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "job-2",
			TorrentHash: "hash2",
			FileName:    "Movie2.mkv",
			Status:      models.ParseJobPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	mockService.On("ListJobs", mock.Anything, 50).Return(jobs, nil)

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/parse-jobs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)

	dataSlice, ok := response.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, dataSlice, 2)
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_ListJobs_WithLimitParam(t *testing.T) {
	mockService := new(MockParseQueueService)
	mockService.On("ListJobs", mock.Anything, 10).Return([]*models.ParseJob{}, nil)

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/parse-jobs?limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_ListJobs_InvalidLimit(t *testing.T) {
	mockService := new(MockParseQueueService)
	// Invalid limit should default to 50
	mockService.On("ListJobs", mock.Anything, 50).Return([]*models.ParseJob{}, nil)

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/parse-jobs?limit=abc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_ListJobs_Empty(t *testing.T) {
	mockService := new(MockParseQueueService)
	mockService.On("ListJobs", mock.Anything, 50).Return([]*models.ParseJob{}, nil)

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/parse-jobs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_ListJobs_ServiceError(t *testing.T) {
	mockService := new(MockParseQueueService)
	mockService.On("ListJobs", mock.Anything, 50).Return(nil, fmt.Errorf("database error"))

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/parse-jobs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	mockService.AssertExpectations(t)
}

// --- RetryParseJob Tests ---

func TestParseJobHandler_RetryParseJob_Success(t *testing.T) {
	mockService := new(MockParseQueueService)
	mockService.On("RetryJob", mock.Anything, "job-1").Return(nil)

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/parse-jobs/job-1/retry", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_RetryParseJob_NotFailed(t *testing.T) {
	mockService := new(MockParseQueueService)
	mockService.On("RetryJob", mock.Anything, "job-1").Return(fmt.Errorf("%w: current status: processing", services.ErrJobNotRetryable))

	handler := NewParseJobHandler(mockService)
	router := setupParseJobRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/parse-jobs/job-1/retry", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	mockService.AssertExpectations(t)
}

func TestParseJobHandler_RetryParseJob_EmptyID(t *testing.T) {
	mockService := new(MockParseQueueService)

	handler := NewParseJobHandler(mockService)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/parse-jobs//retry", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.RetryParseJob(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "RetryJob")
}
