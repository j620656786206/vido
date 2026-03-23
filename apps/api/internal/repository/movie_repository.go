package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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
			status, imdb_id, tmdb_id,
			file_path, parse_status, metadata_source, vote_average,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		movie.FilePath,
		movie.ParseStatus,
		movie.MetadataSource,
		movie.VoteAverage,
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

// FindByFilePath retrieves a movie by its file path (for duplicate detection)
func (r *MovieRepository) FindByFilePath(ctx context.Context, filePath string) (*models.Movie, error) {
	query := `
		SELECT
			id, title, original_title, release_date, genres, rating,
			overview, poster_path, backdrop_path, runtime, original_language,
			status, imdb_id, tmdb_id, created_at, updated_at
		FROM movies
		WHERE file_path = ?
	`

	movie := &models.Movie{}
	var genresJSON string

	err := r.db.QueryRowContext(ctx, query, filePath).Scan(
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
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find movie by file_path: %w", err)
	}

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
		// Map frontend sort fields to actual movie table columns
		sortColumnMap := map[string]string{
			"id":           "id",
			"title":        "title",
			"release_date": "release_date",
			"rating":       "rating",
			"vote_average": "vote_average",
			"created_at":   "created_at",
			"updated_at":   "updated_at",
		}
		if col, ok := sortColumnMap[params.SortBy]; ok {
			sortBy = col
		}
	}

	// Build WHERE clause from filters
	conditions := []string{}
	args := []interface{}{}

	if searchTerm, ok := params.Filters["search"].(string); ok && searchTerm != "" {
		conditions = append(conditions, "title LIKE ?")
		args = append(args, "%"+searchTerm+"%")
	}

	if genres, ok := params.Filters["genres"].([]string); ok {
		for _, g := range genres {
			conditions = append(conditions, `genres LIKE ?`)
			args = append(args, `%"`+g+`"%`)
		}
	}

	if yearMin, ok := params.Filters["year_min"].(string); ok && yearMin != "" {
		conditions = append(conditions, "substr(release_date, 1, 4) >= ?")
		args = append(args, yearMin)
	}

	if yearMax, ok := params.Filters["year_max"].(string); ok && yearMax != "" {
		conditions = append(conditions, "substr(release_date, 1, 4) <= ?")
		args = append(args, yearMax)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
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

// FullTextSearch performs FTS5 search across title, original_title, and overview
func (r *MovieRepository) FullTextSearch(ctx context.Context, query string, params ListParams) ([]models.Movie, *PaginationResult, error) {
	params.Validate()

	if query == "" {
		return r.List(ctx, params)
	}

	// Get total count for FTS results
	countQuery := `
		SELECT COUNT(*)
		FROM movies m
		JOIN movies_fts ON movies_fts.rowid = m.rowid
		WHERE movies_fts MATCH ?
	`
	var totalResults int
	err := r.db.QueryRowContext(ctx, countQuery, query).Scan(&totalResults)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count FTS results: %w", err)
	}

	// FTS5 search query - join with movies table to get full data
	ftsQuery := `
		SELECT
			m.id, m.title, m.original_title, m.release_date, m.genres, m.rating,
			m.overview, m.poster_path, m.backdrop_path, m.runtime, m.original_language,
			m.status, m.imdb_id, m.tmdb_id, m.created_at, m.updated_at
		FROM movies m
		JOIN movies_fts ON movies_fts.rowid = m.rowid
		WHERE movies_fts MATCH ?
		ORDER BY rank
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, ftsQuery, query, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute FTS search: %w", err)
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
			return nil, nil, fmt.Errorf("failed to scan FTS movie: %w", err)
		}

		if err := movie.ScanGenres(genresJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to parse genres: %w", err)
		}

		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating FTS movies: %w", err)
	}

	pagination := NewPaginationResult(params, totalResults)
	return movies, pagination, nil
}

// GetDistinctGenres returns all unique genres from movies
func (r *MovieRepository) GetDistinctGenres(ctx context.Context) ([]string, error) {
	query := `SELECT genres FROM movies WHERE genres != '[]' AND genres != ''`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query movie genres: %w", err)
	}
	defer rows.Close()

	genreSet := make(map[string]struct{})
	for rows.Next() {
		var genresJSON string
		if err := rows.Scan(&genresJSON); err != nil {
			return nil, fmt.Errorf("failed to scan genres: %w", err)
		}

		var genres []string
		if err := json.Unmarshal([]byte(genresJSON), &genres); err != nil {
			continue // skip malformed JSON
		}

		for _, g := range genres {
			genreSet[g] = struct{}{}
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movie genres: %w", err)
	}

	result := make([]string, 0, len(genreSet))
	for g := range genreSet {
		result = append(result, g)
	}

	return result, nil
}

// GetYearRange returns the min and max release years from movies
func (r *MovieRepository) GetYearRange(ctx context.Context) (minYear, maxYear int, err error) {
	query := `SELECT
		COALESCE(MIN(CAST(substr(release_date, 1, 4) AS INTEGER)), 0),
		COALESCE(MAX(CAST(substr(release_date, 1, 4) AS INTEGER)), 0)
		FROM movies WHERE release_date != '' AND release_date IS NOT NULL`

	err = r.db.QueryRowContext(ctx, query).Scan(&minYear, &maxYear)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get movie year range: %w", err)
	}

	return minYear, maxYear, nil
}

// Count returns the total number of movies
func (r *MovieRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM movies").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count movies: %w", err)
	}
	return count, nil
}

// movieSelectColumns defines the column list used by multi-row scan queries.
// This must match the order in scanMovie.
const movieSelectColumns = `
	id, title, original_title, release_date, genres, rating,
	overview, poster_path, backdrop_path, runtime, original_language,
	status, imdb_id, tmdb_id,
	file_path, parse_status, metadata_source,
	subtitle_status, subtitle_path, subtitle_language, subtitle_last_searched, subtitle_search_score,
	vote_average, is_removed,
	created_at, updated_at
`

// scanMovie scans a row into a Movie struct using the standard column order.
func scanMovie(scanner interface{ Scan(dest ...interface{}) error }) (models.Movie, error) {
	var movie models.Movie
	var genresJSON string

	err := scanner.Scan(
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
		&movie.FilePath,
		&movie.ParseStatus,
		&movie.MetadataSource,
		&movie.SubtitleStatus,
		&movie.SubtitlePath,
		&movie.SubtitleLanguage,
		&movie.SubtitleLastSearched,
		&movie.SubtitleSearchScore,
		&movie.VoteAverage,
		&movie.IsRemoved,
		&movie.CreatedAt,
		&movie.UpdatedAt,
	)
	if err != nil {
		return movie, err
	}

	if err := movie.ScanGenres(genresJSON); err != nil {
		return movie, fmt.Errorf("failed to parse genres: %w", err)
	}

	return movie, nil
}

// BulkCreate inserts multiple movies in a single transaction
func (r *MovieRepository) BulkCreate(ctx context.Context, movies []*models.Movie) error {
	if len(movies) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO movies (
			id, title, original_title, release_date, genres, rating,
			overview, poster_path, backdrop_path, runtime, original_language,
			status, imdb_id, tmdb_id,
			file_path, parse_status, metadata_source, vote_average,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, movie := range movies {
		if movie == nil {
			continue
		}

		movie.CreatedAt = now
		movie.UpdatedAt = now

		genresJSON, err := movie.GenresJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal genres: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
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
			movie.FilePath,
			movie.ParseStatus,
			movie.MetadataSource,
			movie.VoteAverage,
			movie.CreatedAt,
			movie.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert movie %s: %w", movie.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByParseStatus retrieves movies matching a given parse status
func (r *MovieRepository) FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Movie, error) {
	query := fmt.Sprintf(`SELECT %s FROM movies WHERE parse_status = ? ORDER BY updated_at DESC`, movieSelectColumns)

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query movies by parse status: %w", err)
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan movie: %w", err)
		}
		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movies: %w", err)
	}

	return movies, nil
}

// UpdateSubtitleStatus updates subtitle-related fields for a movie
func (r *MovieRepository) UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	now := time.Now()

	query := `
		UPDATE movies
		SET subtitle_status = ?, subtitle_path = ?, subtitle_language = ?,
			subtitle_search_score = ?, subtitle_last_searched = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		status,
		sql.NullString{String: path, Valid: path != ""},
		sql.NullString{String: language, Valid: language != ""},
		sql.NullFloat64{Float64: score, Valid: score > 0},
		sql.NullTime{Time: now, Valid: true},
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update movie subtitle status: %w", err)
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

// FindBySubtitleStatus retrieves movies matching a given subtitle status
func (r *MovieRepository) FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Movie, error) {
	query := fmt.Sprintf(`SELECT %s FROM movies WHERE subtitle_status = ? ORDER BY updated_at DESC`, movieSelectColumns)

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query movies by subtitle status: %w", err)
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan movie: %w", err)
		}
		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movies: %w", err)
	}

	return movies, nil
}

// FindNeedingSubtitleSearch retrieves movies not yet searched or last searched before threshold
func (r *MovieRepository) FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Movie, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM movies
		WHERE subtitle_status = 'not_searched'
			OR (subtitle_last_searched IS NOT NULL AND subtitle_last_searched < ?)
		ORDER BY updated_at DESC
	`, movieSelectColumns)

	rows, err := r.db.QueryContext(ctx, query, olderThan)
	if err != nil {
		return nil, fmt.Errorf("failed to query movies needing subtitle search: %w", err)
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan movie: %w", err)
		}
		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movies: %w", err)
	}

	return movies, nil
}

// FindAllWithFilePath retrieves all movies that have a non-null file_path and are not removed
func (r *MovieRepository) FindAllWithFilePath(ctx context.Context) ([]models.Movie, error) {
	query := fmt.Sprintf(`SELECT %s FROM movies WHERE file_path IS NOT NULL AND file_path != '' AND is_removed = 0`, movieSelectColumns)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query movies with file_path: %w", err)
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan movie: %w", err)
		}
		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movies: %w", err)
	}

	return movies, nil
}

// Upsert creates or updates a movie based on TMDb ID
func (r *MovieRepository) Upsert(ctx context.Context, movie *models.Movie) error {
	if movie == nil {
		return fmt.Errorf("movie cannot be nil")
	}

	// If no TMDb ID, just create
	if !movie.TMDbID.Valid {
		return r.Create(ctx, movie)
	}

	// Check if movie with this TMDb ID already exists
	existing, err := r.FindByTMDbID(ctx, movie.TMDbID.Int64)
	if err != nil {
		// If not found, create new movie
		if err.Error() == fmt.Sprintf("movie with tmdb_id %d not found", movie.TMDbID.Int64) {
			return r.Create(ctx, movie)
		}
		return fmt.Errorf("failed to check existing movie: %w", err)
	}

	// Movie exists - update with existing ID
	movie.ID = existing.ID
	movie.CreatedAt = existing.CreatedAt
	return r.Update(ctx, movie)
}
