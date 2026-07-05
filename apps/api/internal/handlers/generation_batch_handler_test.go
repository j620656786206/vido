package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
)

// ─── Mock processor ─────────────────────────────────────────────────────────

var errBoom = errors.New("boom")

type mockGenerationProcessor struct {
	available bool
	running   bool
	batchID   string
	items     []services.GenerationBatchItem
	startErr  error
	progress  *services.GenerationBatchProgress
	preview   int
	prevErr   error

	startedScope string
	startedIDs   []int64
	cancelCalled bool
}

func (m *mockGenerationProcessor) IsAvailable() bool { return m.available }
func (m *mockGenerationProcessor) IsRunning() bool   { return m.running }
func (m *mockGenerationProcessor) Start(_ context.Context, scope string, mediaIDs []int64) (string, []services.GenerationBatchItem, error) {
	m.startedScope = scope
	m.startedIDs = mediaIDs
	if m.startErr != nil {
		return "", nil, m.startErr
	}
	return m.batchID, m.items, nil
}
func (m *mockGenerationProcessor) GetProgress() *services.GenerationBatchProgress { return m.progress }
func (m *mockGenerationProcessor) Cancel()                                        { m.cancelCalled = true }
func (m *mockGenerationProcessor) PreviewMissing(_ context.Context) (int, error) {
	return m.preview, m.prevErr
}

func setupGenerationBatchRouter(p GenerationBatchProcessorInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	NewGenerationBatchHandler(p).RegisterRoutes(api)
	return r
}

func doGenBatchJSON(t *testing.T, r *gin.Engine, method, path, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return w, resp
}

func errCode(t *testing.T, resp map[string]interface{}) string {
	t.Helper()
	errObj, ok := resp["error"].(map[string]interface{})
	require.True(t, ok, "expected error object, got %v", resp)
	return errObj["code"].(string)
}

// ─── POST /subtitles/generation-batch ────────────────────────────────────────

func TestStartGenerationBatch_Disabled503(t *testing.T) {
	r := setupGenerationBatchRouter(&mockGenerationProcessor{available: false})
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"missing"}`)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "TRANSCRIPTION_DISABLED", errCode(t, resp))
}

func TestStartGenerationBatch_BadScope400(t *testing.T) {
	r := setupGenerationBatchRouter(&mockGenerationProcessor{available: true})
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"everything"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_INVALID_FORMAT", errCode(t, resp))
}

func TestStartGenerationBatch_SelectedWithoutIDs400(t *testing.T) {
	r := setupGenerationBatchRouter(&mockGenerationProcessor{available: true})
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"selected"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_REQUIRED_FIELD", errCode(t, resp))
}

func TestStartGenerationBatch_MissingWithIDs400(t *testing.T) {
	r := setupGenerationBatchRouter(&mockGenerationProcessor{available: true})
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"missing","media_ids":[1]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_INVALID_FORMAT", errCode(t, resp))
}

func TestStartGenerationBatch_Running409WithProgress(t *testing.T) {
	p := &mockGenerationProcessor{
		available: true,
		startErr:  services.ErrGenerationBatchRunning,
		progress: &services.GenerationBatchProgress{
			BatchID: "b-1", TotalItems: 38, CurrentIndex: 12,
			Status: services.GenerationBatchStatusRunning,
		},
	}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"missing"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "TRANSCRIPTION_BATCH_RUNNING", errCode(t, resp))

	// Mirror SUBTITLE_BATCH_RUNNING: current progress rides the error body.
	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "409 must carry progress in data")
	assert.Equal(t, "b-1", data["batch_id"])
	assert.Equal(t, float64(38), data["total_items"])
}

func TestStartGenerationBatch_InvalidSelection400(t *testing.T) {
	p := &mockGenerationProcessor{available: true, startErr: services.ErrGenerationSelectionInvalid}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"selected","media_ids":[999]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_INVALID_FORMAT", errCode(t, resp))
}

func TestStartGenerationBatch_StartFailed500(t *testing.T) {
	p := &mockGenerationProcessor{available: true, startErr: errBoom}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"missing"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "TRANSCRIPTION_BATCH_START_FAILED", errCode(t, resp))
}

func TestStartGenerationBatch_Accepted202(t *testing.T) {
	p := &mockGenerationProcessor{
		available: true,
		batchID:   "batch-abc",
		items: []services.GenerationBatchItem{
			{MediaID: 1, Title: "Alpha"},
			{MediaID: 2, Title: "Bravo"},
		},
	}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"missing"}`)
	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Equal(t, true, resp["success"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "batch-abc", data["batch_id"])
	assert.Equal(t, float64(2), data["total_items"])
	items := data["items"].([]interface{})
	require.Len(t, items, 2)
	first := items[0].(map[string]interface{})
	assert.Equal(t, float64(1), first["media_id"])
	assert.Equal(t, "Alpha", first["title"])
	assert.Equal(t, "missing", p.startedScope)
}

func TestStartGenerationBatch_SelectedForwardsIDs(t *testing.T) {
	p := &mockGenerationProcessor{
		available: true,
		batchID:   "batch-sel",
		items:     []services.GenerationBatchItem{{MediaID: 9, Title: "Nine"}},
	}
	r := setupGenerationBatchRouter(p)
	w, _ := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"selected","media_ids":[9,7]}`)
	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Equal(t, "selected", p.startedScope)
	assert.Equal(t, []int64{9, 7}, p.startedIDs)
}

// AC 1: empty missing scope → 200, not an error.
func TestStartGenerationBatch_EmptyMissingScope200(t *testing.T) {
	p := &mockGenerationProcessor{available: true, items: []services.GenerationBatchItem{}}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch", `{"scope":"missing"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, resp["success"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["total_items"])
	items, ok := data["items"].([]interface{})
	require.True(t, ok, "items must serialize as [], not null")
	assert.Empty(t, items)
}

// ─── GET /status ────────────────────────────────────────────────────────────

func TestGetGenerationBatchStatus_Idle(t *testing.T) {
	r := setupGenerationBatchRouter(&mockGenerationProcessor{available: true})
	w, resp := doGenBatchJSON(t, r, "GET", "/api/v1/subtitles/generation-batch/status", "")
	assert.Equal(t, http.StatusOK, w.Code)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, false, data["running"])
	assert.Nil(t, data["progress"])
}

func TestGetGenerationBatchStatus_Running(t *testing.T) {
	p := &mockGenerationProcessor{
		available: true, running: true,
		progress: &services.GenerationBatchProgress{
			BatchID: "b-2", TotalItems: 10, CurrentIndex: 3, CurrentMediaID: 55,
			SuccessCount: 2, Status: services.GenerationBatchStatusRunning,
			SpentUSD: 0.42, BudgetUSD: 5.0,
		},
	}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "GET", "/api/v1/subtitles/generation-batch/status", "")
	assert.Equal(t, http.StatusOK, w.Code)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["running"])
	prog := data["progress"].(map[string]interface{})
	assert.Equal(t, "b-2", prog["batch_id"])
	assert.Equal(t, float64(55), prog["current_media_id"])
	assert.Equal(t, 0.42, prog["spent_usd"])
	assert.Equal(t, 5.0, prog["budget_usd"])
}

// ─── POST /cancel ───────────────────────────────────────────────────────────

func TestCancelGenerationBatch_IdleIdempotent(t *testing.T) {
	p := &mockGenerationProcessor{available: true, running: false}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch/cancel", "")
	assert.Equal(t, http.StatusOK, w.Code)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, false, data["cancelled"])
	assert.Equal(t, false, data["running"])
	assert.False(t, p.cancelCalled)
}

func TestCancelGenerationBatch_Running(t *testing.T) {
	p := &mockGenerationProcessor{available: true, running: true}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "POST", "/api/v1/subtitles/generation-batch/cancel", "")
	assert.Equal(t, http.StatusOK, w.Code)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["cancelled"])
	assert.True(t, p.cancelCalled)
}

// ─── GET /preview ───────────────────────────────────────────────────────────

func TestPreviewGenerationBatch_Missing200(t *testing.T) {
	p := &mockGenerationProcessor{available: true, preview: 38}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "GET", "/api/v1/subtitles/generation-batch/preview?scope=missing", "")
	assert.Equal(t, http.StatusOK, w.Code)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(38), data["total_items"])
}

func TestPreviewGenerationBatch_WrongScope400(t *testing.T) {
	r := setupGenerationBatchRouter(&mockGenerationProcessor{available: true})
	for _, q := range []string{"", "?scope=selected", "?scope=all"} {
		w, resp := doGenBatchJSON(t, r, "GET", "/api/v1/subtitles/generation-batch/preview"+q, "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "VALIDATION_INVALID_FORMAT", errCode(t, resp))
	}
}

func TestPreviewGenerationBatch_QueryFailed500(t *testing.T) {
	p := &mockGenerationProcessor{available: true, prevErr: errBoom}
	r := setupGenerationBatchRouter(p)
	w, resp := doGenBatchJSON(t, r, "GET", "/api/v1/subtitles/generation-batch/preview?scope=missing", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "DB_QUERY_FAILED", errCode(t, resp))
}
