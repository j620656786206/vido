package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// MovieRepository provides data access operations for movies
type MovieRepository struct {
	db *sql.DB
}

// NewMovieRepository creates a new instance of MovieRepository
func NewMovieRepository(db *sql.DB) *MovieRepository {
	return &MovieRepository{
		db: db,
	}
}

// Create inserts a new movie into the database
func (r *MovieRepository) Create(ctx context.Context, movie *models.Movie) error {
	if movie == nil {
		return fmt.Errorf("movie cannot be nil")
	}

	// Set timestamps
	now := time.Now()
	movie.CreatedAt = now
	movie.UpdatedAt = now

	// Convert genres to JSON
	genresJSON, err := movie.GenresJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal genres: %w", err)
	}

	query := `
		INSERT INTO movies (
			id, title, original_title, release_date, genres, rating,
			overview, poster_path, backdrop_path, runtime, original_language,
			status, imdb_id, tmdb_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		movie.ID,
		movie.Title,
		movie.OriginalTitle,
		movie.ReleaseDate,
		genresJSON,
		movie.Rating,
		movie.Overview,
		movie.PosterPath,
		movie.BackdropPath,
		movie.Runtime,
		movie.OriginalLanguage,
		movie.Status,
		movie.IMDbID,
		movie.TMDbID,
		movie.CreatedAt,
		movie.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create movie: %w", err)
	}

	return nil
}

// FindByID retrieves a movie by its primary key
func (r *MovieRepository) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	query := `
		SELECT
			id, title, original_title, release_date, genres, rating,
			overview, poster_path, backdrop_path, runtime, original_language,
			status, imdb_id, tmdb_id, created_at, updated_at
		FROM movies
		WHERE id = ?
	`

	movie := &models.Movie{}
	var genresJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&movie.ID,
		&movie.Title,
		&movie.OriginalTitle,
		&movie.ReleaseDate,
		&genresJSON,
		&movie.Rating,
		&movie.Overview,
		&movie.PosterPath,
		&movie.BackdropPath,
		&movie.Runtime,
		&movie.OriginalLanguage,
		&movie.Status,
		&movie.IMDbID,
		&movie.TMDbID,
		&movie.CreatedAt,
		&movie.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("movie with id %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find movie: %w", err)
	}

	// Parse genres from JSON
	if err := movie.ScanGenres(genresJSON); err != nil {
		return nil, fmt.Errorf("failed to parse genres: %w", err)
	}

	return movie, nil
}

// FindByTMDbID retrieves a movie by its TMDb ID
func (r *MovieRepository) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	query := `
		SELECT
			id, title, original_title, release_date, genres, rating,
			overview, poster_path, backdrop_path, runtime, original_language,
			status, imdb_id, tmdb_id, created_at, updated_at
		FROM movies
		WHERE tmdb_id = ?
	`

	movie := &models.Movie{}
	var genresJSON string

	err := r.db.QueryRowContext(ctx, query, tmdbID).Scan(
		&movie.ID,
		&movie.Title,
		&movie.OriginalTitle,
		&movie.ReleaseDate,
		&genresJSON,
		&movie.Rating,
		&movie.Overview,
		&movie.PosterPath,
		&movie.BackdropPath,
		&movie.Runtime,
		&movie.OriginalLanguage,
		&movie.Status,
		&movie.IMDbID,
		&movie.TMDbID,
		&movie.CreatedAt,
		&movie.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("movie with tmdb_id %d not found", tmdbID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find movie by tmdb_id: %w", err)
	}

	// Parse genres from JSON
	if err := movie.ScanGenres(genresJSON); err != nil {
		return nil, fmt.Errorf("failed to parse genres: %w", err)
	}

	return movie, nil
}

// FindByIMDbID retrieves a movie by its IMDb ID
func (r *MovieRepository) FindByIMDbID(ctx context.Context, imdbID string) (*models.Movie, error) {
	query := `
		SELECT
			id, title, original_title, release_date, genres, rating,
			overview, poster_path, backdrop_path, runtime, original_language,
			status, imdb_id, tmdb_id, created_at, updated_at
		FROM movies
		WHERE imdb_id = ?
	`

	movie := &models.Movie{}
	var genresJSON string

	err := r.db.QueryRowContext(ctx, query, imdbID).Scan(
		&movie.ID,
		&movie.Title,
		&movie.OriginalTitle,
		&movie.ReleaseDate,
		&genresJSON,
		&movie.Rating,
		&movie.Overview,
		&movie.PosterPath,
		&movie.BackdropPath,
		&movie.Runtime,
		&movie.OriginalLanguage,
		&movie.Status,
		&movie.IMDbID,
		&movie.TMDbID,
		&movie.CreatedAt,
		&movie.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("movie with imdb_id %s not found", imdbID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find movie by imdb_id: %w", err)
	}

	// Parse genres from JSON
	if err := movie.ScanGenres(genresJSON); err != nil {
		return nil, fmt.Errorf("failed to parse genres: %w", err)
	}

	return movie, nil
}

// Update modifies an existing movie in the database
func (r *MovieRepository) Update(ctx context.Context, movie *models.Movie) error {
	if movie == nil {
		return fmt.Errorf("movie cannot be nil")
	}

	// Update timestamp
	movie.UpdatedAt = time.Now()

	// Convert genres to JSON
	genresJSON, err := movie.GenresJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal genres: %w", err)
	}

	query := `
		UPDATE movies
		SET
			title = ?,
			original_title = ?,
			release_date = ?,
			genres = ?,
			rating = ?,
			overview = ?,
			poster_path = ?,
			backdrop_path = ?,
			runtime = ?,
			original_language = ?,
			status = ?,
			imdb_id = ?,
			tmdb_id = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		movie.Title,
		movie.OriginalTitle,
		movie.ReleaseDate,
		genresJSON,
		movie.Rating,
		movie.Overview,
		movie.PosterPath,
		movie.BackdropPath,
		movie.Runtime,
		movie.OriginalLanguage,
		movie.Status,
		movie.IMDbID,
		movie.TMDbID,
		movie.UpdatedAt,
		movie.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update movie: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("movie with id %s not found", movie.ID)
	}

	return nil
}

// Delete removes a movie from the database by ID
func (r *MovieRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM movies WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete movie: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("movie with id %s not found", id)
	}

	return nil
}

// List retrieves movies with pagination support
func (r *MovieRepository) List(ctx context.Context, params ListParams) ([]models.Movie, *PaginationResult, error) {
	// Validate parameters
	params.Validate()

	// Default sort column
	sortBy := "created_at"
	if params.SortBy != "" {
		// Validate sort column to prevent SQL injection
		validSortColumns := map[string]bool{
			"id":           true,
			"title":        true,
			"release_date": true,
			"rating":       true,
			"created_at":   true,
			"updated_at":   true,
		}
		if validSortColumns[params.SortBy] {
			sortBy = params.SortBy
		}
	}

	// Build WHERE clause for search filter
	whereClause := ""
	args := []interface{}{}

	if searchTerm, ok := params.Filters["search"].(string); ok && searchTerm != "" {
		whereClause = "WHERE title LIKE ?"
		args = append(args, "%"+searchTerm+"%")
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM movies " + whereClause
	var totalResults int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalResults)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count movies: %w", err)
	}

	// Build and execute list query
	query := fmt.Sprintf(`
		SELECT
			id, title, original_title, release_date, genres, rating,
			overview, poster_path, backdrop_path, runtime, original_language,
			status, imdb_id, tmdb_id, created_at, updated_at
		FROM movies
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortBy, params.SortOrder)

	// Add limit and offset to args
	args = append(args, params.Limit(), params.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list movies: %w", err)
	}
	defer rows.Close()

	movies := []models.Movie{}
	for rows.Next() {
		movie := models.Movie{}
		var genresJSON string

		err := rows.Scan(
			&movie.ID,
			&movie.Title,
			&movie.OriginalTitle,
			&movie.ReleaseDate,
			&genresJSON,
			&movie.Rating,
			&movie.Overview,
			&movie.PosterPath,
			&movie.BackdropPath,
			&movie.Runtime,
			&movie.OriginalLanguage,
			&movie.Status,
			&movie.IMDbID,
			&movie.TMDbID,
			&movie.CreatedAt,
			&movie.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan movie: %w", err)
		}

		// Parse genres from JSON
		if err := movie.ScanGenres(genresJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to parse genres: %w", err)
		}

		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating movies: %w", err)
	}

	// Create pagination result
	pagination := NewPaginationResult(params, totalResults)

	return movies, pagination, nil
}

// SearchByTitle searches for movies by title with pagination
func (r *MovieRepository) SearchByTitle(ctx context.Context, title string, params ListParams) ([]models.Movie, *PaginationResult, error) {
	// Set the search filter
	if params.Filters == nil {
		params.Filters = make(map[string]interface{})
	}
	params.Filters["search"] = title

	return r.List(ctx, params)
}
