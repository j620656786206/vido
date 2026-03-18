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
}
