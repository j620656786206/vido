package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/events"
	"github.com/vido/api/internal/models"
)

// ParseProgressHandler handles parse progress SSE streaming
type ParseProgressHandler struct {
	emitter   events.EventEmitter
	mu        sync.RWMutex
	progress  map[string]*models.ParseProgress
}

// NewParseProgressHandler creates a new ParseProgressHandler
func NewParseProgressHandler(emitter events.EventEmitter) *ParseProgressHandler {
	return &ParseProgressHandler{
		emitter:  emitter,
		progress: make(map[string]*models.ParseProgress),
	}
}

// RegisterRoutes registers the parse progress routes
func (h *ParseProgressHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/parse/progress/:taskId", h.StreamProgress)
	r.GET("/parse/progress/:taskId/status", h.GetProgressStatus)
}

// StreamProgress handles SSE streaming for parse progress
// GET /api/v1/parse/progress/{taskId}
// @Summary Stream parse progress events
// @Description Stream real-time parse progress events using Server-Sent Events
// @Tags Parse
// @Produce text/event-stream
// @Param taskId path string true "Task ID"
// @Success 200 {object} events.ParseEvent
// @Failure 400 {object} APIResponse
// @Router /api/v1/parse/progress/{taskId} [get]
func (h *ParseProgressHandler) StreamProgress(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequestError(c, "PARSE_TASK_ID_REQUIRED", "Task ID is required")
		return
	}

	slog.Info("SSE connection opened", "taskId", taskID)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Subscribe to events for this task
	eventChan := h.emitter.Subscribe(taskID)

	// Ensure cleanup on connection close
	defer func() {
		h.emitter.Unsubscribe(taskID, eventChan)
		slog.Info("SSE connection closed", "taskId", taskID)
	}()

	// Send initial connection event
	h.sendSSEEvent(c.Writer, "connected", map[string]string{
		"taskId":  taskID,
		"message": "Connected to parse progress stream",
	})
	c.Writer.Flush()

	// Stream events
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed
				return false
			}

			h.sendSSEEvent(w, string(event.Type), event)
			return true

		case <-c.Request.Context().Done():
			// Client disconnected
			slog.Info("Client disconnected", "taskId", taskID)
			return false

		case <-time.After(30 * time.Second):
			// Send keepalive ping
			h.sendSSEEvent(w, "ping", map[string]int64{
				"timestamp": time.Now().Unix(),
			})
			return true
		}
	})
}

// sendSSEEvent sends a Server-Sent Event
func (h *ParseProgressHandler) sendSSEEvent(w io.Writer, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to marshal SSE event data", "error", err)
		return
	}

	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))

	// Flush if flusher is available
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// GetProgressStatus returns the current progress status (non-streaming)
// GET /api/v1/parse/progress/{taskId}/status
// @Summary Get current parse progress status
// @Description Get the current progress status for a parse task (non-streaming)
// @Tags Parse
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {object} APIResponse{data=models.ParseProgress}
// @Failure 404 {object} APIResponse
// @Router /api/v1/parse/progress/{taskId}/status [get]
func (h *ParseProgressHandler) GetProgressStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		BadRequestError(c, "PARSE_TASK_ID_REQUIRED", "Task ID is required")
		return
	}

	h.mu.RLock()
	progress, exists := h.progress[taskID]
	h.mu.RUnlock()

	if !exists {
		ErrorResponse(c, http.StatusNotFound, "PARSE_TASK_NOT_FOUND",
			"Parse task not found",
			"The task may have expired or never existed.")
		return
	}

	SuccessResponse(c, progress)
}

// StartProgress starts tracking progress for a new parse task
func (h *ParseProgressHandler) StartProgress(taskID, filename string) *models.ParseProgress {
	progress := models.NewParseProgress(taskID, filename)

	h.mu.Lock()
	h.progress[taskID] = progress
	h.mu.Unlock()

	// Emit parse started event
	h.emitter.Emit(events.NewParseEvent(events.EventParseStarted, taskID, events.ParseStartedData{
		Filename:   filename,
		TotalSteps: len(progress.Steps),
		Steps:      progress.Steps,
	}))

	slog.Info("Parse started", "taskId", taskID, "filename", filename)
	return progress
}

// UpdateStepProgress updates the progress for a specific step
func (h *ParseProgressHandler) UpdateStepProgress(taskID string, stepIndex int, status models.StepStatus, errorMsg string) {
	h.mu.Lock()
	progress, exists := h.progress[taskID]
	if !exists {
		h.mu.Unlock()
		return
	}

	switch status {
	case models.StepInProgress:
		progress.StartStep(stepIndex)
	case models.StepSuccess:
		progress.CompleteStep(stepIndex)
	case models.StepFailed:
		progress.FailStep(stepIndex, errorMsg)
	case models.StepSkipped:
		progress.SkipStep(stepIndex)
	}
	h.mu.Unlock()

	// Determine event type
	var eventType events.ParseEventType
	switch status {
	case models.StepInProgress:
		eventType = events.EventStepStarted
	case models.StepSuccess:
		eventType = events.EventStepCompleted
	case models.StepFailed:
		eventType = events.EventStepFailed
	case models.StepSkipped:
		eventType = events.EventStepSkipped
	default:
		eventType = events.EventProgressUpdate
	}

	// Emit step event
	h.emitter.Emit(events.NewParseEvent(eventType, taskID, events.StepEventData{
		StepIndex: stepIndex,
		Step:      progress.Steps[stepIndex],
		Progress:  progress,
	}))

	slog.Debug("Parse step updated",
		"taskId", taskID,
		"stepIndex", stepIndex,
		"status", status,
	)
}

// CompleteProgress marks the parse task as complete
func (h *ParseProgressHandler) CompleteProgress(taskID string, result *models.ParseResult) {
	h.mu.Lock()
	progress, exists := h.progress[taskID]
	if !exists {
		h.mu.Unlock()
		return
	}
	progress.Complete(result)
	h.mu.Unlock()

	// Emit completion event
	h.emitter.Emit(events.NewParseEvent(events.EventParseCompleted, taskID, events.ParseCompletedData{
		Result:   result,
		Progress: progress,
	}))

	slog.Info("Parse completed", "taskId", taskID, "mediaId", result.MediaID)
}

// FailProgress marks the parse task as failed
func (h *ParseProgressHandler) FailProgress(taskID string, message string) {
	h.mu.Lock()
	progress, exists := h.progress[taskID]
	if !exists {
		h.mu.Unlock()
		return
	}
	progress.Fail(message)
	failedSteps := progress.GetFailedSteps()
	h.mu.Unlock()

	// Emit failure event
	h.emitter.Emit(events.NewParseEvent(events.EventParseFailed, taskID, events.ParseFailedData{
		Message:     message,
		FailedSteps: failedSteps,
		Progress:    progress,
	}))

	slog.Warn("Parse failed", "taskId", taskID, "message", message)
}

// CleanupProgress removes a completed or failed task from memory
func (h *ParseProgressHandler) CleanupProgress(taskID string) {
	h.mu.Lock()
	delete(h.progress, taskID)
	h.mu.Unlock()

	slog.Debug("Parse progress cleaned up", "taskId", taskID)
}

// GetProgress returns the current progress for a task
func (h *ParseProgressHandler) GetProgress(taskID string) (*models.ParseProgress, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	progress, exists := h.progress[taskID]
	return progress, exists
}
