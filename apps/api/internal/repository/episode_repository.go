package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// EpisodeRepository provides data access operations for episodes
type EpisodeRepository struct {
	db *sql.DB
}

// NewEpisodeRepository creates a new instance of EpisodeRepository
func NewEpisodeRepository(db *sql.DB) *EpisodeRepository {
	return &EpisodeRepository{
		db: db,
	}
}

// Create inserts a new episode into the database
func (r *EpisodeRepository) Create(ctx context.Context, episode *models.Episode) error {
	if episode == nil {
		return fmt.Errorf("episode cannot be nil")
	}

	// Set timestamps
	now := time.Now()
	episode.CreatedAt = now
	episode.UpdatedAt = now

	query := `
		INSERT INTO episodes (
			id, series_id, tmdb_id, season_number, episode_number,
			title, overview, air_date, runtime, still_path,
			vote_average, file_path, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		episode.ID,
		episode.SeriesID,
		episode.TMDbID,
		episode.SeasonNumber,
		episode.EpisodeNumber,
		episode.Title,
		episode.Overview,
		episode.AirDate,
		episode.Runtime,
		episode.StillPath,
		episode.VoteAverage,
		episode.FilePath,
		episode.CreatedAt,
		episode.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create episode: %w", err)
	}

	return nil
}

// FindByID retrieves an episode by its primary key
func (r *EpisodeRepository) FindByID(ctx context.Context, id string) (*models.Episode, error) {
	query := `
		SELECT
			id, series_id, tmdb_id, season_number, episode_number,
			title, overview, air_date, runtime, still_path,
			vote_average, file_path, created_at, updated_at
		FROM episodes
		WHERE id = ?
	`

	episode := &models.Episode{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&episode.ID,
		&episode.SeriesID,
		&episode.TMDbID,
		&episode.SeasonNumber,
		&episode.EpisodeNumber,
		&episode.Title,
		&episode.Overview,
		&episode.AirDate,
		&episode.Runtime,
		&episode.StillPath,
		&episode.VoteAverage,
		&episode.FilePath,
		&episode.CreatedAt,
		&episode.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("episode with id %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find episode: %w", err)
	}

	return episode, nil
}

// FindBySeriesID retrieves all episodes for a series
func (r *EpisodeRepository) FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error) {
	query := `
		SELECT
			id, series_id, tmdb_id, season_number, episode_number,
			title, overview, air_date, runtime, still_path,
			vote_average, file_path, created_at, updated_at
		FROM episodes
		WHERE series_id = ?
		ORDER BY season_number, episode_number
	`

	rows, err := r.db.QueryContext(ctx, query, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to find episodes by series_id: %w", err)
	}
	defer rows.Close()

	episodes := []models.Episode{}
	for rows.Next() {
		episode := models.Episode{}
		err := rows.Scan(
			&episode.ID,
			&episode.SeriesID,
			&episode.TMDbID,
			&episode.SeasonNumber,
			&episode.EpisodeNumber,
			&episode.Title,
			&episode.Overview,
			&episode.AirDate,
			&episode.Runtime,
			&episode.StillPath,
			&episode.VoteAverage,
			&episode.FilePath,
			&episode.CreatedAt,
			&episode.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan episode: %w", err)
		}
		episodes = append(episodes, episode)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating episodes: %w", err)
	}

	return episodes, nil
}

// FindBySeasonNumber retrieves all episodes for a specific season of a series
func (r *EpisodeRepository) FindBySeasonNumber(ctx context.Context, seriesID string, seasonNumber int) ([]models.Episode, error) {
	query := `
		SELECT
			id, series_id, tmdb_id, season_number, episode_number,
			title, overview, air_date, runtime, still_path,
			vote_average, file_path, created_at, updated_at
		FROM episodes
		WHERE series_id = ? AND season_number = ?
		ORDER BY episode_number
	`

	rows, err := r.db.QueryContext(ctx, query, seriesID, seasonNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to find episodes by season: %w", err)
	}
	defer rows.Close()

	episodes := []models.Episode{}
	for rows.Next() {
		episode := models.Episode{}
		err := rows.Scan(
			&episode.ID,
			&episode.SeriesID,
			&episode.TMDbID,
			&episode.SeasonNumber,
			&episode.EpisodeNumber,
			&episode.Title,
			&episode.Overview,
			&episode.AirDate,
			&episode.Runtime,
			&episode.StillPath,
			&episode.VoteAverage,
			&episode.FilePath,
			&episode.CreatedAt,
			&episode.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan episode: %w", err)
		}
		episodes = append(episodes, episode)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating episodes: %w", err)
	}

	return episodes, nil
}

// FindBySeriesSeasonEpisode retrieves an episode by series ID, season, and episode number
func (r *EpisodeRepository) FindBySeriesSeasonEpisode(ctx context.Context, seriesID string, season, episode int) (*models.Episode, error) {
	query := `
		SELECT
			id, series_id, tmdb_id, season_number, episode_number,
			title, overview, air_date, runtime, still_path,
			vote_average, file_path, created_at, updated_at
		FROM episodes
		WHERE series_id = ? AND season_number = ? AND episode_number = ?
	`

	ep := &models.Episode{}
	err := r.db.QueryRowContext(ctx, query, seriesID, season, episode).Scan(
		&ep.ID,
		&ep.SeriesID,
		&ep.TMDbID,
		&ep.SeasonNumber,
		&ep.EpisodeNumber,
		&ep.Title,
		&ep.Overview,
		&ep.AirDate,
		&ep.Runtime,
		&ep.StillPath,
		&ep.VoteAverage,
		&ep.FilePath,
		&ep.CreatedAt,
		&ep.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("episode S%02dE%02d for series %s not found", season, episode, seriesID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find episode: %w", err)
	}

	return ep, nil
}

// Update modifies an existing episode in the database
func (r *EpisodeRepository) Update(ctx context.Context, episode *models.Episode) error {
	if episode == nil {
		return fmt.Errorf("episode cannot be nil")
	}

	// Update timestamp
	episode.UpdatedAt = time.Now()

	query := `
		UPDATE episodes
		SET
			series_id = ?,
			tmdb_id = ?,
			season_number = ?,
			episode_number = ?,
			title = ?,
			overview = ?,
			air_date = ?,
			runtime = ?,
			still_path = ?,
			vote_average = ?,
			file_path = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		episode.SeriesID,
		episode.TMDbID,
		episode.SeasonNumber,
		episode.EpisodeNumber,
		episode.Title,
		episode.Overview,
		episode.AirDate,
		episode.Runtime,
		episode.StillPath,
		episode.VoteAverage,
		episode.FilePath,
		episode.UpdatedAt,
		episode.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update episode: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("episode with id %s not found", episode.ID)
	}

	return nil
}

// Delete removes an episode from the database by ID
func (r *EpisodeRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM episodes WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete episode: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("episode with id %s not found", id)
	}

	return nil
}

// Upsert creates or updates an episode based on series_id, season_number, episode_number
func (r *EpisodeRepository) Upsert(ctx context.Context, episode *models.Episode) error {
	if episode == nil {
		return fmt.Errorf("episode cannot be nil")
	}

	// Check if episode already exists
	existing, err := r.FindBySeriesSeasonEpisode(ctx, episode.SeriesID, episode.SeasonNumber, episode.EpisodeNumber)
	if err != nil {
		// If not found, create new episode
		errMsg := fmt.Sprintf("episode S%02dE%02d for series %s not found", episode.SeasonNumber, episode.EpisodeNumber, episode.SeriesID)
		if err.Error() == errMsg {
			return r.Create(ctx, episode)
		}
		return fmt.Errorf("failed to check existing episode: %w", err)
	}

	// Episode exists - update with existing ID
	episode.ID = existing.ID
	episode.CreatedAt = existing.CreatedAt
	return r.Update(ctx, episode)
}
