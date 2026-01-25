package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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
	searchMetadataFunc  func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error)
	getProvidersFunc    func() []services.ProviderInfo
	manualSearchFunc    func(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error)
	applyMetadataFunc   func(ctx context.Context, req *services.ApplyMetadataRequest) (*services.ApplyMetadataResponse, error)
	updateMetadataFunc  func(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error)
	uploadPosterFunc    func(ctx context.Context, req *services.UploadPosterRequest) (*services.UploadPosterResponse, error)
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

func (m *mockMetadataService) ManualSearch(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error) {
	if m.manualSearchFunc != nil {
		return m.manualSearchFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockMetadataService) ApplyMetadata(ctx context.Context, req *services.ApplyMetadataRequest) (*services.ApplyMetadataResponse, error) {
	if m.applyMetadataFunc != nil {
		return m.applyMetadataFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockMetadataService) UpdateMetadata(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error) {
	if m.updateMetadataFunc != nil {
		return m.updateMetadataFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockMetadataService) UploadPoster(ctx context.Context, req *services.UploadPosterRequest) (*services.UploadPosterResponse, error) {
	if m.uploadPosterFunc != nil {
		return m.uploadPosterFunc(ctx, req)
	}
	return nil, nil
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

// [P1] Tests TV search via API endpoint
func TestMetadataHandler_SearchMetadata_TVSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResult := &metadata.SearchResult{
		Items: []metadata.MetadataItem{
			{
				ID:        "1396",
				Title:     "Breaking Bad",
				Year:      2008,
				Rating:    8.9,
				MediaType: metadata.MediaTypeTV,
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
			assert.Equal(t, "Breaking Bad", req.Query)
			assert.Equal(t, "tv", req.MediaType)
			return expectedResult, expectedStatus, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=Breaking+Bad&mediaType=tv", nil)

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	results := data["results"].([]interface{})
	assert.Len(t, results, 1)

	firstResult := results[0].(map[string]interface{})
	// Note: MediaType uses PascalCase in JSON since struct doesn't have json tags
	assert.Equal(t, "tv", firstResult["MediaType"])
	assert.Equal(t, "Breaking Bad", firstResult["Title"])
}

// [P2] Tests search with pagination parameter
func TestMetadataHandler_SearchMetadata_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedPage := 0
	service := &mockMetadataService{
		searchMetadataFunc: func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
			capturedPage = req.Page
			return &metadata.SearchResult{
				Items:      []metadata.MetadataItem{{ID: "1", Title: "Test"}},
				Source:     models.MetadataSourceTMDb,
				TotalCount: 100,
				Page:       req.Page,
			}, &metadata.FallbackStatus{
				Attempts: []metadata.SourceAttempt{{Source: models.MetadataSourceTMDb, Success: true}},
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=Test&mediaType=movie&page=3", nil)

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 3, capturedPage)
}

// [P2] Tests search with year filter parameter
func TestMetadataHandler_SearchMetadata_WithYearFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedYear := 0
	service := &mockMetadataService{
		searchMetadataFunc: func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
			capturedYear = req.Year
			return &metadata.SearchResult{
				Items:      []metadata.MetadataItem{{ID: "1", Title: "Test 2024", Year: 2024}},
				Source:     models.MetadataSourceTMDb,
				TotalCount: 1,
				Page:       1,
			}, &metadata.FallbackStatus{
				Attempts: []metadata.SourceAttempt{{Source: models.MetadataSourceTMDb, Success: true}},
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=Test&mediaType=movie&year=2024", nil)

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 2024, capturedYear)
}

// [P2] Tests search with language parameter
func TestMetadataHandler_SearchMetadata_WithLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedLang := ""
	service := &mockMetadataService{
		searchMetadataFunc: func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
			capturedLang = req.Language
			return &metadata.SearchResult{
				Items:      []metadata.MetadataItem{{ID: "1", Title: "測試電影"}},
				Source:     models.MetadataSourceTMDb,
				TotalCount: 1,
				Page:       1,
			}, &metadata.FallbackStatus{
				Attempts: []metadata.SourceAttempt{{Source: models.MetadataSourceTMDb, Success: true}},
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=測試&mediaType=movie&language=zh-TW", nil)

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "zh-TW", capturedLang)
}

// [P1] Tests fallback status shows source indication (AC4)
func TestMetadataHandler_SearchMetadata_FallbackSourceIndication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResult := &metadata.SearchResult{
		Items: []metadata.MetadataItem{
			{ID: "1", Title: "Test from Douban"},
		},
		Source:     models.MetadataSourceDouban,
		TotalCount: 1,
		Page:       1,
	}

	expectedStatus := &metadata.FallbackStatus{
		Attempts: []metadata.SourceAttempt{
			{Source: models.MetadataSourceTMDb, Success: false},
			{Source: models.MetadataSourceDouban, Success: true},
		},
	}

	service := &mockMetadataService{
		searchMetadataFunc: func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
			return expectedResult, expectedStatus, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=Test&mediaType=movie", nil)

	handler.SearchMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})

	// AC4: Source indication
	assert.Equal(t, "douban", data["source"])

	// Verify fallback status shows the chain
	fallbackStatus := data["fallbackStatus"].(map[string]interface{})
	attempts := fallbackStatus["attempts"].([]interface{})
	assert.Len(t, attempts, 2)

	// First attempt was TMDb (failed)
	attempt1 := attempts[0].(map[string]interface{})
	assert.Equal(t, "tmdb", attempt1["source"])
	assert.False(t, attempt1["success"].(bool))

	// Second attempt was Douban (succeeded)
	attempt2 := attempts[1].(map[string]interface{})
	assert.Equal(t, "douban", attempt2["source"])
	assert.True(t, attempt2["success"].(bool))
}

// [P1] Tests invalid media type returns proper error
func TestMetadataHandler_SearchMetadata_InvalidMediaType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=Test&mediaType=invalid", nil)

	handler.SearchMetadata(c)

	// Should either return error or default to movie
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, w.Code)
}

// [P2] Tests whitespace-only query goes to service for validation
// Note: Handler only checks for empty string, service validates trimmed query
func TestMetadataHandler_SearchMetadata_WhitespaceOnlyQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Service should receive the whitespace query
	service := &mockMetadataService{
		searchMetadataFunc: func(ctx context.Context, req *services.SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
			// Verify the whitespace query was passed through
			assert.Equal(t, "   ", req.Query)
			// Service validates and rejects
			return nil, nil, nil
		},
	}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/metadata/search?query=+++&mediaType=movie", nil)

	handler.SearchMetadata(c)

	// Handler passes to service, gets nil result (empty response)
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Success but empty results
	assert.True(t, response["success"].(bool))
}

// =============================================================================
// Manual Search Handler Tests (Story 3.7)
// =============================================================================

// [P1] Tests manual search with all sources (AC1, AC4)
func TestMetadataHandler_ManualSearch_AllSources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResponse := &services.ManualSearchResponse{
		Results: []services.ManualSearchResultItem{
			{
				ID:        "tmdb-85937",
				Source:    models.MetadataSourceTMDb,
				Title:     "Demon Slayer: Kimetsu no Yaiba",
				TitleZhTW: "鬼滅之刃",
				Year:      2019,
				MediaType: "tv",
				Overview:  "It is the Taisho Period in Japan...",
				PosterURL: "https://image.tmdb.org/t/p/w500/test.jpg",
				Rating:    8.7,
			},
			{
				ID:        "douban-30277296",
				Source:    models.MetadataSourceDouban,
				Title:     "鬼灭之刃",
				TitleZhTW: "鬼滅之刃",
				Year:      2019,
				MediaType: "tv",
				Overview:  "大正時期，少年炭治郎...",
				PosterURL: "https://img.doubanio.com/test.jpg",
				Rating:    8.4,
			},
		},
		TotalCount:      2,
		SearchedSources: []string{"tmdb", "douban"},
	}

	service := &mockMetadataService{
		manualSearchFunc: func(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error) {
			assert.Equal(t, "Demon Slayer", req.Query)
			assert.Equal(t, "tv", req.MediaType)
			assert.Equal(t, "all", req.Source)
			return expectedResponse, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"query":"Demon Slayer","mediaType":"tv","source":"all"}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})

	results := data["results"].([]interface{})
	assert.Len(t, results, 2)

	// AC4: Source indicator in results
	firstResult := results[0].(map[string]interface{})
	assert.Equal(t, "tmdb", firstResult["source"])

	secondResult := results[1].(map[string]interface{})
	assert.Equal(t, "douban", secondResult["source"])

	// Verify searched sources
	searchedSources := data["searchedSources"].([]interface{})
	assert.Len(t, searchedSources, 2)
}

// [P1] Tests manual search with specific source (AC4)
func TestMetadataHandler_ManualSearch_SpecificSource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResponse := &services.ManualSearchResponse{
		Results: []services.ManualSearchResultItem{
			{
				ID:        "tmdb-550",
				Source:    models.MetadataSourceTMDb,
				Title:     "Fight Club",
				Year:      1999,
				MediaType: "movie",
				Rating:    8.4,
			},
		},
		TotalCount:      1,
		SearchedSources: []string{"tmdb"},
	}

	service := &mockMetadataService{
		manualSearchFunc: func(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error) {
			assert.Equal(t, "Fight Club", req.Query)
			assert.Equal(t, "tmdb", req.Source)
			return expectedResponse, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"query":"Fight Club","mediaType":"movie","source":"tmdb"}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})

	// Verify only searched the specified source
	searchedSources := data["searchedSources"].([]interface{})
	assert.Len(t, searchedSources, 1)
	assert.Equal(t, "tmdb", searchedSources[0])
}

// [P1] Tests manual search missing query returns error
func TestMetadataHandler_ManualSearch_MissingQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"mediaType":"movie","source":"all"}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "MANUAL_SEARCH_INVALID_REQUEST", errData["code"])
}

// [P1] Tests manual search invalid JSON returns error
func TestMetadataHandler_ManualSearch_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{invalid json}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// [P1] Tests manual search with year filter (AC1)
func TestMetadataHandler_ManualSearch_WithYearFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedYear := 0
	service := &mockMetadataService{
		manualSearchFunc: func(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error) {
			capturedYear = req.Year
			return &services.ManualSearchResponse{
				Results:         []services.ManualSearchResultItem{},
				TotalCount:      0,
				SearchedSources: []string{"tmdb"},
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"query":"Matrix","mediaType":"movie","year":1999,"source":"tmdb"}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1999, capturedYear)
}

// [P2] Tests manual search no results
func TestMetadataHandler_ManualSearch_NoResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		manualSearchFunc: func(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error) {
			return &services.ManualSearchResponse{
				Results:         []services.ManualSearchResultItem{},
				TotalCount:      0,
				SearchedSources: []string{"tmdb", "douban"},
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"query":"Nonexistent Movie 12345","mediaType":"movie","source":"all"}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	results := data["results"].([]interface{})
	assert.Empty(t, results)
	assert.Equal(t, float64(0), data["totalCount"])
}

// [P2] Tests manual search invalid source returns error
func TestMetadataHandler_ManualSearch_InvalidSource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		manualSearchFunc: func(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error) {
			return nil, services.ErrManualSearchInvalidSource
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"query":"Test","mediaType":"movie","source":"invalid"}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "MANUAL_SEARCH_INVALID_SOURCE", errData["code"])
}

// [P1] Tests manual search route registration
func TestMetadataHandler_RegisterRoutes_IncludesManualSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg)

	routes := router.Routes()

	foundManualSearch := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/api/v1/metadata/manual-search" {
			foundManualSearch = true
			break
		}
	}

	assert.True(t, foundManualSearch, "POST /api/v1/metadata/manual-search route should be registered")
}

// [P1] Tests manual search defaults to movie media type
func TestMetadataHandler_ManualSearch_DefaultsToMovie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedMediaType := ""
	service := &mockMetadataService{
		manualSearchFunc: func(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error) {
			capturedMediaType = req.MediaType
			return &services.ManualSearchResponse{
				Results:         []services.ManualSearchResultItem{},
				TotalCount:      0,
				SearchedSources: []string{"tmdb"},
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No mediaType specified
	body := `{"query":"Test","source":"tmdb"}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "movie", capturedMediaType)
}

// [P1] Tests manual search defaults to all sources
func TestMetadataHandler_ManualSearch_DefaultsToAllSources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedSource := ""
	service := &mockMetadataService{
		manualSearchFunc: func(ctx context.Context, req *services.ManualSearchRequest) (*services.ManualSearchResponse, error) {
			capturedSource = req.Source
			return &services.ManualSearchResponse{
				Results:         []services.ManualSearchResultItem{},
				TotalCount:      0,
				SearchedSources: []string{"tmdb", "douban", "wikipedia"},
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No source specified
	body := `{"query":"Test","mediaType":"movie"}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/manual-search", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ManualSearch(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "all", capturedSource)
}

// =============================================================================
// Apply Metadata Handler Tests (Story 3.7 - AC3)
// =============================================================================

// [P1] Tests apply metadata success for movie (AC3)
func TestMetadataHandler_ApplyMetadata_MovieSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResponse := &services.ApplyMetadataResponse{
		Success:   true,
		MediaID:   "test-movie-id",
		MediaType: "movie",
		Title:     "Fight Club",
		Source:    models.MetadataSourceTMDb,
	}

	service := &mockMetadataService{
		applyMetadataFunc: func(ctx context.Context, req *services.ApplyMetadataRequest) (*services.ApplyMetadataResponse, error) {
			assert.Equal(t, "test-movie-id", req.MediaID)
			assert.Equal(t, "movie", req.MediaType)
			assert.Equal(t, "tmdb-550", req.SelectedItem.ID)
			assert.Equal(t, "tmdb", req.SelectedItem.Source)
			return expectedResponse, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{
		"mediaId": "test-movie-id",
		"mediaType": "movie",
		"selectedItem": {
			"id": "tmdb-550",
			"source": "tmdb"
		}
	}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/apply", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ApplyMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test-movie-id", data["mediaId"])
	assert.Equal(t, "Fight Club", data["title"])
	assert.Equal(t, "tmdb", data["source"])
}

// [P1] Tests apply metadata missing mediaId returns error
func TestMetadataHandler_ApplyMetadata_MissingMediaId(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{
		"mediaType": "movie",
		"selectedItem": {
			"id": "tmdb-550",
			"source": "tmdb"
		}
	}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/apply", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ApplyMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "APPLY_METADATA_INVALID_REQUEST", errData["code"])
}

// [P1] Tests apply metadata missing selectedItem returns error
func TestMetadataHandler_ApplyMetadata_MissingSelectedItem(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{
		"mediaId": "test-movie-id",
		"mediaType": "movie"
	}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/apply", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ApplyMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// [P1] Tests apply metadata invalid JSON returns error
func TestMetadataHandler_ApplyMetadata_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{invalid json}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/apply", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ApplyMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// [P1] Tests apply metadata media not found
func TestMetadataHandler_ApplyMetadata_MediaNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		applyMetadataFunc: func(ctx context.Context, req *services.ApplyMetadataRequest) (*services.ApplyMetadataResponse, error) {
			return nil, services.ErrApplyMetadataNotFound
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{
		"mediaId": "nonexistent-id",
		"mediaType": "movie",
		"selectedItem": {
			"id": "tmdb-550",
			"source": "tmdb"
		}
	}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/apply", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ApplyMetadata(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "APPLY_METADATA_NOT_FOUND", errData["code"])
}

// [P2] Tests apply metadata with learnPattern flag
func TestMetadataHandler_ApplyMetadata_WithLearnPattern(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedLearnPattern := false
	service := &mockMetadataService{
		applyMetadataFunc: func(ctx context.Context, req *services.ApplyMetadataRequest) (*services.ApplyMetadataResponse, error) {
			capturedLearnPattern = req.LearnPattern
			return &services.ApplyMetadataResponse{
				Success:   true,
				MediaID:   req.MediaID,
				MediaType: req.MediaType,
				Title:     "Test",
				Source:    models.MetadataSourceTMDb,
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{
		"mediaId": "test-id",
		"mediaType": "movie",
		"selectedItem": {
			"id": "tmdb-550",
			"source": "tmdb"
		},
		"learnPattern": true
	}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/apply", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ApplyMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, capturedLearnPattern)
}

// [P1] Tests apply metadata route registration
func TestMetadataHandler_RegisterRoutes_IncludesApplyMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg)

	routes := router.Routes()

	foundApplyMetadata := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/api/v1/metadata/apply" {
			foundApplyMetadata = true
			break
		}
	}

	assert.True(t, foundApplyMetadata, "POST /api/v1/metadata/apply route should be registered")
}

// [P2] Tests apply metadata for series
func TestMetadataHandler_ApplyMetadata_SeriesSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResponse := &services.ApplyMetadataResponse{
		Success:   true,
		MediaID:   "test-series-id",
		MediaType: "series",
		Title:     "Breaking Bad",
		Source:    models.MetadataSourceTMDb,
	}

	service := &mockMetadataService{
		applyMetadataFunc: func(ctx context.Context, req *services.ApplyMetadataRequest) (*services.ApplyMetadataResponse, error) {
			assert.Equal(t, "test-series-id", req.MediaID)
			assert.Equal(t, "series", req.MediaType)
			return expectedResponse, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{
		"mediaId": "test-series-id",
		"mediaType": "series",
		"selectedItem": {
			"id": "tmdb-1396",
			"source": "tmdb"
		}
	}`
	c.Request = httptest.NewRequest("POST", "/api/v1/metadata/apply", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ApplyMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "series", data["mediaType"])
}

// =============================================================================
// Update Metadata Handler Tests (Story 3.8 - AC2)
// =============================================================================

// [P1] Tests update metadata success for movie (AC2)
func TestMetadataHandler_UpdateMetadata_MovieSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResponse := &services.UpdateMetadataResponse{
		ID:             "test-movie-id",
		Title:          "鬼滅之刃",
		MetadataSource: models.MetadataSourceManual,
		UpdatedAt:      "2026-01-25T12:00:00Z",
	}

	service := &mockMetadataService{
		updateMetadataFunc: func(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error) {
			assert.Equal(t, "test-movie-id", req.ID)
			assert.Equal(t, "movie", req.MediaType)
			assert.Equal(t, "鬼滅之刃", req.Title)
			assert.Equal(t, "Demon Slayer", req.TitleEnglish)
			assert.Equal(t, 2019, req.Year)
			assert.Contains(t, req.Genres, "動作")
			assert.Equal(t, "外崎春雄", req.Director)
			assert.Contains(t, req.Cast, "花江夏樹")
			return expectedResponse, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-movie-id"}}
	body := `{
		"mediaType": "movie",
		"title": "鬼滅之刃",
		"titleEnglish": "Demon Slayer",
		"year": 2019,
		"genres": ["動作", "奇幻", "冒險"],
		"director": "外崎春雄",
		"cast": ["花江夏樹", "鬼頭明里", "下野紘"],
		"overview": "大正時代的日本，善良的少年炭治郎..."
	}`
	c.Request = httptest.NewRequest("PUT", "/api/v1/media/test-movie-id/metadata", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test-movie-id", data["id"])
	assert.Equal(t, "鬼滅之刃", data["title"])
	assert.Equal(t, "manual", data["metadataSource"])
}

// [P1] Tests update metadata missing title returns error (AC4)
func TestMetadataHandler_UpdateMetadata_MissingTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		updateMetadataFunc: func(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error) {
			return nil, services.ErrUpdateMetadataTitleRequired
		},
	}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	body := `{
		"mediaType": "movie",
		"year": 2019
	}`
	c.Request = httptest.NewRequest("PUT", "/api/v1/media/test-id/metadata", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_REQUIRED_FIELD", errData["code"])
}

// [P1] Tests update metadata missing year returns error (AC4)
func TestMetadataHandler_UpdateMetadata_MissingYear(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		updateMetadataFunc: func(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error) {
			return nil, services.ErrUpdateMetadataYearRequired
		},
	}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	body := `{
		"mediaType": "movie",
		"title": "Test Movie"
	}`
	c.Request = httptest.NewRequest("PUT", "/api/v1/media/test-id/metadata", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_REQUIRED_FIELD", errData["code"])
}

// [P1] Tests update metadata invalid JSON returns error
func TestMetadataHandler_UpdateMetadata_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	body := `{invalid json}`
	c.Request = httptest.NewRequest("PUT", "/api/v1/media/test-id/metadata", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMetadata(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
}

// [P1] Tests update metadata media not found
func TestMetadataHandler_UpdateMetadata_MediaNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		updateMetadataFunc: func(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error) {
			return nil, services.ErrUpdateMetadataNotFound
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent-id"}}
	body := `{
		"mediaType": "movie",
		"title": "Test",
		"year": 2020
	}`
	c.Request = httptest.NewRequest("PUT", "/api/v1/media/nonexistent-id/metadata", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMetadata(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "METADATA_UPDATE_NOT_FOUND", errData["code"])
}

// [P2] Tests update metadata for series
func TestMetadataHandler_UpdateMetadata_SeriesSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResponse := &services.UpdateMetadataResponse{
		ID:             "test-series-id",
		Title:          "Breaking Bad",
		MetadataSource: models.MetadataSourceManual,
		UpdatedAt:      "2026-01-25T12:00:00Z",
	}

	service := &mockMetadataService{
		updateMetadataFunc: func(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error) {
			assert.Equal(t, "test-series-id", req.ID)
			assert.Equal(t, "series", req.MediaType)
			return expectedResponse, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-series-id"}}
	body := `{
		"mediaType": "series",
		"title": "Breaking Bad",
		"year": 2008
	}`
	c.Request = httptest.NewRequest("PUT", "/api/v1/media/test-series-id/metadata", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
}

// [P2] Tests update metadata with poster URL
func TestMetadataHandler_UpdateMetadata_WithPosterURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedPosterURL := ""
	service := &mockMetadataService{
		updateMetadataFunc: func(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error) {
			capturedPosterURL = req.PosterURL
			return &services.UpdateMetadataResponse{
				ID:             req.ID,
				Title:          req.Title,
				MetadataSource: models.MetadataSourceManual,
				UpdatedAt:      "2026-01-25T12:00:00Z",
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	body := `{
		"mediaType": "movie",
		"title": "Test Movie",
		"year": 2020,
		"posterUrl": "https://example.com/poster.jpg"
	}`
	c.Request = httptest.NewRequest("PUT", "/api/v1/media/test-id/metadata", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://example.com/poster.jpg", capturedPosterURL)
}

// [P1] Tests update metadata route registration
func TestMetadataHandler_RegisterRoutes_IncludesUpdateMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg)

	routes := router.Routes()

	foundUpdateMetadata := false
	for _, route := range routes {
		if route.Method == "PUT" && route.Path == "/api/v1/media/:id/metadata" {
			foundUpdateMetadata = true
			break
		}
	}

	assert.True(t, foundUpdateMetadata, "PUT /api/v1/media/:id/metadata route should be registered")
}

// [P2] Tests update metadata defaults media type to movie
func TestMetadataHandler_UpdateMetadata_DefaultsToMovie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	capturedMediaType := ""
	service := &mockMetadataService{
		updateMetadataFunc: func(ctx context.Context, req *services.UpdateMetadataRequest) (*services.UpdateMetadataResponse, error) {
			capturedMediaType = req.MediaType
			return &services.UpdateMetadataResponse{
				ID:             req.ID,
				Title:          req.Title,
				MetadataSource: models.MetadataSourceManual,
				UpdatedAt:      "2026-01-25T12:00:00Z",
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	// No mediaType specified
	body := `{
		"title": "Test Movie",
		"year": 2020
	}`
	c.Request = httptest.NewRequest("PUT", "/api/v1/media/test-id/metadata", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMetadata(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "movie", capturedMediaType)
}

// =============================================================================
// Upload Poster Handler Tests (Story 3.8 - AC3)
// =============================================================================

// [P1] Tests upload poster success (AC3)
func TestMetadataHandler_UploadPoster_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedResponse := &services.UploadPosterResponse{
		PosterURL:    "/posters/test-movie-id.webp",
		ThumbnailURL: "/posters/test-movie-id-thumb.webp",
	}

	service := &mockMetadataService{
		uploadPosterFunc: func(ctx context.Context, req *services.UploadPosterRequest) (*services.UploadPosterResponse, error) {
			assert.Equal(t, "test-movie-id", req.MediaID)
			assert.Equal(t, "movie", req.MediaType)
			assert.Equal(t, "image/jpeg", req.ContentType)
			assert.NotEmpty(t, req.FileData)
			return expectedResponse, nil
		},
	}

	handler := NewMetadataHandler(service)

	// Create a multipart form with a fake image
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "poster.jpg")
	// Write a minimal valid JPEG header
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46})
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-movie-id"}}
	c.Request = httptest.NewRequest("POST", "/api/v1/media/test-movie-id/poster?mediaType=movie", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UploadPoster(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "/posters/test-movie-id.webp", data["posterUrl"])
	assert.Equal(t, "/posters/test-movie-id-thumb.webp", data["thumbnailUrl"])
}

// [P1] Tests upload poster missing file returns error
func TestMetadataHandler_UploadPoster_MissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest("POST", "/api/v1/media/test-id/poster", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UploadPoster(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "POSTER_UPLOAD_INVALID_REQUEST", errData["code"])
}

// [P1] Tests upload poster invalid file type returns error (AC3)
func TestMetadataHandler_UploadPoster_InvalidFileType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		uploadPosterFunc: func(ctx context.Context, req *services.UploadPosterRequest) (*services.UploadPosterResponse, error) {
			return nil, services.ErrPosterInvalidFormat
		},
	}

	handler := NewMetadataHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "poster.gif")
	part.Write([]byte("GIF89a")) // GIF header
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest("POST", "/api/v1/media/test-id/poster", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UploadPoster(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "POSTER_INVALID_FORMAT", errData["code"])
}

// [P1] Tests upload poster file too large returns error (AC3)
func TestMetadataHandler_UploadPoster_FileTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		uploadPosterFunc: func(ctx context.Context, req *services.UploadPosterRequest) (*services.UploadPosterResponse, error) {
			return nil, services.ErrPosterTooLarge
		},
	}

	handler := NewMetadataHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "poster.jpg")
	// Write a fake large file
	part.Write(make([]byte, 1024)) // 1KB for testing
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest("POST", "/api/v1/media/test-id/poster", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UploadPoster(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "POSTER_TOO_LARGE", errData["code"])
}

// [P1] Tests upload poster media not found
func TestMetadataHandler_UploadPoster_MediaNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		uploadPosterFunc: func(ctx context.Context, req *services.UploadPosterRequest) (*services.UploadPosterResponse, error) {
			return nil, services.ErrUploadPosterNotFound
		},
	}

	handler := NewMetadataHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "poster.jpg")
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent-id"}}
	c.Request = httptest.NewRequest("POST", "/api/v1/media/nonexistent-id/poster", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UploadPoster(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// [P1] Tests upload poster route registration
func TestMetadataHandler_RegisterRoutes_IncludesUploadPoster(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{}
	handler := NewMetadataHandler(service)

	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg)

	routes := router.Routes()

	foundUploadPoster := false
	for _, route := range routes {
		if route.Method == "POST" && route.Path == "/api/v1/media/:id/poster" {
			foundUploadPoster = true
			break
		}
	}

	assert.True(t, foundUploadPoster, "POST /api/v1/media/:id/poster route should be registered")
}

// [P2] Tests upload poster for series
func TestMetadataHandler_UploadPoster_SeriesSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockMetadataService{
		uploadPosterFunc: func(ctx context.Context, req *services.UploadPosterRequest) (*services.UploadPosterResponse, error) {
			assert.Equal(t, "series", req.MediaType)
			return &services.UploadPosterResponse{
				PosterURL:    "/posters/test-series-id.webp",
				ThumbnailURL: "/posters/test-series-id-thumb.webp",
			}, nil
		},
	}

	handler := NewMetadataHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "poster.jpg")
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-series-id"}}
	c.Request = httptest.NewRequest("POST", "/api/v1/media/test-series-id/poster?mediaType=series", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UploadPoster(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
