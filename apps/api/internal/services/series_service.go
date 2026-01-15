package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// SeriesService provides business logic for TV series operations.
// It uses SeriesRepositoryInterface for data access, enabling
// testing with mock implementations and future database migrations.
type SeriesService struct {
	repo repository.SeriesRepositoryInterface
}

// NewSeriesService creates a new SeriesService with the given repository.
func NewSeriesService(repo repository.SeriesRepositoryInterface) *SeriesService {
	return &SeriesService{
		repo: repo,
	}
}

// Create validates and creates a new series.
// It generates a UUID if the series doesn't have an ID.
func (s *SeriesService) Create(ctx context.Context, series *models.Series) error {
	if series == nil {
		return fmt.Errorf("series cannot be nil")
	}

	if series.Title == "" {
		return fmt.Errorf("series title cannot be empty")
	}

	// Generate ID if not provided
	if series.ID == "" {
		series.ID = uuid.New().String()
	}

	slog.Info("Creating series", "series_id", series.ID, "title", series.Title)

	if err := s.repo.Create(ctx, series); err != nil {
		slog.Error("Failed to create series", "error", err, "series_id", series.ID)
		return fmt.Errorf("failed to create series: %w", err)
	}

	return nil
}

// GetByID retrieves a series by its ID.
func (s *SeriesService) GetByID(ctx context.Context, id string) (*models.Series, error) {
	if id == "" {
		return nil, fmt.Errorf("series id cannot be empty")
	}

	series, err := s.repo.FindByID(ctx, id)
	if err != nil {
		slog.Error("Failed to get series", "error", err, "series_id", id)
		return nil, err
	}

	return series, nil
}

// GetByTMDbID retrieves a series by its TMDb ID.
func (s *SeriesService) GetByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	if tmdbID <= 0 {
		return nil, fmt.Errorf("tmdb id must be positive")
	}

	series, err := s.repo.FindByTMDbID(ctx, tmdbID)
	if err != nil {
		slog.Error("Failed to get series by TMDb ID", "error", err, "tmdb_id", tmdbID)
		return nil, err
	}

	return series, nil
}

// GetByIMDbID retrieves a series by its IMDb ID.
func (s *SeriesService) GetByIMDbID(ctx context.Context, imdbID string) (*models.Series, error) {
	if imdbID == "" {
		return nil, fmt.Errorf("imdb id cannot be empty")
	}

	series, err := s.repo.FindByIMDbID(ctx, imdbID)
	if err != nil {
		slog.Error("Failed to get series by IMDb ID", "error", err, "imdb_id", imdbID)
		return nil, err
	}

	return series, nil
}

// Update validates and updates an existing series.
func (s *SeriesService) Update(ctx context.Context, series *models.Series) error {
	if series == nil {
		return fmt.Errorf("series cannot be nil")
	}

	if series.ID == "" {
		return fmt.Errorf("series id cannot be empty")
	}

	if series.Title == "" {
		return fmt.Errorf("series title cannot be empty")
	}

	slog.Info("Updating series", "series_id", series.ID, "title", series.Title)

	if err := s.repo.Update(ctx, series); err != nil {
		slog.Error("Failed to update series", "error", err, "series_id", series.ID)
		return fmt.Errorf("failed to update series: %w", err)
	}

	return nil
}

// Delete removes a series by its ID.
func (s *SeriesService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("series id cannot be empty")
	}

	slog.Info("Deleting series", "series_id", id)

	if err := s.repo.Delete(ctx, id); err != nil {
		slog.Error("Failed to delete series", "error", err, "series_id", id)
		return fmt.Errorf("failed to delete series: %w", err)
	}

	return nil
}

// List retrieves series with pagination support.
func (s *SeriesService) List(ctx context.Context, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	series, pagination, err := s.repo.List(ctx, params)
	if err != nil {
		slog.Error("Failed to list series", "error", err)
		return nil, nil, fmt.Errorf("failed to list series: %w", err)
	}

	return series, pagination, nil
}

// SearchByTitle searches for series by title with pagination.
func (s *SeriesService) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	if title == "" {
		return nil, nil, fmt.Errorf("search title cannot be empty")
	}

	series, pagination, err := s.repo.SearchByTitle(ctx, title, params)
	if err != nil {
		slog.Error("Failed to search series", "error", err, "title", title)
		return nil, nil, fmt.Errorf("failed to search series: %w", err)
	}

	return series, pagination, nil
}
