package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/services"
)

type MockExportService struct {
	mock.Mock
}

func (m *MockExportService) ExportJSON(ctx context.Context) (*services.ExportResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ExportResult), args.Error(1)
}
func (m *MockExportService) ExportYAML(ctx context.Context) (*services.ExportResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ExportResult), args.Error(1)
}
func (m *MockExportService) ExportNFO(ctx context.Context) (*services.ExportResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ExportResult), args.Error(1)
}
func (m *MockExportService) GetExportStatus(ctx context.Context) (*services.ExportResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ExportResult), args.Error(1)
}
func (m *MockExportService) GetExportFilePath(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func setupExportRouter(svc services.ExportServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewExportHandler(svc)
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestExportHandler_TriggerExport(t *testing.T) {
	t.Run("JSON export success", func(t *testing.T) {
		mockSvc := new(MockExportService)
		result := &services.ExportResult{
			ExportID:  "e1",
			Format:    services.ExportFormatJSON,
			Status:    services.ExportStatusCompleted,
			ItemCount: 5,
		}
		mockSvc.On("ExportJSON", mock.Anything).Return(result, nil)

		router := setupExportRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/export", strings.NewReader(`{"format":"json"}`))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body APIResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.True(t, body.Success)
	})

	t.Run("invalid format", func(t *testing.T) {
		mockSvc := new(MockExportService)
		router := setupExportRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/export", strings.NewReader(`{"format":"csv"}`))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("missing format", func(t *testing.T) {
		mockSvc := new(MockExportService)
		router := setupExportRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/export", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestExportHandler_GetExportStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockExportService)
		result := &services.ExportResult{Status: services.ExportStatusCompleted, Message: "done"}
		mockSvc.On("GetExportStatus", mock.Anything).Return(result, nil)

		router := setupExportRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/export/status", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}
