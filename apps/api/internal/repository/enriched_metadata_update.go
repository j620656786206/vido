package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// Enriched-metadata writers — the NARROW counterpart to the wide Update.
//
// WHY THIS FILE EXISTS (9R-10b CR-249 finding B).
//
// MovieRepository.Update and SeriesRepository.Update write EVERY mutable column,
// including the five subtitle-delivery columns (subtitle_status, subtitle_path,
// subtitle_language, subtitle_last_searched, subtitle_search_score). That is
// correct for callers that own the whole row, and wrong for one that does not.
//
// EnrichmentService does not own those columns — an audit of every assignment it
// makes shows it never sets one. But it loads a row, then spends seconds to tens
// of seconds on it (NFO read, filename parse which may call an LLM, TMDB search)
// before writing the whole thing back. Anything that touched the subtitle columns
// during that window is silently reverted.
//
// That window is not hypothetical, and it is not rare: enrichment enumerates rows
// with parse_status pending/empty — newly scanned files — and story 9R-10b's
// auto-trigger runs the free subtitle lane over rows missing zh-Hant subtitles —
// also newly scanned files — CONCURRENTLY, off the same scan-complete callback.
// The overlap is precisely the feature's headline case: drop files in at night,
// wake up to subtitles. The observable symptom is the worst kind — the .srt is
// written to disk, and then the database is told it does not exist.
//
// These writers persist EXACTLY the columns enrichment computes, so a concurrent
// subtitle write can no longer be clobbered by a stale in-memory copy.
//
// SCOPE, STATED HONESTLY: this closes the enrichment writer only. The wide Update
// has other callers (scanner, library, metadata-edit, movie services) that carry
// the same stale-copy hazard in principle; each needs its own audit of what it
// legitimately owns, which is tracked separately rather than guessed at here.

// UpdateEnrichedMetadata persists only the columns EnrichmentService computes.
//
// Deliberately NOT written here (enrichment never assigns them, so writing them
// back from a possibly-stale copy could only ever lose someone else's work):
// the five subtitle-delivery columns, file_path, file_size, library_id,
// is_removed, rating, original_language, status, production_countries, credits,
// spoken_languages. subtitle_tracks IS written — it is ffprobe/NFO technical
// data that enrichment genuinely produces, not delivery state.
func (r *MovieRepository) UpdateEnrichedMetadata(ctx context.Context, movie *models.Movie) error {
	if movie == nil {
		return fmt.Errorf("movie cannot be nil")
	}
	if movie.ID == "" {
		return fmt.Errorf("movie id cannot be empty")
	}

	genresJSON, err := movie.GenresJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal genres: %w", err)
	}

	movie.UpdatedAt = time.Now()

	const query = `
		UPDATE movies
		SET
			title = ?,
			original_title = ?,
			release_date = ?,
			genres = ?,
			overview = ?,
			poster_path = ?,
			backdrop_path = ?,
			runtime = ?,
			duration_seconds = ?,
			imdb_id = ?,
			tmdb_id = ?,
			parse_status = ?,
			metadata_source = ?,
			vote_average = ?,
			vote_count = ?,
			popularity = ?,
			video_codec = ?,
			video_resolution = ?,
			audio_codec = ?,
			audio_channels = ?,
			-- NULL is "unknown" and never replaces a stored answer: enrichment
			-- that skipped the probe (NFO tech info) still holds the NULL it
			-- loaded, and must not erase what the subtitle-track backfill
			-- wrote meanwhile (disc-2026-10-movie-subtitle-tracks-unknown-refresh).
			subtitle_tracks = COALESCE(?, subtitle_tracks),
			hdr_format = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		movie.Title, movie.OriginalTitle, movie.ReleaseDate, genresJSON,
		movie.Overview, movie.PosterPath, movie.BackdropPath, movie.Runtime,
		movie.DurationSeconds,
		movie.IMDbID, movie.TMDbID, movie.ParseStatus, movie.MetadataSource,
		movie.VoteAverage, movie.VoteCount, movie.Popularity,
		movie.VideoCodec, movie.VideoResolution, movie.AudioCodec, movie.AudioChannels,
		movie.SubtitleTracks, movie.HDRFormat,
		movie.UpdatedAt, movie.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update movie metadata: %w", err)
	}
	return requireOneRow(result, "movie", movie.ID)
}

// UpdateEnrichedMetadata is the series counterpart. Enrichment assigns a smaller
// set on a series than on a movie — no runtime, no tech info, no imdb_id, no
// popularity — and this writer matches that audit exactly.
func (r *SeriesRepository) UpdateEnrichedMetadata(ctx context.Context, series *models.Series) error {
	if series == nil {
		return fmt.Errorf("series cannot be nil")
	}
	if series.ID == "" {
		return fmt.Errorf("series id cannot be empty")
	}

	genresJSON, err := series.GenresJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal genres: %w", err)
	}

	series.UpdatedAt = time.Now()

	const query = `
		UPDATE series
		SET
			title = ?,
			original_title = ?,
			first_air_date = ?,
			genres = ?,
			overview = ?,
			poster_path = ?,
			backdrop_path = ?,
			tmdb_id = ?,
			parse_status = ?,
			metadata_source = ?,
			vote_average = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		series.Title, series.OriginalTitle, series.FirstAirDate, genresJSON,
		series.Overview, series.PosterPath, series.BackdropPath,
		series.TMDbID, series.ParseStatus, series.MetadataSource, series.VoteAverage,
		series.UpdatedAt, series.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update series metadata: %w", err)
	}
	return requireOneRow(result, "series", series.ID)
}

// requireOneRow turns "no such row" into an error rather than a silent no-op —
// an enrichment write that hits zero rows means the media was removed underneath
// us, and the caller deserves to know.
func requireOneRow(result sql.Result, kind, id string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%s with id %s not found", kind, id)
	}
	return nil
}

// UpdateCredits persists ONLY the credits column (sub-7-3). Enrichment now
// fetches a matched title's cast from TMDb, and — by this file's discipline —
// that is a column of its own rather than a widening of UpdateEnrichedMetadata:
// the caller decides ownership (models.ShouldOverwrite against the row's
// metadata_source) BEFORE calling, so a Metadata-Editor cast is never touched,
// and a concurrent editor write cannot be reverted by the wide copy either.
// A nil credits clears the column.
func (r *MovieRepository) UpdateCredits(ctx context.Context, id string, credits *models.Credits) error {
	return updateCreditsColumn(ctx, r.db, "movies", id, credits)
}

// UpdateCredits is the series counterpart of MovieRepository.UpdateCredits.
func (r *SeriesRepository) UpdateCredits(ctx context.Context, id string, credits *models.Credits) error {
	return updateCreditsColumn(ctx, r.db, "series", id, credits)
}

func updateCreditsColumn(ctx context.Context, db *sql.DB, table, id string, credits *models.Credits) error {
	if id == "" {
		return fmt.Errorf("%s id cannot be empty", table)
	}
	var value models.NullString
	if credits != nil && (len(credits.Cast) > 0 || len(credits.Crew) > 0) {
		// Serialise through the model so the column shape stays identical to
		// what the Metadata Editor writes (Movie.SetCredits / Series.SetCredits).
		var m models.Movie
		if err := m.SetCredits(credits); err != nil {
			return fmt.Errorf("failed to marshal credits: %w", err)
		}
		value = m.CreditsJSON
	}
	result, err := db.ExecContext(ctx,
		`UPDATE `+table+` SET credits = ?, updated_at = ? WHERE id = ?`,
		value, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to update %s credits: %w", table, err)
	}
	return requireOneRow(result, table, id)
}

// FindMissingCredits lists matched movies whose cast was never stored
// (disc-2026-10-series-credits-empty): rows matched before sub-7-3 began
// writing TMDb cast (2026-09-07) never got one, and nothing re-queued them.
// NULL and ” are both "missing" (UpdateCredits writes NULL for an empty cast).
func (r *MovieRepository) FindMissingCredits(ctx context.Context, limit int) ([]models.Movie, error) {
	query := fmt.Sprintf(`SELECT %s FROM movies
		WHERE tmdb_id IS NOT NULL AND tmdb_id > 0
		  AND (credits IS NULL OR credits = '')
		  AND is_removed = 0
		ORDER BY updated_at ASC LIMIT ?`, movieSelectColumns)
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query movies missing credits: %w", err)
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
		return nil, fmt.Errorf("error iterating movies missing credits: %w", err)
	}
	return movies, nil
}

// FindMissingCredits is the series counterpart of MovieRepository.FindMissingCredits.
func (r *SeriesRepository) FindMissingCredits(ctx context.Context, limit int) ([]models.Series, error) {
	query := fmt.Sprintf(`SELECT %s FROM series
		WHERE tmdb_id IS NOT NULL AND tmdb_id > 0
		  AND (credits IS NULL OR credits = '')
		  AND is_removed = 0
		ORDER BY updated_at ASC LIMIT ?`, seriesSelectColumns)
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query series missing credits: %w", err)
	}
	defer rows.Close()
	var list []models.Series
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan series: %w", err)
		}
		list = append(list, s)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating series missing credits: %w", err)
	}
	return list, nil
}

// FindMissingSubtitleTracks lists movies with a file whose subtitle_tracks is
// still NULL — "we don't know whether it has Chinese subtitles"
// (disc-2026-10-movie-subtitle-tracks-unknown-refresh). Scans before
// disc-2026-10-subtitle-filter-disagrees-with-badges left an empty probe as
// NULL, and enrichment still skips the probe when NFO already supplied the
// tech info, so nothing else ever re-reads these rows. `[]` (probed, no
// subtitles) is a known answer and is NOT listed.
//
// Paged by id (afterID "" = from the start): rows that stay NULL — a file on
// an unmounted share, a probe that fails — are listed every pass, and a
// fixed first page would let them shut the rest out for good.
func (r *MovieRepository) FindMissingSubtitleTracks(ctx context.Context, afterID string, limit int) ([]models.Movie, error) {
	query := fmt.Sprintf(`SELECT %s FROM movies
		WHERE file_path IS NOT NULL AND file_path != ''
		  AND subtitle_tracks IS NULL
		  AND is_removed = 0
		  AND id > ?
		ORDER BY id ASC LIMIT ?`, movieSelectColumns)
	rows, err := r.db.QueryContext(ctx, query, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query movies missing subtitle tracks: %w", err)
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
		return nil, fmt.Errorf("error iterating movies missing subtitle tracks: %w", err)
	}
	return movies, nil
}

// UpdateSubtitleTracksIfMissing writes subtitle_tracks ONLY while the row
// still has none, so a scan that stored a value while the backfill was
// probing is never overwritten. written=false means another writer got there
// first (or the row is gone) — not an error. updated_at is left alone: this
// is a derived field filled in the background, not an edit of the movie.
func (r *MovieRepository) UpdateSubtitleTracksIfMissing(ctx context.Context, id, tracksJSON string) (written bool, err error) {
	if id == "" {
		return false, fmt.Errorf("movie id cannot be empty")
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE movies SET subtitle_tracks = ? WHERE id = ? AND subtitle_tracks IS NULL`,
		tracksJSON, id,
	)
	if err != nil {
		return false, fmt.Errorf("failed to update movie subtitle tracks: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}
