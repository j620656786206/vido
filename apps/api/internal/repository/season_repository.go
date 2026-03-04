package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// SeasonRepository provides data access operations for seasons
type SeasonRepository struct {
	db *sql.DB
}

// NewSeasonRepository creates a new instance of SeasonRepository
func NewSeasonRepository(db *sql.DB) *SeasonRepository {
	return &SeasonRepository{
		db: db,
	}
}

// Create inserts a new season into the database
func (r *SeasonRepository) Create(ctx context.Context, season *models.Season) error {
	if season == nil {
		return fmt.Errorf("season cannot be nil")
	}

	now := time.Now()
	season.CreatedAt = now
	season.UpdatedAt = now

	query := `
		INSERT INTO seasons (
			id, series_id, tmdb_id, season_number,
			name, overview, poster_path, air_date,
			episode_count, vote_average, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		season.ID,
		season.SeriesID,
		season.TMDbID,
		season.SeasonNumber,
		season.Name,
		season.Overview,
		season.PosterPath,
		season.AirDate,
		season.EpisodeCount,
		season.VoteAverage,
		season.CreatedAt,
		season.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create season: %w", err)
	}

	return nil
}

// FindByID retrieves a season by its primary key
func (r *SeasonRepository) FindByID(ctx context.Context, id string) (*models.Season, error) {
	query := `
		SELECT
			id, series_id, tmdb_id, season_number,
			name, overview, poster_path, air_date,
			episode_count, vote_average, created_at, updated_at
		FROM seasons
		WHERE id = ?
	`

	season := &models.Season{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&season.ID,
		&season.SeriesID,
		&season.TMDbID,
		&season.SeasonNumber,
		&season.Name,
		&season.Overview,
		&season.PosterPath,
		&season.AirDate,
		&season.EpisodeCount,
		&season.VoteAverage,
		&season.CreatedAt,
		&season.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("season with id %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find season: %w", err)
	}

	return season, nil
}

// FindBySeriesID retrieves all seasons for a series
func (r *SeasonRepository) FindBySeriesID(ctx context.Context, seriesID string) ([]models.Season, error) {
	query := `
		SELECT
			id, series_id, tmdb_id, season_number,
			name, overview, poster_path, air_date,
			episode_count, vote_average, created_at, updated_at
		FROM seasons
		WHERE series_id = ?
		ORDER BY season_number
	`

	rows, err := r.db.QueryContext(ctx, query, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to find seasons by series_id: %w", err)
	}
	defer rows.Close()

	seasons := []models.Season{}
	for rows.Next() {
		season := models.Season{}
		err := rows.Scan(
			&season.ID,
			&season.SeriesID,
			&season.TMDbID,
			&season.SeasonNumber,
			&season.Name,
			&season.Overview,
			&season.PosterPath,
			&season.AirDate,
			&season.EpisodeCount,
			&season.VoteAverage,
			&season.CreatedAt,
			&season.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan season: %w", err)
		}
		seasons = append(seasons, season)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating seasons: %w", err)
	}

	return seasons, nil
}

// FindBySeriesAndNumber retrieves a season by series ID and season number
func (r *SeasonRepository) FindBySeriesAndNumber(ctx context.Context, seriesID string, seasonNumber int) (*models.Season, error) {
	query := `
		SELECT
			id, series_id, tmdb_id, season_number,
			name, overview, poster_path, air_date,
			episode_count, vote_average, created_at, updated_at
		FROM seasons
		WHERE series_id = ? AND season_number = ?
	`

	season := &models.Season{}
	err := r.db.QueryRowContext(ctx, query, seriesID, seasonNumber).Scan(
		&season.ID,
		&season.SeriesID,
		&season.TMDbID,
		&season.SeasonNumber,
		&season.Name,
		&season.Overview,
		&season.PosterPath,
		&season.AirDate,
		&season.EpisodeCount,
		&season.VoteAverage,
		&season.CreatedAt,
		&season.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("season %d for series %s not found", seasonNumber, seriesID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find season: %w", err)
	}

	return season, nil
}

// Update modifies an existing season in the database
func (r *SeasonRepository) Update(ctx context.Context, season *models.Season) error {
	if season == nil {
		return fmt.Errorf("season cannot be nil")
	}

	season.UpdatedAt = time.Now()

	query := `
		UPDATE seasons
		SET
			series_id = ?,
			tmdb_id = ?,
			season_number = ?,
			name = ?,
			overview = ?,
			poster_path = ?,
			air_date = ?,
			episode_count = ?,
			vote_average = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		season.SeriesID,
		season.TMDbID,
		season.SeasonNumber,
		season.Name,
		season.Overview,
		season.PosterPath,
		season.AirDate,
		season.EpisodeCount,
		season.VoteAverage,
		season.UpdatedAt,
		season.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update season: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("season with id %s not found", season.ID)
	}

	return nil
}

// Delete removes a season from the database by ID
func (r *SeasonRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM seasons WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete season: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("season with id %s not found", id)
	}

	return nil
}

// Upsert creates or updates a season based on UNIQUE(series_id, season_number)
func (r *SeasonRepository) Upsert(ctx context.Context, season *models.Season) error {
	if season == nil {
		return fmt.Errorf("season cannot be nil")
	}

	existing, err := r.FindBySeriesAndNumber(ctx, season.SeriesID, season.SeasonNumber)
	if err != nil {
		// Not found — create new
		errMsg := fmt.Sprintf("season %d for series %s not found", season.SeasonNumber, season.SeriesID)
		if err.Error() == errMsg {
			return r.Create(ctx, season)
		}
		return fmt.Errorf("failed to check existing season: %w", err)
	}

	// Season exists — update with existing ID
	season.ID = existing.ID
	season.CreatedAt = existing.CreatedAt
	return r.Update(ctx, season)
}
