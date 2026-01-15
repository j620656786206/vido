package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCacheRepository is a mock implementation of CacheRepositoryInterface for testing
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (*CacheEntry, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CacheEntry), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value string, cacheType string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, cacheType, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheRepository) ClearExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheRepository) ClearByType(ctx context.Context, cacheType string) (int64, error) {
	args := m.Called(ctx, cacheType)
	return args.Get(0).(int64), args.Error(1)
}

// Verify mock implements interface
var _ CacheRepositoryInterface = (*MockCacheRepository)(nil)

func TestCacheRepository_ValidationErrors(t *testing.T) {
	repo := NewCacheRepository(nil)
	ctx := context.Background()

	t.Run("Get with empty key returns error", func(t *testing.T) {
		_, err := repo.Get(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache key cannot be empty")
	})

	t.Run("Set with empty key returns error", func(t *testing.T) {
		err := repo.Set(ctx, "", "value", "type", time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache key cannot be empty")
	})

	t.Run("Set with zero TTL returns error", func(t *testing.T) {
		err := repo.Set(ctx, "key", "value", "type", 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache TTL must be positive")
	})

	t.Run("Set with negative TTL returns error", func(t *testing.T) {
		err := repo.Set(ctx, "key", "value", "type", -time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache TTL must be positive")
	})

	t.Run("Delete with empty key returns error", func(t *testing.T) {
		err := repo.Delete(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache key cannot be empty")
	})

	t.Run("ClearByType with empty type returns error", func(t *testing.T) {
		_, err := repo.ClearByType(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache type cannot be empty")
	})
}

func TestMockCacheRepository_Interface(t *testing.T) {
	mockRepo := new(MockCacheRepository)
	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		mockRepo.On("Set", ctx, "test-key", "test-value", "tmdb", time.Hour).Return(nil)
		mockRepo.On("Get", ctx, "test-key").Return(&CacheEntry{
			Key:   "test-key",
			Value: "test-value",
			Type:  "tmdb",
		}, nil)

		err := mockRepo.Set(ctx, "test-key", "test-value", "tmdb", time.Hour)
		assert.NoError(t, err)

		entry, err := mockRepo.Get(ctx, "test-key")
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, "test-value", entry.Value)
	})

	t.Run("Delete", func(t *testing.T) {
		mockRepo.On("Delete", ctx, "delete-key").Return(nil)

		err := mockRepo.Delete(ctx, "delete-key")
		assert.NoError(t, err)
	})

	t.Run("Clear", func(t *testing.T) {
		mockRepo.On("Clear", ctx).Return(nil)

		err := mockRepo.Clear(ctx)
		assert.NoError(t, err)
	})

	t.Run("ClearExpired", func(t *testing.T) {
		mockRepo.On("ClearExpired", ctx).Return(int64(5), nil)

		count, err := mockRepo.ClearExpired(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("ClearByType", func(t *testing.T) {
		mockRepo.On("ClearByType", ctx, "tmdb").Return(int64(10), nil)

		count, err := mockRepo.ClearByType(ctx, "tmdb")
		assert.NoError(t, err)
		assert.Equal(t, int64(10), count)
	})

	mockRepo.AssertExpectations(t)
}
