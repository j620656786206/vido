package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
	RemovePath(ctx context.Context, libraryID string, pathID string) error
	RefreshPathStatuses(ctx context.Context, libraryID string) ([]models.MediaLibraryPath, error)
}

// CreateLibraryRequest is the input for creating a library.
type CreateLibraryRequest struct {
	Name        string   `json:"name"`
	ContentType string   `json:"content_type"`
	Paths       []string `json:"paths"`
	// AutoSubtitle carries the free-auto-generation opt-in through CREATE too
	// (CR M2). A plain bool, unlike the update request's pointer: absent means
	// false, which is the correct default for a brand-new library. Without this
	// the modal rendered a checkbox whose value was silently discarded on
	// create — the user ticked it, pressed 建立, and nothing said otherwise.
	AutoSubtitle bool `json:"auto_subtitle"`
}

// UpdateLibraryRequest is the input for updating a library.
type UpdateLibraryRequest struct {
	Name        *string `json:"name,omitempty"`
	ContentType *string `json:"content_type,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	// AutoSubtitle opts this library in to free subtitle auto-generation
	// (Story 9R-10b AC #2). A POINTER like its siblings: absent means "leave
	// as-is", so a form that does not know about the setting cannot silently
	// switch it off — or, worse, on.
	AutoSubtitle *bool `json:"auto_subtitle,omitempty"`
}

// MediaLibraryService implements MediaLibraryServiceInterface.
type MediaLibraryService struct {
	repo repository.MediaLibraryRepositoryInterface

	// Type-change rebuild wiring (setter-injected; see SetMediaPurgers /
	// SetScanTrigger).
	moviePurger  libraryMediaPurger
	seriesPurger libraryMediaPurger
	scanTrigger  func()
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
		Name:         req.Name,
		ContentType:  models.MediaLibraryContentType(req.ContentType),
		AutoSubtitle: req.AutoSubtitle,
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

// libraryMediaPurger is the narrow port a media-row table exposes for the
// type-change rebuild. Both MovieRepository and SeriesRepository satisfy it
// (series deletion cascades seasons/episodes via FK).
type libraryMediaPurger interface {
	DeleteByLibraryID(ctx context.Context, libraryID string) (int64, error)
}

// SetMediaPurgers wires the movie/series purgers used by the content-type
// rebuild. Setter (not constructor) because main.go builds this service before
// the repos-consuming wiring settles — mirrors SetSeriesRepo precedent.
func (s *MediaLibraryService) SetMediaPurgers(movies, series libraryMediaPurger) {
	s.moviePurger = movies
	s.seriesPurger = series
}

// SetScanTrigger wires an async full-scan trigger (the same code path as the
// manual 掃描媒體庫 button). Fired after a type-change purge so the library is
// rebuilt under its new type without a manual step.
func (s *MediaLibraryService) SetScanTrigger(trigger func()) {
	s.scanTrigger = trigger
}

// rebuildLibraryMedia purges every media row belonging to the library and, when
// wired, kicks a rescan. Research note (2026-08-31): Plex forbids changing a
// library's type outright — the official path is delete-and-recreate the
// library; Jellyfin nominally allows it but community guidance is likewise to
// recreate. Vido allows the change AND automates the rebuild: purge + rescan is
// the delete-and-recreate, done for the user (⚖️ Alexyu 2026-08-31,內測實測
// bugfix-library-type-change-no-reclassify).
func (s *MediaLibraryService) rebuildLibraryMedia(ctx context.Context, libraryID string) {
	if s.moviePurger == nil || s.seriesPurger == nil {
		slog.Warn("Library rebuild skipped: media purgers not wired", "library_id", libraryID)
		return
	}
	movies, err := s.moviePurger.DeleteByLibraryID(ctx, libraryID)
	if err != nil {
		slog.Error("Library rebuild: movie purge failed", "library_id", libraryID, "error", err)
	}
	series, err := s.seriesPurger.DeleteByLibraryID(ctx, libraryID)
	if err != nil {
		slog.Error("Library rebuild: series purge failed", "library_id", libraryID, "error", err)
	}
	slog.Info("Library media purged for type change",
		"library_id", libraryID, "movies_removed", movies, "series_removed", series)
	if s.scanTrigger != nil {
		s.scanTrigger()
	}
}

func (s *MediaLibraryService) UpdateLibrary(ctx context.Context, id string, req UpdateLibraryRequest) (*models.MediaLibrary, error) {
	lib, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get library for update: %w", err)
	}
	oldType := lib.ContentType

	if req.Name != nil {
		lib.Name = *req.Name
	}
	if req.ContentType != nil {
		lib.ContentType = models.MediaLibraryContentType(*req.ContentType)
	}
	if req.SortOrder != nil {
		lib.SortOrder = *req.SortOrder
	}
	if req.AutoSubtitle != nil {
		lib.AutoSubtitle = *req.AutoSubtitle
	}

	if err := lib.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	if err := s.repo.Update(ctx, lib); err != nil {
		return nil, fmt.Errorf("update library: %w", err)
	}

	// Content type actually changed → existing rows were classified under the
	// old type and are now all wrong. Rebuild: purge this library's rows and
	// rescan under the new type.
	if req.ContentType != nil && lib.ContentType != oldType {
		s.rebuildLibraryMedia(ctx, lib.ID)
	}

	slog.Info("Library updated", "id", lib.ID, "name", lib.Name)
	return lib, nil
}

func (s *MediaLibraryService) DeleteLibrary(ctx context.Context, id string, removeMedia bool) error {
	if removeMedia && s.moviePurger != nil && s.seriesPurger != nil {
		slog.Info("Deleting library with media removal", "id", id)
		if _, err := s.moviePurger.DeleteByLibraryID(ctx, id); err != nil {
			slog.Error("Library delete: movie purge failed", "library_id", id, "error", err)
		}
		if _, err := s.seriesPurger.DeleteByLibraryID(ctx, id); err != nil {
			slog.Error("Library delete: series purge failed", "library_id", id, "error", err)
		}
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
	// Sanitize and validate path
	cleaned := filepath.Clean(pathStr)
	if !filepath.IsAbs(cleaned) {
		return nil, fmt.Errorf("validation: path must be absolute")
	}

	p := &models.MediaLibraryPath{
		LibraryID: libraryID,
		Path:      cleaned,
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

func (s *MediaLibraryService) RemovePath(ctx context.Context, libraryID string, pathID string) error {
	// Verify path belongs to the specified library
	paths, err := s.repo.GetPathsByLibraryID(ctx, libraryID)
	if err != nil {
		return fmt.Errorf("get paths for ownership check: %w", err)
	}

	found := false
	for _, p := range paths {
		if p.ID == pathID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("path %s does not belong to library %s: %w", pathID, libraryID, repository.ErrLibraryPathNotFound)
	}

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
