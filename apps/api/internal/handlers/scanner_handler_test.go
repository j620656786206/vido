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
	"github.com/vido/api/internal/services"
)

// MockScannerService implements ScannerServiceInterface for testing
type MockScannerService struct {
	mock.Mock
}

func (m *MockScannerService) IsScanActive() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockScannerService) StartScan(ctx context.Context) (*services.ScanResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ScanResult), args.Error(1)
}

func (m *MockScannerService) CancelScan() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockScannerService) GetProgress() services.ScanProgress {
	args := m.Called()
	return args.Get(0).(services.ScanProgress)
}

func setupScannerRouter(svc ScannerServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewScannerHandler(svc)
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestScannerHandler_TriggerScan_Success(t *testing.T) {
	mockSvc := new(MockScannerService)
	// Handler no longer calls IsScanActive — StartScan's mutex is the gate
	mockSvc.On("StartScan", mock.Anything).Return(&services.ScanResult{
		FilesFound:   10,
		FilesCreated: 10,
		Duration:     "1s",
	}, nil)

	router := setupScannerRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/scanner/scan", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body.Success)

	dataMap, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Scan started", dataMap["message"])

	// Give goroutine time to call StartScan
	time.Sleep(50 * time.Millisecond)
	mockSvc.AssertCalled(t, "StartScan", mock.Anything)
}

func TestScannerHandler_GetStatus_NoScan(t *testing.T) {
	mockSvc := new(MockScannerService)
	mockSvc.On("GetProgress").Return(services.ScanProgress{
		IsActive:   false,
		FilesFound: 0,
	})

	router := setupScannerRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/scanner/status", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body.Success)

	dataMap, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, dataMap["isActive"])
}

func TestScannerHandler_GetStatus_ActiveScan(t *testing.T) {
	mockSvc := new(MockScannerService)
	mockSvc.On("GetProgress").Return(services.ScanProgress{
		IsActive:    true,
		FilesFound:  42,
		CurrentFile: "/media/movies/test.mkv",
		PercentDone: 50,
		ErrorCount:  1,
	})

	router := setupScannerRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/scanner/status", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body.Success)

	dataMap, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, dataMap["isActive"])
	assert.Equal(t, float64(42), dataMap["filesFound"])
	assert.Equal(t, "/media/movies/test.mkv", dataMap["currentFile"])
	assert.Equal(t, float64(50), dataMap["percentDone"])
}

func TestScannerHandler_CancelScan_Success(t *testing.T) {
	mockSvc := new(MockScannerService)
	mockSvc.On("CancelScan").Return(nil)

	router := setupScannerRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/scanner/cancel", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body.Success)

	dataMap, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Scan cancelled", dataMap["message"])
}

func TestScannerHandler_CancelScan_NoActiveScan(t *testing.T) {
	mockSvc := new(MockScannerService)
	mockSvc.On("CancelScan").Return(services.ErrScanNotActive)

	router := setupScannerRouter(mockSvc)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/scanner/cancel", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	var body APIResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.False(t, body.Success)
	assert.NotNil(t, body.Error)
	assert.Equal(t, "SCANNER_NOT_ACTIVE", body.Error.Code)
}
