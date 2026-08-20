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
			subtitle_tracks = ?,
			hdr_format = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		movie.Title, movie.OriginalTitle, movie.ReleaseDate, genresJSON,
		movie.Overview, movie.PosterPath, movie.BackdropPath, movie.Runtime,
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
