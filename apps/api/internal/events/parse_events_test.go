package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func TestParseEventType_Constants(t *testing.T) {
	assert.Equal(t, ParseEventType("parse_started"), EventParseStarted)
	assert.Equal(t, ParseEventType("step_started"), EventStepStarted)
	assert.Equal(t, ParseEventType("step_completed"), EventStepCompleted)
	assert.Equal(t, ParseEventType("step_failed"), EventStepFailed)
	assert.Equal(t, ParseEventType("step_skipped"), EventStepSkipped)
	assert.Equal(t, ParseEventType("parse_completed"), EventParseCompleted)
	assert.Equal(t, ParseEventType("parse_failed"), EventParseFailed)
	assert.Equal(t, ParseEventType("progress_update"), EventProgressUpdate)
}

func TestNewParseEvent(t *testing.T) {
	data := ParseStartedData{
		Filename:   "test.mkv",
		TotalSteps: 6,
	}

	event := NewParseEvent(EventParseStarted, "task-123", data)

	assert.Equal(t, EventParseStarted, event.Type)
	assert.Equal(t, "task-123", event.TaskID)
	assert.NotZero(t, event.Timestamp)
	assert.Equal(t, data, event.Data)
}

func TestNewChannelEmitter(t *testing.T) {
	emitter := NewChannelEmitter()

	require.NotNil(t, emitter)
	assert.NotNil(t, emitter.subscribers)
	assert.False(t, emitter.closed)
}

func TestChannelEmitter_Subscribe(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	ch := emitter.Subscribe("task-123")

	require.NotNil(t, ch)
	assert.Equal(t, 1, emitter.SubscriberCount("task-123"))
}

func TestChannelEmitter_MultipleSubscribers(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	ch1 := emitter.Subscribe("task-123")
	ch2 := emitter.Subscribe("task-123")
	ch3 := emitter.Subscribe("task-456")

	require.NotNil(t, ch1)
	require.NotNil(t, ch2)
	require.NotNil(t, ch3)
	assert.Equal(t, 2, emitter.SubscriberCount("task-123"))
	assert.Equal(t, 1, emitter.SubscriberCount("task-456"))
	assert.Equal(t, 3, emitter.TotalSubscribers())
}

func TestChannelEmitter_Emit(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	ch := emitter.Subscribe("task-123")

	event := NewParseEvent(EventParseStarted, "task-123", nil)
	emitter.Emit(event)

	select {
	case received := <-ch:
		assert.Equal(t, EventParseStarted, received.Type)
		assert.Equal(t, "task-123", received.TaskID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}
}

func TestChannelEmitter_EmitToMultipleSubscribers(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	ch1 := emitter.Subscribe("task-123")
	ch2 := emitter.Subscribe("task-123")

	event := NewParseEvent(EventStepCompleted, "task-123", nil)
	emitter.Emit(event)

	// Both subscribers should receive the event
	for i, ch := range []<-chan ParseEvent{ch1, ch2} {
		select {
		case received := <-ch:
			assert.Equal(t, EventStepCompleted, received.Type)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Subscriber %d: Timeout waiting for event", i+1)
		}
	}
}

func TestChannelEmitter_EmitToCorrectTask(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	ch1 := emitter.Subscribe("task-123")
	ch2 := emitter.Subscribe("task-456")

	event := NewParseEvent(EventParseStarted, "task-123", nil)
	emitter.Emit(event)

	// ch1 should receive the event
	select {
	case received := <-ch1:
		assert.Equal(t, "task-123", received.TaskID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for event on ch1")
	}

	// ch2 should NOT receive the event (different task)
	select {
	case <-ch2:
		t.Fatal("ch2 should not receive event for different task")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event
	}
}

func TestChannelEmitter_Unsubscribe(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	ch := emitter.Subscribe("task-123")
	assert.Equal(t, 1, emitter.SubscriberCount("task-123"))

	emitter.Unsubscribe("task-123", ch)
	assert.Equal(t, 0, emitter.SubscriberCount("task-123"))
}

func TestChannelEmitter_UnsubscribeNonexistent(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	ch := make(chan ParseEvent)

	// Should not panic
	emitter.Unsubscribe("task-123", ch)
	emitter.Unsubscribe("nonexistent", ch)
}

func TestChannelEmitter_Close(t *testing.T) {
	emitter := NewChannelEmitter()

	ch := emitter.Subscribe("task-123")
	emitter.Close()

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed")

	// Emit should not panic after close
	emitter.Emit(NewParseEvent(EventParseStarted, "task-123", nil))

	// Subscribe after close should return closed channel
	ch2 := emitter.Subscribe("task-456")
	_, ok = <-ch2
	assert.False(t, ok, "Channel from closed emitter should be closed")
}

func TestChannelEmitter_Concurrent(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 100

	// Subscribe
	ch := emitter.Subscribe("task-123")

	// Emit events concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := NewParseEvent(EventProgressUpdate, "task-123", goroutineID)
				emitter.Emit(event)
			}
		}(i)
	}

	// Consume events
	received := 0
	done := make(chan bool)
	go func() {
		for range ch {
			received++
		}
		done <- true
	}()

	wg.Wait()
	emitter.Close()
	<-done

	// Due to buffered channel and non-blocking send, we may not receive all events
	// but we should receive some
	assert.Greater(t, received, 0)
}

func TestParseStartedData(t *testing.T) {
	steps := models.StandardParseSteps()
	data := ParseStartedData{
		Filename:   "test-movie.mkv",
		TotalSteps: 6,
		Steps:      steps,
	}

	assert.Equal(t, "test-movie.mkv", data.Filename)
	assert.Equal(t, 6, data.TotalSteps)
	assert.Len(t, data.Steps, 6)
}

func TestStepEventData(t *testing.T) {
	step := models.ParseStep{
		Name:   "tmdb_search",
		Label:  "搜尋 TMDb",
		Status: models.StepSuccess,
	}

	progress := models.NewParseProgress("task-123", "test.mkv")

	data := StepEventData{
		StepIndex: 1,
		Step:      step,
		Progress:  progress,
	}

	assert.Equal(t, 1, data.StepIndex)
	assert.Equal(t, "tmdb_search", data.Step.Name)
	assert.NotNil(t, data.Progress)
}

func TestParseCompletedData(t *testing.T) {
	result := &models.ParseResult{
		MediaID:        "movie-123",
		Title:          "Test Movie",
		Year:           2024,
		MetadataSource: models.MetadataSourceTMDb,
	}

	progress := models.NewParseProgress("task-123", "test.mkv")
	progress.Complete(result)

	data := ParseCompletedData{
		Result:   result,
		Progress: progress,
	}

	require.NotNil(t, data.Result)
	assert.Equal(t, "movie-123", data.Result.MediaID)
	assert.Equal(t, models.ParseStatusSuccess, data.Progress.Status)
}

func TestParseFailedData(t *testing.T) {
	progress := models.NewParseProgress("task-123", "test.mkv")
	progress.FailStep(1, "TMDb timeout")
	progress.FailStep(2, "Douban unavailable")

	data := ParseFailedData{
		Message:     "All sources failed",
		FailedSteps: progress.GetFailedSteps(),
		Progress:    progress,
	}

	assert.Equal(t, "All sources failed", data.Message)
	assert.Len(t, data.FailedSteps, 2)
	assert.NotNil(t, data.Progress)
}

func TestProgressUpdateData(t *testing.T) {
	progress := models.NewParseProgress("task-123", "test.mkv")
	progress.CompleteStep(0)
	progress.StartStep(1)

	data := ProgressUpdateData{
		Percentage:  16,
		CurrentStep: 1,
		Progress:    progress,
	}

	assert.Equal(t, 16, data.Percentage)
	assert.Equal(t, 1, data.CurrentStep)
	assert.NotNil(t, data.Progress)
}

func TestChannelEmitter_NonBlockingEmit(t *testing.T) {
	emitter := NewChannelEmitter()
	defer emitter.Close()

	// Subscribe with a full channel (buffer size 100, send 101 events)
	ch := emitter.Subscribe("task-123")

	// Fill the channel buffer
	for i := 0; i < 150; i++ {
		event := NewParseEvent(EventProgressUpdate, "task-123", i)
		emitter.Emit(event)
	}

	// Should not block - some events may be dropped
	// Drain the channel
	count := 0
	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case <-ch:
			count++
		case <-timeout:
			goto done
		}
	}
done:
	// Buffer size is 100, so we should have received ~100 events
	assert.LessOrEqual(t, count, 100)
	assert.Greater(t, count, 0)
}
