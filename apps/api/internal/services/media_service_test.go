package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/media"
)

func TestNewMediaService_WithValidDirectories(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIDO_MEDIA_DIRS", dir)

	service := NewMediaService()

	assert.NotNil(t, service)
	config := service.GetConfig()
	assert.Equal(t, 1, config.TotalCount)
	assert.Equal(t, 1, config.ValidCount)
	assert.False(t, config.SearchOnlyMode)
}

func TestNewMediaService_SearchOnlyMode(t *testing.T) {
	os.Unsetenv("VIDO_MEDIA_DIRS")
	t.Setenv("VIDO_MEDIA_DIRS", "")

	service := NewMediaService()

	assert.NotNil(t, service)
	assert.True(t, service.IsSearchOnlyMode())
}

func TestMediaService_GetConfiguredDirectories(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	t.Setenv("VIDO_MEDIA_DIRS", dir1+","+dir2)

	service := NewMediaService()
	dirs := service.GetConfiguredDirectories()

	assert.Len(t, dirs, 2)
}

func TestMediaService_GetAccessibleDirectories(t *testing.T) {
	validDir := t.TempDir()
	invalidDir := "/nonexistent/path/for/testing"
	t.Setenv("VIDO_MEDIA_DIRS", validDir+","+invalidDir)

	service := NewMediaService()
	dirs := service.GetAccessibleDirectories()

	assert.Len(t, dirs, 1)
	assert.Equal(t, validDir, dirs[0].Path)
	assert.Equal(t, media.StatusAccessible, dirs[0].Status)
}

func TestMediaService_RefreshDirectoryStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIDO_MEDIA_DIRS", dir)

	service := NewMediaService()

	// Initial check
	config := service.GetConfig()
	assert.Equal(t, 1, config.ValidCount)

	// Refresh should return same result
	refreshed := service.RefreshDirectoryStatus()
	assert.Equal(t, 1, refreshed.ValidCount)
}

func TestMediaService_IsSearchOnlyMode_True(t *testing.T) {
	t.Setenv("VIDO_MEDIA_DIRS", "/nonexistent1,/nonexistent2")

	service := NewMediaService()

	assert.True(t, service.IsSearchOnlyMode())
}

func TestMediaService_IsSearchOnlyMode_False(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIDO_MEDIA_DIRS", dir)

	service := NewMediaService()

	assert.False(t, service.IsSearchOnlyMode())
}

func TestMediaService_ThreadSafety(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIDO_MEDIA_DIRS", dir)

	service := NewMediaService()

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
