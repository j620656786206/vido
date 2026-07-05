package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
)

// ErrGlossaryTermNotFound is returned when a glossary term lookup finds no row.
var ErrGlossaryTermNotFound = errors.New("glossary term not found")

// GlossaryRepositoryInterface defines glossary data access (Story 9R-6).
type GlossaryRepositoryInterface interface {
	// Upsert inserts a term or, on the (media_id, term_src, language) unique
	// conflict, updates the existing rendering/source/confirmed flag. Used both
	// by the generation pipeline (auto-mined terms) and manual edits.
	Upsert(ctx context.Context, term *models.GlossaryTerm) error
	// ListByMedia returns all terms for a media id, term_src ascending.
	ListByMedia(ctx context.Context, mediaID string) ([]models.GlossaryTerm, error)
	// LookupByMedia returns a term_src→term_zh map for a media id — the shape
	// the translation service injects into prompts (9R-7). Only CONFIRMED terms
	// are returned when confirmedOnly is true.
	LookupByMedia(ctx context.Context, mediaID string, confirmedOnly bool) (map[string]string, error)
	// Update changes the rendering/confirmed flag of an existing term by id.
	Update(ctx context.Context, id, termZh string, confirmed bool) (time.Time, error)
	// Confirm marks a term confirmed by id (F6 review action).
	Confirm(ctx context.Context, id string) (time.Time, error)
	// Delete removes a term by id.
	Delete(ctx context.Context, id string) error
}

// GlossaryRepository provides SQLite data access for per-show glossary terms.
type GlossaryRepository struct {
	db *sql.DB
}

// NewGlossaryRepository creates a new GlossaryRepository.
func NewGlossaryRepository(db *sql.DB) *GlossaryRepository {
	return &GlossaryRepository{db: db}
}

// Compile-time interface verification.
var _ GlossaryRepositoryInterface = (*GlossaryRepository)(nil)

// glossaryColumns keeps INSERT/SELECT/scan in sync (Rule 15 DB Column Sync).
const glossaryColumns = `id, media_id, term_src, term_zh, language, source, confirmed, created_at, updated_at`

func scanGlossaryTerm(scanner interface{ Scan(dest ...any) error }) (models.GlossaryTerm, error) {
	var g models.GlossaryTerm
	err := scanner.Scan(
		&g.ID, &g.MediaID, &g.TermSrc, &g.TermZh, &g.Language,
		&g.Source, &g.Confirmed, &g.CreatedAt, &g.UpdatedAt,
	)
	return g, err
}

func (r *GlossaryRepository) Upsert(ctx context.Context, term *models.GlossaryTerm) error {
	if term == nil {
		return fmt.Errorf("glossary term cannot be nil")
	}
	if err := term.Validate(); err != nil {
		return err
	}
	if term.ID == "" {
		term.ID = uuid.New().String()
	}
	if term.Language == "" {
		term.Language = models.GlossaryDefaultLanguage
	}
	if term.Source == "" {
		term.Source = models.GlossarySourceManual
	}
	now := time.Now()
	term.CreatedAt = now
	term.UpdatedAt = now

	// ON CONFLICT keeps the first row's id/created_at, updates the mutable
	// fields. A re-mined term therefore refreshes its rendering without a
	// duplicate; a manual edit that races an auto-mine last-writer-wins.
	query := `INSERT INTO show_glossary (` + glossaryColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(media_id, term_src, language) DO UPDATE SET
			term_zh = excluded.term_zh,
			source = excluded.source,
			confirmed = excluded.confirmed,
			updated_at = excluded.updated_at`
	_, err := r.db.ExecContext(ctx, query,
		term.ID, term.MediaID, term.TermSrc, term.TermZh, term.Language,
		term.Source, term.Confirmed, term.CreatedAt, term.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert glossary term: %w", err)
	}
	return nil
}

func (r *GlossaryRepository) ListByMedia(ctx context.Context, mediaID string) ([]models.GlossaryTerm, error) {
	query := `SELECT ` + glossaryColumns + ` FROM show_glossary WHERE media_id = ? ORDER BY term_src ASC`
	rows, err := r.db.QueryContext(ctx, query, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to list glossary terms: %w", err)
	}
	defer rows.Close()

	var terms []models.GlossaryTerm
	for rows.Next() {
		t, err := scanGlossaryTerm(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan glossary term: %w", err)
		}
		terms = append(terms, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating glossary terms: %w", err)
	}
	return terms, nil
}

func (r *GlossaryRepository) LookupByMedia(ctx context.Context, mediaID string, confirmedOnly bool) (map[string]string, error) {
	query := `SELECT term_src, term_zh FROM show_glossary WHERE media_id = ?`
	if confirmedOnly {
		query += ` AND confirmed = 1`
	}
	rows, err := r.db.QueryContext(ctx, query, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up glossary: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var src, zh string
		if err := rows.Scan(&src, &zh); err != nil {
			return nil, fmt.Errorf("failed to scan glossary lookup: %w", err)
		}
		out[src] = zh
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating glossary lookup: %w", err)
	}
	return out, nil
}

func (r *GlossaryRepository) Update(ctx context.Context, id, termZh string, confirmed bool) (time.Time, error) {
	if strings.TrimSpace(termZh) == "" {
		return time.Time{}, &models.ValidationError{Field: "term_zh", Message: "term_zh is required"}
	}
	now := time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE show_glossary SET term_zh = ?, confirmed = ?, updated_at = ? WHERE id = ?`,
		termZh, confirmed, now, id)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to update glossary term: %w", err)
	}
	return now, affectedOrNotFound(res, id)
}

func (r *GlossaryRepository) Confirm(ctx context.Context, id string) (time.Time, error) {
	now := time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE show_glossary SET confirmed = 1, updated_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to confirm glossary term: %w", err)
	}
	return now, affectedOrNotFound(res, id)
}

func (r *GlossaryRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM show_glossary WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete glossary term: %w", err)
	}
	return affectedOrNotFound(res, id)
}

// affectedOrNotFound maps a zero-rows result to ErrGlossaryTermNotFound.
func affectedOrNotFound(res sql.Result, id string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read glossary update result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("glossary term %s: %w", id, ErrGlossaryTermNotFound)
	}
	return nil
}
