package logger

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// mockLogRepo is a test mock for LogRepositoryInterface
type mockLogRepo struct {
	mu   sync.Mutex
	logs []models.SystemLog
}

func (m *mockLogRepo) GetLogs(_ context.Context, _ models.LogFilter) ([]models.SystemLog, int, error) {
	return nil, 0, nil
}

func (m *mockLogRepo) CreateLog(_ context.Context, log *models.SystemLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, *log)
	return nil
}

func (m *mockLogRepo) CreateLogBatch(_ context.Context, logs []models.SystemLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logs...)
	return nil
}

func (m *mockLogRepo) DeleteOlderThan(_ context.Context, _ int) (int64, error) {
	return 0, nil
}

func (m *mockLogRepo) getLogs() []models.SystemLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.SystemLog, len(m.logs))
	copy(result, m.logs)
	return result
}

func TestDBHandler_Handle(t *testing.T) {
	repo := &mockLogRepo{}
	handler := NewDBHandler(repo)
	defer handler.Close()

	logger := slog.New(handler)

	logger.Info("Test message", "key", "value")

	// Manually flush
	handler.flush()

	logs := repo.getLogs()
	require.Len(t, logs, 1)
	assert.Equal(t, models.LogLevelInfo, logs[0].Level)
	assert.Equal(t, "Test message", logs[0].Message)
	assert.Contains(t, logs[0].ContextJSON, `"key"`)
}

func TestDBHandler_LevelMapping(t *testing.T) {
	repo := &mockLogRepo{}
	handler := NewDBHandler(repo)
	defer handler.Close()

	logger := slog.New(handler)

	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	handler.flush()

	logs := repo.getLogs()
	require.Len(t, logs, 4)

	assert.Equal(t, models.LogLevelDebug, logs[0].Level)
	assert.Equal(t, models.LogLevelInfo, logs[1].Level)
	assert.Equal(t, models.LogLevelWarn, logs[2].Level)
	assert.Equal(t, models.LogLevelError, logs[3].Level)
}

func TestDBHandler_MasksSensitiveData(t *testing.T) {
	repo := &mockLogRepo{}
	handler := NewDBHandler(repo)
	defer handler.Close()

	logger := slog.New(handler)

	t.Run("masks api_key attribute", func(t *testing.T) {
		logger.Info("Config loaded", "api_key", "abcdef123456789")
		handler.flush()

		logs := repo.getLogs()
		require.NotEmpty(t, logs)
		last := logs[len(logs)-1]
		assert.Contains(t, last.ContextJSON, "abcd****")
		assert.NotContains(t, last.ContextJSON, "abcdef123456789")
	})

	t.Run("masks password attribute", func(t *testing.T) {
		logger.Info("Login attempt", "password", "mysecretpass")
		handler.flush()

		logs := repo.getLogs()
		require.NotEmpty(t, logs)
		last := logs[len(logs)-1]
		assert.Contains(t, last.ContextJSON, "myse****")
		assert.NotContains(t, last.ContextJSON, "mysecretpass")
	})

	t.Run("masks sensitive patterns in message", func(t *testing.T) {
		logger.Info("api_key=abcdef123456789 found in config")
		handler.flush()

		logs := repo.getLogs()
		require.NotEmpty(t, logs)
		last := logs[len(logs)-1]
		assert.NotContains(t, last.Message, "abcdef123456789")
	})
}

func TestDBHandler_BufferFlush(t *testing.T) {
	repo := &mockLogRepo{}
	handler := NewDBHandler(repo)
	defer handler.Close()

	logger := slog.New(handler)

	// Log 100 entries to trigger auto-flush
	for i := 0; i < maxBufferSize; i++ {
		logger.Info("Buffered log", "index", i)
	}

	// Give the handler a moment to flush
	time.Sleep(100 * time.Millisecond)

	logs := repo.getLogs()
	assert.GreaterOrEqual(t, len(logs), maxBufferSize)
}

func TestDBHandler_WithAttrs(t *testing.T) {
	repo := &mockLogRepo{}
	handler := NewDBHandler(repo)
	defer handler.Close()

	childHandler := handler.WithAttrs([]slog.Attr{slog.String("service", "test")})
	logger := slog.New(childHandler)

	logger.Info("With attrs test")
	handler.flush()

	logs := repo.getLogs()
	require.NotEmpty(t, logs)
	assert.Contains(t, logs[len(logs)-1].ContextJSON, "service")
}

func TestDBHandler_WithGroup(t *testing.T) {
	repo := &mockLogRepo{}
	handler := NewDBHandler(repo)
	defer handler.Close()

	childHandler := handler.WithGroup("mygroup")
	logger := slog.New(childHandler)

	logger.Info("With group test", "field", "val")
	handler.flush()

	logs := repo.getLogs()
	require.NotEmpty(t, logs)
	assert.Contains(t, logs[len(logs)-1].ContextJSON, "mygroup")
}

func TestMapLevel(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected models.LogLevel
	}{
		{slog.LevelDebug, models.LogLevelDebug},
		{slog.LevelInfo, models.LogLevelInfo},
		{slog.LevelWarn, models.LogLevelWarn},
		{slog.LevelError, models.LogLevelError},
		{slog.Level(-8), models.LogLevelDebug},  // below debug
		{slog.Level(12), models.LogLevelError},   // above error
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, mapLevel(tt.level))
	}
}

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{"api_key", "mykey12345", "myke****"},
		{"password", "pass", "****"},
		{"token", "tok123456", "tok1****"},
		{"username", "john", "john"},
	}

	for _, tt := range tests {
		result := maskSensitiveValue(tt.key, tt.value)
		assert.Equal(t, tt.expected, result, "key=%s value=%s", tt.key, tt.value)
	}
}
