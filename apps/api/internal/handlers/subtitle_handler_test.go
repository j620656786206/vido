package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/providers"
)

// --- Mock Provider for handler tests ---

type handlerMockProvider struct {
	name         string
	searchResult []providers.SubtitleResult
	searchErr    error
	downloadData []byte
	downloadErr  error
}

func (m *handlerMockProvider) Name() string { return m.name }
func (m *handlerMockProvider) Search(_ context.Context, _ providers.SubtitleQuery) ([]providers.SubtitleResult, error) {
	return m.searchResult, m.searchErr
}
func (m *handlerMockProvider) Download(_ context.Context, _ string) ([]byte, error) {
	return m.downloadData, m.downloadErr
}

// --- Mock Engine ---

type mockEngine struct {
	result subtitle.EngineResult
}

func (m *mockEngine) Process(_ context.Context, _, _, _ string, _ providers.SubtitleQuery, _ string) subtitle.EngineResult {
	return m.result
}

// --- Test Setup ---

func setupSubtitleHandler(t *testing.T) (*SubtitleHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	prov := &handlerMockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant", Filename: "sub.srt", Downloads: 100},
		},
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n測試\n"),
	}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	handler := NewSubtitleHandler(&mockEngine{}, []providers.SubtitleProvider{prov}, scorer, nil, nil)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	return handler, router
}

// --- Search Tests ---

func TestSubtitleHandler_Search_Success(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(SubtitleSearchRequest{
		MediaID:   "movie-1",
		MediaType: "movie",
		Query:     "Test Movie",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}

func TestSubtitleHandler_Search_MissingMediaID(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(map[string]string{
		"mediaType": "movie",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubtitleHandler_Search_InvalidMediaType(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(SubtitleSearchRequest{
		MediaID:   "1",
		MediaType: "invalid",
		Query:     "test",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubtitleHandler_Search_ProviderFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Two providers
	prov1 := &handlerMockProvider{name: "assrt", searchResult: []providers.SubtitleResult{
		{ID: "1", Source: "assrt", Language: "zh-Hant"},
	}}
	prov2 := &handlerMockProvider{name: "zimuku", searchResult: []providers.SubtitleResult{
		{ID: "2", Source: "zimuku", Language: "zh-Hant"},
	}}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	handler := NewSubtitleHandler(&mockEngine{}, []providers.SubtitleProvider{prov1, prov2}, scorer, nil, nil)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	// Request only assrt
	body, _ := json.Marshal(SubtitleSearchRequest{
		MediaID:   "1",
		MediaType: "movie",
		Providers: []string{"assrt"},
		Query:     "test",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubtitleHandler_Search_EmptyResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prov := &handlerMockProvider{name: "assrt", searchResult: nil}
	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	handler := NewSubtitleHandler(&mockEngine{}, []providers.SubtitleProvider{prov}, scorer, nil, nil)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body, _ := json.Marshal(SubtitleSearchRequest{
		MediaID:   "1",
		MediaType: "movie",
		Query:     "nonexistent",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Preview Tests ---

func TestSubtitleHandler_Preview_Success(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(SubtitlePreviewRequest{
		SubtitleID: "1",
		Provider:   "assrt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubtitleHandler_Preview_ProviderNotFound(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(SubtitlePreviewRequest{
		SubtitleID: "1",
		Provider:   "nonexistent",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubtitleHandler_Preview_DownloadFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prov := &handlerMockProvider{
		name:        "assrt",
		downloadErr: errors.New("download failed"),
	}

	handler := NewSubtitleHandler(&mockEngine{}, []providers.SubtitleProvider{prov},
		subtitle.NewScorer(subtitle.NewDefaultScorerConfig()), nil, nil)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body, _ := json.Marshal(SubtitlePreviewRequest{
		SubtitleID: "1",
		Provider:   "assrt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Helper function tests ---

func TestExtractFirstLines(t *testing.T) {
	data := []byte("1\n00:00:01,000 --> 00:00:03,000\nHello World\n\n2\n00:00:04,000 --> 00:00:06,000\nSecond line\n")
	lines := extractFirstLines(data, 5)
	assert.Len(t, lines, 5)
	assert.Equal(t, "1", lines[0])
	assert.Contains(t, lines[1], "-->")
}

func TestExtractFirstLines_Empty(t *testing.T) {
	lines := extractFirstLines([]byte{}, 10)
	assert.Empty(t, lines)
}
