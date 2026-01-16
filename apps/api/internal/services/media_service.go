package services

import (
	"log/slog"
	"sync"

	"github.com/vido/api/internal/media"
)

// MediaServiceInterface defines the contract for media directory operations.
// This interface enables testing handlers with mock services.
type MediaServiceInterface interface {
	// GetConfig returns the current media configuration
	GetConfig() *media.MediaConfig
	// GetConfiguredDirectories returns all configured directories
	GetConfiguredDirectories() []media.MediaDirectory
	// GetAccessibleDirectories returns only accessible directories
	GetAccessibleDirectories() []media.MediaDirectory
	// RefreshDirectoryStatus re-validates all directories and returns updated config
	RefreshDirectoryStatus() *media.MediaConfig
	// IsSearchOnlyMode returns true if no accessible directories are configured
	IsSearchOnlyMode() bool
}

// MediaService provides business logic for media directory operations.
// It caches the directory validation results and provides thread-safe access.
type MediaService struct {
	config *media.MediaConfig
	mu     sync.RWMutex
}

// NewMediaService creates a new MediaService and loads the initial configuration.
// It validates all configured directories and logs their status.
func NewMediaService() *MediaService {
	config := media.LoadMediaConfig()
	media.LogMediaConfigStatus(config)

	return &MediaService{
		config: config,
	}
}

// GetConfig returns the current media configuration with all directories and their status.
func (s *MediaService) GetConfig() *media.MediaConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetConfiguredDirectories returns all configured directories regardless of status.
func (s *MediaService) GetConfiguredDirectories() []media.MediaDirectory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Directories
}

// GetAccessibleDirectories returns only the directories that are accessible.
func (s *MediaService) GetAccessibleDirectories() []media.MediaDirectory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.GetAccessibleDirectories()
}

// RefreshDirectoryStatus re-validates all configured directories.
// This is useful when directories may have been mounted/unmounted at runtime.
func (s *MediaService) RefreshDirectoryStatus() *media.MediaConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("Refreshing media directory status")
	s.config = media.LoadMediaConfig()
	media.LogMediaConfigStatus(s.config)

	return s.config
}

// IsSearchOnlyMode returns true if no accessible directories are configured.
// In this mode, the application operates in search-only mode without library features.
func (s *MediaService) IsSearchOnlyMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.SearchOnlyMode
}
