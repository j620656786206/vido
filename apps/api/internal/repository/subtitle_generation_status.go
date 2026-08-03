package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// updateSubtitleGenerationStatus is the shared body behind the movie and series
// generation-status writers (story sub-1-6, closing
// `backlog-subtitle-status-writer-search-columns`).
//
// It writes subtitle_status / subtitle_path / subtitle_language and NOTHING
// else. The two search columns — `subtitle_search_score` and
// `subtitle_last_searched` — are deliberately absent from the SET list: a
// subtitle produced by extract-and-translate was never scored against provider
// results and never searched for, and stamping them would make it
// indistinguishable from a provider hit in the library UI (sub-1-5b AC #6.3).
//
// The table name is an internal constant supplied by the two callers, never
// user input.
func updateSubtitleGenerationStatus(
	ctx context.Context,
	db *sql.DB,
	table, id string,
	status models.SubtitleStatus,
	path, language string,
) error {
	if id == "" {
		return fmt.Errorf("%s id cannot be empty", table)
	}

	query := `
		UPDATE ` + table + `
		SET subtitle_status = ?, subtitle_path = ?, subtitle_language = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := db.ExecContext(ctx, query,
		string(status),
		sql.NullString{String: path, Valid: path != ""},
		sql.NullString{String: language, Valid: language != ""},
		time.Now(),
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update %s subtitle generation status: %w", table, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// A silent zero-rows update would let the pipeline believe it recorded
		// a status it never wrote — and the run row would then claim a
		// provenance the media row does not reflect.
		return fmt.Errorf("%s with id %s not found", table, id)
	}

	return nil
}
