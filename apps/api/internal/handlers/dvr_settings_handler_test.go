package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/services"
)

// mockDVRSettingsService implements services.DVRSettingsServiceInterface via
// swappable funcs.
type mockDVRSettingsService struct {
	getConfig          func(ctx context.Context, plugin string) (*services.DVRConfigStatus, error)
	saveConfig         func(ctx context.Context, plugin string, input services.DVRConfigInput) error
	testConnection     func(ctx context.Context, plugin string, input *services.DVRConfigInput) error
	getQualityProfiles func(ctx context.Context, plugin string) ([]plugins.QualityProfile, error)
	getRootFolders     func(ctx context.Context, plugin string) ([]plugins.RootFolder, error)
}

func (m *mockDVRSettingsService) GetConfig(ctx context.Context, plugin string) (*services.DVRConfigStatus, error) {
	return m.getConfig(ctx, plugin)
}
func (m *mockDVRSettingsService) SaveConfig(ctx context.Context, plugin string, input services.DVRConfigInput) error {
	return m.saveConfig(ctx, plugin, input)
}
func (m *mockDVRSettingsService) TestConnection(ctx context.Context, plugin string, input *services.DVRConfigInput) error {
	return m.testConnection(ctx, plugin, input)
}
func (m *mockDVRSettingsService) GetQualityProfiles(ctx context.Context, plugin string) ([]plugins.QualityProfile, error) {
	return m.getQualityProfiles(ctx, plugin)
}
func (m *mockDVRSettingsService) GetRootFolders(ctx context.Context, plugin string) ([]plugins.RootFolder, error) {
	return m.getRootFolders(ctx, plugin)
}

var _ services.DVRSettingsServiceInterface = (*mockDVRSettingsService)(nil)

func setupDVRRouter(svc services.DVRSettingsServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler := NewDVRSettingsHandler(svc, "radarr")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestDVRSettingsHandler_GetConfig(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	svc := &mockDVRSettingsService{
		getConfig: func(ctx context.Context, plugin string) (*services.DVRConfigStatus, error) {
			assert.Equal(t, "radarr", plugin)
			return &services.DVRConfigStatus{
				URL:              "http://radarr:7878",
				Enabled:          true,
				QualityProfileID: 4,
				RootFolderPath:   "/movies",
				HasAPIKey:        true,
				Health:           plugins.PluginHealth{Status: plugins.HealthStatusHealthy, LastCheckedAt: &now},
			}, nil
		},
	}
	router := setupDVRRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/radarr", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			URL       string `json:"url"`
			Enabled   bool   `json:"enabled"`
			HasAPIKey bool   `json:"has_api_key"`
			Health    struct {
				Status        string  `json:"status"`
				LastCheckedAt *string `json:"last_checked_at"`
				Message       string  `json:"message"`
			} `json:"health"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "http://radarr:7878", resp.Data.URL)
	assert.True(t, resp.Data.HasAPIKey)
	assert.Equal(t, "healthy", resp.Data.Health.Status)
	require.NotNil(t, resp.Data.Health.LastCheckedAt)
	// The key itself must never appear anywhere in the GET response (AC #4, Rule 27 ⑤).
	assert.NotContains(t, w.Body.String(), "\"api_key\"")
}

func TestDVRSettingsHandler_SaveConfig_Success(t *testing.T) {
	var captured services.DVRConfigInput
	svc := &mockDVRSettingsService{
		saveConfig: func(ctx context.Context, plugin string, input services.DVRConfigInput) error {
			assert.Equal(t, "radarr", plugin)
			captured = input
			return nil
		},
	}
	router := setupDVRRouter(svc)

	body := `{"url":"http://radarr:7878","api_key":"secret","enabled":true,"quality_profile_id":4,"root_folder_path":"/movies"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/radarr", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, services.DVRConfigInput{
		URL:              "http://radarr:7878",
		APIKey:           "secret",
		Enabled:          true,
		QualityProfileID: 4,
		RootFolderPath:   "/movies",
	}, captured)
}

func TestDVRSettingsHandler_SaveConfig_TestFailedIs409(t *testing.T) {
	svc := &mockDVRSettingsService{
		saveConfig: func(ctx context.Context, plugin string, input services.DVRConfigInput) error {
			return &plugins.PluginError{Code: plugins.ErrCodeTestFailed, Message: "radarr connection test failed — config not saved"}
		},
	}
	router := setupDVRRouter(svc)

	body := `{"url":"http://radarr:7878","api_key":"bad","enabled":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/radarr", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	assert.Equal(t, "DVR_TEST_FAILED", resp.Error.Code)
}

func TestDVRSettingsHandler_SaveConfig_InvalidBody(t *testing.T) {
	svc := &mockDVRSettingsService{}
	router := setupDVRRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/radarr", bytes.NewBufferString(`{"url": 123`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDVRSettingsHandler_TestConnection_NoBodyUsesSaved(t *testing.T) {
	var receivedInput *services.DVRConfigInput = &services.DVRConfigInput{URL: "sentinel"}
	svc := &mockDVRSettingsService{
		testConnection: func(ctx context.Context, plugin string, input *services.DVRConfigInput) error {
			receivedInput = input
			return nil
		},
	}
	router := setupDVRRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/radarr/test", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, receivedInput, "no body → service must receive nil (test saved config)")
}

func TestDVRSettingsHandler_TestConnection_BodyConfig(t *testing.T) {
	var receivedInput *services.DVRConfigInput
	svc := &mockDVRSettingsService{
		testConnection: func(ctx context.Context, plugin string, input *services.DVRConfigInput) error {
			receivedInput = input
			return nil
		},
	}
	router := setupDVRRouter(svc)

	body := `{"url":"http://candidate:7878","api_key":"k"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/radarr/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, receivedInput)
	assert.Equal(t, "http://candidate:7878", receivedInput.URL)
}

func TestDVRSettingsHandler_TestConnection_Failure(t *testing.T) {
	svc := &mockDVRSettingsService{
		testConnection: func(ctx context.Context, plugin string, input *services.DVRConfigInput) error {
			return &plugins.PluginError{Code: plugins.ErrCodeAuthFailed, Message: "radarr rejected the API key"}
		},
	}
	router := setupDVRRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/radarr/test", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "DVR_AUTH_FAILED", resp.Error.Code)
}

func TestDVRSettingsHandler_GetQualityProfiles(t *testing.T) {
	svc := &mockDVRSettingsService{
		getQualityProfiles: func(ctx context.Context, plugin string) ([]plugins.QualityProfile, error) {
			return []plugins.QualityProfile{{ID: 1, Name: "HD-1080p"}}, nil
		},
	}
	router := setupDVRRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/radarr/quality-profiles", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			QualityProfiles []plugins.QualityProfile `json:"quality_profiles"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.QualityProfiles, 1)
	assert.Equal(t, "HD-1080p", resp.Data.QualityProfiles[0].Name)
}

func TestDVRSettingsHandler_GetQualityProfiles_NotConfigured(t *testing.T) {
	svc := &mockDVRSettingsService{
		getQualityProfiles: func(ctx context.Context, plugin string) ([]plugins.QualityProfile, error) {
			return nil, &plugins.PluginError{Code: plugins.ErrCodeNotConfigured, Message: "radarr is not configured"}
		},
	}
	router := setupDVRRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/radarr/quality-profiles", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "DVR_NOT_CONFIGURED", resp.Error.Code)
}

func TestDVRSettingsHandler_GetRootFolders(t *testing.T) {
	svc := &mockDVRSettingsService{
		getRootFolders: func(ctx context.Context, plugin string) ([]plugins.RootFolder, error) {
			return []plugins.RootFolder{{ID: 1, Path: "/movies"}}, nil
		},
	}
	router := setupDVRRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/radarr/root-folders", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			RootFolders []plugins.RootFolder `json:"root_folders"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.RootFolders, 1)
	assert.Equal(t, "/movies", resp.Data.RootFolders[0].Path)
}
