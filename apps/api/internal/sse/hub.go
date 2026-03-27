package sse

import (
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// EventType represents the type of SSE event
type EventType string

const (
	EventScanProgress          EventType = "scan_progress"
	EventScanComplete          EventType = "scan_complete"
	EventScanCancelled         EventType = "scan_cancelled"
	EventSubtitleProgress      EventType = "subtitle_progress"
	EventSubtitleBatchProgress EventType = "subtitle_batch_progress"
	EventNotification          EventType = "notification"
	EventEnrichProgress        EventType = "enrich_progress"
	EventEnrichComplete        EventType = "enrich_complete"
)

// Event represents an SSE event to broadcast
type Event struct {
	ID   string      `json:"id,omitempty"`
	Type EventType   `json:"type"`
	Data interface{} `json:"data"`
}

// Client represents a connected SSE client
type Client struct {
	ID     string
	Events chan Event
}

// Hub manages SSE client connections and event broadcasting
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
	closed     atomic.Bool
}

// NewHub creates a new Hub and starts the Run goroutine
func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		done:       make(chan struct{}),
	}
	go h.Run()
	return h
}

// Run is the main select loop that handles register/unregister/broadcast operations.
// It runs until Close is called.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			count := len(h.clients)
			h.mu.Unlock()
			slog.Info("SSE client registered", "client_id", client.ID, "total_clients", count)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Events)
			}
			count := len(h.clients)
			h.mu.Unlock()
			slog.Info("SSE client unregistered", "client_id", client.ID, "total_clients", count)

		case event := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Events <- event:
				default:
					slog.Warn("SSE client buffer full, dropping event",
						"client_id", client.ID,
						"event_type", event.Type,
					)
				}
			}
			h.mu.RUnlock()

		case <-h.done:
			// Close all client channels on shutdown
			h.mu.Lock()
			for id, client := range h.clients {
				close(client.Events)
				delete(h.clients, id)
			}
			h.mu.Unlock()
			slog.Info("SSE hub stopped")
			return
		}
	}
}

// Register creates a new client with a UUID ID and buffered channel,
// registers it with the hub, and returns the client.
func (h *Hub) Register() *Client {
	client := &Client{
		ID:     uuid.New().String(),
		Events: make(chan Event, 100),
	}
	h.register <- client
	return client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends an event to all connected clients.
// Uses non-blocking send to avoid blocking the caller if the broadcast channel is full.
func (h *Hub) Broadcast(event Event) {
	select {
	case h.broadcast <- event:
	default:
		data, _ := json.Marshal(event)
		slog.Warn("SSE broadcast channel full, dropping event", "event", string(data))
	}
}

// Close stops the Run loop and cleans up all client connections.
func (h *Hub) Close() {
	if h.closed.CompareAndSwap(false, true) {
		close(h.done)
	}
}

// ClientCount returns the current number of connected clients (thread-safe).
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
