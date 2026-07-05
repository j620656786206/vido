package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// mockGlossaryService records calls and returns canned results.
type mockGlossaryService struct {
	listResp      []models.GlossaryTerm
	listErr       error
	addErr        error
	editErr       error
	confirmErr    error
	confirmAll    int64
	confirmAllErr error
	deleteErr     error

	lastAdd        *models.GlossaryTerm
	lastEditMedia  string
	lastEditID     string
	lastConfirmAll string
	lastDeleteID   string
}

func (m *mockGlossaryService) List(ctx context.Context, mediaID string) ([]models.GlossaryTerm, error) {
	return m.listResp, m.listErr
}
func (m *mockGlossaryService) Add(ctx context.Context, term *models.GlossaryTerm) error {
	m.lastAdd = term
	return m.addErr
}
func (m *mockGlossaryService) Edit(ctx context.Context, mediaID, id, termZh string, confirmed bool) error {
	m.lastEditMedia, m.lastEditID = mediaID, id
	return m.editErr
}
func (m *mockGlossaryService) Confirm(ctx context.Context, mediaID, id string) error {
	return m.confirmErr
}
func (m *mockGlossaryService) ConfirmAll(ctx context.Context, mediaID string) (int64, error) {
	m.lastConfirmAll = mediaID
	return m.confirmAll, m.confirmAllErr
}
func (m *mockGlossaryService) Delete(ctx context.Context, mediaID, id string) error {
	m.lastDeleteID = id
	return m.deleteErr
}

func setupGlossaryRouter(svc services.GlossaryServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewGlossaryHandler(svc).RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestGlossaryHandler_List(t *testing.T) {
	svc := &mockGlossaryService{listResp: []models.GlossaryTerm{{ID: "g1", TermSrc: "Vecna", TermZh: "維克那"}}}
	r := setupGlossaryRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/media/42/glossary", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data struct {
			Terms []models.GlossaryTerm `json:"terms"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Terms, 1)
	assert.Equal(t, "維克那", body.Data.Terms[0].TermZh)
}

func TestGlossaryHandler_List_NeverNull(t *testing.T) {
	r := setupGlossaryRouter(&mockGlossaryService{listResp: nil})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/media/42/glossary", nil))
	assert.Contains(t, w.Body.String(), `"terms":[]`)
}

func TestGlossaryHandler_Add_RouteMediaWins(t *testing.T) {
	svc := &mockGlossaryService{}
	r := setupGlossaryRouter(svc)
	// Body tries to smuggle a different media_id — must be ignored.
	body := `{"term_src":"Demogorgon","term_zh":"魔王獸","media_id":"999"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media/42/glossary", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, svc.lastAdd)
	assert.Equal(t, "42", svc.lastAdd.MediaID, "route media id is authoritative")
	assert.Equal(t, "Demogorgon", svc.lastAdd.TermSrc)
}

func TestGlossaryHandler_ConfirmAll(t *testing.T) {
	svc := &mockGlossaryService{confirmAll: 3}
	r := setupGlossaryRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media/42/glossary/confirm-all", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "42", svc.lastConfirmAll)
	assert.Contains(t, w.Body.String(), `"confirmed":3`)
}

func TestGlossaryHandler_Edit_And_Delete(t *testing.T) {
	svc := &mockGlossaryService{}
	r := setupGlossaryRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/media/42/glossary/g1", strings.NewReader(`{"term_zh":"新譯","confirmed":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "g1", svc.lastEditID)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/media/42/glossary/g1", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "g1", svc.lastDeleteID)
}

func TestGlossaryHandler_NotFound(t *testing.T) {
	svc := &mockGlossaryService{confirmErr: repository.ErrGlossaryTermNotFound}
	r := setupGlossaryRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/media/42/glossary/nope/confirm", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "DB_NOT_FOUND")
}

func TestGlossaryHandler_ValidationError(t *testing.T) {
	svc := &mockGlossaryService{addErr: &models.ValidationError{Field: "term_src", Message: "term_src is required"}}
	r := setupGlossaryRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media/42/glossary", strings.NewReader(`{"term_zh":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
}
