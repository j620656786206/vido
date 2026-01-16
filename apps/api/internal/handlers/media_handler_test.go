package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/media"
)

// mockMediaService is a mock implementation of MediaServiceInterface for testing
type mockMediaService struct {
	config *media.MediaConfig
}

func (m *mockMediaService) GetConfig() *media.MediaConfig {
	return m.config
}

func (m *mockMediaService) GetConfiguredDirectories() []media.MediaDirectory {
	return m.config.Directories
}

func (m *mockMediaService) GetAccessibleDirectories() []media.MediaDirectory {
	accessible := make([]media.MediaDirectory, 0)
	for _, d := range m.config.Directories {
		if d.Status == media.StatusAccessible {
			accessible = append(accessible, d)
		}
	}
	return accessible
}

func (m *mockMediaService) RefreshDirectoryStatus() *media.MediaConfig {
	return m.config
}

func (m *mockMediaService) IsSearchOnlyMode() bool {
	return m.config.SearchOnlyMode
}

func newMockMediaConfig(dirs []media.MediaDirectory, validCount int, searchOnlyMode bool) *media.MediaConfig {
	return &media.MediaConfig{
		Directories:    dirs,
		ValidCount:     validCount,
		TotalCount:     len(dirs),
		SearchOnlyMode: searchOnlyMode,
	}
}

func setupMediaRouter(handler *MediaHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)
	return router
}

func TestMediaHandler_GetMediaDirectories_Success(t *testing.T) {
	dirs := []media.MediaDirectory{
		{
			Path:      "/media/movies",
			Type:      "movies",
			Status:    media.StatusAccessible,
			FileCount: 100,
		},
		{
			Path:   "/media/tv",
			Type:   "tv",
			Status: media.StatusNotFound,
			Error:  "directory does not exist",
		},
	}
	mockSvc := &mockMediaService{config: newMockMediaConfig(dirs, 1, false)}
	handler := NewMediaHandler(mockSvc)
	router := setupMediaRouter(handler)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/settings/media-directories", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)

	// Check the data structure
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, float64(2), data["total_count"])
	assert.Equal(t, float64(1), data["valid_count"])
	assert.Equal(t, false, data["search_only_mode"])
}

func TestMediaHandler_GetMediaDirectories_SearchOnlyMode(t *testing.T) {
	mockSvc := &mockMediaService{config: newMockMediaConfig([]media.MediaDirectory{}, 0, true)}
	handler := NewMediaHandler(mockSvc)
	router := setupMediaRouter(handler)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/settings/media-directories", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, float64(0), data["total_count"])
	assert.Equal(t, float64(0), data["valid_count"])
	assert.Equal(t, true, data["search_only_mode"])
}

func TestMediaHandler_RefreshMediaDirectories(t *testing.T) {
	dirs := []media.MediaDirectory{
		{
			Path:      "/media/movies",
			Type:      "movies",
			Status:    media.StatusAccessible,
			FileCount: 50,
		},
	}
	mockSvc := &mockMediaService{config: newMockMediaConfig(dirs, 1, false)}
	handler := NewMediaHandler(mockSvc)
	router := setupMediaRouter(handler)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/settings/media-directories/refresh", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
}

func TestMediaHandler_RegisterRoutes(t *testing.T) {
	mockSvc := &mockMediaService{config: newMockMediaConfig([]media.MediaDirectory{}, 0, true)}
	handler := NewMediaHandler(mockSvc)
	router := setupMediaRouter(handler)

	// Test GET endpoint exists
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/media-directories", nil)
	wGet := httptest.NewRecorder()
	router.ServeHTTP(wGet, reqGet)
	assert.Equal(t, http.StatusOK, wGet.Code)

	// Test POST endpoint exists
	reqPost, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/media-directories/refresh", nil)
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)
	assert.Equal(t, http.StatusOK, wPost.Code)
}

func TestMediaHandler_DirectoryStatuses(t *testing.T) {
	// Test all possible directory statuses
	dirs := []media.MediaDirectory{
		{Path: "/accessible", Status: media.StatusAccessible, FileCount: 10},
		{Path: "/notfound", Status: media.StatusNotFound, Error: "directory does not exist"},
		{Path: "/notdir", Status: media.StatusNotDir, Error: "path is not a directory"},
		{Path: "/notreadable", Status: media.StatusNotReadable, Error: "cannot read directory contents"},
	}
	mockSvc := &mockMediaService{config: newMockMediaConfig(dirs, 1, false)}
	handler := NewMediaHandler(mockSvc)
	router := setupMediaRouter(handler)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/settings/media-directories", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)

	directories, ok := data["directories"].([]interface{})
	require.True(t, ok)
	assert.Len(t, directories, 4)
}
