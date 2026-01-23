package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// mockMetadataService implements services.MetadataServiceInterface for testing
type mockMetadataService struct {
	searchMetadataFunc func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error)
	getProvidersFunc   func() []services.ProviderInfo
}

func (m *mockMetadataService) SearchMetadata(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
	if m.searchMetadataFunc != nil {
		return m.searchMetadataFunc(ctx, req)
	}
	return nil, nil, nil
}

func (m *mockMetadataService) GetProviders() []services.ProviderInfo {
	if m.getProvidersFunc != nil {
		return m.getProvidersFunc()
	}
	return []services.ProviderInfo{}
}

func TestNewMetadataHandler(t *testing.T) {
	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	assert.NotNil(t, handler)
}

func TestMetadataHandler_SearchMetadata_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResult := &metadata.SearchResult{
		Items: []metadata.MetadataItem{
			{
				ID:          "550",
				Title:       "Fight Club",
				Year:        1999,
				Rating:      8.4,
				MediaType:   metadata.MediaTypeMovie,
			},
		},
		Source:     models.MetadataSourceTMDb,
		TotalCount: 1,
		Page:       1,
	}

	expectedStatus := &metadata.FallbackStatus{
		Attempts: []metadata.SourceAttempt{
			{Source: models.MetadataSourceTMDb, Success: true},
		},
	}

	service := &mockMetadataService{
		searchMetadataFunc: func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
			assert.Equal(t, "Fight Club", req.Query)
			assert.Equal(t, "movie", req.MediaType)
			return expectedResult, expectedStatus, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=Fight+Club&mediaType=movie", nil)

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "tmdb", data["source"])

	results := data["results"].([]interface{})
	assert.Len(t, results, 1)

	fallbackStatus := data["fallbackStatus"].(map[string]interface{})
	attempts := fallbackStatus["attempts"].([]interface{})
	assert.Len(t, attempts, 1)
}

func TestMetadataHandler_SearchMetadata_MissingQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?mediaType=movie", nil)

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "METADATA_INVALID_REQUEST", errData["code"])
}

func TestMetadataHandler_SearchMetadata_AllProvidersFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedStatus := &metadata.FallbackStatus{
		Attempts: []metadata.SourceAttempt{
			{Source: models.MetadataSourceTMDb, Success: false},
			{Source: models.MetadataSourceDouban, Success: false},
		},
	}

	service := &mockMetadataService{
		searchMetadataFunc: func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
			return nil, expectedStatus, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=Nonexistent&mediaType=movie", nil)

	handler.SearchMetadata(c)

	// Should return 200 with no results but include fallback status
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})

	results := data["results"].([]interface{})
	assert.Empty(t, results)

	// Should still have fallback status
	fallbackStatus := data["fallbackStatus"].(map[string]interface{})
	assert.NotNil(t, fallbackStatus)
}

func TestMetadataHandler_GetProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedProviders := []services.ProviderInfo{
		{
			Name:      "TMDb",
			Source:    models.MetadataSourceTMDb,
			Available: true,
			Status:    metadata.ProviderStatusAvailable,
		},
		{
			Name:      "Douban",
			Source:    models.MetadataSourceDouban,
			Available: true,
			Status:    metadata.ProviderStatusAvailable,
		},
	}

	service := &mockMetadataService{
		getProvidersFunc: func() []services.ProviderInfo {
			return expectedProviders
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/providers", nil)

	handler.GetProviders(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)
}

func TestMetadataHandler_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg)

	routes := router.Routes()

	expectedRoutes := map[string]string{
		"GET:/api/v1/metadata/search":    "SearchMetadata",
		"GET:/api/v1/metadata/providers": "GetProviders",
	}

	for _, route := range routes {
		key := route.Method + ":" + route.Path
		delete(expectedRoutes, key)
	}

	assert.Empty(t, expectedRoutes, "Missing routes: %v", expectedRoutes)
}
