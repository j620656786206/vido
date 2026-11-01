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
	subtitle_tracks, subtitle_tracks_file_sig, burned_in_subtitle, burned_in_checked_at,
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
		&episode.SubtitleTracks,
		&episode.SubtitleTracksFileSig,
		&episode.BurnedInSubtitle,
		&episode.BurnedInCheckedAt,
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

	episodes, err := scanEpisodeRows(rows)
	if err != nil {
		return nil, err
	}
	// disc-2026-10-candidates-ignore-existing-chinese: the column only knows
	// what Vido itself placed. An embedded Traditional track or a .zh-TW
	// sidecar the sweep stored in subtitle_tracks is a subtitle too — the
	// same verdict the season-list badge renders. Chinese burned into the
	// picture, of any script, also needs nothing generated
	// (disc-2026-11-burned-in-subs-b2, ruling 2A).
	kept := episodes[:0]
	for _, e := range episodes {
		if models.HasTraditionalOrBurnedInChinese(string(e.SubtitleStatus), e.SubtitleLanguage.String, e.SubtitleTracks.String, e.BurnedInSubtitle.String) {
			continue
		}
		kept = append(kept, e)
	}
	return kept, nil
}

// CountMissingZhHantSubtitle returns the number of episodes
// FindMissingZhHantSubtitle would enumerate — the episode half of the
// generation-batch preview count (sub-5-1 AC #7), mirroring
// MovieRepository.CountMissingZhHantSubtitle over the same shared predicate.
func (r *EpisodeRepository) CountMissingZhHantSubtitle(ctx context.Context) (int, error) {
	// The four columns the verdict reads, nothing else: the count must drop
	// exactly the rows FindMissingZhHantSubtitle drops.
	query := `SELECT subtitle_status, subtitle_language, subtitle_tracks, burned_in_subtitle FROM episodes WHERE ` + missingZhHantSubtitleEpisodeWhere
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count episodes missing zh-Hant subtitle: %w", err)
	}
	defer rows.Close()
	return countMissingZhHantRows(rows)
}

// countMissingZhHantRows counts the (status, language, tracks, burned) rows
// that are neither Traditional Chinese already nor burned-in Chinese — the
// rows FindMissingZhHantSubtitle keeps; shared by the movie and episode
// counts (movies select NULL for burned: they have no such column).
func countMissingZhHantRows(rows *sql.Rows) (int, error) {
	count := 0
	for rows.Next() {
		var status sql.NullString
		var language, tracks, burned sql.NullString
		if err := rows.Scan(&status, &language, &tracks, &burned); err != nil {
			return 0, fmt.Errorf("failed to scan subtitle columns: %w", err)
		}
		if models.HasTraditionalOrBurnedInChinese(status.String, language.String, tracks.String, burned.String) {
			continue
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating subtitle columns: %w", err)
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

// FindWithFiles lists every episode that has a media file, in broadcast order
// — the background subtitle-track sweep's worklist
// (disc-2026-10-episode-list-subtitle-badge-a). The sweep decides per row
// whether it needs a probe (file signature), so this does not filter on it.
func (r *EpisodeRepository) FindWithFiles(ctx context.Context) ([]models.Episode, error) {
	query := `SELECT ` + episodeSelectColumns + `
		FROM episodes
		WHERE file_path IS NOT NULL AND file_path != ''
		ORDER BY series_id, season_number, episode_number, id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes with files: %w", err)
	}
	defer rows.Close()

	return scanEpisodeRows(rows)
}

// EpisodeFileRef is one episode file the scanner checks for removal: the
// episode, the path it holds, and the library of its series (episodes carry
// no library of their own).
type EpisodeFileRef struct {
	ID        string
	FilePath  string
	LibraryID string
}

// FindFilesForRemovalCheck lists every episode that holds a file path, with
// its series' library — the scanner's removed-file pass
// (disc-2026-10-episode-rows-outlive-deleted-files).
func (r *EpisodeRepository) FindFilesForRemovalCheck(ctx context.Context) ([]EpisodeFileRef, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT e.id, e.file_path, COALESCE(s.library_id, '')
		FROM episodes e LEFT JOIN series s ON s.id = e.series_id
		WHERE e.file_path IS NOT NULL AND e.file_path != ''
		ORDER BY e.id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query episode files: %w", err)
	}
	defer rows.Close()
	var refs []EpisodeFileRef
	for rows.Next() {
		var ref EpisodeFileRef
		if err := rows.Scan(&ref.ID, &ref.FilePath, &ref.LibraryID); err != nil {
			return nil, fmt.Errorf("failed to scan episode file: %w", err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating episode files: %w", err)
	}
	return refs, nil
}

// ClearMissingFile turns an episode whose file is gone back into "known
// episode, no local file": file_path and the columns read FROM that file
// (subtitle_tracks, subtitle_tracks_file_sig, and the burned-in answer —
// disc-2026-11-burned-in-subs-b2 AC #6) are cleared. Everything else —
// TMDb data, subtitle delivery state — stays: other flows own it, and they
// already skip episodes without a path. The next scan that finds the file
// again writes the path back onto this same row (Upsert matches series +
// season + episode).
//
// Compare-and-set on the path the caller checked, so an episode re-ingested
// at a new path while the scan was stat-ing is left alone. Returns whether
// it cleared.
func (r *EpisodeRepository) ClearMissingFile(ctx context.Context, episodeID, filePath string) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE episodes SET file_path = NULL, subtitle_tracks = NULL, subtitle_tracks_file_sig = NULL,
		        burned_in_subtitle = NULL, burned_in_checked_at = NULL
		 WHERE id = ? AND file_path = ?`,
		episodeID, filePath)
	if err != nil {
		return false, fmt.Errorf("failed to clear missing episode file: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return n > 0, nil
}

// ReleaseFileFromOtherEpisodes clears filePath from every OTHER episode row of
// the series that still holds it — one file is one episode. A file that used to
// collapse onto the episode-0 placeholder (before bare SxxExx names parsed) and
// now lands on its real number must not show up twice
// (disc-2026-10-bare-episode-filename-uses-folder-title). Same columns as
// ClearMissingFile; the rows stay.
func (r *EpisodeRepository) ReleaseFileFromOtherEpisodes(ctx context.Context, seriesID, filePath, keepEpisodeID string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE episodes SET file_path = NULL, subtitle_tracks = NULL, subtitle_tracks_file_sig = NULL,
		        burned_in_subtitle = NULL, burned_in_checked_at = NULL
		 WHERE series_id = ? AND file_path = ? AND id <> ?`,
		seriesID, filePath, keepEpisodeID)
	if err != nil {
		return 0, fmt.Errorf("failed to release episode file: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return n, nil
}

// UpdateSubtitleTracks records what the sweep read for one episode: the
// merged embedded+sidecar tracks and the file signature they were read from
// (migration 044). Narrow write, like UpdateDurationSeconds: `updated_at` is
// not touched — the file did not change, we only looked at it.
//
// When the signature differs from the stored one the file was replaced, so
// the burned-in answer read from the old picture is cleared and the season is
// checked again (disc-2026-11-burned-in-subs-b2 AC #6). A re-read of the same
// file (a script peek) keeps it. SQLite evaluates every SET expression against
// the row as it was, so the CASE sees the OLD signature.
func (r *EpisodeRepository) UpdateSubtitleTracks(ctx context.Context, episodeID, tracksJSON, fileSig string) error {
	if episodeID == "" {
		return fmt.Errorf("episode id cannot be empty")
	}
	sig := newNullableString(fileSig)
	result, err := r.db.ExecContext(ctx,
		`UPDATE episodes SET subtitle_tracks = ?, subtitle_tracks_file_sig = ?,
		        burned_in_subtitle = CASE WHEN subtitle_tracks_file_sig IS ? THEN burned_in_subtitle ELSE NULL END,
		        burned_in_checked_at = CASE WHEN subtitle_tracks_file_sig IS ? THEN burned_in_checked_at ELSE NULL END
		 WHERE id = ?`,
		tracksJSON, sig, sig, sig, episodeID)
	if err != nil {
		return fmt.Errorf("failed to update episode subtitle tracks: %w", err)
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

// RefreshSubtitleTracks replaces subtitle_tracks only if it still holds
// `previous` — the season list's sidecar refresh. The compare-and-set keeps a
// list opened with a stale copy from writing an old embedded half over a
// re-probe the sweep committed a moment ago. Returns whether it wrote.
func (r *EpisodeRepository) RefreshSubtitleTracks(ctx context.Context, episodeID, previous, next string) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE episodes SET subtitle_tracks = ? WHERE id = ? AND subtitle_tracks = ?`,
		next, episodeID, previous)
	if err != nil {
		return false, fmt.Errorf("failed to refresh episode subtitle tracks: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return n > 0, nil
}

// SetSeasonBurnedInSubtitle writes one burned-in check's result onto every
// episode of the season whose tracks were read and found empty
// (subtitle_tracks = '[]') — disc-2026-11-burned-in-subs-b1: one episode is
// checked and stands for the season, but an episode that HAS tracks is left
// alone (its tracks already answer). Narrow write like UpdateSubtitleTracks:
// updated_at is not touched. Returns how many episodes it wrote.
func (r *EpisodeRepository) SetSeasonBurnedInSubtitle(ctx context.Context, seriesID string, season int, value string, checkedAt time.Time) (int64, error) {
	if !models.IsBurnedInValue(value) {
		return 0, fmt.Errorf("invalid burned-in subtitle value %q", value)
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE episodes SET burned_in_subtitle = ?, burned_in_checked_at = ?
		 WHERE series_id = ? AND season_number = ? AND subtitle_tracks = '[]'`,
		value, checkedAt.UTC(), seriesID, season)
	if err != nil {
		return 0, fmt.Errorf("failed to set season burned-in subtitle: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return n, nil
}

// MarkSeasonBurnedInChecked stamps burned_in_checked_at on the season's
// still-unanswered trackless episodes, leaving burned_in_subtitle NULL: the
// check ran but could not decide (disc-2026-11-burned-in-subs-b2 AC #2), and
// the sweep should not pay to ask again before the retry window passes.
// Returns how many episodes it stamped.
func (r *EpisodeRepository) MarkSeasonBurnedInChecked(ctx context.Context, seriesID string, season int, checkedAt time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE episodes SET burned_in_checked_at = ?
		 WHERE series_id = ? AND season_number = ? AND subtitle_tracks = '[]' AND burned_in_subtitle IS NULL`,
		checkedAt.UTC(), seriesID, season)
	if err != nil {
		return 0, fmt.Errorf("failed to mark season burned-in checked: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return n, nil
}

// BurnedInSeasonRef names one season the burned-in sweep should check.
type BurnedInSeasonRef struct {
	SeriesID    string
	SeriesTitle string
	Season      int
}

// FindSeasonsNeedingBurnedInCheck lists the seasons with at least one episode
// that has a file, no subtitle track (subtitle_tracks = '[]') and no
// burned-in answer — skipping episodes whose last undecided check is newer
// than retryBefore (disc-2026-11-burned-in-subs-b2 AC #1). Removed series are
// left out. Ordered by series title, then season, so the sweep reads like the
// library.
func (r *EpisodeRepository) FindSeasonsNeedingBurnedInCheck(ctx context.Context, retryBefore time.Time) ([]BurnedInSeasonRef, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.series_id, COALESCE(s.title, ''), e.season_number
		   FROM episodes e
		   JOIN series s ON s.id = e.series_id
		  WHERE e.file_path IS NOT NULL AND e.file_path != ''
		    AND e.subtitle_tracks = '[]'
		    AND e.burned_in_subtitle IS NULL
		    AND (e.burned_in_checked_at IS NULL OR e.burned_in_checked_at < ?)
		    AND (s.is_removed = 0 OR s.is_removed IS NULL)
		  GROUP BY e.series_id, e.season_number
		  ORDER BY COALESCE(s.title, '') COLLATE NOCASE, e.series_id, e.season_number`,
		retryBefore.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to find seasons needing a burned-in check: %w", err)
	}
	defer rows.Close()
	var out []BurnedInSeasonRef
	for rows.Next() {
		var ref BurnedInSeasonRef
		if err := rows.Scan(&ref.SeriesID, &ref.SeriesTitle, &ref.Season); err != nil {
			return nil, fmt.Errorf("failed to scan burned-in season: %w", err)
		}
		out = append(out, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating burned-in seasons: %w", err)
	}
	return out, nil
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
