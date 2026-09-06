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
//
// Since sub-7-1 every read/write is keyed by SCOPE (models.GlossaryScope*),
// never by a media id: callers resolve the media id through
// services.GlossaryScopeResolver first. The repository does not know what a
// media id is any more — that is the whole point (eval-1 架構評估).
type GlossaryRepositoryInterface interface {
	// Upsert inserts a term or, on the (scope, term_src NOCASE, language) unique
	// conflict, updates the existing rendering/source/confirmed flag. Used both
	// by the generation pipeline (auto-mined terms) and manual edits.
	Upsert(ctx context.Context, term *models.GlossaryTerm) error
	// InsertIfAbsent inserts a term ONLY when no (scope, term_src NOCASE, language)
	// row exists — ON CONFLICT DO NOTHING (sub-5-5 AC #4, red line 2). The
	// auto-harvest write path MUST use this, never Upsert: Upsert's DO UPDATE
	// would clobber a user-corrected/confirmed rendering back to the machine
	// version and un-confirm it. Returns whether a row was actually inserted.
	InsertIfAbsent(ctx context.Context, term *models.GlossaryTerm) (bool, error)
	// ListByScope returns all terms in a scope, term_src ascending.
	ListByScope(ctx context.Context, scope string) ([]models.GlossaryTerm, error)
	// IsScopeSeeded reports whether the scope carries a seed mark — the
	// durable "has this drawer been seeded from TMDb yet" answer the
	// seed-on-first-resolve seam asks (one primary-key lookup). A mark, not
	// EXISTS on live terms, so a user who deletes every seeded term keeps
	// them deleted across restarts.
	IsScopeSeeded(ctx context.Context, scope string) (bool, error)
	// MarkScopeSeeded records that the scope was seeded (seeded = rows
	// planted, may be 0 for a title with nothing translatable).
	MarkScopeSeeded(ctx context.Context, scope string, seeded int) error
	// LookupByScope returns a term_src→term_zh map for a scope — the shape the
	// translation service injects into prompts (9R-7). Only CONFIRMED terms are
	// returned when confirmedOnly is true.
	LookupByScope(ctx context.Context, scope string, confirmedOnly bool) (map[string]string, error)
	// Update changes the rendering/confirmed flag of an existing term by id.
	Update(ctx context.Context, id, termZh string, confirmed bool) (time.Time, error)
	// Confirm marks a term confirmed by id (F6 review action).
	Confirm(ctx context.Context, id string) (time.Time, error)
	// ConfirmAllByScope marks every unconfirmed term in a scope confirmed in
	// one statement (F6 「全部確認」); returns the number of rows changed.
	ConfirmAllByScope(ctx context.Context, scope string) (int64, error)
	// Delete removes a term by id.
	Delete(ctx context.Context, id string) error
	// MigrateScope moves every term from one scope to another in ONE
	// transaction (sub-7-1 AC #3: a `local:<id>` drawer being upgraded to its
	// `tmdb:*` drawer once the match lands). Insert-if-absent semantics: a term
	// that already exists under `to` (same term_src NOCASE + language) is NOT
	// overwritten — that row stays behind under `from` and is counted in
	// `skipped`, so nothing the user confirmed in the shared drawer is ever
	// clobbered by a stale local one. Returns (moved, skipped).
	MigrateScope(ctx context.Context, from, to string) (moved, skipped int64, err error)
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
const glossaryColumns = `id, media_id, scope, term_src, term_zh, language, source, confirmed, created_at, updated_at`

// glossaryConflictTarget MUST spell the unique index exactly, collation
// included — SQLite refuses an ON CONFLICT target that does not match an
// index, and idx_show_glossary_scope_unique (migration 036) is NOCASE on term_src.
const glossaryConflictTarget = `ON CONFLICT(scope, term_src COLLATE NOCASE, language)`

func scanGlossaryTerm(scanner interface{ Scan(dest ...any) error }) (models.GlossaryTerm, error) {
	var g models.GlossaryTerm
	err := scanner.Scan(
		&g.ID, &g.MediaID, &g.Scope, &g.TermSrc, &g.TermZh, &g.Language,
		&g.Source, &g.Confirmed, &g.CreatedAt, &g.UpdatedAt,
	)
	return g, err
}

// normalizeGlossaryTerm applies the write-side rules every path shares
// (sub-7-1 AC #1): a trimmed term_src, and the id/language/source defaults.
// Case is deliberately NOT folded — the stored spelling is the one the user
// (or the subtitle) wrote; the NOCASE index only decides what counts as the
// same term.
func normalizeGlossaryTerm(term *models.GlossaryTerm) error {
	if term == nil {
		return fmt.Errorf("glossary term cannot be nil")
	}
	term.TermSrc = strings.TrimSpace(term.TermSrc)
	term.Scope = strings.TrimSpace(term.Scope)
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
	return nil
}

func (r *GlossaryRepository) Upsert(ctx context.Context, term *models.GlossaryTerm) error {
	if err := normalizeGlossaryTerm(term); err != nil {
		return err
	}

	// ON CONFLICT keeps the first row's id/created_at/term_src spelling,
	// updates the mutable fields. A re-mined term therefore refreshes its
	// rendering without a duplicate; a manual edit that races an auto-mine
	// last-writer-wins.
	query := `INSERT INTO show_glossary (` + glossaryColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		` + glossaryConflictTarget + ` DO UPDATE SET
			term_zh = excluded.term_zh,
			source = excluded.source,
			confirmed = excluded.confirmed,
			updated_at = excluded.updated_at`
	_, err := r.db.ExecContext(ctx, query,
		term.ID, term.MediaID, term.Scope, term.TermSrc, term.TermZh, term.Language,
		term.Source, term.Confirmed, term.CreatedAt, term.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert glossary term: %w", err)
	}
	return nil
}

// InsertIfAbsent is the auto-harvest write path (sub-5-5 AC #4): insert-only,
// existing rows stay byte-identical. Two concurrent harvests of the same new
// term are naturally safe — the second INSERT hits the UNIQUE conflict and
// does nothing.
func (r *GlossaryRepository) InsertIfAbsent(ctx context.Context, term *models.GlossaryTerm) (bool, error) {
	if err := normalizeGlossaryTerm(term); err != nil {
		return false, err
	}

	query := `INSERT INTO show_glossary (` + glossaryColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		` + glossaryConflictTarget + ` DO NOTHING`
	res, err := r.db.ExecContext(ctx, query,
		term.ID, term.MediaID, term.Scope, term.TermSrc, term.TermZh, term.Language,
		term.Source, term.Confirmed, term.CreatedAt, term.UpdatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("failed to insert glossary term: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read glossary insert result: %w", err)
	}
	return affected > 0, nil
}

func (r *GlossaryRepository) ListByScope(ctx context.Context, scope string) ([]models.GlossaryTerm, error) {
	query := `SELECT ` + glossaryColumns + ` FROM show_glossary WHERE scope = ? ORDER BY term_src ASC`
	rows, err := r.db.QueryContext(ctx, query, scope)
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

func (r *GlossaryRepository) LookupByScope(ctx context.Context, scope string, confirmedOnly bool) (map[string]string, error) {
	query := `SELECT term_src, term_zh FROM show_glossary WHERE scope = ?`
	if confirmedOnly {
		query += ` AND confirmed = 1`
	}
	rows, err := r.db.QueryContext(ctx, query, scope)
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

func (r *GlossaryRepository) ConfirmAllByScope(ctx context.Context, scope string) (int64, error) {
	now := time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE show_glossary SET confirmed = 1, updated_at = ? WHERE scope = ? AND confirmed = 0`,
		now, scope)
	if err != nil {
		return 0, fmt.Errorf("failed to confirm all glossary terms: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read confirm-all result: %w", err)
	}
	return affected, nil
}

// MigrateScope — see the interface doc. One transaction: the UPDATE moves
// only the rows that would not collide in `to`; whatever is left under `from`
// afterwards is the skipped set.
func (r *GlossaryRepository) MigrateScope(ctx context.Context, from, to string) (int64, int64, error) {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" || to == "" || from == to {
		return 0, 0, fmt.Errorf("migrate glossary scope: from %q to %q is not a move", from, to)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin glossary scope migration: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck — no-op after Commit

	now := time.Now()
	res, err := tx.ExecContext(ctx, `
		UPDATE show_glossary SET scope = ?, updated_at = ?
		WHERE scope = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM show_glossary t
		    WHERE t.scope = ?
		      AND t.term_src = show_glossary.term_src COLLATE NOCASE
		      AND t.language = show_glossary.language)`,
		to, now, from, to)
	if err != nil {
		return 0, 0, fmt.Errorf("move glossary terms %s → %s: %w", from, to, err)
	}
	moved, err := res.RowsAffected()
	if err != nil {
		return 0, 0, fmt.Errorf("read glossary scope move result: %w", err)
	}
	var skipped int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM show_glossary WHERE scope = ?`, from).Scan(&skipped); err != nil {
		return 0, 0, fmt.Errorf("count glossary terms left in %s: %w", from, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit glossary scope migration: %w", err)
	}
	return moved, skipped, nil
}

// ListByMedia, LookupByMedia and ConfirmAll are the pre-sub-7-1 names, kept
// for ONE story as delegates so an un-migrated caller still compiles. They
// treat the argument as an UNRESOLVED local id and therefore only ever see the
// `local:<id>` drawer — a caller that wants the shared drawer must resolve
// through services.GlossaryScopeResolver and use the *ByScope methods.
//
// Deprecated: resolve a scope and call ListByScope.
func (r *GlossaryRepository) ListByMedia(ctx context.Context, mediaID string) ([]models.GlossaryTerm, error) {
	return r.ListByScope(ctx, models.GlossaryScopeLocal(mediaID))
}

// Deprecated: resolve a scope and call LookupByScope.
func (r *GlossaryRepository) LookupByMedia(ctx context.Context, mediaID string, confirmedOnly bool) (map[string]string, error) {
	return r.LookupByScope(ctx, models.GlossaryScopeLocal(mediaID), confirmedOnly)
}

// Deprecated: resolve a scope and call ConfirmAllByScope.
func (r *GlossaryRepository) ConfirmAll(ctx context.Context, mediaID string) (int64, error) {
	return r.ConfirmAllByScope(ctx, models.GlossaryScopeLocal(mediaID))
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

// IsScopeSeeded implements GlossaryRepositoryInterface.
func (r *GlossaryRepository) IsScopeSeeded(ctx context.Context, scope string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM glossary_seed_marks WHERE scope = ?)`, strings.TrimSpace(scope),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to read glossary seed mark: %w", err)
	}
	return exists, nil
}

// MarkScopeSeeded implements GlossaryRepositoryInterface.
func (r *GlossaryRepository) MarkScopeSeeded(ctx context.Context, scope string, seeded int) error {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return &models.ValidationError{Field: "scope", Message: "scope is required"}
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO glossary_seed_marks (scope, seeded, seeded_at) VALUES (?, ?, ?)
		 ON CONFLICT(scope) DO UPDATE SET seeded = excluded.seeded, seeded_at = excluded.seeded_at`,
		scope, seeded, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to write glossary seed mark: %w", err)
	}
	return nil
}
