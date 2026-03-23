package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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
			file_path = ?,
			parse_status = ?,
			metadata_source = ?,
			vote_average = ?,
			is_removed = ?,
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
		series.FilePath,
		series.ParseStatus,
		series.MetadataSource,
		series.VoteAverage,
		series.IsRemoved,
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
		// Map frontend sort fields to actual series table columns
		sortColumnMap := map[string]string{
			"id":             "id",
			"title":          "title",
			"first_air_date": "first_air_date",
			"release_date":   "first_air_date", // alias: frontend uses release_date for both movie/series
			"rating":         "rating",
			"vote_average":   "vote_average",
			"created_at":     "created_at",
			"updated_at":     "updated_at",
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
		conditions = append(conditions, "substr(first_air_date, 1, 4) >= ?")
		args = append(args, yearMin)
	}

	if yearMax, ok := params.Filters["year_max"].(string); ok && yearMax != "" {
		conditions = append(conditions, "substr(first_air_date, 1, 4) <= ?")
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

// GetDistinctGenres returns all unique genres from series
func (r *SeriesRepository) GetDistinctGenres(ctx context.Context) ([]string, error) {
	query := `SELECT genres FROM series WHERE genres != '[]' AND genres != ''`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query series genres: %w", err)
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
		return nil, fmt.Errorf("error iterating series genres: %w", err)
	}

	result := make([]string, 0, len(genreSet))
	for g := range genreSet {
		result = append(result, g)
	}

	return result, nil
}

// GetYearRange returns the min and max first_air_date years from series
func (r *SeriesRepository) GetYearRange(ctx context.Context) (minYear, maxYear int, err error) {
	query := `SELECT
		COALESCE(MIN(CAST(substr(first_air_date, 1, 4) AS INTEGER)), 0),
		COALESCE(MAX(CAST(substr(first_air_date, 1, 4) AS INTEGER)), 0)
		FROM series WHERE first_air_date != '' AND first_air_date IS NOT NULL`

	err = r.db.QueryRowContext(ctx, query).Scan(&minYear, &maxYear)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get series year range: %w", err)
	}

	return minYear, maxYear, nil
}

// Count returns the total number of series
func (r *SeriesRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM series").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count series: %w", err)
	}
	return count, nil
}

// seriesSelectColumns defines the column list used by multi-row scan queries.
// This must match the order in scanSeries.
const seriesSelectColumns = `
	id, title, original_title, first_air_date, last_air_date, genres, rating,
	overview, poster_path, backdrop_path, number_of_seasons, number_of_episodes,
	status, original_language, imdb_id, tmdb_id, in_production,
	file_path, parse_status, metadata_source,
	subtitle_status, subtitle_path, subtitle_language, subtitle_last_searched, subtitle_search_score,
	vote_average,
	created_at, updated_at
`

// scanSeries scans a row into a Series struct using the standard column order.
func scanSeries(scanner interface{ Scan(dest ...interface{}) error }) (models.Series, error) {
	var s models.Series
	var genresJSON string

	err := scanner.Scan(
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
		&s.FilePath,
		&s.ParseStatus,
		&s.MetadataSource,
		&s.SubtitleStatus,
		&s.SubtitlePath,
		&s.SubtitleLanguage,
		&s.SubtitleLastSearched,
		&s.SubtitleSearchScore,
		&s.VoteAverage,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return s, err
	}

	if err := s.ScanGenres(genresJSON); err != nil {
		return s, fmt.Errorf("failed to parse genres: %w", err)
	}

	return s, nil
}

// BulkCreate inserts multiple series in a single transaction
func (r *SeriesRepository) BulkCreate(ctx context.Context, seriesList []*models.Series) error {
	if len(seriesList) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO series (
			id, title, original_title, first_air_date, last_air_date, genres, rating,
			overview, poster_path, backdrop_path, number_of_seasons, number_of_episodes,
			status, original_language, imdb_id, tmdb_id, in_production, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, series := range seriesList {
		if series == nil {
			continue
		}

		series.CreatedAt = now
		series.UpdatedAt = now

		genresJSON, err := series.GenresJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal genres: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
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
			return fmt.Errorf("failed to insert series %s: %w", series.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByParseStatus retrieves series matching a given parse status
func (r *SeriesRepository) FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Series, error) {
	query := fmt.Sprintf(`SELECT %s FROM series WHERE parse_status = ? ORDER BY updated_at DESC`, seriesSelectColumns)

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query series by parse status: %w", err)
	}
	defer rows.Close()

	var seriesList []models.Series
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan series: %w", err)
		}
		seriesList = append(seriesList, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating series: %w", err)
	}

	return seriesList, nil
}

// UpdateSubtitleStatus updates subtitle-related fields for a series
func (r *SeriesRepository) UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	now := time.Now()

	query := `
		UPDATE series
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
		return fmt.Errorf("failed to update series subtitle status: %w", err)
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

// FindBySubtitleStatus retrieves series matching a given subtitle status
func (r *SeriesRepository) FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Series, error) {
	query := fmt.Sprintf(`SELECT %s FROM series WHERE subtitle_status = ? ORDER BY updated_at DESC`, seriesSelectColumns)

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query series by subtitle status: %w", err)
	}
	defer rows.Close()

	var seriesList []models.Series
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan series: %w", err)
		}
		seriesList = append(seriesList, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating series: %w", err)
	}

	return seriesList, nil
}

// FindNeedingSubtitleSearch retrieves series not yet searched or last searched before threshold
func (r *SeriesRepository) FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Series, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM series
		WHERE subtitle_status = 'not_searched'
			OR (subtitle_last_searched IS NOT NULL AND subtitle_last_searched < ?)
		ORDER BY updated_at DESC
	`, seriesSelectColumns)

	rows, err := r.db.QueryContext(ctx, query, olderThan)
	if err != nil {
		return nil, fmt.Errorf("failed to query series needing subtitle search: %w", err)
	}
	defer rows.Close()

	var seriesList []models.Series
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan series: %w", err)
		}
		seriesList = append(seriesList, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating series: %w", err)
	}

	return seriesList, nil
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
