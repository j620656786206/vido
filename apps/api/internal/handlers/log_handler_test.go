package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// MockLogService
type MockLogService struct {
	mock.Mock
}

func (m *MockLogService) GetLogs(ctx context.Context, filter models.LogFilter) (*services.LogsResponse, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LogsResponse), args.Error(1)
}

func (m *MockLogService) ClearLogs(ctx context.Context, days int) (*services.LogClearResult, error) {
	args := m.Called(ctx, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LogClearResult), args.Error(1)
}

func setupLogRouter(svc services.LogServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewLogHandler(svc)
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestLogHandler_GetLogs_Success(t *testing.T) {
	mockSvc := new(MockLogService)

	expected := &services.LogsResponse{
		Logs: []models.SystemLog{
			{ID: 1, Level: models.LogLevelError, Message: "Test error", CreatedAt: time.Now()},
		},
		Total:   1,
		Page:    1,
		PerPage: 50,
	}

	mockSvc.On("GetLogs", mock.Anything, models.LogFilter{}).Return(expected, nil)

	router := setupLogRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/logs", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body.Success)
	mockSvc.AssertExpectations(t)
}

func TestLogHandler_GetLogs_WithFilters(t *testing.T) {
	mockSvc := new(MockLogService)

	expectedFilter := models.LogFilter{
		Level:   models.LogLevelError,
		Keyword: "tmdb",
		Page:    2,
		PerPage: 25,
	}

	mockSvc.On("GetLogs", mock.Anything, expectedFilter).Return(&services.LogsResponse{
		Logs:    []models.SystemLog{},
		Total:   0,
		Page:    2,
		PerPage: 25,
	}, nil)

	router := setupLogRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/logs?level=ERROR&keyword=tmdb&page=2&per_page=25", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestLogHandler_GetLogs_InvalidLevel(t *testing.T) {
	mockSvc := new(MockLogService)

	router := setupLogRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/logs?level=INVALID", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.False(t, body.Success)
	assert.Equal(t, "VALIDATION_INVALID_FORMAT", body.Error.Code)
}

func TestLogHandler_GetLogs_ServiceError(t *testing.T) {
	mockSvc := new(MockLogService)
	mockSvc.On("GetLogs", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	router := setupLogRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/logs", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestLogHandler_ClearLogs_Success(t *testing.T) {
	mockSvc := new(MockLogService)

	mockSvc.On("ClearLogs", mock.Anything, 30).Return(&services.LogClearResult{
		EntriesRemoved: 42,
		Days:           30,
	}, nil)

	router := setupLogRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/logs?older_than_days=30", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body.Success)
	mockSvc.AssertExpectations(t)
}

func TestLogHandler_ClearLogs_MissingDays(t *testing.T) {
	mockSvc := new(MockLogService)

	router := setupLogRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/logs", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "VALIDATION_REQUIRED_FIELD", body.Error.Code)
}

func TestLogHandler_ClearLogs_InvalidDays(t *testing.T) {
	mockSvc := new(MockLogService)

	tests := []struct {
		name string
		days string
	}{
		{"zero", "0"},
		{"negative", "-5"},
		{"non-numeric", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupLogRouter(mockSvc)
			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/logs?older_than_days="+tt.days, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	}
}

func TestLogHandler_ClearLogs_ServiceError(t *testing.T) {
	mockSvc := new(MockLogService)
	mockSvc.On("ClearLogs", mock.Anything, 7).Return(nil, errors.New("db error"))

	router := setupLogRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/logs?older_than_days=7", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}
