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

// Regression for the 9R-15 startup panic: glossary routes must share the :id
// wildcard with the pre-existing /media/:id/* routes (metadata_handler) — gin
// panics at registration when wildcard names differ on the same segment.
func TestGlossaryHandler_RegisterRoutes_CoexistsWithMediaIDRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	apiV1 := r.Group("/api/v1")
	apiV1.PUT("/media/:id/metadata", func(c *gin.Context) {}) // mirrors metadata_handler
	require.NotPanics(t, func() {
		NewGlossaryHandler(&mockGlossaryService{}).RegisterRoutes(apiV1)
	})
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

// ─── sub-8-1: export / import ───────────────────────────────────────────────

type stubExchange struct {
	exportErr error
	importErr error
	gotFile   *services.GlossaryExport
}

func (s *stubExchange) Export(_ context.Context, mediaID string) (*services.GlossaryExport, error) {
	if s.exportErr != nil {
		return nil, s.exportErr
	}
	return &services.GlossaryExport{Format: "vido-glossary", Version: 1, Scope: "tmdb:tv:75006", Title: "Shadow and Bone",
		Terms: []services.GlossaryExportTerm{{TermSrc: "Ravka", TermZh: "拉夫卡", Source: "official_subtitle"}}}, nil
}
func (s *stubExchange) Import(_ context.Context, _ string, f *services.GlossaryExport) (*services.GlossaryImportResult, error) {
	s.gotFile = f
	if s.importErr != nil {
		return nil, s.importErr
	}
	return &services.GlossaryImportResult{Imported: 2, Skipped: 1, Conflicts: []services.GlossaryImportConflict{{ID: "g1", TermSrc: "Darkling", Mine: "闇之手", Theirs: "黑暗之主"}}}, nil
}

func exchangeRouter(x GlossaryExchanger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewGlossaryHandler(&mockGlossaryService{})
	if x != nil {
		h.WithExchange(x)
	}
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

const exportJSON = `{"format":"vido-glossary","version":1,"scope":"tmdb:tv:75006","terms":[{"term_src":"Grisha","term_zh":"格里沙","source":"official_subtitle","confirmed":true}]}`

func TestGlossaryHandler_ExportIsAFileDownload(t *testing.T) {
	w := httptest.NewRecorder()
	exchangeRouter(&stubExchange{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/media/s1/glossary/export", nil))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `attachment; filename="vido-glossary-tmdb-tv-75006.json"`, w.Header().Get("Content-Disposition"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "vido-glossary", body["format"], "the file itself, not an APIResponse envelope")
	_, wrapped := body["success"]
	assert.False(t, wrapped)

	w = httptest.NewRecorder()
	exchangeRouter(&stubExchange{exportErr: services.ErrGlossaryNotShareable}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/media/s1/glossary/export", nil))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "GLOSSARY_NOT_SHAREABLE")

	w = httptest.NewRecorder()
	exchangeRouter(nil).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/media/s1/glossary/export", nil))
	assert.Equal(t, http.StatusNotFound, w.Code, "exchange not wired → 404")
}

func TestGlossaryHandler_ImportJSONAndMultipart(t *testing.T) {
	x := &stubExchange{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media/s1/glossary/import", strings.NewReader(exportJSON))
	req.Header.Set("Content-Type", "application/json")
	exchangeRouter(x).ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, x.gotFile)
	assert.Equal(t, "Grisha", x.gotFile.Terms[0].TermSrc)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	assert.Equal(t, 2.0, data["imported"])
	assert.Equal(t, "黑暗之主", data["conflicts"].([]any)[0].(map[string]any)["theirs"])

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "vido-glossary-tmdb-tv-75006.json")
	_, _ = fw.Write([]byte(exportJSON))
	_ = mw.Close()
	x = &stubExchange{}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/media/s1/glossary/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	exchangeRouter(x).ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "tmdb:tv:75006", x.gotFile.Scope)
}

func TestGlossaryHandler_ImportErrors(t *testing.T) {
	post := func(x GlossaryExchanger, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/s1/glossary/import", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		exchangeRouter(x).ServeHTTP(w, req)
		return w
	}
	cases := []struct {
		x      *stubExchange
		body   string
		status int
		code   string
	}{
		{&stubExchange{}, `not json`, http.StatusBadRequest, "GLOSSARY_IMPORT_INVALID"},
		{&stubExchange{importErr: services.ErrGlossaryImportInvalid}, exportJSON, http.StatusBadRequest, "GLOSSARY_IMPORT_INVALID"},
		{&stubExchange{importErr: services.ErrGlossaryScopeMismatch}, exportJSON, http.StatusBadRequest, "GLOSSARY_SCOPE_MISMATCH"},
		{&stubExchange{importErr: services.ErrGlossaryNotShareable}, exportJSON, http.StatusConflict, "GLOSSARY_NOT_SHAREABLE"},
	}
	for _, tc := range cases {
		w := post(tc.x, tc.body)
		assert.Equal(t, tc.status, w.Code, tc.code)
		assert.Contains(t, w.Body.String(), tc.code)
	}
	big := `{"format":"vido-glossary","version":1,"scope":"tmdb:tv:1","terms":[{"term_src":"` + strings.Repeat("x", 3<<20) + `"}]}`
	w := post(&stubExchange{}, big)
	assert.Equal(t, http.StatusBadRequest, w.Code, "over 2 MB is refused before decoding the whole thing")
}
