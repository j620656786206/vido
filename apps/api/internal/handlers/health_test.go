package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/health"
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

func (m *MockHealthChecker) CheckTMDb(ctx context.Context) error      { return nil }
func (m *MockHealthChecker) CheckDouban(ctx context.Context) error    { return nil }
func (m *MockHealthChecker) CheckWikipedia(ctx context.Context) error { return nil }
func (m *MockHealthChecker) CheckAI(ctx context.Context) error        { return nil }

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
