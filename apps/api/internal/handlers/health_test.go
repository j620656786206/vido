package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/health"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", HealthCheck)

	t.Run("returns healthy status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "healthy")
		assert.Contains(t, w.Body.String(), "vido-api")
	})
}

// MockHealthChecker for testing
type MockHealthChecker struct{}

func (m *MockHealthChecker) CheckTMDb(ctx context.Context) error        { return nil }
func (m *MockHealthChecker) CheckDouban(ctx context.Context) error      { return nil }
func (m *MockHealthChecker) CheckWikipedia(ctx context.Context) error   { return nil }
func (m *MockHealthChecker) CheckAI(ctx context.Context) error          { return nil }
func (m *MockHealthChecker) CheckQBittorrent(ctx context.Context) error { return nil }

func TestServiceHealthHandler_GetServicesHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns healthy status when all services are up", func(t *testing.T) {
		checker := &MockHealthChecker{}
		monitor := health.NewHealthMonitor(checker)
		degradationService := services.NewDegradationService(monitor)
		handler := NewServiceHealthHandler(degradationService)

		router := gin.New()
		router.GET("/api/v1/health/services", handler.GetServicesHealth)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/services", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "success")
		assert.Contains(t, w.Body.String(), "normal")
	})

	t.Run("returns 503 when service not configured", func(t *testing.T) {
		handler := NewServiceHealthHandler(nil)

		router := gin.New()
		router.GET("/api/v1/health/services", handler.GetServicesHealth)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/services", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "HEALTH_NOT_CONFIGURED")
	})
}

// MockConnectionHistoryService for testing
type MockConnectionHistoryService struct {
	events []models.ConnectionEvent
	err    error
}

func (m *MockConnectionHistoryService) RecordEvent(_ context.Context, _ string, _ models.ConnectionEventType, _ string, _ string) error {
	return m.err
}

func (m *MockConnectionHistoryService) GetHistory(_ context.Context, _ string, _ int) ([]models.ConnectionEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func TestServiceHealthHandler_GetConnectionHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns history events", func(t *testing.T) {
		checker := &MockHealthChecker{}
		monitor := health.NewHealthMonitor(checker)
		degradationService := services.NewDegradationService(monitor)
		handler := NewServiceHealthHandler(degradationService)

		mockSvc := &MockConnectionHistoryService{
			events: []models.ConnectionEvent{
				{ID: "evt-1", Service: "qbittorrent", EventType: models.EventDisconnected, Status: models.ServiceStatusDown, Message: "timeout", CreatedAt: time.Now()},
				{ID: "evt-2", Service: "qbittorrent", EventType: models.EventConnected, Status: models.ServiceStatusHealthy, CreatedAt: time.Now()},
			},
		}
		handler.SetHistoryService(mockSvc)

		router := gin.New()
		router.GET("/api/v1/health/services/:service/history", handler.GetConnectionHistory)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/services/qbittorrent/history", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "success")
		assert.Contains(t, w.Body.String(), "evt-1")
		assert.Contains(t, w.Body.String(), "evt-2")
	})

	t.Run("returns empty array when no history", func(t *testing.T) {
		checker := &MockHealthChecker{}
		monitor := health.NewHealthMonitor(checker)
		degradationService := services.NewDegradationService(monitor)
		handler := NewServiceHealthHandler(degradationService)

		mockSvc := &MockConnectionHistoryService{events: nil}
		handler.SetHistoryService(mockSvc)

		router := gin.New()
		router.GET("/api/v1/health/services/:service/history", handler.GetConnectionHistory)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/services/qbittorrent/history", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"data":[]`)
	})

	t.Run("returns 503 when history repo not configured", func(t *testing.T) {
		checker := &MockHealthChecker{}
		monitor := health.NewHealthMonitor(checker)
		degradationService := services.NewDegradationService(monitor)
		handler := NewServiceHealthHandler(degradationService)

		router := gin.New()
		router.GET("/api/v1/health/services/:service/history", handler.GetConnectionHistory)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/services/qbittorrent/history", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "CONNECTION_HISTORY_ERROR")
	})

	t.Run("returns 500 when repo errors", func(t *testing.T) {
		checker := &MockHealthChecker{}
		monitor := health.NewHealthMonitor(checker)
		degradationService := services.NewDegradationService(monitor)
		handler := NewServiceHealthHandler(degradationService)

		mockSvc := &MockConnectionHistoryService{err: fmt.Errorf("db error")}
		handler.SetHistoryService(mockSvc)

		router := gin.New()
		router.GET("/api/v1/health/services/:service/history", handler.GetConnectionHistory)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/services/qbittorrent/history", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "CONNECTION_HISTORY_ERROR")
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		checker := &MockHealthChecker{}
		monitor := health.NewHealthMonitor(checker)
		degradationService := services.NewDegradationService(monitor)
		handler := NewServiceHealthHandler(degradationService)

		mockSvc := &MockConnectionHistoryService{events: []models.ConnectionEvent{}}
		handler.SetHistoryService(mockSvc)

		router := gin.New()
		router.GET("/api/v1/health/services/:service/history", handler.GetConnectionHistory)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/services/qbittorrent/history?limit=5", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 400 for invalid service name", func(t *testing.T) {
		checker := &MockHealthChecker{}
		monitor := health.NewHealthMonitor(checker)
		degradationService := services.NewDegradationService(monitor)
		handler := NewServiceHealthHandler(degradationService)

		mockSvc := &MockConnectionHistoryService{events: []models.ConnectionEvent{}}
		handler.SetHistoryService(mockSvc)

		router := gin.New()
		router.GET("/api/v1/health/services/:service/history", handler.GetConnectionHistory)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/services/unknown-service/history", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_INVALID_FORMAT")
	})
}
