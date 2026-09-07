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
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// settingsRepoStub is an in-memory repository.SettingsRepositoryInterface.
type settingsRepoStub struct{ values map[string]string }

func (r *settingsRepoStub) Set(_ context.Context, s *models.Setting) error {
	r.values[s.Key] = s.Value
	return nil
}
func (r *settingsRepoStub) Get(_ context.Context, key string) (*models.Setting, error) {
	v, ok := r.values[key]
	if !ok {
		return nil, fmt.Errorf("setting with key %s not found", key)
	}
	return &models.Setting{Key: key, Value: v, Type: "string"}, nil
}
func (r *settingsRepoStub) GetAll(context.Context) ([]models.Setting, error) { return nil, nil }
func (r *settingsRepoStub) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}
func (r *settingsRepoStub) GetString(ctx context.Context, key string) (string, error) {
	s, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}
func (r *settingsRepoStub) GetInt(context.Context, string) (int, error)   { return 0, nil }
func (r *settingsRepoStub) GetBool(context.Context, string) (bool, error) { return false, nil }
func (r *settingsRepoStub) SetString(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}
func (r *settingsRepoStub) SetInt(context.Context, string, int) error   { return nil }
func (r *settingsRepoStub) SetBool(context.Context, string, bool) error { return nil }

func setupLocalizationRouter(t *testing.T, stored string, env string) (*gin.Engine, *settingsRepoStub) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := &settingsRepoStub{values: map[string]string{}}
	if stored != "" {
		repo.values[services.SettingKeyLocalizationLevel] = stored
	}
	r := gin.New()
	NewLocalizationHandler(services.NewLocalizationSettingsService(repo, env, nil)).RegisterRoutes(r.Group("/api/v1"))
	return r, repo
}

func TestLocalizationHandler_Get(t *testing.T) {
	r, _ := setupLocalizationRouter(t, "", "ott")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subtitles/localization", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data LocalizationResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ott", string(body.Data.Level))
	assert.Equal(t, "env", string(body.Data.Source))
	assert.Len(t, body.Data.Levels, 3)
}

func TestLocalizationHandler_Put(t *testing.T) {
	r, repo := setupLocalizationRouter(t, "", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/v1/subtitles/localization", strings.NewReader(`{"level":"literal"}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "literal", repo.values[services.SettingKeyLocalizationLevel])
	var body struct {
		Data LocalizationResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "settings", string(body.Data.Source))

	// Unknown level → 400, nothing stored.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/v1/subtitles/localization", strings.NewReader(`{"level":"netflix"}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "literal", repo.values[services.SettingKeyLocalizationLevel])

	// Missing body → 400.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/v1/subtitles/localization", strings.NewReader(`{}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
