package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepositories(t *testing.T) {
	// Test with nil db - this is just to verify the factory function works
	// In real usage, a valid *sql.DB would be passed
	repos := NewRepositories(nil)

	assert.NotNil(t, repos)
	assert.NotNil(t, repos.Movies)
	assert.NotNil(t, repos.Series)
	assert.NotNil(t, repos.Settings)
	assert.Nil(t, repos.Cache) // Cache is not initialized yet
}

func TestRepositories_SetCacheRepository(t *testing.T) {
	repos := NewRepositories(nil)

	// Initially cache is nil
	assert.Nil(t, repos.Cache)

	// After setting, cache should be available
	// Note: We can't test with real CacheRepository yet as it's not implemented
	// This test will be expanded in Task 4
}
