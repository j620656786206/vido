package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// --- Mock service ---

type mockFilterPresetService struct {
	getAllResp []models.FilterPreset
	getAllErr  error

	createReq  services.CreateFilterPresetRequest
	createResp *models.FilterPreset
	createErr  error

	deleteID  string
	deleteErr error
}

func (m *mockFilterPresetService) GetAllPresets(_ context.Context) ([]models.FilterPreset, error) {
	return m.getAllResp, m.getAllErr
}
func (m *mockFilterPresetService) CreatePreset(_ context.Context, req services.CreateFilterPresetRequest) (*models.FilterPreset, error) {
	m.createReq = req
	return m.createResp, m.createErr
}
func (m *mockFilterPresetService) DeletePreset(_ context.Context, id string) error {
	m.deleteID = id
	return m.deleteErr
}

var _ services.FilterPresetServiceInterface = (*mockFilterPresetService)(nil)

func setupFilterPresetRouter(svc services.FilterPresetServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewFilterPresetsHandler(svc)
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestFilterPresetsHandler_ListPresets(t *testing.T) {
	svc := &mockFilterPresetService{getAllResp: []models.FilterPreset{
		{ID: "p1", Name: "韓劇", Filters: `{"region":"KR"}`},
	}}
	r := setupFilterPresetRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/filter-presets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Presets []models.FilterPreset `json:"presets"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	require.Len(t, resp.Data.Presets, 1)
	assert.Equal(t, "韓劇", resp.Data.Presets[0].Name)
}

func TestFilterPresetsHandler_ListPresets_EmptyArrayNotNull(t *testing.T) {
	svc := &mockFilterPresetService{getAllResp: nil}
	r := setupFilterPresetRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/filter-presets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"presets":[]`)
}

func TestFilterPresetsHandler_CreatePreset(t *testing.T) {
	svc := &mockFilterPresetService{createResp: &models.FilterPreset{ID: "new", Name: "高評分動畫"}}
	r := setupFilterPresetRouter(svc)

	body, _ := json.Marshal(map[string]string{"name": "高評分動畫", "filters": `{"genre":"16"}`})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/filter-presets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "高評分動畫", svc.createReq.Name)
	assert.Equal(t, `{"genre":"16"}`, svc.createReq.Filters)
}

func TestFilterPresetsHandler_CreatePreset_ValidationError(t *testing.T) {
	svc := &mockFilterPresetService{createErr: &models.ValidationError{Field: "name", Message: "required"}}
	r := setupFilterPresetRouter(svc)

	body, _ := json.Marshal(map[string]string{"name": "", "filters": "{}"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/filter-presets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), errCodeFilterPresetValidation)
}

func TestFilterPresetsHandler_CreatePreset_LimitReached(t *testing.T) {
	svc := &mockFilterPresetService{createErr: services.ErrFilterPresetLimitReached}
	r := setupFilterPresetRouter(svc)

	body, _ := json.Marshal(map[string]string{"name": "超出", "filters": "{}"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/filter-presets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), errCodeFilterPresetLimit)
}

func TestFilterPresetsHandler_DeletePreset(t *testing.T) {
	svc := &mockFilterPresetService{}
	r := setupFilterPresetRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/filter-presets/p1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "p1", svc.deleteID)
}

func TestFilterPresetsHandler_DeletePreset_NotFound(t *testing.T) {
	svc := &mockFilterPresetService{deleteErr: repository.ErrFilterPresetNotFound}
	r := setupFilterPresetRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/filter-presets/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), errCodeFilterPresetNotFound)
}
