package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// Single-intent writers for scan / library / poster state — the NARROW
// counterparts to the wide Update, second batch.
//
// WHY THIS FILE EXISTS (bugfix-wide-update-stale-copy-other-callers, the
// follow-up to 9R-10b CR-249 finding B).
//
// MovieRepository.Update writes 37 columns (including the five
// subtitle-delivery columns, file_path, file_size, library_id, is_removed);
// SeriesRepository.Update writes 32. A caller that loads a row, does slow work,
// then writes the whole row back reverts whatever anyone else wrote in between.
// 9R-10b closed that for EnrichmentService (enriched_metadata_update.go). The
// audit of the REMAINING callers (story §audit) found:
//
//   - scanner detectRemovedFiles: loads EVERY movie, then os.Stat's each one on
//     the NAS before writing `is_removed` back with the whole row. Runs right
//     after scan-complete — concurrently with enrichment and the free subtitle
//     lane. Seconds-long window, one-column intent.
//   - scanner aggregateSeriesFileSizes: loads EVERY series, stats every episode
//     file, writes `file_size` back with the whole row. Same shape.
//   - scanner processVideoFile, BatchReparse, poster upload: microsecond
//     windows, but one- or two-column intents writing 37 columns during the
//     very scan whose callback starts the subtitle lane.
//
// Every writer here persists EXACTLY the columns its caller computes, plus
// updated_at. The user-edit paths (metadata edit, PUT /movies, PUT /series)
// deliberately keep the wide Update: they own the fields they write.
//
// 0 ROWS AFFECTED. The wide Update returns an error when no row matched
// (movie_repository.go: "movie with id %s not found"); these writers do the
// same but wrap sql.ErrNoRows so callers can errors.Is it. A narrow write to a
// vanished row is never silent.

func execNarrow(ctx context.Context, db *sql.DB, table, id, query string, args ...any) error {
	if id == "" {
		return fmt.Errorf("%s id cannot be empty", table)
	}
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update %s %s: %w", table, id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%s with id %s not found: %w", table, id, sql.ErrNoRows)
	}
	return nil
}

// ─── movies ─────────────────────────────────────────────────────────────────

// UpdateScanFileInfo records what a scan learned about the file on disk:
// its size, and that the row must be re-parsed. (scanner processVideoFile)
func (r *MovieRepository) UpdateScanFileInfo(ctx context.Context, id string, fileSize int64, parseStatus models.ParseStatus) error {
	return execNarrow(ctx, r.db, "movie", id,
		`UPDATE movies SET file_size = ?, parse_status = ?, updated_at = ? WHERE id = ?`,
		fileSize, parseStatus, time.Now(), id)
}

// MarkRemoved flags a movie whose file is gone from disk. (scanner
// detectRemovedFiles) Nothing else on the row is touched — a subtitle
// delivered while the removed-file pass was walking the library survives.
func (r *MovieRepository) MarkRemoved(ctx context.Context, id string) error {
	return execNarrow(ctx, r.db, "movie", id,
		`UPDATE movies SET is_removed = 1, updated_at = ? WHERE id = ?`,
		time.Now(), id)
}

// UpdateParseStatus queues or settles a parse without touching anything
// else. (BatchReparse)
func (r *MovieRepository) UpdateParseStatus(ctx context.Context, id string, status models.ParseStatus) error {
	return execNarrow(ctx, r.db, "movie", id,
		`UPDATE movies SET parse_status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id)
}

// UpdatePosterPath stores a new poster location. (poster upload / fetch)
// An empty path clears the column to NULL, matching what the wide Update
// did through models.NewNullString — never an empty string.
func (r *MovieRepository) UpdatePosterPath(ctx context.Context, id, posterPath string) error {
	return execNarrow(ctx, r.db, "movie", id,
		`UPDATE movies SET poster_path = ?, updated_at = ? WHERE id = ?`,
		models.NewNullString(posterPath), time.Now(), id)
}

// ─── series ─────────────────────────────────────────────────────────────────

// UpdateFileSize stores the aggregated size of a series' episode files.
// (scanner aggregateSeriesFileSizes)
func (r *SeriesRepository) UpdateFileSize(ctx context.Context, id string, fileSize int64) error {
	return execNarrow(ctx, r.db, "series", id,
		`UPDATE series SET file_size = ?, updated_at = ? WHERE id = ?`,
		fileSize, time.Now(), id)
}

// UpdateParseStatus queues or settles a parse without touching anything
// else. (BatchReparse)
func (r *SeriesRepository) UpdateParseStatus(ctx context.Context, id string, status models.ParseStatus) error {
	return execNarrow(ctx, r.db, "series", id,
		`UPDATE series SET parse_status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id)
}

// UpdatePosterPath stores a new poster location; "" clears to NULL.
func (r *SeriesRepository) UpdatePosterPath(ctx context.Context, id, posterPath string) error {
	return execNarrow(ctx, r.db, "series", id,
		`UPDATE series SET poster_path = ?, updated_at = ? WHERE id = ?`,
		models.NewNullString(posterPath), time.Now(), id)
}
