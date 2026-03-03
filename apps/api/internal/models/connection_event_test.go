package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnectionEventType_Constants(t *testing.T) {
	assert.Equal(t, ConnectionEventType("connected"), EventConnected)
	assert.Equal(t, ConnectionEventType("disconnected"), EventDisconnected)
	assert.Equal(t, ConnectionEventType("error"), EventError)
	assert.Equal(t, ConnectionEventType("recovered"), EventRecovered)
}

func TestValidEventTypes(t *testing.T) {
	types := ValidEventTypes()
	assert.Len(t, types, 4)
	assert.Contains(t, types, EventConnected)
	assert.Contains(t, types, EventDisconnected)
	assert.Contains(t, types, EventError)
	assert.Contains(t, types, EventRecovered)
}

func TestIsValidEventType(t *testing.T) {
	assert.True(t, IsValidEventType(EventConnected))
	assert.True(t, IsValidEventType(EventDisconnected))
	assert.True(t, IsValidEventType(EventError))
	assert.True(t, IsValidEventType(EventRecovered))
	assert.False(t, IsValidEventType(ConnectionEventType("invalid")))
	assert.False(t, IsValidEventType(ConnectionEventType("")))
}

func TestConnectionEvent_Fields(t *testing.T) {
	now := time.Now()
	event := ConnectionEvent{
		ID:        "test-id",
		Service:   "qbittorrent",
		EventType: EventDisconnected,
		Status:    ServiceStatusDown,
		Message:   "connection refused",
		CreatedAt: now,
	}

	assert.Equal(t, "test-id", event.ID)
	assert.Equal(t, "qbittorrent", event.Service)
	assert.Equal(t, EventDisconnected, event.EventType)
	assert.Equal(t, ServiceStatusDown, event.Status)
	assert.Equal(t, "connection refused", event.Message)
	assert.Equal(t, now, event.CreatedAt)
}
