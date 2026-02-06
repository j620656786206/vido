package events

import (
	"sync"
	"time"

	"github.com/vido/api/internal/models"
)

// ParseEventType represents the type of parse event
type ParseEventType string

const (
	EventParseStarted   ParseEventType = "parse_started"
	EventStepStarted    ParseEventType = "step_started"
	EventStepCompleted  ParseEventType = "step_completed"
	EventStepFailed     ParseEventType = "step_failed"
	EventStepSkipped    ParseEventType = "step_skipped"
	EventParseCompleted ParseEventType = "parse_completed"
	EventParseFailed    ParseEventType = "parse_failed"
	EventProgressUpdate ParseEventType = "progress_update"
)

// ParseEvent represents a real-time parse event
type ParseEvent struct {
	Type      ParseEventType `json:"type"`
	TaskID    string         `json:"taskId"`
	Timestamp time.Time      `json:"timestamp"`
	Data      interface{}    `json:"data,omitempty"`
}

// ParseStartedData contains data for parse_started event
type ParseStartedData struct {
	Filename   string              `json:"filename"`
	TotalSteps int                 `json:"totalSteps"`
	Steps      []models.ParseStep  `json:"steps"`
}

// StepEventData contains data for step-related events
type StepEventData struct {
	StepIndex int                `json:"stepIndex"`
	Step      models.ParseStep   `json:"step"`
	Progress  *models.ParseProgress `json:"progress,omitempty"`
}

// ParseCompletedData contains data for parse_completed event
type ParseCompletedData struct {
	Result   *models.ParseResult   `json:"result,omitempty"`
	Progress *models.ParseProgress `json:"progress"`
}

// ParseFailedData contains data for parse_failed event
type ParseFailedData struct {
	Message      string              `json:"message"`
	FailedSteps  []models.ParseStep  `json:"failedSteps"`
	Progress     *models.ParseProgress `json:"progress"`
}

// ProgressUpdateData contains data for progress_update event
type ProgressUpdateData struct {
	Percentage  int                   `json:"percentage"`
	CurrentStep int                   `json:"currentStep"`
	Progress    *models.ParseProgress `json:"progress"`
}

// NewParseEvent creates a new parse event
func NewParseEvent(eventType ParseEventType, taskID string, data interface{}) ParseEvent {
	return ParseEvent{
		Type:      eventType,
		TaskID:    taskID,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// EventEmitter defines the interface for broadcasting parse events
type EventEmitter interface {
	// Emit sends an event to all subscribers of the given task
	Emit(event ParseEvent)
	// Subscribe returns a channel that receives events for the given task
	Subscribe(taskID string) <-chan ParseEvent
	// Unsubscribe removes a subscription for the given task
	Unsubscribe(taskID string, ch <-chan ParseEvent)
	// Close closes the emitter and all subscriptions
	Close()
}

// ChannelEmitter implements EventEmitter using Go channels
type ChannelEmitter struct {
	mu          sync.RWMutex
	subscribers map[string][]chan ParseEvent
	closed      bool
}

// NewChannelEmitter creates a new ChannelEmitter
func NewChannelEmitter() *ChannelEmitter {
	return &ChannelEmitter{
		subscribers: make(map[string][]chan ParseEvent),
	}
}

// Emit sends an event to all subscribers of the given task
func (e *ChannelEmitter) Emit(event ParseEvent) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.closed {
		return
	}

	taskSubs, ok := e.subscribers[event.TaskID]
	if !ok {
		return
	}

	for _, ch := range taskSubs {
		select {
		case ch <- event:
		default:
			// Channel is full, skip this event (non-blocking)
		}
	}
}

// Subscribe returns a channel that receives events for the given task
func (e *ChannelEmitter) Subscribe(taskID string) <-chan ParseEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		ch := make(chan ParseEvent)
		close(ch)
		return ch
	}

	ch := make(chan ParseEvent, 100) // Buffered channel to prevent blocking
	e.subscribers[taskID] = append(e.subscribers[taskID], ch)
	return ch
}

// Unsubscribe removes a subscription for the given task
func (e *ChannelEmitter) Unsubscribe(taskID string, ch <-chan ParseEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	taskSubs, ok := e.subscribers[taskID]
	if !ok {
		return
	}

	// Find and remove the channel
	for i, sub := range taskSubs {
		if sub == ch {
			// Remove by swapping with last element and truncating
			taskSubs[i] = taskSubs[len(taskSubs)-1]
			e.subscribers[taskID] = taskSubs[:len(taskSubs)-1]
			close(sub)
			break
		}
	}

	// Clean up empty task entries
	if len(e.subscribers[taskID]) == 0 {
		delete(e.subscribers, taskID)
	}
}

// Close closes the emitter and all subscriptions
func (e *ChannelEmitter) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return
	}

	e.closed = true

	for taskID, subs := range e.subscribers {
		for _, ch := range subs {
			close(ch)
		}
		delete(e.subscribers, taskID)
	}
}

// SubscriberCount returns the number of subscribers for a task
func (e *ChannelEmitter) SubscriberCount(taskID string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return len(e.subscribers[taskID])
}

// TotalSubscribers returns the total number of subscribers across all tasks
func (e *ChannelEmitter) TotalSubscribers() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	total := 0
	for _, subs := range e.subscribers {
		total += len(subs)
	}
	return total
}
