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

// MockServiceStatusService implements ServiceStatusServiceInterface for testing
type MockServiceStatusService struct {
	mock.Mock
}

func (m *MockServiceStatusService) GetAllStatuses(ctx context.Context) ([]models.ServiceStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ServiceStatus), args.Error(1)
}

func (m *MockServiceStatusService) TestService(ctx context.Context, serviceName string) (*models.ServiceStatus, error) {
	args := m.Called(ctx, serviceName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServiceStatus), args.Error(1)
}

func setupStatusRouter(svc services.ServiceStatusServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewStatusHandler(svc)
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestStatusHandler_GetAllServiceStatuses_Success(t *testing.T) {
	mockSvc := new(MockServiceStatusService)
	now := time.Now()
	expected := []models.ServiceStatus{
		{
			Name:           "tmdb",
			DisplayName:    "TMDb API",
			Status:         models.StatusConnected,
			Message:        "已連線",
			LastSuccessAt:  &now,
			LastCheckAt:    now,
			ResponseTimeMs: 45,
		},
		{
			Name:           "ai",
			DisplayName:    "AI 服務",
			Status:         models.StatusUnconfigured,
			Message:        "未設定",
			LastCheckAt:    now,
			ResponseTimeMs: 0,
		},
	}
	mockSvc.On("GetAllStatuses", mock.Anything).Return(expected, nil)

	router := setupStatusRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Parse the data to verify services array
	dataBytes, _ := json.Marshal(resp.Data)
	var data struct {
		Services []models.ServiceStatus `json:"services"`
	}
	err = json.Unmarshal(dataBytes, &data)
	require.NoError(t, err)
	assert.Len(t, data.Services, 2)
	assert.Equal(t, "tmdb", data.Services[0].Name)
	assert.Equal(t, models.StatusConnected, data.Services[0].Status)
}

func TestStatusHandler_GetAllServiceStatuses_Error(t *testing.T) {
	mockSvc := new(MockServiceStatusService)
	mockSvc.On("GetAllStatuses", mock.Anything).Return(nil, errors.New("internal error"))

	router := setupStatusRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestStatusHandler_TestServiceConnection_Success(t *testing.T) {
	mockSvc := new(MockServiceStatusService)
	now := time.Now()
	expected := &models.ServiceStatus{
		Name:           "tmdb",
		DisplayName:    "TMDb API",
		Status:         models.StatusConnected,
		Message:        "已連線",
		LastSuccessAt:  &now,
		LastCheckAt:    now,
		ResponseTimeMs: 123,
	}
	mockSvc.On("TestService", mock.Anything, "tmdb").Return(expected, nil)

	router := setupStatusRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/services/tmdb/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestStatusHandler_TestServiceConnection_NotFound(t *testing.T) {
	mockSvc := new(MockServiceStatusService)
	mockSvc.On("TestService", mock.Anything, "unknown").Return(nil, services.ErrServiceNotFound)

	router := setupStatusRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/services/unknown/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "SERVICE_NOT_FOUND", resp.Error.Code)
}

func TestStatusHandler_TestServiceConnection_InternalError(t *testing.T) {
	mockSvc := new(MockServiceStatusService)
	mockSvc.On("TestService", mock.Anything, "tmdb").Return(nil, errors.New("check failed"))

	router := setupStatusRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/services/tmdb/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStatusHandler_RegisterRoutes(t *testing.T) {
	mockSvc := new(MockServiceStatusService)
	mockSvc.On("GetAllStatuses", mock.Anything).Return([]models.ServiceStatus{}, nil)

	router := setupStatusRouter(mockSvc)

	// Test GET endpoint exists
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
