package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/media"
)

func TestNewMediaService_WithValidDirectories(t *testing.T) {
	dir := t.TempDir()

	service := NewMediaService([]string{dir})

	assert.NotNil(t, service)
	config := service.GetConfig()
	assert.Equal(t, 1, config.TotalCount)
	assert.Equal(t, 1, config.ValidCount)
	assert.False(t, config.SearchOnlyMode)
}

func TestNewMediaService_SearchOnlyMode(t *testing.T) {
	service := NewMediaService([]string{})

	assert.NotNil(t, service)
	assert.True(t, service.IsSearchOnlyMode())
}

func TestNewMediaService_NilDirectories(t *testing.T) {
	service := NewMediaService(nil)

	assert.NotNil(t, service)
	assert.True(t, service.IsSearchOnlyMode())
	assert.Equal(t, 0, service.GetConfig().TotalCount)
}

func TestMediaService_GetConfiguredDirectories(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	service := NewMediaService([]string{dir1, dir2})
	dirs := service.GetConfiguredDirectories()

	assert.Len(t, dirs, 2)
}

func TestMediaService_GetAccessibleDirectories(t *testing.T) {
	validDir := t.TempDir()
	invalidDir := "/nonexistent/path/for/testing"

	service := NewMediaService([]string{validDir, invalidDir})
	dirs := service.GetAccessibleDirectories()

	assert.Len(t, dirs, 1)
	assert.Equal(t, validDir, dirs[0].Path)
	assert.Equal(t, media.StatusAccessible, dirs[0].Status)
}

func TestMediaService_RefreshDirectoryStatus(t *testing.T) {
	dir := t.TempDir()

	service := NewMediaService([]string{dir})

	// Initial check
	config := service.GetConfig()
	assert.Equal(t, 1, config.ValidCount)

	// Refresh should return same result
	refreshed := service.RefreshDirectoryStatus()
	assert.Equal(t, 1, refreshed.ValidCount)
}

func TestMediaService_IsSearchOnlyMode_True(t *testing.T) {
	service := NewMediaService([]string{"/nonexistent1", "/nonexistent2"})

	assert.True(t, service.IsSearchOnlyMode())
}

func TestMediaService_IsSearchOnlyMode_False(t *testing.T) {
	dir := t.TempDir()

	service := NewMediaService([]string{dir})

	assert.False(t, service.IsSearchOnlyMode())
}

func TestMediaService_ThreadSafety(t *testing.T) {
	dir := t.TempDir()

	service := NewMediaService([]string{dir})

	// Run concurrent operations to test thread safety
	done := make(chan bool, 10)

	for i := 0; i < 5; i++ {
		go func() {
			_ = service.GetConfig()
			done <- true
		}()
		go func() {
			_ = service.RefreshDirectoryStatus()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMediaService_StoresDirsForRefresh(t *testing.T) {
	dir := t.TempDir()

	service := NewMediaService([]string{dir})

	// Verify dirs are stored for refresh
	assert.NotNil(t, service.dirs)
	assert.Len(t, service.dirs, 1)
	assert.Equal(t, dir, service.dirs[0])
}
