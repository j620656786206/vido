package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func setupBackupRouter(svc services.BackupServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewBackupHandler(svc)
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
