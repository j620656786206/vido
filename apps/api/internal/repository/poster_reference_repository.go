package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// PosterReferenceRepository answers one question for the poster orphan sweep
// (bugfix-poster-orphan-sweep): which user-uploaded posters does any row still
// point at? It reads movies AND series in one place so the sweep never has to
// know how many tables can hold an upload.
type PosterReferenceRepository struct {
	db *sql.DB
}

// NewPosterReferenceRepository creates a PosterReferenceRepository.
func NewPosterReferenceRepository(db *sql.DB) *PosterReferenceRepository {
	return &PosterReferenceRepository{db: db}
}

// ListUploadedPosterPaths returns every stored poster_path that may name an
// upload: "/posters/<id>.jpg" (possibly with "?v=…"), and also an absolute URL
// someone pasted of this app's own poster route (".../api/v1/posters/<id>.jpg")
// — the sweep errs towards keeping. Soft-removed rows (is_removed) are
// deliberately INCLUDED: a restored item must still have its poster.
func (r *PosterReferenceRepository) ListUploadedPosterPaths(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT poster_path FROM movies WHERE poster_path LIKE '%/posters/%'
		UNION ALL
		SELECT poster_path FROM series WHERE poster_path LIKE '%/posters/%'`)
	if err != nil {
		return nil, fmt.Errorf("list uploaded poster paths: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan uploaded poster path: %w", err)
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate uploaded poster paths: %w", err)
	}
	return paths, nil
}
