package sse

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := hub.Register()
	require.NotNil(t, client)
	assert.NotEmpty(t, client.ID)

	// Give the Run goroutine time to process the register
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	hub.Unregister(client)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := hub.Register()
	time.Sleep(20 * time.Millisecond)

	event := Event{
		Type: EventScanProgress,
		Data: map[string]string{"status": "scanning"},
	}
	hub.Broadcast(event)

	select {
	case received := <-client.Events:
		assert.Equal(t, EventScanProgress, received.Type)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for broadcast event")
	}
}

func TestHub_BroadcastToMultipleClients(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client1 := hub.Register()
	client2 := hub.Register()
	time.Sleep(20 * time.Millisecond)

	event := Event{
		Type: EventNotification,
		Data: map[string]string{"message": "hello"},
	}
	hub.Broadcast(event)

	for _, client := range []*Client{client1, client2} {
		select {
		case received := <-client.Events:
			assert.Equal(t, EventNotification, received.Type)
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("Timeout waiting for event on client %s", client.ID)
		}
	}
}

func TestHub_UnregisterClosesChannel(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := hub.Register()
	time.Sleep(20 * time.Millisecond)

	hub.Unregister(client)
	time.Sleep(20 * time.Millisecond)

	// Channel should be closed after unregister
	_, ok := <-client.Events
	assert.False(t, ok, "Client channel should be closed after unregister")
}

func TestHub_NonBlockingSend(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := hub.Register()
	time.Sleep(20 * time.Millisecond)

	// Fill the client's event buffer (capacity 100)
	for i := 0; i < 150; i++ {
		hub.Broadcast(Event{
			Type: EventScanProgress,
			Data: i,
		})
	}

	// Give Run goroutine time to process all broadcasts
	time.Sleep(50 * time.Millisecond)

	// Drain and count received events
	count := 0
	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case <-client.Events:
			count++
		case <-timeout:
			goto done
		}
	}
done:
	// Should receive up to buffer capacity (100), not all 150
	assert.LessOrEqual(t, count, 100)
	assert.Greater(t, count, 0)
}

func TestHub_ConcurrentBroadcast(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := hub.Register()
	time.Sleep(20 * time.Millisecond)

	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				hub.Broadcast(Event{
					Type: EventNotification,
					Data: id*100 + j,
				})
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	// Drain events
	count := 0
	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case <-client.Events:
			count++
		case <-timeout:
			goto done
		}
	}
done:
	assert.Greater(t, count, 0)
	_ = client
}

func TestHub_ClientCount(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	assert.Equal(t, 0, hub.ClientCount())

	c1 := hub.Register()
	c2 := hub.Register()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, hub.ClientCount())

	hub.Unregister(c1)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	hub.Unregister(c2)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_Close(t *testing.T) {
	hub := NewHub()

	client := hub.Register()
	time.Sleep(20 * time.Millisecond)

	hub.Close()
	time.Sleep(20 * time.Millisecond)

	// Client channel should be closed
	_, ok := <-client.Events
	assert.False(t, ok, "Client channel should be closed after hub.Close()")

	// Double close should not panic
	hub.Close()
}

func TestEventRequestProgress_Value(t *testing.T) {
	// Story 13-3a AC #4 [@contract-v1] — the request pipeline event type.
	assert.Equal(t, EventType("request_progress"), EventRequestProgress)
}

// A dropped event is logged by TYPE and SIZE, never by payload: the
// generation-batch terminal event carries the whole queue (thousands of items
// on a select-all — dsr-6d-a AC #2), and one dropped event must not write a
// multi-hundred-KB log line.
func TestHub_DroppedBroadcastLogsTypeAndSizeNotThePayload(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	// A hub whose Run loop is not started: the buffer fills and the next
	// Broadcast takes the drop path.
	h := &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan Event, 1),
		register:   make(chan *Client, 1),
		unregister: make(chan *Client, 1),
		done:       make(chan struct{}),
	}

	queue := make([]map[string]string, 0, 200)
	for i := 0; i < 200; i++ {
		queue = append(queue, map[string]string{"media_id": "0a54a9e2-3a67-4f3e-9f8e-a1c2d3e4f501", "title": "怪奇物語 S04E07"})
	}
	event := Event{Type: EventGenerationBatchProgress, Data: map[string]interface{}{"status": "complete", "items": queue}}

	h.Broadcast(event) // fills the buffer
	h.Broadcast(event) // dropped

	logged := buf.String()
	require.Contains(t, logged, "SSE broadcast channel full")
	assert.Contains(t, logged, "event_type=generation_batch_progress")
	assert.Contains(t, logged, "payload_bytes=")
	assert.NotContains(t, logged, "怪奇物語", "the payload itself must never reach the log")
	assert.Less(t, len(logged), 500, "one short line, not the queue")
	assert.Equal(t, 1, strings.Count(logged, "SSE broadcast channel full"))
}
