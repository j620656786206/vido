package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// SeriesRepository provides data access operations for TV series
type SeriesRepository struct {
	db *sql.DB
}

// NewSeriesRepository creates a new instance of SeriesRepository
func NewSeriesRepository(db *sql.DB) *SeriesRepository {
	return &SeriesRepository{
		db: db,
	}
}

// Create inserts a new series into the database
func (r *SeriesRepository) Create(ctx context.Context, series *models.Series) error {
	if series == nil {
		return fmt.Errorf("series cannot be nil")
	}

	// Set timestamps
	now := time.Now()
	series.CreatedAt = now
	series.UpdatedAt = now

	// Convert genres to JSON
	genresJSON, err := series.GenresJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal genres: %w", err)
	}

	query := `
		INSERT INTO series (
			id, title, original_title, first_air_date, last_air_date, genres, rating,
			overview, poster_path, backdrop_path, number_of_seasons, number_of_episodes,
			status, original_language, imdb_id, tmdb_id, in_production, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		series.ID,
		series.Title,
		series.OriginalTitle,
		series.FirstAirDate,
		series.LastAirDate,
		genresJSON,
		series.Rating,
		series.Overview,
		series.PosterPath,
		series.BackdropPath,
		series.NumberOfSeasons,
		series.NumberOfEpisodes,
		series.Status,
		series.OriginalLanguage,
		series.IMDbID,
		series.TMDbID,
		series.InProduction,
		series.CreatedAt,
		series.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create series: %w", err)
	}

	return nil
}

// FindByID retrieves a series by its primary key
func (r *SeriesRepository) FindByID(ctx context.Context, id string) (*models.Series, error) {
	query := `
		SELECT
			id, title, original_title, first_air_date, last_air_date, genres, rating,
			overview, poster_path, backdrop_path, number_of_seasons, number_of_episodes,
			status, original_language, imdb_id, tmdb_id, in_production, created_at, updated_at
		FROM series
		WHERE id = ?
	`

	series := &models.Series{}
	var genresJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&series.ID,
		&series.Title,
		&series.OriginalTitle,
		&series.FirstAirDate,
		&series.LastAirDate,
		&genresJSON,
		&series.Rating,
		&series.Overview,
		&series.PosterPath,
		&series.BackdropPath,
		&series.NumberOfSeasons,
		&series.NumberOfEpisodes,
		&series.Status,
		&series.OriginalLanguage,
		&series.IMDbID,
		&series.TMDbID,
		&series.InProduction,
		&series.CreatedAt,
		&series.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("series with id %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find series: %w", err)
	}

	// Parse genres from JSON
	if err := series.ScanGenres(genresJSON); err != nil {
		return nil, fmt.Errorf("failed to parse genres: %w", err)
	}

	return series, nil
}

// FindByTMDbID retrieves a series by its TMDb ID
func (r *SeriesRepository) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	query := `
		SELECT
			id, title, original_title, first_air_date, last_air_date, genres, rating,
			overview, poster_path, backdrop_path, number_of_seasons, number_of_episodes,
			status, original_language, imdb_id, tmdb_id, in_production, created_at, updated_at
		FROM series
		WHERE tmdb_id = ?
	`

	series := &models.Series{}
	var genresJSON string

	err := r.db.QueryRowContext(ctx, query, tmdbID).Scan(
		&series.ID,
		&series.Title,
		&series.OriginalTitle,
		&series.FirstAirDate,
		&series.LastAirDate,
		&genresJSON,
		&series.Rating,
		&series.Overview,
		&series.PosterPath,
		&series.BackdropPath,
		&series.NumberOfSeasons,
		&series.NumberOfEpisodes,
		&series.Status,
		&series.OriginalLanguage,
		&series.IMDbID,
		&series.TMDbID,
		&series.InProduction,
		&series.CreatedAt,
		&series.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("series with tmdb_id %d not found", tmdbID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find series by tmdb_id: %w", err)
	}

	// Parse genres from JSON
	if err := series.ScanGenres(genresJSON); err != nil {
		return nil, fmt.Errorf("failed to parse genres: %w", err)
	}

	return series, nil
}

// FindByIMDbID retrieves a series by its IMDb ID
func (r *SeriesRepository) FindByIMDbID(ctx context.Context, imdbID string) (*models.Series, error) {
	query := `
		SELECT
			id, title, original_title, first_air_date, last_air_date, genres, rating,
			overview, poster_path, backdrop_path, number_of_seasons, number_of_episodes,
			status, original_language, imdb_id, tmdb_id, in_production, created_at, updated_at
		FROM series
		WHERE imdb_id = ?
	`

	series := &models.Series{}
	var genresJSON string

	err := r.db.QueryRowContext(ctx, query, imdbID).Scan(
		&series.ID,
		&series.Title,
		&series.OriginalTitle,
		&series.FirstAirDate,
		&series.LastAirDate,
		&genresJSON,
		&series.Rating,
		&series.Overview,
		&series.PosterPath,
		&series.BackdropPath,
		&series.NumberOfSeasons,
		&series.NumberOfEpisodes,
		&series.Status,
		&series.OriginalLanguage,
		&series.IMDbID,
		&series.TMDbID,
		&series.InProduction,
		&series.CreatedAt,
		&series.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("series with imdb_id %s not found", imdbID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find series by imdb_id: %w", err)
	}

	// Parse genres from JSON
	if err := series.ScanGenres(genresJSON); err != nil {
		return nil, fmt.Errorf("failed to parse genres: %w", err)
	}

	return series, nil
}

// Update modifies an existing series in the database
func (r *SeriesRepository) Update(ctx context.Context, series *models.Series) error {
	if series == nil {
		return fmt.Errorf("series cannot be nil")
	}

	// Update timestamp
	series.UpdatedAt = time.Now()

	// Convert genres to JSON
	genresJSON, err := series.GenresJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal genres: %w", err)
	}

	query := `
		UPDATE series
		SET
			title = ?,
			original_title = ?,
			first_air_date = ?,
			last_air_date = ?,
			genres = ?,
			rating = ?,
			overview = ?,
			poster_path = ?,
			backdrop_path = ?,
			number_of_seasons = ?,
			number_of_episodes = ?,
			status = ?,
			original_language = ?,
			imdb_id = ?,
			tmdb_id = ?,
			in_production = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		series.Title,
		series.OriginalTitle,
		series.FirstAirDate,
		series.LastAirDate,
		genresJSON,
		series.Rating,
		series.Overview,
		series.PosterPath,
		series.BackdropPath,
		series.NumberOfSeasons,
		series.NumberOfEpisodes,
		series.Status,
		series.OriginalLanguage,
		series.IMDbID,
		series.TMDbID,
		series.InProduction,
		series.UpdatedAt,
		series.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update series: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("series with id %s not found", series.ID)
	}

	return nil
}

// Delete removes a series from the database by ID
func (r *SeriesRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM series WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete series: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("series with id %s not found", id)
	}

	return nil
}

// List retrieves series with pagination support
func (r *SeriesRepository) List(ctx context.Context, params ListParams) ([]models.Series, *PaginationResult, error) {
	// Validate parameters
	params.Validate()

	// Default sort column
	sortBy := "created_at"
	if params.SortBy != "" {
		// Validate sort column to prevent SQL injection
		validSortColumns := map[string]bool{
			"id":             true,
			"title":          true,
			"first_air_date": true,
			"rating":         true,
			"created_at":     true,
			"updated_at":     true,
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
	countQuery := "SELECT COUNT(*) FROM series " + whereClause
	var totalResults int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalResults)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count series: %w", err)
	}

	// Build and execute list query
	query := fmt.Sprintf(`
		SELECT
			id, title, original_title, first_air_date, last_air_date, genres, rating,
			overview, poster_path, backdrop_path, number_of_seasons, number_of_episodes,
			status, original_language, imdb_id, tmdb_id, in_production, created_at, updated_at
		FROM series
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortBy, params.SortOrder)

	// Add limit and offset to args
	args = append(args, params.Limit(), params.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list series: %w", err)
	}
	defer rows.Close()

	series := []models.Series{}
	for rows.Next() {
		s := models.Series{}
		var genresJSON string

		err := rows.Scan(
			&s.ID,
			&s.Title,
			&s.OriginalTitle,
			&s.FirstAirDate,
			&s.LastAirDate,
			&genresJSON,
			&s.Rating,
			&s.Overview,
			&s.PosterPath,
			&s.BackdropPath,
			&s.NumberOfSeasons,
			&s.NumberOfEpisodes,
			&s.Status,
			&s.OriginalLanguage,
			&s.IMDbID,
			&s.TMDbID,
			&s.InProduction,
			&s.CreatedAt,
			&s.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan series: %w", err)
		}

		// Parse genres from JSON
		if err := s.ScanGenres(genresJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to parse genres: %w", err)
		}

		series = append(series, s)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating series: %w", err)
	}

	// Create pagination result
	pagination := NewPaginationResult(params, totalResults)

	return series, pagination, nil
}

// SearchByTitle searches for series by title with pagination
func (r *SeriesRepository) SearchByTitle(ctx context.Context, title string, params ListParams) ([]models.Series, *PaginationResult, error) {
	// Set the search filter
	if params.Filters == nil {
		params.Filters = make(map[string]interface{})
	}
	params.Filters["search"] = title

	return r.List(ctx, params)
}

// FullTextSearch performs FTS5 search across title, original_title, and overview
func (r *SeriesRepository) FullTextSearch(ctx context.Context, query string, params ListParams) ([]models.Series, *PaginationResult, error) {
	params.Validate()

	if query == "" {
		return r.List(ctx, params)
	}

	// Get total count for FTS results
	countQuery := `
		SELECT COUNT(*)
		FROM series s
		JOIN series_fts ON series_fts.rowid = s.rowid
		WHERE series_fts MATCH ?
	`
	var totalResults int
	err := r.db.QueryRowContext(ctx, countQuery, query).Scan(&totalResults)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count FTS results: %w", err)
	}

	// FTS5 search query - join with series table to get full data
	ftsQuery := `
		SELECT
			s.id, s.title, s.original_title, s.first_air_date, s.last_air_date, s.genres, s.rating,
			s.overview, s.poster_path, s.backdrop_path, s.number_of_seasons, s.number_of_episodes,
			s.status, s.original_language, s.imdb_id, s.tmdb_id, s.in_production, s.created_at, s.updated_at
		FROM series s
		JOIN series_fts ON series_fts.rowid = s.rowid
		WHERE series_fts MATCH ?
		ORDER BY rank
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, ftsQuery, query, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute FTS search: %w", err)
	}
	defer rows.Close()

	seriesList := []models.Series{}
	for rows.Next() {
		s := models.Series{}
		var genresJSON string

		err := rows.Scan(
			&s.ID,
			&s.Title,
			&s.OriginalTitle,
			&s.FirstAirDate,
			&s.LastAirDate,
			&genresJSON,
			&s.Rating,
			&s.Overview,
			&s.PosterPath,
			&s.BackdropPath,
			&s.NumberOfSeasons,
			&s.NumberOfEpisodes,
			&s.Status,
			&s.OriginalLanguage,
			&s.IMDbID,
			&s.TMDbID,
			&s.InProduction,
			&s.CreatedAt,
			&s.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan FTS series: %w", err)
		}

		if err := s.ScanGenres(genresJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to parse genres: %w", err)
		}

		seriesList = append(seriesList, s)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating FTS series: %w", err)
	}

	pagination := NewPaginationResult(params, totalResults)
	return seriesList, pagination, nil
}

// Upsert creates or updates a series based on TMDb ID
func (r *SeriesRepository) Upsert(ctx context.Context, series *models.Series) error {
	if series == nil {
		return fmt.Errorf("series cannot be nil")
	}

	// If no TMDb ID, just create
	if !series.TMDbID.Valid {
		return r.Create(ctx, series)
	}

	// Check if series with this TMDb ID already exists
	existing, err := r.FindByTMDbID(ctx, series.TMDbID.Int64)
	if err != nil {
		// If not found, create new series
		if err.Error() == fmt.Sprintf("series with tmdb_id %d not found", series.TMDbID.Int64) {
			return r.Create(ctx, series)
		}
		return fmt.Errorf("failed to check existing series: %w", err)
	}

	// Series exists - update with existing ID
	series.ID = existing.ID
	series.CreatedAt = existing.CreatedAt
	return r.Update(ctx, series)
}
