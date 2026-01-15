package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MovieService provides business logic for movie operations.
// It uses MovieRepositoryInterface for data access, enabling
// testing with mock implementations and future database migrations.
type MovieService struct {
	repo repository.MovieRepositoryInterface
}

// NewMovieService creates a new MovieService with the given repository.
func NewMovieService(repo repository.MovieRepositoryInterface) *MovieService {
	return &MovieService{
		repo: repo,
	}
}

// Create validates and creates a new movie.
// It generates a UUID if the movie doesn't have an ID.
func (s *MovieService) Create(ctx context.Context, movie *models.Movie) error {
	if movie == nil {
		return fmt.Errorf("movie cannot be nil")
	}

	if movie.Title == "" {
		return fmt.Errorf("movie title cannot be empty")
	}

	// Generate ID if not provided
	if movie.ID == "" {
		movie.ID = uuid.New().String()
	}

	slog.Info("Creating movie", "movie_id", movie.ID, "title", movie.Title)

	if err := s.repo.Create(ctx, movie); err != nil {
		slog.Error("Failed to create movie", "error", err, "movie_id", movie.ID)
		return fmt.Errorf("failed to create movie: %w", err)
	}

	return nil
}

// GetByID retrieves a movie by its ID.
func (s *MovieService) GetByID(ctx context.Context, id string) (*models.Movie, error) {
	if id == "" {
		return nil, fmt.Errorf("movie id cannot be empty")
	}

	movie, err := s.repo.FindByID(ctx, id)
	if err != nil {
		slog.Error("Failed to get movie", "error", err, "movie_id", id)
		return nil, err
	}

	return movie, nil
}

// GetByTMDbID retrieves a movie by its TMDb ID.
func (s *MovieService) GetByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	if tmdbID <= 0 {
		return nil, fmt.Errorf("tmdb id must be positive")
	}

	movie, err := s.repo.FindByTMDbID(ctx, tmdbID)
	if err != nil {
		slog.Error("Failed to get movie by TMDb ID", "error", err, "tmdb_id", tmdbID)
		return nil, err
	}

	return movie, nil
}

// GetByIMDbID retrieves a movie by its IMDb ID.
func (s *MovieService) GetByIMDbID(ctx context.Context, imdbID string) (*models.Movie, error) {
	if imdbID == "" {
		return nil, fmt.Errorf("imdb id cannot be empty")
	}

	movie, err := s.repo.FindByIMDbID(ctx, imdbID)
	if err != nil {
		slog.Error("Failed to get movie by IMDb ID", "error", err, "imdb_id", imdbID)
		return nil, err
	}

	return movie, nil
}

// Update validates and updates an existing movie.
func (s *MovieService) Update(ctx context.Context, movie *models.Movie) error {
	if movie == nil {
		return fmt.Errorf("movie cannot be nil")
	}

	if movie.ID == "" {
		return fmt.Errorf("movie id cannot be empty")
	}

	if movie.Title == "" {
		return fmt.Errorf("movie title cannot be empty")
	}

	slog.Info("Updating movie", "movie_id", movie.ID, "title", movie.Title)

	if err := s.repo.Update(ctx, movie); err != nil {
		slog.Error("Failed to update movie", "error", err, "movie_id", movie.ID)
		return fmt.Errorf("failed to update movie: %w", err)
	}

	return nil
}

// Delete removes a movie by its ID.
func (s *MovieService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("movie id cannot be empty")
	}

	slog.Info("Deleting movie", "movie_id", id)

	if err := s.repo.Delete(ctx, id); err != nil {
		slog.Error("Failed to delete movie", "error", err, "movie_id", id)
		return fmt.Errorf("failed to delete movie: %w", err)
	}

	return nil
}

// List retrieves movies with pagination support.
func (s *MovieService) List(ctx context.Context, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	movies, pagination, err := s.repo.List(ctx, params)
	if err != nil {
		slog.Error("Failed to list movies", "error", err)
		return nil, nil, fmt.Errorf("failed to list movies: %w", err)
	}

	return movies, pagination, nil
}

// SearchByTitle searches for movies by title with pagination.
func (s *MovieService) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	if title == "" {
		return nil, nil, fmt.Errorf("search title cannot be empty")
	}

	movies, pagination, err := s.repo.SearchByTitle(ctx, title, params)
	if err != nil {
		slog.Error("Failed to search movies", "error", err, "title", title)
		return nil, nil, fmt.Errorf("failed to search movies: %w", err)
	}

	return movies, pagination, nil
}
