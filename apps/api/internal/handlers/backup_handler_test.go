package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// MockBackupService implements BackupServiceInterface for handler tests
type MockBackupService struct {
	mock.Mock
}

func (m *MockBackupService) CreateBackup(ctx context.Context) (*models.Backup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Backup), args.Error(1)
}

func (m *MockBackupService) ListBackups(ctx context.Context) (*models.BackupListResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BackupListResponse), args.Error(1)
}

func (m *MockBackupService) GetBackup(ctx context.Context, id string) (*models.Backup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Backup), args.Error(1)
}

func (m *MockBackupService) DeleteBackup(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBackupService) GetBackupFilePath(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *MockBackupService) VerifyBackup(ctx context.Context, id string) (*models.VerificationResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VerificationResult), args.Error(1)
}

func (m *MockBackupService) RestoreBackup(ctx context.Context, id string) (*models.RestoreResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RestoreResult), args.Error(1)
}

func (m *MockBackupService) GetRestoreStatus(ctx context.Context) (*models.RestoreResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RestoreResult), args.Error(1)
}

// MockScheduler implements BackupSchedulerInterface
type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) Start(ctx context.Context) {
	m.Called(ctx)
}
func (m *MockScheduler) Stop() {
	m.Called()
}
func (m *MockScheduler) SetSchedule(ctx context.Context, schedule services.BackupSchedule) error {
	return m.Called(ctx, schedule).Error(0)
}
func (m *MockScheduler) GetSchedule(ctx context.Context) (*services.BackupScheduleResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.BackupScheduleResponse), args.Error(1)
}

func setupBackupRouter(svc services.BackupServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewBackupHandler(svc)
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func setupBackupRouterWithScheduler(svc services.BackupServiceInterface, scheduler services.BackupSchedulerInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewBackupHandler(svc)
	handler.SetScheduler(scheduler)
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestBackupHandler_RestoreBackup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		result := &models.RestoreResult{
			RestoreID:  "r1",
			Status:     models.RestoreStatusCompleted,
			SnapshotID: "snap1",
			Message:    "還原完成",
		}
		mockSvc.On("RestoreBackup", mock.Anything, "b1").Return(result, nil)

		router := setupBackupRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/backups/b1/restore", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var body APIResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.True(t, body.Success)
		mockSvc.AssertExpectations(t)
	})

	t.Run("backup not found", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		mockSvc.On("RestoreBackup", mock.Anything, "nonexistent").Return(nil, services.ErrBackupNotFound)

		router := setupBackupRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/backups/nonexistent/restore", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)

		var body APIResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.False(t, body.Success)
		assert.Equal(t, "BACKUP_NOT_FOUND", body.Error.Code)
	})

	t.Run("restore in progress", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		mockSvc.On("RestoreBackup", mock.Anything, "b1").Return(nil, services.ErrRestoreInProgress)

		router := setupBackupRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/backups/b1/restore", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)

		var body APIResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.False(t, body.Success)
		assert.Equal(t, "RESTORE_IN_PROGRESS", body.Error.Code)
	})
}

func TestBackupHandler_RestoreBackup_InternalError(t *testing.T) {
	t.Run("generic error returns 500", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		mockSvc.On("RestoreBackup", mock.Anything, "b1").Return(nil, assert.AnError)

		router := setupBackupRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/backups/b1/restore", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)

		var body APIResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.False(t, body.Success)
		assert.Equal(t, "RESTORE_FAILED", body.Error.Code)
	})
}

func TestBackupHandler_RestoreBackup_ResponseFields(t *testing.T) {
	t.Run("response contains all restore result fields", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		result := &models.RestoreResult{
			RestoreID:  "r-uuid",
			Status:     models.RestoreStatusCompleted,
			SnapshotID: "snap-uuid",
			Message:    "還原完成，資料庫已恢復",
		}
		mockSvc.On("RestoreBackup", mock.Anything, "b1").Return(result, nil)

		router := setupBackupRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/backups/b1/restore", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var body struct {
			Success bool `json:"success"`
			Data    struct {
				RestoreID  string `json:"restoreId"`
				Status     string `json:"status"`
				SnapshotID string `json:"snapshotId"`
				Message    string `json:"message"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.True(t, body.Success)
		assert.Equal(t, "r-uuid", body.Data.RestoreID)
		assert.Equal(t, "completed", body.Data.Status)
		assert.Equal(t, "snap-uuid", body.Data.SnapshotID)
		assert.Equal(t, "還原完成，資料庫已恢復", body.Data.Message)
	})
}

func TestBackupHandler_GetRestoreStatus_Error(t *testing.T) {
	t.Run("internal error returns 500", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		mockSvc.On("GetRestoreStatus", mock.Anything).Return(nil, assert.AnError)

		router := setupBackupRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/restore/status", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
}

func TestBackupHandler_GetRestoreStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		result := &models.RestoreResult{
			Status:  models.RestoreStatusCompleted,
			Message: "沒有進行中的還原作業",
		}
		mockSvc.On("GetRestoreStatus", mock.Anything).Return(result, nil)

		router := setupBackupRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/restore/status", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var body APIResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.True(t, body.Success)
		mockSvc.AssertExpectations(t)
	})
}

func TestBackupHandler_GetSchedule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		mockScheduler := new(MockScheduler)
		resp := &services.BackupScheduleResponse{
			BackupSchedule: services.BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3},
		}
		mockScheduler.On("GetSchedule", mock.Anything).Return(resp, nil)

		router := setupBackupRouterWithScheduler(mockSvc, mockScheduler)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/backups/schedule", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body APIResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.True(t, body.Success)
	})
}

func TestBackupHandler_UpdateSchedule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		mockScheduler := new(MockScheduler)
		mockScheduler.On("SetSchedule", mock.Anything, mock.AnythingOfType("services.BackupSchedule")).Return(nil)
		resp := &services.BackupScheduleResponse{
			BackupSchedule: services.BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3},
		}
		mockScheduler.On("GetSchedule", mock.Anything).Return(resp, nil)

		router := setupBackupRouterWithScheduler(mockSvc, mockScheduler)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/settings/backups/schedule", strings.NewReader(`{"enabled":true,"frequency":"daily","hour":3}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid schedule", func(t *testing.T) {
		mockSvc := new(MockBackupService)
		mockScheduler := new(MockScheduler)
		mockScheduler.On("SetSchedule", mock.Anything, mock.AnythingOfType("services.BackupSchedule")).Return(fmt.Errorf("SCHEDULE_INVALID: frequency must be daily, weekly, or disabled"))

		router := setupBackupRouterWithScheduler(mockSvc, mockScheduler)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/settings/backups/schedule", strings.NewReader(`{"enabled":true,"frequency":"hourly","hour":3}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
