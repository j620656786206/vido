package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// MockLogRepo
type MockLogRepo struct {
	mock.Mock
}

func (m *MockLogRepo) GetLogs(ctx context.Context, filter models.LogFilter) ([]models.SystemLog, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]models.SystemLog), args.Int(1), args.Error(2)
}

func (m *MockLogRepo) CreateLog(ctx context.Context, log *models.SystemLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockLogRepo) CreateLogBatch(ctx context.Context, logs []models.SystemLog) error {
	args := m.Called(ctx, logs)
	return args.Error(0)
}

func (m *MockLogRepo) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	args := m.Called(ctx, days)
	return args.Get(0).(int64), args.Error(1)
}

func TestLogService_GetLogs(t *testing.T) {
	ctx := context.Background()

	t.Run("success with enrichment", func(t *testing.T) {
		repo := new(MockLogRepo)
		svc := NewLogService(repo)

		now := time.Now()
		logs := []models.SystemLog{
			{
				ID:          1,
				Level:       models.LogLevelError,
				Message:     "Failed to fetch metadata",
				Source:      "tmdb",
				ContextJSON: `{"error_code": "TMDB_TIMEOUT", "movie_id": "123"}`,
				CreatedAt:   now,
			},
			{
				ID:        2,
				Level:     models.LogLevelInfo,
				Message:   "Server started",
				CreatedAt: now,
			},
		}

		filter := models.LogFilter{Page: 1, PerPage: 50}
		repo.On("GetLogs", mock.Anything, filter).Return(logs, 2, nil)

		result, err := svc.GetLogs(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Total)
		assert.Len(t, result.Logs, 2)

		// ERROR entry should have hint
		assert.NotEmpty(t, result.Logs[0].Hint)
		assert.Contains(t, result.Logs[0].Hint, "TMDb")

		// INFO entry should have no hint
		assert.Empty(t, result.Logs[1].Hint)

		// Context should be parsed
		assert.NotNil(t, result.Logs[0].Context)

		repo.AssertExpectations(t)
	})

	t.Run("invalid level", func(t *testing.T) {
		repo := new(MockLogRepo)
		svc := NewLogService(repo)

		_, err := svc.GetLogs(ctx, models.LogFilter{Level: "INVALID"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid log level")
	})

	t.Run("defaults pagination", func(t *testing.T) {
		repo := new(MockLogRepo)
		svc := NewLogService(repo)

		expectedFilter := models.LogFilter{Page: 1, PerPage: 50}
		repo.On("GetLogs", mock.Anything, expectedFilter).Return([]models.SystemLog{}, 0, nil)

		result, err := svc.GetLogs(ctx, models.LogFilter{})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 50, result.PerPage)
	})
}

func TestLogService_ClearLogs(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := new(MockLogRepo)
		svc := NewLogService(repo)

		repo.On("DeleteOlderThan", mock.Anything, 30).Return(int64(42), nil)

		result, err := svc.ClearLogs(ctx, 30)
		require.NoError(t, err)
		assert.Equal(t, int64(42), result.EntriesRemoved)
		assert.Equal(t, 30, result.Days)

		repo.AssertExpectations(t)
	})

	t.Run("invalid days", func(t *testing.T) {
		repo := new(MockLogRepo)
		svc := NewLogService(repo)

		_, err := svc.ClearLogs(ctx, 0)
		assert.Error(t, err)

		_, err = svc.ClearLogs(ctx, -1)
		assert.Error(t, err)
	})
}

func TestFindHint(t *testing.T) {
	t.Run("matches error_code key", func(t *testing.T) {
		hint := findHint(map[string]interface{}{"error_code": "TMDB_TIMEOUT"}, "")
		assert.Contains(t, hint, "TMDb")
	})

	t.Run("matches code key", func(t *testing.T) {
		hint := findHint(map[string]interface{}{"code": "QBT_CONNECTION"}, "")
		assert.Contains(t, hint, "qBittorrent")
	})

	t.Run("matches message fallback", func(t *testing.T) {
		hint := findHint(map[string]interface{}{}, "AI_QUOTA_EXCEEDED error occurred")
		assert.Contains(t, hint, "AI")
	})

	t.Run("no match returns empty", func(t *testing.T) {
		hint := findHint(map[string]interface{}{}, "random message")
		assert.Empty(t, hint)
	})

	// All known error codes produce hints
	t.Run("all known error codes", func(t *testing.T) {
		knownCodes := []string{
			"TMDB_TIMEOUT", "AI_QUOTA_EXCEEDED", "DB_QUERY_FAILED",
			"QBT_CONNECTION", "TMDB_NOT_FOUND", "TMDB_RATE_LIMIT",
			"AI_TIMEOUT", "AUTH_TOKEN_EXPIRED",
		}
		for _, code := range knownCodes {
			hint := findHint(map[string]interface{}{"error_code": code}, "")
			assert.NotEmpty(t, hint, "expected hint for code: %s", code)
		}
	})

	t.Run("error_code takes precedence over code", func(t *testing.T) {
		hint := findHint(map[string]interface{}{
			"error_code": "TMDB_TIMEOUT",
			"code":       "QBT_CONNECTION",
		}, "")
		assert.Contains(t, hint, "TMDb")
	})

	t.Run("non-string error_code is ignored", func(t *testing.T) {
		hint := findHint(map[string]interface{}{"error_code": 12345}, "")
		assert.Empty(t, hint)
	})

	t.Run("empty context empty message", func(t *testing.T) {
		hint := findHint(map[string]interface{}{}, "")
		assert.Empty(t, hint)
	})

	t.Run("nil context map", func(t *testing.T) {
		hint := findHint(nil, "some error")
		assert.Empty(t, hint)
	})
}

func TestLogService_GetLogs_MalformedJSON(t *testing.T) {
	ctx := context.Background()
	repo := new(MockLogRepo)
	svc := NewLogService(repo)

	logs := []models.SystemLog{
		{
			ID:          1,
			Level:       models.LogLevelError,
			Message:     "Error with bad context",
			ContextJSON: `{invalid json`,
			CreatedAt:   time.Now(),
		},
	}

	repo.On("GetLogs", mock.Anything, mock.Anything).Return(logs, 1, nil)

	result, err := svc.GetLogs(ctx, models.LogFilter{Page: 1, PerPage: 50})
	require.NoError(t, err)
	assert.Len(t, result.Logs, 1)
	// Malformed JSON should not crash — Context stays nil
	assert.Nil(t, result.Logs[0].Context)
	assert.Empty(t, result.Logs[0].Hint)
}

func TestLogService_GetLogs_EmptyResult(t *testing.T) {
	ctx := context.Background()
	repo := new(MockLogRepo)
	svc := NewLogService(repo)

	repo.On("GetLogs", mock.Anything, mock.Anything).Return([]models.SystemLog{}, 0, nil)

	result, err := svc.GetLogs(ctx, models.LogFilter{Page: 1, PerPage: 50})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Logs)
}

func TestLogService_GetLogs_RepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockLogRepo)
	svc := NewLogService(repo)

	repo.On("GetLogs", mock.Anything, mock.Anything).Return([]models.SystemLog(nil), 0, assert.AnError)

	_, err := svc.GetLogs(ctx, models.LogFilter{Page: 1, PerPage: 50})
	assert.Error(t, err)
}

func TestLogService_ClearLogs_RepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockLogRepo)
	svc := NewLogService(repo)

	repo.On("DeleteOlderThan", mock.Anything, 7).Return(int64(0), assert.AnError)

	_, err := svc.ClearLogs(ctx, 7)
	assert.Error(t, err)
}
