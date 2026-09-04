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

	"github.com/vido/api/internal/subtitle"
)

// ─── Fakes ─────────────────────────────────────────────────────────────────

type fakePipelineQueue struct {
	running bool
	outcome subtitle.EnqueueOutcome
	calls   []subtitle.MediaRef
	opts    []subtitle.ProcessItemOptions
}

func (q *fakePipelineQueue) IsRunning() bool { return q.running }

func (q *fakePipelineQueue) EnqueueItem(ref subtitle.MediaRef, opts subtitle.ProcessItemOptions) subtitle.EnqueueOutcome {
	q.calls = append(q.calls, ref)
	q.opts = append(q.opts, opts)
	return q.outcome
}

type fakeMediaLookup struct {
	item *subtitle.MediaItem
	err  error
	refs []subtitle.MediaRef
}

func (l *fakeMediaLookup) Load(_ context.Context, ref subtitle.MediaRef) (*subtitle.MediaItem, error) {
	l.refs = append(l.refs, ref)
	if l.err != nil {
		return nil, l.err
	}
	return l.item, nil
}

func runningQueue() *fakePipelineQueue {
	return &fakePipelineQueue{running: true, outcome: subtitle.EnqueueAccepted}
}

func foundMedia() *fakeMediaLookup {
	return &fakeMediaLookup{item: &subtitle.MediaItem{FilePath: "/media/Movie.mkv"}}
}

func newPipelineRouter(h *SubtitlePipelineHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func postRun(t *testing.T, h *SubtitlePipelineHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/pipeline/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newPipelineRouter(h).ServeHTTP(w, req)
	return w
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var resp APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

const runMovieID = "3f1c9a54-0b2e-4d7a-9a11-6c5d4e3f2a10"

// ─── AC #4: the endpoint table ─────────────────────────────────────────────

func TestRunPipeline_QueuesAndReturns202(t *testing.T) {
	queue := runningQueue()
	h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusAccepted, w.Code,
		"never synchronous — a translate run is minutes and progress flows over SSE")
	resp := decodeEnvelope(t, w)
	assert.True(t, resp.Success)
	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "queued", data["status"])
	assert.Equal(t, runMovieID, data["media_id"])

	require.Len(t, queue.calls, 1)
	assert.Equal(t, subtitle.MediaRef{ID: runMovieID, MediaType: "movie"}, queue.calls[0])
	assert.False(t, queue.opts[0].Force)
}

func TestRunPipeline_AlreadyQueuedIsStill202(t *testing.T) {
	queue := &fakePipelineQueue{running: true, outcome: subtitle.EnqueueDuplicate}
	h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusAccepted, w.Code,
		"a duplicate submit is not an error — the item IS being processed, which is what the caller wanted")
	data := decodeEnvelope(t, w).Data.(map[string]interface{})
	assert.Equal(t, "already_queued", data["status"])
}

// TestRunPipeline_QueueFullIs503NotAlreadyQueued — CR M1. A dropped item must
// never be reported as "already_queued": that answer promises work that was in
// fact discarded, and the caller would wait for SSE progress that never comes.
func TestRunPipeline_QueueFullIs503NotAlreadyQueued(t *testing.T) {
	queue := &fakePipelineQueue{running: true, outcome: subtitle.EnqueueQueueFull}
	h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	body := decodeEnvelope(t, w)
	assert.Equal(t, "SUBTITLE_QUEUE_FULL", body.Error.Code)
	assert.Equal(t, "字幕生成佇列已滿", body.Error.Message)
}

// TestRunPipeline_PoolStoppedMidRequestIs409 — the IsRunning pre-check can race
// a shutdown; the enqueue outcome is the truth and must not read as a 202.
func TestRunPipeline_PoolStoppedMidRequestIs409(t *testing.T) {
	queue := &fakePipelineQueue{running: true, outcome: subtitle.EnqueueStopped}
	h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "AI_NOT_CONFIGURED", decodeEnvelope(t, w).Error.Code)
}

func TestRunPipeline_ForceRidesThroughToTheQueue(t *testing.T) {
	queue := runningQueue()
	h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie","force":true}`)

	require.Equal(t, http.StatusAccepted, w.Code)
	require.Len(t, queue.opts, 1)
	assert.True(t, queue.opts[0].Force, "FR32's re-run switch must reach ProcessItem")
}

func TestRunPipeline_RejectsUnknownMediaType(t *testing.T) {
	queue := runningQueue()
	h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true }, nil)

	// `tv` is the TMDB/request vocabulary; the pipeline's is movie|series|episode
	// (sub-1-2 AC #1). Accepting it would enqueue an id that resolves to nothing.
	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"tv"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, queue.calls, "nothing may be enqueued on a rejected request")
	assert.Equal(t, "VALIDATION_INVALID_FORMAT", decodeEnvelope(t, w).Error.Code)
}

func TestRunPipeline_RejectsMissingMediaID(t *testing.T) {
	h := NewSubtitlePipelineHandler(runningQueue(), foundMedia(), func() bool { return true }, nil)

	w := postRun(t, h, `{"media_type":"movie"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_INVALID_FORMAT", decodeEnvelope(t, w).Error.Code)
}

func TestRunPipeline_404WhenTheItemDoesNotExist(t *testing.T) {
	queue := runningQueue()
	lookup := &fakeMediaLookup{err: subtitle.ErrMediaNotFound}
	h := NewSubtitlePipelineHandler(queue, lookup, func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, queue.calls, "a non-existent item must never occupy a queue slot")
}

func TestRunPipeline_500OnLookupFailure(t *testing.T) {
	queue := runningQueue()
	lookup := &fakeMediaLookup{err: errors.New("database is locked")}
	h := NewSubtitlePipelineHandler(queue, lookup, func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code,
		"a locked database is not a 404 — reporting it as one would send the operator hunting for a missing row")
	assert.Empty(t, queue.calls)
}

// ─── AC #5: the capability gate, endpoint entry point ──────────────────────

func TestRunPipeline_409AndNoEnqueueWhenTranslationKeyMissing(t *testing.T) {
	queue := runningQueue()
	lookup := foundMedia()
	h := NewSubtitlePipelineHandler(queue, lookup, func() bool { return false }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	body := decodeEnvelope(t, w)
	assert.Equal(t, "AI_NOT_CONFIGURED", body.Error.Code, "reuse — this story adds ZERO Rule 7 codes")
	assert.Equal(t, "尚未設定翻譯服務金鑰", body.Error.Message)
	assert.Contains(t, body.Error.Suggestion, "CLAUDE_API_KEY")

	assert.Empty(t, queue.calls, "the gate short-circuits BEFORE enqueue — no queued work that can only fail")
	assert.Empty(t, lookup.refs, "and before the lookup: an unconfigured install should not touch the DB per request")
}

func TestRunPipeline_409WhenPipelineModeIsLegacy(t *testing.T) {
	// Legacy mode leaves the pool nil in main.go, so the handler is constructed
	// with a nil queue. The route still exists (stable API surface) and answers
	// honestly instead of panicking.
	h := NewSubtitlePipelineHandler(nil, foundMedia(), func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	body := decodeEnvelope(t, w)
	assert.Equal(t, "AI_NOT_CONFIGURED", body.Error.Code)
	assert.Contains(t, body.Error.Suggestion, "VIDO_SUBTITLE_PIPELINE_MODE")
}

func TestRunPipeline_409WhenPoolIsNotRunning(t *testing.T) {
	h := NewSubtitlePipelineHandler(&fakePipelineQueue{running: false}, foundMedia(), func() bool { return true }, nil)

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie"}`)

	require.Equal(t, http.StatusConflict, w.Code,
		"a stopped pool would accept nothing — answering 202 would be a lie")
}

func TestRunPipeline_AcceptsEveryPipelineMediaType(t *testing.T) {
	for _, mediaType := range []string{"movie", "series", "episode"} {
		t.Run(mediaType, func(t *testing.T) {
			queue := runningQueue()
			h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true }, nil)

			w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"`+mediaType+`"}`)

			require.Equal(t, http.StatusAccepted, w.Code)
			require.Len(t, queue.calls, 1)
			assert.Equal(t, mediaType, queue.calls[0].MediaType)
		})
	}
}

// ─── sub-6-8a AC #4: model_id on the FR12 single-item trigger ──────────────

func TestRunPipeline_PassesTheChosenModelIntoTheQueuedOptions(t *testing.T) {
	queue := runningQueue()
	h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true },
		&stubModelValidator{allowed: []string{"claude-haiku-4-5"}})

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie","model_id":"claude-haiku-4-5"}`)

	require.Equal(t, http.StatusAccepted, w.Code)
	require.Len(t, queue.opts, 1)
	assert.Equal(t, "claude-haiku-4-5", queue.opts[0].ModelID,
		"the pipeline must run the model the caller asked for, not the deployment default")
}

func TestRunPipeline_UnsupportedModelIsRejected(t *testing.T) {
	queue := runningQueue()
	h := NewSubtitlePipelineHandler(queue, foundMedia(), func() bool { return true },
		&stubModelValidator{allowed: []string{"claude-haiku-4-5"}})

	w := postRun(t, h, `{"media_id":"`+runMovieID+`","media_type":"movie","model_id":"gpt-4o"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, queue.calls, "nothing may be queued for a model this install cannot run")
}
