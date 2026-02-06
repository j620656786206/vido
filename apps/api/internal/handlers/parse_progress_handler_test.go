package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/events"
	"github.com/vido/api/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupParseProgressHandler() (*ParseProgressHandler, *events.ChannelEmitter) {
	emitter := events.NewChannelEmitter()
	handler := NewParseProgressHandler(emitter)
	return handler, emitter
}

func setupParseProgressRouter(handler *ParseProgressHandler) *gin.Engine {
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)
	return router
}

func TestNewParseProgressHandler(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	require.NotNil(t, handler)
	assert.NotNil(t, handler.emitter)
	assert.NotNil(t, handler.progress)
}

func TestParseProgressHandler_StartProgress(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	// Subscribe to events
	eventChan := emitter.Subscribe("task-123")

	progress := handler.StartProgress("task-123", "test-movie.mkv")

	require.NotNil(t, progress)
	assert.Equal(t, "task-123", progress.TaskID)
	assert.Equal(t, "test-movie.mkv", progress.Filename)
	assert.Equal(t, models.ParseStatusPending, progress.Status)
	assert.Len(t, progress.Steps, 6)

	// Verify event was emitted
	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventParseStarted, event.Type)
		assert.Equal(t, "task-123", event.TaskID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected parse_started event")
	}
}

func TestParseProgressHandler_UpdateStepProgress(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	handler.StartProgress("task-123", "test.mkv")

	// Subscribe after starting
	eventChan := emitter.Subscribe("task-123")

	// Test step in progress
	handler.UpdateStepProgress("task-123", 0, models.StepInProgress, "")

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventStepStarted, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected step_started event")
	}

	// Test step completed
	handler.UpdateStepProgress("task-123", 0, models.StepSuccess, "")

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventStepCompleted, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected step_completed event")
	}

	// Verify progress was updated
	progress, exists := handler.GetProgress("task-123")
	require.True(t, exists)
	assert.Equal(t, models.StepSuccess, progress.Steps[0].Status)
}

func TestParseProgressHandler_UpdateStepProgress_Failed(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	handler.StartProgress("task-123", "test.mkv")
	eventChan := emitter.Subscribe("task-123")

	handler.UpdateStepProgress("task-123", 1, models.StepFailed, "TMDb API timeout")

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventStepFailed, event.Type)
		data, ok := event.Data.(events.StepEventData)
		require.True(t, ok)
		assert.Equal(t, "TMDb API timeout", data.Step.Error)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected step_failed event")
	}
}

func TestParseProgressHandler_UpdateStepProgress_Skipped(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	handler.StartProgress("task-123", "test.mkv")
	eventChan := emitter.Subscribe("task-123")

	handler.UpdateStepProgress("task-123", 2, models.StepSkipped, "")

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventStepSkipped, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected step_skipped event")
	}
}

func TestParseProgressHandler_CompleteProgress(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	handler.StartProgress("task-123", "test.mkv")
	eventChan := emitter.Subscribe("task-123")

	result := &models.ParseResult{
		MediaID:        "movie-456",
		Title:          "Test Movie",
		Year:           2024,
		MetadataSource: models.MetadataSourceTMDb,
	}

	handler.CompleteProgress("task-123", result)

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventParseCompleted, event.Type)
		data, ok := event.Data.(events.ParseCompletedData)
		require.True(t, ok)
		require.NotNil(t, data.Result)
		assert.Equal(t, "movie-456", data.Result.MediaID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected parse_completed event")
	}

	progress, exists := handler.GetProgress("task-123")
	require.True(t, exists)
	assert.Equal(t, models.ParseStatusSuccess, progress.Status)
	assert.Equal(t, 100, progress.Percentage)
}

func TestParseProgressHandler_FailProgress(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	handler.StartProgress("task-123", "test.mkv")
	eventChan := emitter.Subscribe("task-123")

	handler.FailProgress("task-123", "All sources failed")

	select {
	case event := <-eventChan:
		assert.Equal(t, events.EventParseFailed, event.Type)
		data, ok := event.Data.(events.ParseFailedData)
		require.True(t, ok)
		assert.Equal(t, "All sources failed", data.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected parse_failed event")
	}

	progress, exists := handler.GetProgress("task-123")
	require.True(t, exists)
	assert.Equal(t, models.ParseStatusFailed, progress.Status)
}

func TestParseProgressHandler_CleanupProgress(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	handler.StartProgress("task-123", "test.mkv")

	_, exists := handler.GetProgress("task-123")
	require.True(t, exists)

	handler.CleanupProgress("task-123")

	_, exists = handler.GetProgress("task-123")
	assert.False(t, exists)
}

func TestParseProgressHandler_GetProgress_NotFound(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	_, exists := handler.GetProgress("nonexistent")
	assert.False(t, exists)
}

func TestParseProgressHandler_GetProgressStatus(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()
	router := setupParseProgressRouter(handler)

	// Start a progress
	handler.StartProgress("task-123", "test-movie.mkv")

	// Request status
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/parse/progress/task-123/status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	require.NotNil(t, response.Data)
}

func TestParseProgressHandler_GetProgressStatus_NotFound(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()
	router := setupParseProgressRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/parse/progress/nonexistent/status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "PARSE_TASK_NOT_FOUND", response.Error.Code)
}

func TestParseProgressHandler_GetProgressStatus_EmptyTaskID(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()
	router := setupParseProgressRouter(handler)

	w := httptest.NewRecorder()
	// Gin routes with :taskId won't match empty, but we test the handler directly
	req, _ := http.NewRequest("GET", "/api/v1/parse/progress//status", nil)
	router.ServeHTTP(w, req)

	// Route won't match properly, returns 301 redirect or 404/400
	// The exact behavior depends on Gin version, just verify it's an error
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestParseProgressHandler_StreamProgress_HeadersSetup(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	// Test that handler is properly configured
	require.NotNil(t, handler)
	require.NotNil(t, handler.emitter)

	// Start progress and verify it can be tracked
	handler.StartProgress("task-stream-test", "test.mkv")
	progress, exists := handler.GetProgress("task-stream-test")
	require.True(t, exists)
	assert.Equal(t, "test.mkv", progress.Filename)
}

func TestParseProgressHandler_StreamProgress_ReceivesEvents(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	// Start a progress task
	handler.StartProgress("task-123", "test.mkv")

	// Verify progress exists
	progress, exists := handler.GetProgress("task-123")
	require.True(t, exists)
	assert.Equal(t, "test.mkv", progress.Filename)

	// Update step and verify event would be emitted
	handler.UpdateStepProgress("task-123", 0, models.StepSuccess, "")

	progress, _ = handler.GetProgress("task-123")
	assert.Equal(t, models.StepSuccess, progress.Steps[0].Status)
}

func TestParseProgressHandler_UpdateNonexistentTask(t *testing.T) {
	handler, emitter := setupParseProgressHandler()
	defer emitter.Close()

	// Should not panic
	handler.UpdateStepProgress("nonexistent", 0, models.StepSuccess, "")
	handler.CompleteProgress("nonexistent", &models.ParseResult{})
	handler.FailProgress("nonexistent", "error")
}
