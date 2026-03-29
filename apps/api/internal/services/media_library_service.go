package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MediaLibraryServiceInterface defines the contract for media library operations.
type MediaLibraryServiceInterface interface {
	GetAllLibraries(ctx context.Context) ([]models.MediaLibraryWithPaths, error)
	GetLibrary(ctx context.Context, id string) (*models.MediaLibraryWithPaths, error)
	CreateLibrary(ctx context.Context, req CreateLibraryRequest) (*models.MediaLibrary, error)
	UpdateLibrary(ctx context.Context, id string, req UpdateLibraryRequest) (*models.MediaLibrary, error)
	DeleteLibrary(ctx context.Context, id string, removeMedia bool) error
	AddPath(ctx context.Context, libraryID string, path string) (*models.MediaLibraryPath, error)
	RemovePath(ctx context.Context, pathID string) error
	RefreshPathStatuses(ctx context.Context, libraryID string) ([]models.MediaLibraryPath, error)
}

// CreateLibraryRequest is the input for creating a library.
type CreateLibraryRequest struct {
	Name        string   `json:"name"`
	ContentType string   `json:"content_type"`
	Paths       []string `json:"paths"`
}

// UpdateLibraryRequest is the input for updating a library.
type UpdateLibraryRequest struct {
	Name        *string `json:"name,omitempty"`
	ContentType *string `json:"content_type,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// MediaLibraryService implements MediaLibraryServiceInterface.
type MediaLibraryService struct {
	repo repository.MediaLibraryRepositoryInterface
}

// NewMediaLibraryService creates a new MediaLibraryService.
func NewMediaLibraryService(repo repository.MediaLibraryRepositoryInterface) *MediaLibraryService {
	return &MediaLibraryService{repo: repo}
}

func (s *MediaLibraryService) GetAllLibraries(ctx context.Context) ([]models.MediaLibraryWithPaths, error) {
	libraries, err := s.repo.GetAllWithPathsAndCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all libraries: %w", err)
	}
	return libraries, nil
}

func (s *MediaLibraryService) GetLibrary(ctx context.Context, id string) (*models.MediaLibraryWithPaths, error) {
	lib, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get library: %w", err)
	}

	paths, err := s.repo.GetPathsByLibraryID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get library paths: %w", err)
	}

	return &models.MediaLibraryWithPaths{
		MediaLibrary: *lib,
		Paths:        paths,
	}, nil
}

func (s *MediaLibraryService) CreateLibrary(ctx context.Context, req CreateLibraryRequest) (*models.MediaLibrary, error) {
	lib := &models.MediaLibrary{
		Name:        req.Name,
		ContentType: models.MediaLibraryContentType(req.ContentType),
	}

	if err := lib.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	if err := s.repo.Create(ctx, lib); err != nil {
		return nil, fmt.Errorf("create library: %w", err)
	}

	slog.Info("Library created", "id", lib.ID, "name", lib.Name, "type", lib.ContentType)

	// Add paths
	for _, pathStr := range req.Paths {
		if _, err := s.addPathInternal(ctx, lib.ID, pathStr); err != nil {
			slog.Warn("Failed to add path during library creation", "path", pathStr, "error", err)
		}
	}

	return lib, nil
}

func (s *MediaLibraryService) UpdateLibrary(ctx context.Context, id string, req UpdateLibraryRequest) (*models.MediaLibrary, error) {
	lib, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get library for update: %w", err)
	}

	if req.Name != nil {
		lib.Name = *req.Name
	}
	if req.ContentType != nil {
		lib.ContentType = models.MediaLibraryContentType(*req.ContentType)
	}
	if req.SortOrder != nil {
		lib.SortOrder = *req.SortOrder
	}

	if err := lib.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	if err := s.repo.Update(ctx, lib); err != nil {
		return nil, fmt.Errorf("update library: %w", err)
	}

	slog.Info("Library updated", "id", lib.ID, "name", lib.Name)
	return lib, nil
}

func (s *MediaLibraryService) DeleteLibrary(ctx context.Context, id string, removeMedia bool) error {
	if removeMedia {
		slog.Info("Deleting library with media removal", "id", id)
		// TODO: remove associated movies/series records in Story 7b-5
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete library: %w", err)
	}

	slog.Info("Library deleted", "id", id)
	return nil
}

func (s *MediaLibraryService) AddPath(ctx context.Context, libraryID string, pathStr string) (*models.MediaLibraryPath, error) {
	// Verify library exists
	if _, err := s.repo.GetByID(ctx, libraryID); err != nil {
		return nil, fmt.Errorf("get library: %w", err)
	}

	return s.addPathInternal(ctx, libraryID, pathStr)
}

func (s *MediaLibraryService) addPathInternal(ctx context.Context, libraryID string, pathStr string) (*models.MediaLibraryPath, error) {
	p := &models.MediaLibraryPath{
		LibraryID: libraryID,
		Path:      pathStr,
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// Check path accessibility
	p.Status = checkPathStatus(pathStr)

	if err := s.repo.AddPath(ctx, p); err != nil {
		return nil, fmt.Errorf("add path: %w", err)
	}

	slog.Info("Path added to library", "library_id", libraryID, "path", pathStr, "status", p.Status)
	return p, nil
}

func (s *MediaLibraryService) RemovePath(ctx context.Context, pathID string) error {
	if err := s.repo.RemovePath(ctx, pathID); err != nil {
		return fmt.Errorf("remove path: %w", err)
	}
	return nil
}

func (s *MediaLibraryService) RefreshPathStatuses(ctx context.Context, libraryID string) ([]models.MediaLibraryPath, error) {
	paths, err := s.repo.GetPathsByLibraryID(ctx, libraryID)
	if err != nil {
		return nil, fmt.Errorf("get paths: %w", err)
	}

	for i := range paths {
		newStatus := checkPathStatus(paths[i].Path)
		if err := s.repo.UpdatePathStatus(ctx, paths[i].ID, newStatus); err != nil {
			slog.Warn("Failed to update path status", "path_id", paths[i].ID, "error", err)
			continue
		}
		paths[i].Status = newStatus
	}

	return paths, nil
}

// checkPathStatus validates a filesystem path and returns its status.
func checkPathStatus(path string) models.MediaLibraryPathStatus {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return models.PathStatusNotFound
		}
		if os.IsPermission(err) {
			return models.PathStatusNotReadable
		}
		return models.PathStatusNotFound
	}
	if !info.IsDir() {
		return models.PathStatusNotDirectory
	}
	return models.PathStatusAccessible
}
