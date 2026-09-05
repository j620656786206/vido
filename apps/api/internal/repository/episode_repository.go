package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// ErrEpisodeNotFound is returned when an episode lookup finds no matching record.
var ErrEpisodeNotFound = errors.New("episode not found")

// episodeSelectColumns is the canonical column list for episode SELECTs. The
// subtitle_* columns (Story 12-2, migration 025) are included so every episode
// read carries its per-episode subtitle status for the detail-page accordion.
const episodeSelectColumns = `
	id, series_id, season_id, tmdb_id, season_number, episode_number,
	title, overview, air_date, runtime, duration_seconds, still_path,
	vote_average, file_path, subtitle_status, subtitle_path, subtitle_language,
	created_at, updated_at`

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

// scanEpisode scans a single row (from *sql.Row or *sql.Rows) into an Episode,
// matching the column order of episodeSelectColumns.
func scanEpisode(s interface{ Scan(...any) error }, episode *models.Episode) error {
	return s.Scan(
		&episode.ID,
		&episode.SeriesID,
		&episode.SeasonID,
		&episode.TMDbID,
		&episode.SeasonNumber,
		&episode.EpisodeNumber,
		&episode.Title,
		&episode.Overview,
		&episode.AirDate,
		&episode.Runtime,
		&episode.DurationSeconds,
		&episode.StillPath,
		&episode.VoteAverage,
		&episode.FilePath,
		&episode.SubtitleStatus,
		&episode.SubtitlePath,
		&episode.SubtitleLanguage,
		&episode.CreatedAt,
		&episode.UpdatedAt,
	)
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
			id, series_id, season_id, tmdb_id, season_number, episode_number,
			title, overview, air_date, runtime, duration_seconds, still_path,
			vote_average, file_path, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		episode.ID,
		episode.SeriesID,
		episode.SeasonID,
		episode.TMDbID,
		episode.SeasonNumber,
		episode.EpisodeNumber,
		episode.Title,
		episode.Overview,
		episode.AirDate,
		episode.Runtime,
		episode.DurationSeconds,
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
	query := `SELECT ` + episodeSelectColumns + ` FROM episodes WHERE id = ?`

	episode := &models.Episode{}
	err := scanEpisode(r.db.QueryRowContext(ctx, query, id), episode)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("episode with id %s: %w", id, ErrEpisodeNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find episode: %w", err)
	}

	return episode, nil
}

// FindBySeriesID retrieves all episodes for a series
func (r *EpisodeRepository) FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error) {
	query := `SELECT ` + episodeSelectColumns + `
		FROM episodes
		WHERE series_id = ?
		ORDER BY season_number, episode_number`

	rows, err := r.db.QueryContext(ctx, query, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to find episodes by series_id: %w", err)
	}
	defer rows.Close()

	return scanEpisodeRows(rows)
}

// missingZhHantSubtitleEpisodeWhere is the episode-grain twin of
// missingZhHantSubtitleWhere (movie_repository.go): "needs generation" means no
// zh-Hant subtitle on record AND a media file to extract from.
//
// Deliberately BROADER than a subtitle_status filter, for the same reason as
// the movie predicate: an episode with a found ENGLISH subtitle still lacks
// zh-Hant and is in scope. Completed items self-exclude once generation writes
// subtitle_language='zh-Hant', which is what makes a re-scan free.
//
// Unlike movies, the episodes table has no is_removed column — removal is
// modelled by deleting the row (migration 006), so there is no soft-delete
// clause to mirror.
const missingZhHantSubtitleEpisodeWhere = `
	(subtitle_language IS NULL OR subtitle_language != 'zh-Hant')
	AND file_path IS NOT NULL AND file_path != ''
`

// FindMissingZhHantSubtitle retrieves episodes that need generation
// (story sub-1-6 AC #11). Ordered by series then season/episode so a batch
// queue reads in broadcast order — which also groups a show's episodes
// together, giving the D10 per-show gate its warm prompt prefix.
func (r *EpisodeRepository) FindMissingZhHantSubtitle(ctx context.Context) ([]models.Episode, error) {
	query := `SELECT ` + episodeSelectColumns + `
		FROM episodes
		WHERE ` + missingZhHantSubtitleEpisodeWhere + `
		ORDER BY series_id, season_number, episode_number, id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes missing zh-Hant subtitle: %w", err)
	}
	defer rows.Close()

	return scanEpisodeRows(rows)
}

// CountMissingZhHantSubtitle returns the number of episodes
// FindMissingZhHantSubtitle would enumerate — the episode half of the
// generation-batch preview count (sub-5-1 AC #7), mirroring
// MovieRepository.CountMissingZhHantSubtitle over the same shared predicate.
func (r *EpisodeRepository) CountMissingZhHantSubtitle(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM episodes WHERE ` + missingZhHantSubtitleEpisodeWhere

	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count episodes missing zh-Hant subtitle: %w", err)
	}
	return count, nil
}

// FindBySeasonNumber retrieves all episodes for a specific season of a series
func (r *EpisodeRepository) FindBySeasonNumber(ctx context.Context, seriesID string, seasonNumber int) ([]models.Episode, error) {
	query := `SELECT ` + episodeSelectColumns + `
		FROM episodes
		WHERE series_id = ? AND season_number = ?
		ORDER BY episode_number`

	rows, err := r.db.QueryContext(ctx, query, seriesID, seasonNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to find episodes by season: %w", err)
	}
	defer rows.Close()

	return scanEpisodeRows(rows)
}

// FindBySeasonID retrieves all episodes for a specific season
func (r *EpisodeRepository) FindBySeasonID(ctx context.Context, seasonID string) ([]models.Episode, error) {
	query := `SELECT ` + episodeSelectColumns + `
		FROM episodes
		WHERE season_id = ?
		ORDER BY episode_number`

	rows, err := r.db.QueryContext(ctx, query, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to find episodes by season_id: %w", err)
	}
	defer rows.Close()

	return scanEpisodeRows(rows)
}

// scanEpisodeRows iterates a *sql.Rows result set into a slice of episodes.
func scanEpisodeRows(rows *sql.Rows) ([]models.Episode, error) {
	episodes := []models.Episode{}
	for rows.Next() {
		episode := models.Episode{}
		if err := scanEpisode(rows, &episode); err != nil {
			return nil, fmt.Errorf("failed to scan episode: %w", err)
		}
		episodes = append(episodes, episode)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating episodes: %w", err)
	}

	return episodes, nil
}

// FindBySeriesSeasonEpisode retrieves an episode by series ID, season, and episode number
func (r *EpisodeRepository) FindBySeriesSeasonEpisode(ctx context.Context, seriesID string, season, episode int) (*models.Episode, error) {
	query := `SELECT ` + episodeSelectColumns + `
		FROM episodes
		WHERE series_id = ? AND season_number = ? AND episode_number = ?`

	ep := &models.Episode{}
	err := scanEpisode(r.db.QueryRowContext(ctx, query, seriesID, season, episode), ep)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("episode S%02dE%02d for series %s: %w", season, episode, seriesID, ErrEpisodeNotFound)
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
			season_id = ?,
			tmdb_id = ?,
			season_number = ?,
			episode_number = ?,
			title = ?,
			overview = ?,
			air_date = ?,
			runtime = ?,
			duration_seconds = ?,
			still_path = ?,
			vote_average = ?,
			file_path = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		episode.SeriesID,
		episode.SeasonID,
		episode.TMDbID,
		episode.SeasonNumber,
		episode.EpisodeNumber,
		episode.Title,
		episode.Overview,
		episode.AirDate,
		episode.Runtime,
		episode.DurationSeconds,
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

// UpdateEpisodeSubtitleStatus updates only the subtitle tracking columns for an
// episode (Story 12-2 Task 5.3). The subtitle engine (Epic 8, currently
// series-level) will call this once per-episode subtitle search is implemented;
// this story writes the status that the detail-page accordion displays.
func (r *EpisodeRepository) UpdateEpisodeSubtitleStatus(ctx context.Context, episodeID string, status models.SubtitleStatus, path, language string) error {
	if episodeID == "" {
		return fmt.Errorf("episode id cannot be empty")
	}

	query := `
		UPDATE episodes
		SET subtitle_status = ?, subtitle_path = ?, subtitle_language = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		string(status),
		newNullableString(path),
		newNullableString(language),
		time.Now(),
		episodeID,
	)
	if err != nil {
		return fmt.Errorf("failed to update episode subtitle status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("episode with id %s: %w", episodeID, ErrEpisodeNotFound)
	}

	return nil
}

// UpdateDurationSeconds records the container duration measured for one
// episode (sub-6-10a AC #1, migration 035).
//
// A narrow single-column write, mirroring UpdateEpisodeSubtitleStatus above,
// rather than a read-modify-write through Update(): the caller is the candidate
// sweep, which holds up to a couple of thousand rows in flight and must not
// stamp its own stale copy of a row's title or subtitle status over whatever
// the scanner wrote a moment ago.
//
// `updated_at` is deliberately NOT touched: this is a measurement of a file
// that did not change, and bumping the row's timestamp would tell every
// "recently updated" surface that something happened to it.
//
// seconds must be positive — 0 means "the probe found no duration", and
// storing that would turn "unknown" into "zero-length", which prices as free.
func (r *EpisodeRepository) UpdateDurationSeconds(ctx context.Context, episodeID string, seconds int64) error {
	if episodeID == "" {
		return fmt.Errorf("episode id cannot be empty")
	}
	if seconds <= 0 {
		return fmt.Errorf("episode %s: duration must be positive, got %d", episodeID, seconds)
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE episodes SET duration_seconds = ? WHERE id = ?`, seconds, episodeID)
	if err != nil {
		return fmt.Errorf("failed to update episode duration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("episode with id %s: %w", episodeID, ErrEpisodeNotFound)
	}

	return nil
}

// newNullableString returns a sql-friendly value: NULL for empty strings, the
// string otherwise. Keeps subtitle_path/subtitle_language NULL when unset.
func newNullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
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
func (r *EpisodeRepository) Upsert(ctx context.Context, episode *models.Episode) (bool, error) {
	if episode == nil {
		return false, fmt.Errorf("episode cannot be nil")
	}

	// Check if episode already exists
	existing, err := r.FindBySeriesSeasonEpisode(ctx, episode.SeriesID, episode.SeasonNumber, episode.EpisodeNumber)
	if err != nil {
		if errors.Is(err, ErrEpisodeNotFound) {
			return true, r.Create(ctx, episode)
		}
		return false, fmt.Errorf("failed to check existing episode: %w", err)
	}

	// Episode exists - update with existing ID
	episode.ID = existing.ID
	episode.CreatedAt = existing.CreatedAt
	return false, r.Update(ctx, episode)
}
