package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDegradationLevel_String(t *testing.T) {
	tests := []struct {
		level    DegradationLevel
		expected string
	}{
		{DegradationNormal, "normal"},
		{DegradationPartial, "partial"},
		{DegradationMinimal, "minimal"},
		{DegradationOffline, "offline"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.level))
		})
	}
}

func TestServiceHealth_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"healthy status", ServiceStatusHealthy, true},
		{"degraded status", ServiceStatusDegraded, false},
		{"down status", ServiceStatusDown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := ServiceHealth{Status: tt.status}
			assert.Equal(t, tt.expected, health.IsHealthy())
		})
	}
}

func TestServiceHealth_IsDegraded(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"healthy status", ServiceStatusHealthy, false},
		{"degraded status", ServiceStatusDegraded, true},
		{"down status", ServiceStatusDown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := ServiceHealth{Status: tt.status}
			assert.Equal(t, tt.expected, health.IsDegraded())
		})
	}
}

func TestServiceHealth_IsDown(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"healthy status", ServiceStatusHealthy, false},
		{"degraded status", ServiceStatusDegraded, false},
		{"down status", ServiceStatusDown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := ServiceHealth{Status: tt.status}
			assert.Equal(t, tt.expected, health.IsDown())
		})
	}
}

func TestServiceHealth_RecordError(t *testing.T) {
	health := NewServiceHealth("test", "Test Service")
	assert.Equal(t, ServiceStatusHealthy, health.Status)
	assert.Equal(t, 0, health.ErrorCount)

	// First error - should be degraded
	health.RecordError("connection timeout")
	assert.Equal(t, ServiceStatusDegraded, health.Status)
	assert.Equal(t, 1, health.ErrorCount)
	assert.Equal(t, "connection timeout", health.Message)

	// Second error - still degraded
	health.RecordError("connection refused")
	assert.Equal(t, ServiceStatusDegraded, health.Status)
	assert.Equal(t, 2, health.ErrorCount)

	// Third error - should be down
	health.RecordError("service unavailable")
	assert.Equal(t, ServiceStatusDown, health.Status)
	assert.Equal(t, 3, health.ErrorCount)
}

func TestServiceHealth_RecordSuccess(t *testing.T) {
	health := NewServiceHealth("test", "Test Service")

	// Simulate errors
	health.RecordError("error 1")
	health.RecordError("error 2")
	health.RecordError("error 3")
	assert.Equal(t, ServiceStatusDown, health.Status)
	assert.Equal(t, 3, health.ErrorCount)

	// Record success
	health.RecordSuccess()
	assert.Equal(t, ServiceStatusHealthy, health.Status)
	assert.Equal(t, 0, health.ErrorCount)
	assert.Empty(t, health.Message)
	assert.True(t, health.LastSuccess.After(time.Time{}))
}

func TestNewServiceHealth(t *testing.T) {
	health := NewServiceHealth("tmdb", "TMDb API")

	assert.Equal(t, "tmdb", health.Name)
	assert.Equal(t, "TMDb API", health.DisplayName)
	assert.Equal(t, ServiceStatusHealthy, health.Status)
	assert.Equal(t, 0, health.ErrorCount)
	assert.Empty(t, health.Message)
}

func TestDegradedResult_HasMissingFields(t *testing.T) {
	tests := []struct {
		name          string
		missingFields []string
		expected      bool
	}{
		{"no missing fields", nil, false},
		{"empty missing fields", []string{}, false},
		{"with missing fields", []string{"title", "overview"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DegradedResult{MissingFields: tt.missingFields}
			assert.Equal(t, tt.expected, result.HasMissingFields())
		})
	}
}

func TestDegradedResult_HasFallbackUsed(t *testing.T) {
	tests := []struct {
		name         string
		fallbackUsed []string
		expected     bool
	}{
		{"no fallback used", nil, false},
		{"empty fallback used", []string{}, false},
		{"with fallback used", []string{"regex_fallback"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DegradedResult{FallbackUsed: tt.fallbackUsed}
			assert.Equal(t, tt.expected, result.HasFallbackUsed())
		})
	}
}

func TestDegradedResult_IsDegraded(t *testing.T) {
	tests := []struct {
		name     string
		level    DegradationLevel
		expected bool
	}{
		{"normal level", DegradationNormal, false},
		{"partial level", DegradationPartial, true},
		{"minimal level", DegradationMinimal, true},
		{"offline level", DegradationOffline, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DegradedResult{DegradationLevel: tt.level}
			assert.Equal(t, tt.expected, result.IsDegraded())
		})
	}
}

func TestFieldAvailability_String(t *testing.T) {
	field := FieldAvailability{
		Field:     "title",
		Available: true,
		Source:    "tmdb",
	}

	assert.Equal(t, "title", field.Field)
	assert.True(t, field.Available)
	assert.Equal(t, "tmdb", field.Source)
}

func TestServiceName_Constants(t *testing.T) {
	assert.Equal(t, ServiceName("tmdb"), ServiceNameTMDb)
	assert.Equal(t, ServiceName("douban"), ServiceNameDouban)
	assert.Equal(t, ServiceName("wikipedia"), ServiceNameWikipedia)
	assert.Equal(t, ServiceName("ai"), ServiceNameAI)
}
