package sse

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler returns a Gin handler for SSE streaming.
// Endpoint: GET /api/v1/events
func Handler(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

		// Register client
		client := hub.Register()
		defer hub.Unregister(client)

		slog.Info("SSE events stream opened", "client_id", client.ID)
		defer slog.Info("SSE events stream closed", "client_id", client.ID)

		// Send initial connection event
		sendSSEEvent(c.Writer, "connected", map[string]string{
			"clientId": client.ID,
			"message":  "Connected to event stream",
		})
		c.Writer.Flush()

		// Stream events
		c.Stream(func(w io.Writer) bool {
			select {
			case event, ok := <-client.Events:
				if !ok {
					return false
				}
				sendSSEEvent(w, string(event.Type), event)
				return true

			case <-c.Request.Context().Done():
				return false

			case <-time.After(30 * time.Second):
				// Send keepalive ping
				sendSSEEvent(w, "ping", map[string]int64{
					"timestamp": time.Now().Unix(),
				})
				return true
			}
		})
	}
}

// sendSSEEvent writes a single SSE event to the writer
func sendSSEEvent(w io.Writer, eventType string, data interface{}) {
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
