package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
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

// --- Mock Status Updater ---

type mockStatusUpdater struct {
	lastID       string
	lastStatus   models.SubtitleStatus
	lastPath     string
	lastLanguage string
	lastScore    float64
	updateErr    error
}

func (m *mockStatusUpdater) UpdateSubtitleStatus(_ context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	m.lastID = id
	m.lastStatus = status
	m.lastPath = path
	m.lastLanguage = language
	m.lastScore = score
	return m.updateErr
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
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)

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

	// Verify snake_case JSON keys in response
	var rawResp map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rawResp))
	var dataItems []map[string]interface{}
	require.NoError(t, json.Unmarshal(rawResp["data"], &dataItems))
	require.Len(t, dataItems, 1)
	// Check snake_case keys exist
	assert.Contains(t, dataItems[0], "download_url")
	assert.Contains(t, dataItems[0], "score_breakdown")
}

func TestSubtitleHandler_Search_MissingMediaID(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(map[string]string{
		"media_type": "movie",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubtitleHandler_Search_InvalidMediaType(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(map[string]string{
		"media_id":   "1",
		"media_type": "invalid",
		"query":      "test",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubtitleHandler_Search_ProviderFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prov1 := &handlerMockProvider{name: "assrt", searchResult: []providers.SubtitleResult{
		{ID: "1", Source: "assrt", Language: "zh-Hant"},
	}}
	prov2 := &handlerMockProvider{name: "zimuku", searchResult: []providers.SubtitleResult{
		{ID: "2", Source: "zimuku", Language: "zh-Hant"},
	}}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov1, prov2},
		scorer, nil, placer, nil, nil, nil,
	)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

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
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)

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

// --- Download Tests ---

func TestSubtitleHandler_Download_ProviderNotFound(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(SubtitleDownloadRequest{
		MediaID:       "1",
		MediaType:     "movie",
		MediaFilePath: "/media/movie.mkv",
		SubtitleID:    "sub-1",
		Provider:      "nonexistent",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubtitleHandler_Download_DownloadFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prov := &handlerMockProvider{
		name:        "assrt",
		downloadErr: errors.New("download failed"),
	}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body, _ := json.Marshal(SubtitleDownloadRequest{
		MediaID:       "1",
		MediaType:     "movie",
		MediaFilePath: "/media/movie.mkv",
		SubtitleID:    "sub-1",
		Provider:      "assrt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubtitleHandler_Download_MissingFields(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	// Missing media_file_path
	body, _ := json.Marshal(map[string]string{
		"media_id":    "1",
		"media_type":  "movie",
		"subtitle_id": "sub-1",
		"provider":    "assrt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubtitleHandler_Download_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a temp dir with a fake media file
	tmpDir := t.TempDir()
	mediaPath := tmpDir + "/movie.mkv"
	if err := os.WriteFile(mediaPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	prov := &handlerMockProvider{
		name:         "assrt",
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n測試字幕\n"),
	}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	mockMovieRepo := &mockStatusUpdater{}
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, mockMovieRepo, nil,
	)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body, _ := json.Marshal(SubtitleDownloadRequest{
		MediaID:       "movie-1",
		MediaType:     "movie",
		MediaFilePath: mediaPath,
		SubtitleID:    "sub-1",
		Provider:      "assrt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Verify DB was updated
	assert.Equal(t, "movie-1", mockMovieRepo.lastID)
	assert.Equal(t, models.SubtitleStatusFound, mockMovieRepo.lastStatus)
}

func TestSubtitleHandler_Download_WithConvertToTraditionalFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	mediaPath := tmpDir + "/movie.mkv"
	os.WriteFile(mediaPath, []byte("fake"), 0644)

	// Simplified Chinese content
	prov := &handlerMockProvider{
		name:         "assrt",
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n这是简体中文测试\n"),
	}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	convertFalse := false
	body, _ := json.Marshal(SubtitleDownloadRequest{
		MediaID:              "movie-1",
		MediaType:            "movie",
		MediaFilePath:        mediaPath,
		SubtitleID:           "sub-1",
		Provider:             "assrt",
		ConvertToTraditional: &convertFalse,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify response contains language info
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	// Without conversion, language should be zh-Hans (simplified detected)
	assert.Contains(t, data["language"], "zh")
}

// --- CN Conversion Policy Tests ---

func TestSubtitleHandler_ShouldConvert(t *testing.T) {
	h := &SubtitleHandler{}
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name         string
		detectedLang string
		userOverride *bool
		want         bool
	}{
		{"zh-Hans no override → convert", "zh-Hans", nil, true},
		{"zh-Hant → no convert", "zh-Hant", nil, false},
		{"en → no convert", "en", nil, false},
		{"zh-Hans with CN override OFF → no convert", "zh-Hans", boolPtr(false), false},
		{"zh-Hans with override ON → convert", "zh-Hans", boolPtr(true), true},
		{"zh-Hant with override ON → no convert (already traditional)", "zh-Hant", boolPtr(true), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.shouldConvert(tt.detectedLang, tt.userOverride)
			assert.Equal(t, tt.want, got)
		})
	}
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

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)

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

// --- Acceptance Tests: Coverage Gaps ---

// AC #2: Verify search returns scored results with all expected DTO fields
func TestSubtitleHandler_Search_ResponseFields(t *testing.T) {
	_, router := setupSubtitleHandler(t)

	body, _ := json.Marshal(SubtitleSearchRequest{
		MediaID:   "movie-1",
		MediaType: "movie",
		Query:     "Test",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rawResp))
	var items []map[string]interface{}
	require.NoError(t, json.Unmarshal(rawResp["data"], &items))
	require.NotEmpty(t, items)

	item := items[0]
	// AC #2: Scored results must include all DTO fields (snake_case)
	for _, field := range []string{"id", "source", "filename", "language", "download_url", "downloads", "group", "resolution", "format", "score", "score_breakdown"} {
		assert.Contains(t, item, field, "missing field: %s", field)
	}
	// score_breakdown must have all sub-fields
	sb := item["score_breakdown"].(map[string]interface{})
	for _, field := range []string{"language", "resolution", "source_trust", "group", "downloads"} {
		assert.Contains(t, sb, field, "missing score_breakdown field: %s", field)
	}
}

// AC #4: Preview returns exactly first 10 non-empty lines
func TestSubtitleHandler_Preview_ReturnsFirst10Lines(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create SRT content with more than 10 non-empty lines
	srtContent := "1\n00:00:01,000 --> 00:00:03,000\nLine A\n\n2\n00:00:04,000 --> 00:00:06,000\nLine B\n\n3\n00:00:07,000 --> 00:00:09,000\nLine C\n\n4\n00:00:10,000 --> 00:00:12,000\nLine D\n\n5\n00:00:13,000 --> 00:00:15,000\nLine E\n\n6\n00:00:16,000 --> 00:00:18,000\nLine F\n"

	prov := &handlerMockProvider{
		name:         "assrt",
		downloadData: []byte(srtContent),
	}
	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)
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

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	lines := data["lines"].([]interface{})
	// AC #4: exactly 10 non-empty lines
	assert.Len(t, lines, 10)
	assert.Contains(t, data, "language")
}

// AC #8: Download response contains all required fields with values
func TestSubtitleHandler_Download_ResponseFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	mediaPath := tmpDir + "/movie.mkv"
	os.WriteFile(mediaPath, []byte("fake"), 0644)

	prov := &handlerMockProvider{
		name:         "assrt",
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n字幕測試\n"),
	}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body, _ := json.Marshal(SubtitleDownloadRequest{
		MediaID:       "movie-1",
		MediaType:     "movie",
		MediaFilePath: mediaPath,
		SubtitleID:    "sub-1",
		Provider:      "assrt",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	// AC #8: response must contain all 3 fields
	assert.Contains(t, data, "subtitle_path")
	assert.Contains(t, data, "language")
	assert.Contains(t, data, "score")
	// language should be detected and non-empty
	assert.NotEmpty(t, data["language"])
}

// AC #2: Multiple providers searched (verify both contribute results)
func TestSubtitleHandler_Search_MultipleProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prov1 := &handlerMockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "a1", Source: "assrt", Language: "zh-Hant", Filename: "a.srt"},
		},
	}
	prov2 := &handlerMockProvider{
		name: "zimuku",
		searchResult: []providers.SubtitleResult{
			{ID: "z1", Source: "zimuku", Language: "zh-Hant", Filename: "z.srt"},
		},
	}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov1, prov2},
		scorer, nil, placer, nil, nil, nil,
	)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	// Search without provider filter = all providers
	body, _ := json.Marshal(SubtitleSearchRequest{
		MediaID:   "1",
		MediaType: "movie",
		Query:     "test",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &rawResp)
	var items []map[string]interface{}
	json.Unmarshal(rawResp["data"], &items)
	// AC #2: Both providers contribute results (at least 2)
	assert.GreaterOrEqual(t, len(items), 2, "Expected results from both providers")

	// Verify both sources are represented
	sources := make(map[string]bool)
	for _, item := range items {
		sources[item["source"].(string)] = true
	}
	assert.True(t, sources["assrt"], "Missing results from assrt provider")
	assert.True(t, sources["zimuku"], "Missing results from zimuku provider")
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

// --- Batch Handler Tests (Story 8-9) ---

func setupBatchHandler(t *testing.T) (*SubtitleHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	prov := &handlerMockProvider{
		name:         "assrt",
		searchResult: []providers.SubtitleResult{{ID: "1", Source: "assrt", Language: "zh-Hant"}},
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n測試\n"),
	}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)

	// Wire batch processor with empty collector (no items)
	collector := &emptyBatchCollector{}
	bp := subtitle.NewBatchProcessor(
		subtitle.NewEngine([]providers.SubtitleProvider{prov}, scorer, nil, nil, nil, &mockStatusUpdater{}, &mockStatusUpdater{}),
		nil, collector, subtitle.BatchConfig{DelayBetweenItems: 1 * time.Millisecond},
	)
	handler.SetBatchProcessor(bp)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	return handler, router
}

// emptyBatchCollector returns no items.
type emptyBatchCollector struct{}

func (e *emptyBatchCollector) CollectMoviesNeedingSubtitles(_ context.Context) ([]subtitle.BatchItem, error) {
	return nil, nil
}
func (e *emptyBatchCollector) CollectSeriesNeedingSubtitles(_ context.Context) ([]subtitle.BatchItem, error) {
	return nil, nil
}

// AC #4: POST /batch returns 202 with batch_id + total_items
func TestSubtitleHandler_StartBatch_Returns202(t *testing.T) {
	_, router := setupBatchHandler(t)

	body, _ := json.Marshal(BatchStartRequest{
		Scope: "library",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 202, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Contains(t, data, "batch_id")
	assert.Contains(t, data, "total_items")
}

// AC #4: Invalid scope returns 400
func TestSubtitleHandler_StartBatch_InvalidScope(t *testing.T) {
	_, router := setupBatchHandler(t)

	body, _ := json.Marshal(map[string]string{"scope": "invalid"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// AC #4: Season scope requires season_id
func TestSubtitleHandler_StartBatch_SeasonRequiresID(t *testing.T) {
	_, router := setupBatchHandler(t)

	body, _ := json.Marshal(BatchStartRequest{Scope: "season"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// GET /batch/status returns running:false when no batch
func TestSubtitleHandler_GetBatchStatus_Idle(t *testing.T) {
	_, router := setupBatchHandler(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subtitles/batch/status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, false, data["running"])
}

// --- TA 8-9: Additional Handler Coverage Tests ---

// batchCollectorWithItems returns configurable items for batch tests.
type batchCollectorWithItems struct {
	movies []subtitle.BatchItem
}

func (b *batchCollectorWithItems) CollectMoviesNeedingSubtitles(_ context.Context) ([]subtitle.BatchItem, error) {
	return b.movies, nil
}
func (b *batchCollectorWithItems) CollectSeriesNeedingSubtitles(_ context.Context) ([]subtitle.BatchItem, error) {
	return nil, nil
}

// setupBatchHandlerWithItems creates a handler with a batch processor that has items to process.
func setupBatchHandlerWithItems(t *testing.T) (*SubtitleHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	prov := &handlerMockProvider{
		name:         "assrt",
		searchResult: []providers.SubtitleResult{{ID: "1", Source: "assrt", Language: "zh-Hant"}},
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n測試\n"),
	}

	scorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		scorer, nil, placer, nil, nil, nil,
	)

	items := []subtitle.BatchItem{
		{MediaID: "m1", MediaType: "movie", Title: "Movie 1"},
		{MediaID: "m2", MediaType: "movie", Title: "Movie 2"},
		{MediaID: "m3", MediaType: "movie", Title: "Movie 3"},
	}
	collector := &batchCollectorWithItems{movies: items}
	bp := subtitle.NewBatchProcessor(
		subtitle.NewEngine([]providers.SubtitleProvider{prov}, scorer, nil, nil, nil, &mockStatusUpdater{}, &mockStatusUpdater{}),
		nil, collector, subtitle.BatchConfig{DelayBetweenItems: 100 * time.Millisecond},
	)
	handler.SetBatchProcessor(bp)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	return handler, router
}

// AC #7: [P0] POST /batch while running returns 409 with progress data
func TestSubtitleHandler_StartBatch_Conflict409(t *testing.T) {
	_, router := setupBatchHandlerWithItems(t)

	body, _ := json.Marshal(BatchStartRequest{Scope: "library"})

	// Start first batch
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, req1)
	assert.Equal(t, 202, w1.Code)

	// Brief wait to ensure batch is running
	time.Sleep(20 * time.Millisecond)

	// Start second batch — should get 409
	w2 := httptest.NewRecorder()
	body2, _ := json.Marshal(BatchStartRequest{Scope: "library"})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)

	assert.Equal(t, 409, w2.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["success"])

	// Verify error structure
	errObj, ok := resp["error"].(map[string]interface{})
	require.True(t, ok, "Expected error object in response")
	assert.Equal(t, "SUBTITLE_BATCH_RUNNING", errObj["code"])

	// Verify progress data is included
	assert.Contains(t, resp, "data", "409 response should include progress data")

	// Wait for completion
	time.Sleep(500 * time.Millisecond)
}

// [P1] POST /batch with empty body returns 400
func TestSubtitleHandler_StartBatch_EmptyBody(t *testing.T) {
	_, router := setupBatchHandler(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// [P1] POST /batch with no Content-Type returns 400
func TestSubtitleHandler_StartBatch_NoContentType(t *testing.T) {
	_, router := setupBatchHandler(t)

	body, _ := json.Marshal(BatchStartRequest{Scope: "library"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader(body))
	// No Content-Type header
	router.ServeHTTP(w, req)

	// Gin should still accept it or return 400 — key is no panic
	assert.Contains(t, []int{200, 202, 400}, w.Code)
}

// [P1] GET /batch/status while batch running returns progress
func TestSubtitleHandler_GetBatchStatus_Running(t *testing.T) {
	_, router := setupBatchHandlerWithItems(t)

	// Start a batch first
	body, _ := json.Marshal(BatchStartRequest{Scope: "library"})
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, req1)
	require.Equal(t, 202, w1.Code)

	// Brief wait for background processing to start
	time.Sleep(20 * time.Millisecond)

	// Check status
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/subtitles/batch/status", nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["running"])
	assert.Contains(t, data, "progress", "Running response should include progress object")

	progress, ok := data["progress"].(map[string]interface{})
	require.True(t, ok, "progress should be a JSON object")
	assert.Contains(t, progress, "batch_id")
	assert.Contains(t, progress, "total_items")
	assert.Equal(t, "running", progress["status"])

	// Wait for completion
	time.Sleep(500 * time.Millisecond)
}

// No batch processor returns error
func TestSubtitleHandler_StartBatch_NoBatchProcessor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSubtitleHandler(nil, nil, nil, nil, nil, nil, nil)
	// Don't set batch processor

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body, _ := json.Marshal(BatchStartRequest{Scope: "library"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
