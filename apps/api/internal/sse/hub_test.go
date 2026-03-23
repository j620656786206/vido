package sse

import (
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
