package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/services"
)

// MockCacheStatsService
type MockCacheStatsService struct {
	mock.Mock
}

func (m *MockCacheStatsService) GetCacheStats(ctx context.Context) (*services.CacheStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.CacheStats), args.Error(1)
}

// MockCacheCleanupService
type MockCacheCleanupService struct {
	mock.Mock
}

func (m *MockCacheCleanupService) ClearCacheByAge(ctx context.Context, days int) (*services.CleanupResult, error) {
	args := m.Called(ctx, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.CleanupResult), args.Error(1)
}

func (m *MockCacheCleanupService) ClearCacheByType(ctx context.Context, cacheType string) (*services.CleanupResult, error) {
	args := m.Called(ctx, cacheType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.CleanupResult), args.Error(1)
}

func setupCacheRouter(stats CacheStatsServiceInterface, cleanup CacheCleanupServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewCacheHandler(stats, cleanup)
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestCacheHandler_GetCacheStats_Success(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	expected := &services.CacheStats{
		CacheTypes: []services.CacheTypeInfo{
			{Type: "image", Label: "圖片快取", SizeBytes: 1024, EntryCount: 10},
			{Type: "ai", Label: "AI 解析快取", SizeBytes: 512, EntryCount: 5},
		},
		TotalSizeBytes: 1536,
	}
	mockStats.On("GetCacheStats", mock.Anything).Return(expected, nil)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/cache", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))

	data := body["data"].(map[string]interface{})
	cacheTypes := data["cacheTypes"].([]interface{})
	assert.Len(t, cacheTypes, 2)
	assert.Equal(t, float64(1536), data["totalSizeBytes"])
}

func TestCacheHandler_GetCacheStats_Error(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	mockStats.On("GetCacheStats", mock.Anything).Return(nil, errors.New("db error"))

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/cache", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestCacheHandler_ClearCacheByType_Success(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	mockCleanup.On("ClearCacheByType", mock.Anything, "ai").Return(
		&services.CleanupResult{Type: "ai", EntriesRemoved: 5, BytesReclaimed: 1024}, nil,
	)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache/ai", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "ai", data["type"])
	assert.Equal(t, float64(5), data["entriesRemoved"])
}

func TestCacheHandler_ClearCacheByType_InvalidType(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	mockCleanup.On("ClearCacheByType", mock.Anything, "bogus").Return(
		nil, services.ErrInvalidCacheType,
	)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache/bogus", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "CACHE_TYPE_INVALID", errObj["code"])
}

func TestCacheHandler_ClearAllCache_WithDaysParam(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	mockCleanup.On("ClearCacheByAge", mock.Anything, 30).Return(
		&services.CleanupResult{Type: "all", EntriesRemoved: 10, BytesReclaimed: 5000}, nil,
	)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache?older_than_days=30", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(10), data["entriesRemoved"])
}

func TestCacheHandler_ClearAllCache_InvalidDaysParam(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache?older_than_days=abc", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCacheHandler_ClearAllCache_NegativeDaysParam(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache?older_than_days=-5", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCacheHandler_ClearAllCache_NoParam(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	// Should clear each type
	for _, ct := range services.ValidCacheTypes {
		mockCleanup.On("ClearCacheByType", mock.Anything, ct).Return(
			&services.CleanupResult{Type: ct, EntriesRemoved: 1, BytesReclaimed: 100}, nil,
		)
	}

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "all", data["type"])
	assert.Equal(t, float64(5), data["entriesRemoved"])     // 5 types * 1
	assert.Equal(t, float64(500), data["bytesReclaimed"])    // 5 types * 100
}

func TestCacheHandler_ClearCacheByType_ServerError(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	mockCleanup.On("ClearCacheByType", mock.Anything, "ai").Return(
		nil, errors.New("db failure"),
	)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache/ai", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "CACHE_CLEAR_FAILED", errObj["code"])
}

func TestCacheHandler_ClearAllCache_ZeroDaysParam(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache?older_than_days=0", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCacheHandler_ClearAllCache_EmptyDaysParam_FallsToAllClear(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	// Empty older_than_days= → gin returns "" → handler treats as no param → clear all
	for _, ct := range services.ValidCacheTypes {
		mockCleanup.On("ClearCacheByType", mock.Anything, ct).Return(
			&services.CleanupResult{Type: ct, EntriesRemoved: 1, BytesReclaimed: 0}, nil,
		)
	}

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache?older_than_days=", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Falls through to clear-all since "" != "" is false
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "all", data["type"])
}

func TestCacheHandler_ClearAllCache_FloatDaysParam(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache?older_than_days=30.5", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// strconv.Atoi rejects floats
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCacheHandler_ClearAllCache_PartialTypeFailure(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	// Some types succeed, some fail
	mockCleanup.On("ClearCacheByType", mock.Anything, "image").Return(
		&services.CleanupResult{Type: "image", EntriesRemoved: 3, BytesReclaimed: 500}, nil,
	)
	mockCleanup.On("ClearCacheByType", mock.Anything, "ai").Return(
		nil, errors.New("db locked"),
	)
	mockCleanup.On("ClearCacheByType", mock.Anything, "metadata").Return(
		&services.CleanupResult{Type: "metadata", EntriesRemoved: 2, BytesReclaimed: 0}, nil,
	)
	mockCleanup.On("ClearCacheByType", mock.Anything, "douban").Return(
		nil, errors.New("table missing"),
	)
	mockCleanup.On("ClearCacheByType", mock.Anything, "wikipedia").Return(
		&services.CleanupResult{Type: "wikipedia", EntriesRemoved: 1, BytesReclaimed: 0}, nil,
	)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Should still return 200 with partial results
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "all", data["type"])
	// Only successful types: 3 + 2 + 1 = 6
	assert.Equal(t, float64(6), data["entriesRemoved"])
	assert.Equal(t, float64(500), data["bytesReclaimed"])
}

func TestCacheHandler_ClearCacheByAge_ServiceError(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	mockCleanup.On("ClearCacheByAge", mock.Anything, 30).Return(
		nil, errors.New("disk full"),
	)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/cache?older_than_days=30", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestCacheHandler_ResponseStructure(t *testing.T) {
	mockStats := new(MockCacheStatsService)
	mockCleanup := new(MockCacheCleanupService)

	mockStats.On("GetCacheStats", mock.Anything).Return(&services.CacheStats{
		CacheTypes: []services.CacheTypeInfo{
			{Type: "image", Label: "圖片快取", SizeBytes: 100, EntryCount: 1},
		},
		TotalSizeBytes: 100,
	}, nil)

	router := setupCacheRouter(mockStats, mockCleanup)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/cache", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &raw))

	// Verify top-level structure
	assert.Contains(t, raw, "success")
	assert.Contains(t, raw, "data")
	assert.True(t, raw["success"].(bool))

	data := raw["data"].(map[string]interface{})
	assert.Contains(t, data, "cacheTypes")
	assert.Contains(t, data, "totalSizeBytes")

	// Verify cache type structure
	cacheTypes := data["cacheTypes"].([]interface{})
	ct := cacheTypes[0].(map[string]interface{})
	assert.Contains(t, ct, "type")
	assert.Contains(t, ct, "label")
	assert.Contains(t, ct, "sizeBytes")
	assert.Contains(t, ct, "entryCount")
}
