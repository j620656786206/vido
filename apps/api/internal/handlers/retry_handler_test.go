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
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/retry"
)

// MockRetryService is a mock implementation of RetryServiceInterface
type MockRetryService struct {
	items           []*retry.RetryItem
	stats           *retry.RetryStats
	triggerErr      error
	cancelErr       error
	getPendingErr   error
	getByIDErr      error
	getByIDNotFound bool
}

func NewMockRetryService() *MockRetryService {
	return &MockRetryService{
		items: []*retry.RetryItem{},
		stats: &retry.RetryStats{
			TotalPending:   0,
			TotalSucceeded: 0,
			TotalFailed:    0,
		},
	}
}

func (m *MockRetryService) QueueRetry(ctx context.Context, taskID, taskType string, payload interface{}, err error) error {
	return nil
}

func (m *MockRetryService) CancelRetry(ctx context.Context, itemID string) error {
	return m.cancelErr
}

func (m *MockRetryService) TriggerImmediate(ctx context.Context, itemID string) error {
	return m.triggerErr
}

func (m *MockRetryService) GetPendingRetries(ctx context.Context) ([]*retry.RetryItem, error) {
	return m.items, m.getPendingErr
}

func (m *MockRetryService) GetRetryStats(ctx context.Context) (*retry.RetryStats, error) {
	return m.stats, nil
}

func (m *MockRetryService) IsRetryableError(err error) bool {
	return true
}

func (m *MockRetryService) StartScheduler(ctx context.Context) error {
	return nil
}

func (m *MockRetryService) StopScheduler() {}

func (m *MockRetryService) GetByID(ctx context.Context, id string) (*retry.RetryItem, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if m.getByIDNotFound {
		return nil, nil
	}
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, nil
}

func setupRetryTestRouter(service *MockRetryService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewRetryHandler(service)
	api := router.Group("/api/v1")
	RegisterRetryRoutes(api, handler)
	return router
}

func createMockRetryItem(id, taskID string) *retry.RetryItem {
	return &retry.RetryItem{
		ID:            id,
		TaskID:        taskID,
		TaskType:      retry.TaskTypeParse,
		AttemptCount:  1,
		MaxAttempts:   4,
		LastError:     "TMDb timeout",
		NextAttemptAt: time.Now().Add(2 * time.Second),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestRetryHandler_GetPending_Success(t *testing.T) {
	service := NewMockRetryService()
	service.items = []*retry.RetryItem{
		createMockRetryItem("retry-1", "task-1"),
		createMockRetryItem("retry-2", "task-2"),
	}
	service.stats = &retry.RetryStats{
		TotalPending:   2,
		TotalSucceeded: 5,
		TotalFailed:    1,
	}

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/retry/pending", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}

func TestRetryHandler_GetPending_Error(t *testing.T) {
	service := NewMockRetryService()
	service.getPendingErr = errors.New("database error")

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/retry/pending", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestRetryHandler_GetPending_Empty(t *testing.T) {
	service := NewMockRetryService()
	service.items = []*retry.RetryItem{}

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/retry/pending", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestRetryHandler_TriggerImmediate_Success(t *testing.T) {
	service := NewMockRetryService()
	service.items = []*retry.RetryItem{
		createMockRetryItem("retry-1", "task-1"),
	}

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/retry/retry-1/trigger", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestRetryHandler_TriggerImmediate_NotFound(t *testing.T) {
	service := NewMockRetryService()
	service.triggerErr = errors.New("retry item not found")

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/retry/nonexistent/trigger", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "RETRY_NOT_FOUND", resp.Error.Code)
}

func TestRetryHandler_TriggerImmediate_Error(t *testing.T) {
	service := NewMockRetryService()
	service.triggerErr = errors.New("database error")

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/retry/retry-1/trigger", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRetryHandler_Cancel_Success(t *testing.T) {
	service := NewMockRetryService()
	service.items = []*retry.RetryItem{
		createMockRetryItem("retry-1", "task-1"),
	}

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/retry/retry-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRetryHandler_Cancel_NotFound(t *testing.T) {
	service := NewMockRetryService()
	service.cancelErr = errors.New("retry item not found")

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/retry/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "RETRY_NOT_FOUND", resp.Error.Code)
}

func TestRetryHandler_Cancel_Error(t *testing.T) {
	service := NewMockRetryService()
	service.cancelErr = errors.New("database error")

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/retry/retry-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRetryHandler_GetByID_Success(t *testing.T) {
	service := NewMockRetryService()
	service.items = []*retry.RetryItem{
		createMockRetryItem("retry-1", "task-1"),
	}

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/retry/retry-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestRetryHandler_GetByID_NotFound(t *testing.T) {
	service := NewMockRetryService()
	service.getByIDNotFound = true

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/retry/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "RETRY_NOT_FOUND", resp.Error.Code)
}

func TestRetryHandler_GetByID_Error(t *testing.T) {
	service := NewMockRetryService()
	service.getByIDErr = errors.New("database error")

	router := setupRetryTestRouter(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/retry/retry-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"not found", "not found", true},
		{"item not found", "not found", true},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		result := containsString(tt.s, tt.substr)
		assert.Equal(t, tt.expected, result, "containsString(%q, %q)", tt.s, tt.substr)
	}
}
