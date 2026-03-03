package models

import "time"

// ConnectionEventType represents the type of connection event
type ConnectionEventType string

const (
	EventConnected    ConnectionEventType = "connected"
	EventDisconnected ConnectionEventType = "disconnected"
	EventError        ConnectionEventType = "error"
	EventRecovered    ConnectionEventType = "recovered"
)

// ConnectionEvent represents a connection status change event
type ConnectionEvent struct {
	ID        string              `json:"id"`
	Service   string              `json:"service"`
	EventType ConnectionEventType `json:"eventType"`
	Status    string              `json:"status"` // healthy, degraded, down
	Message   string              `json:"message,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
}

// ValidEventTypes returns all valid connection event types
func ValidEventTypes() []ConnectionEventType {
	return []ConnectionEventType{EventConnected, EventDisconnected, EventError, EventRecovered}
}

// IsValidEventType checks if the given event type is valid
func IsValidEventType(et ConnectionEventType) bool {
	for _, valid := range ValidEventTypes() {
		if et == valid {
			return true
		}
	}
	return false
}
