package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
)

// ErrSubtitleRunNotFound is returned when a subtitle-run lookup by id finds no
// row. FindCompletedRun deliberately does NOT use it — see its doc comment.
var ErrSubtitleRunNotFound = errors.New("subtitle run not found")

// SubtitleRunRepositoryInterface defines item-grain provenance data access
// (Story sub-1-2, architecture D2). Cue-grain identity is NOT stored here; it
// lives in the tiered cache keyed by hash(cue) + RunVersion (D4, sub-1-5b).
type SubtitleRunRepositoryInterface interface {
	// Create inserts a run, assigning an id and started_at when unset.
	Create(ctx context.Context, run *models.SubtitleRun) error
	// Update overwrites every mutable column of an existing run by id.
	Update(ctx context.Context, run *models.SubtitleRun) error
	// FindByID returns one run, or ErrSubtitleRunNotFound.
	FindByID(ctx context.Context, id string) (*models.SubtitleRun, error)
	// FindCompletedRun is the RESUME PREDICATE: the most recent 'completed' run
	// for this media whose version tuple matches. A prompt or model bump yields
	// no match, so a pilot re-run is never silently skipped. Returns (nil, nil)
	// when there is none — absence is the normal case, not an error.
	FindCompletedRun(ctx context.Context, mediaID, mediaType string, v models.RunVersion) (*models.SubtitleRun, error)
	// ListByStatus returns runs in the given state, newest first. limit <= 0
	// means no limit.
	ListByStatus(ctx context.Context, status models.SubtitleRunStatus, limit int) ([]models.SubtitleRun, error)
	// CountByStatus counts runs in the given state (ux3-1-6 attention cell).
	CountByStatus(ctx context.Context, status models.SubtitleRunStatus) (int, error)
	// CompletedMediaRefsSince returns the distinct (media_id, media_type) pairs
	// of runs completed at/after `since` (ux3-1-6 processed-today cell). The
	// day window bounds the result set — no LIMIT by design (tech-spec D2).
	CompletedMediaRefsSince(ctx context.Context, since time.Time) ([]SubtitleRunMediaRef, error)
	// LatestWithSpend returns the most recently completed terminal run that has
	// a recorded spend, or (nil, nil) when none exists yet — absence is the
	// normal pre-migration-032 state, not an error (ux3-1-6 spend readout).
	LatestWithSpend(ctx context.Context) (*models.SubtitleRun, error)
}

// SubtitleRunMediaRef is one distinct media identity touched by a run —
// the processed-today dedupe grain (ux3-1-6).
type SubtitleRunMediaRef struct {
	MediaID   string
	MediaType string
}

// SubtitleRunRepository provides SQLite data access for subtitle-run provenance.
type SubtitleRunRepository struct {
	db *sql.DB
}

// NewSubtitleRunRepository creates a new SubtitleRunRepository.
func NewSubtitleRunRepository(db *sql.DB) *SubtitleRunRepository {
	return &SubtitleRunRepository{db: db}
}

// Compile-time interface verification.
var _ SubtitleRunRepositoryInterface = (*SubtitleRunRepository)(nil)

// subtitleRunColumns keeps INSERT/UPDATE/SELECT/scan in sync (Rule 15 DB Column
// Sync). All 19 columns — the 16 of migration 030 in table order, the two
// spend columns of migration 032, and stubborn_count of migration 034. The bugfix-20-1 precedent — series.seasons
// was never added to the select list, so GetSeasons silently returned [] for
// every series — is why this is one constant used everywhere rather than four
// hand-written lists.
const subtitleRunColumns = `id, media_id, media_type, tmdb_id, metadata_hash, glossary_version, ` +
	`prompt_version, model_id, status, source_language, output_path, cue_count, ` +
	`cache_enabled, error_message, started_at, completed_at, spent_usd, budget_usd, stubborn_count`

// subtitleRunInsertPlaceholders matches subtitleRunColumns 1:1 (19 values).
const subtitleRunInsertPlaceholders = `?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?`

// subtitleRunUpdateAssignments covers every column except the id key, so an
// Update can never leave a column stale.
const subtitleRunUpdateAssignments = `media_id = ?, media_type = ?, tmdb_id = ?, metadata_hash = ?, ` +
	`glossary_version = ?, prompt_version = ?, model_id = ?, status = ?, source_language = ?, ` +
	`output_path = ?, cue_count = ?, cache_enabled = ?, error_message = ?, started_at = ?, completed_at = ?, ` +
	`spent_usd = ?, budget_usd = ?, stubborn_count = ?`

// subtitleRunValues returns the 19 column values in subtitleRunColumns order.
// Both time columns are normalized to UTC before storage: the driver stores a
// time.Time as text, and FindCompletedRun / ListByStatus ORDER BY that text —
// a local-time value ("… +0800 CST") would compare by wall-clock digits and
// mis-order against UTC rows (including CURRENT_TIMESTAMP-defaulted ones).
func subtitleRunValues(run *models.SubtitleRun) []any {
	var completedAt *time.Time
	if run.CompletedAt != nil {
		utc := run.CompletedAt.UTC()
		completedAt = &utc
	}
	return []any{
		run.ID, run.MediaID, run.MediaType, run.TMDbID, run.MetadataHash, run.GlossaryVersion,
		run.PromptVersion, run.ModelID, run.Status, run.SourceLanguage, run.OutputPath, run.CueCount,
		run.CacheEnabled, run.ErrorMessage, run.StartedAt.UTC(), completedAt, run.SpentUSD, run.BudgetUSD,
		run.StubbornCount,
	}
}

// scanSubtitleRun reads all 19 columns in subtitleRunColumns order. The four
// nullable TEXT/INTEGER columns go through sql.Null* so a row written by any
// other path (e.g. a bare INSERT) still scans; the nullable columns modelled
// as pointers stay pointers so "unset" survives the round trip.
func scanSubtitleRun(scanner interface{ Scan(dest ...any) error }) (models.SubtitleRun, error) {
	var run models.SubtitleRun
	var sourceLanguage, outputPath, errorMessage sql.NullString
	var cueCount sql.NullInt64

	err := scanner.Scan(
		&run.ID, &run.MediaID, &run.MediaType, &run.TMDbID, &run.MetadataHash, &run.GlossaryVersion,
		&run.PromptVersion, &run.ModelID, &run.Status, &sourceLanguage, &outputPath, &cueCount,
		&run.CacheEnabled, &errorMessage, &run.StartedAt, &run.CompletedAt, &run.SpentUSD, &run.BudgetUSD,
		&run.StubbornCount,
	)
	if err != nil {
		return run, err
	}

	run.SourceLanguage = sourceLanguage.String
	run.OutputPath = outputPath.String
	run.CueCount = int(cueCount.Int64)
	run.ErrorMessage = errorMessage.String
	return run, nil
}

func (r *SubtitleRunRepository) Create(ctx context.Context, run *models.SubtitleRun) error {
	if run == nil {
		return fmt.Errorf("subtitle run cannot be nil")
	}
	if err := run.Validate(); err != nil {
		return err
	}
	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	if run.Status == "" {
		run.Status = models.SubtitleRunPending
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now().UTC()
	}

	query := `INSERT INTO subtitle_runs (` + subtitleRunColumns + `) VALUES (` + subtitleRunInsertPlaceholders + `)`
	if _, err := r.db.ExecContext(ctx, query, subtitleRunValues(run)...); err != nil {
		return fmt.Errorf("failed to create subtitle run: %w", err)
	}
	return nil
}

func (r *SubtitleRunRepository) Update(ctx context.Context, run *models.SubtitleRun) error {
	if run == nil {
		return fmt.Errorf("subtitle run cannot be nil")
	}
	if run.ID == "" {
		return &models.ValidationError{Field: "id", Message: "id is required to update a subtitle run"}
	}
	// Validate() permits an empty status (Create backfills the DB default), but
	// an existing row can never legitimately return to "" — without this guard
	// the empty string reaches the status CHECK and surfaces as a raw driver
	// constraint error, the exact class Validate exists to replace.
	if run.Status == "" {
		return &models.ValidationError{Field: "status", Message: "status is required to update a subtitle run"}
	}
	// Update overwrites all 18 non-id columns, so a sparsely-populated struct
	// would silently zero started_at and corrupt the ORDER BY started_at
	// resume/listing semantics.
	if run.StartedAt.IsZero() {
		return &models.ValidationError{Field: "started_at", Message: "started_at is required to update a subtitle run"}
	}
	if err := run.Validate(); err != nil {
		return err
	}

	// Values are in subtitleRunColumns order; the id moves from the head of the
	// list to the WHERE clause at the tail.
	values := subtitleRunValues(run)
	args := append(values[1:], run.ID)

	res, err := r.db.ExecContext(ctx, `UPDATE subtitle_runs SET `+subtitleRunUpdateAssignments+` WHERE id = ?`, args...)
	if err != nil {
		return fmt.Errorf("failed to update subtitle run: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read subtitle run update result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("subtitle run %s: %w", run.ID, ErrSubtitleRunNotFound)
	}
	return nil
}

func (r *SubtitleRunRepository) FindByID(ctx context.Context, id string) (*models.SubtitleRun, error) {
	query := `SELECT ` + subtitleRunColumns + ` FROM subtitle_runs WHERE id = ?`
	run, err := scanSubtitleRun(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("subtitle run %s: %w", id, ErrSubtitleRunNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find subtitle run: %w", err)
	}
	return &run, nil
}

func (r *SubtitleRunRepository) FindCompletedRun(ctx context.Context, mediaID, mediaType string, v models.RunVersion) (*models.SubtitleRun, error) {
	// All four tuple columns participate. Dropping any one of them would let a
	// re-run with a bumped prompt or model match a stale row and be skipped —
	// exactly the silent-failure trap the M1 pilot instrumentation exists to
	// avoid.
	query := `SELECT ` + subtitleRunColumns + ` FROM subtitle_runs
		WHERE media_id = ? AND media_type = ? AND status = ?
		  AND metadata_hash = ? AND glossary_version = ? AND prompt_version = ? AND model_id = ?
		ORDER BY started_at DESC LIMIT 1`

	run, err := scanSubtitleRun(r.db.QueryRowContext(ctx, query,
		mediaID, mediaType, models.SubtitleRunCompleted,
		v.MetadataHash, v.GlossaryVersion, v.PromptVersion, v.ModelID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		// Intentional swallow: "no prior matching run" is the normal first-run
		// case, and the caller's decision is resume-or-run, not error handling.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find completed subtitle run: %w", err)
	}
	return &run, nil
}

func (r *SubtitleRunRepository) ListByStatus(ctx context.Context, status models.SubtitleRunStatus, limit int) ([]models.SubtitleRun, error) {
	query := `SELECT ` + subtitleRunColumns + ` FROM subtitle_runs WHERE status = ? ORDER BY started_at DESC`
	args := []any{status}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list subtitle runs: %w", err)
	}
	defer rows.Close()

	var runs []models.SubtitleRun
	for rows.Next() {
		run, err := scanSubtitleRun(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subtitle run: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subtitle runs: %w", err)
	}
	return runs, nil
}

func (r *SubtitleRunRepository) CountByStatus(ctx context.Context, status models.SubtitleRunStatus) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subtitle_runs WHERE status = ?`, status).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count subtitle runs by status: %w", err)
	}
	return count, nil
}

// CompletedMediaRefsSince compares lexicographically over the driver's time
// text — correct because this table ALWAYS writes UTC (subtitleRunValues
// normalizes both time columns) and the parameter is UTC-normalized too.
// SQLite's datetime() cannot parse the driver's Go String() format, so a
// datetime() wrapper would return NULL and match nothing.
func (r *SubtitleRunRepository) CompletedMediaRefsSince(ctx context.Context, since time.Time) ([]SubtitleRunMediaRef, error) {
	query := `SELECT DISTINCT media_id, media_type FROM subtitle_runs
		WHERE status = ? AND completed_at IS NOT NULL AND completed_at >= ?`

	rows, err := r.db.QueryContext(ctx, query, models.SubtitleRunCompleted, since.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query completed subtitle runs since: %w", err)
	}
	defer rows.Close()

	var refs []SubtitleRunMediaRef
	for rows.Next() {
		var ref SubtitleRunMediaRef
		if err := rows.Scan(&ref.MediaID, &ref.MediaType); err != nil {
			return nil, fmt.Errorf("failed to scan completed subtitle run ref: %w", err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating completed subtitle run refs: %w", err)
	}
	return refs, nil
}

func (r *SubtitleRunRepository) LatestWithSpend(ctx context.Context) (*models.SubtitleRun, error) {
	query := `SELECT ` + subtitleRunColumns + ` FROM subtitle_runs
		WHERE spent_usd IS NOT NULL AND completed_at IS NOT NULL
		ORDER BY completed_at DESC LIMIT 1`

	run, err := scanSubtitleRun(r.db.QueryRowContext(ctx, query))
	if errors.Is(err, sql.ErrNoRows) {
		// Absence is the normal state until the first post-032 pipeline run.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find latest subtitle run with spend: %w", err)
	}
	return &run, nil
}
