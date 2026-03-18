package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceHealth_ToServiceStatus_Connected(t *testing.T) {
	svc := NewServiceHealth("tmdb", "TMDb API")
	svc.RecordSuccess()

	status := svc.ToServiceStatus()
	assert.Equal(t, "tmdb", status.Name)
	assert.Equal(t, "TMDb API", status.DisplayName)
	assert.Equal(t, StatusConnected, status.Status)
	assert.Equal(t, "已連線", status.Message)
	assert.NotNil(t, status.LastSuccessAt)
	assert.Empty(t, status.ErrorMessage)
}

func TestServiceHealth_ToServiceStatus_Error(t *testing.T) {
	svc := NewServiceHealth("ai", "AI 服務")
	svc.RecordError("connection refused")

	status := svc.ToServiceStatus()
	assert.Equal(t, StatusError, status.Status)
	assert.Equal(t, "connection refused", status.ErrorMessage)
}

func TestServiceHealth_ToServiceStatus_Disconnected(t *testing.T) {
	svc := NewServiceHealth("qbittorrent", "qBittorrent")
	// 3+ errors → down → disconnected
	svc.RecordError("err1")
	svc.RecordError("err2")
	svc.RecordError("err3")

	status := svc.ToServiceStatus()
	assert.Equal(t, StatusDisconnected, status.Status)
	assert.Equal(t, "err3", status.ErrorMessage)
}

func TestServiceHealth_ToServiceStatus_RateLimited(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"rate limit text", "TMDb API rate limit exceeded"},
		{"429 status", "HTTP 429 Too Many Requests"},
		{"rate_limit underscore", "rate_limit reached"},
		{"too many requests", "too many requests"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewServiceHealth("tmdb", "TMDb API")
			svc.RecordError(tt.message)

			status := svc.ToServiceStatus()
			assert.Equal(t, StatusRateLimited, status.Status)
			assert.Equal(t, "速率限制中", status.Message)
			assert.Equal(t, tt.message, status.ErrorMessage)
		})
	}
}

func TestServiceHealth_ToServiceStatus_Unconfigured(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"not configured", "AI provider not configured"},
		{"not enabled", "Douban service not enabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewServiceHealth("ai", "AI 服務")
			svc.RecordError(tt.message)

			status := svc.ToServiceStatus()
			assert.Equal(t, StatusUnconfigured, status.Status)
			assert.Equal(t, "未設定", status.Message)
		})
	}
}

func TestServiceHealth_ToServiceStatus_UnconfiguredDown(t *testing.T) {
	svc := NewServiceHealth("ai", "AI 服務")
	svc.RecordError("AI provider not configured")
	svc.RecordError("AI provider not configured")
	svc.RecordError("AI provider not configured")

	status := svc.ToServiceStatus()
	assert.Equal(t, StatusUnconfigured, status.Status)
}

func TestServiceHealth_ToServiceStatus_LastSuccessAt_Nil(t *testing.T) {
	svc := NewServiceHealth("tmdb", "TMDb API")
	// Never had a success
	status := svc.ToServiceStatus()
	assert.Nil(t, status.LastSuccessAt)
}

func TestServiceHealth_SetResponseTime(t *testing.T) {
	svc := NewServiceHealth("tmdb", "TMDb API")
	svc.SetResponseTime(150)

	assert.Equal(t, int64(150), svc.ResponseTimeMs)

	status := svc.ToServiceStatus()
	assert.Equal(t, int64(150), status.ResponseTimeMs)
}

func TestServicesHealth_AllServiceStatuses(t *testing.T) {
	services := NewServicesHealth()
	services.TMDb.RecordSuccess()
	services.TMDb.SetResponseTime(45)
	services.AI.RecordError("not configured")

	statuses := services.AllServiceStatuses()
	require.Len(t, statuses, 5)

	// Find TMDb
	var tmdb *ServiceStatus
	for i := range statuses {
		if statuses[i].Name == "tmdb" {
			tmdb = &statuses[i]
			break
		}
	}
	require.NotNil(t, tmdb)
	assert.Equal(t, StatusConnected, tmdb.Status)
	assert.Equal(t, int64(45), tmdb.ResponseTimeMs)

	// Find AI
	var ai *ServiceStatus
	for i := range statuses {
		if statuses[i].Name == "ai" {
			ai = &statuses[i]
			break
		}
	}
	require.NotNil(t, ai)
	assert.Equal(t, StatusUnconfigured, ai.Status)
}

func TestServiceHealth_ToServiceStatus_LastCheckAt(t *testing.T) {
	svc := NewServiceHealth("tmdb", "TMDb API")
	svc.RecordSuccess()

	status := svc.ToServiceStatus()
	assert.False(t, status.LastCheckAt.IsZero())
	// Should be within last second
	assert.WithinDuration(t, time.Now(), status.LastCheckAt, time.Second)
}
