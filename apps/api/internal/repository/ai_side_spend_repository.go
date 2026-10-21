package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/vido/api/internal/models"
)

// AISideSpendTotals is one window's parse-path spend: how many paid calls and
// what they cost, summed exactly (decimal) before anyone rounds it.
type AISideSpendTotals struct {
	Calls int
	Spent decimal.Decimal
}

// AISideSpendRepositoryInterface is the ai_side_spend ledger
// (backlog-ai-parse-reads-env-keys-only-a).
type AISideSpendRepositoryInterface interface {
	// Create inserts one row, stamping created_at with now when unset.
	Create(ctx context.Context, row *models.AISideSpend) error
	// TotalsBetween sums the rows whose created_at is in [from, to).
	TotalsBetween(ctx context.Context, from, to time.Time) (AISideSpendTotals, error)
}

// AISideSpendRepository is the SQLite implementation.
type AISideSpendRepository struct {
	db *sql.DB
}

var _ AISideSpendRepositoryInterface = (*AISideSpendRepository)(nil)

// NewAISideSpendRepository builds the repository.
func NewAISideSpendRepository(db *sql.DB) *AISideSpendRepository {
	return &AISideSpendRepository{db: db}
}

// Create inserts one row. created_at is bound as a time.Time so the app's
// driver writes the post-042 UTC text (dbtime.Format), the shape the window
// query compares against.
func (r *AISideSpendRepository) Create(ctx context.Context, row *models.AISideSpend) error {
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO ai_side_spend
		(kind, provider, model_id, input_tokens, output_tokens, spent_usd, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(row.Kind), row.Provider, row.ModelID, row.InputTokens, row.OutputTokens, row.SpentUSD, row.CreatedAt.UTC())
	if err != nil {
		return fmt.Errorf("insert ai side spend: %w", err)
	}
	if id, err := res.LastInsertId(); err == nil {
		row.ID = id
	}
	return nil
}

// TotalsBetween reads the window's rows and sums them in decimal — the same
// exact-arithmetic rule the subtitle ledger follows (a float SUM drifts).
// Lexicographic comparison on the UTC text, both bounds UTC-normalized (the
// CompletedRunsBetween precedent).
func (r *AISideSpendRepository) TotalsBetween(ctx context.Context, from, to time.Time) (AISideSpendTotals, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT spent_usd FROM ai_side_spend WHERE created_at >= ? AND created_at < ?`, from.UTC(), to.UTC())
	if err != nil {
		return AISideSpendTotals{}, fmt.Errorf("read ai side spend: %w", err)
	}
	defer rows.Close()

	totals := AISideSpendTotals{Spent: decimal.Zero}
	for rows.Next() {
		var usd float64
		if err := rows.Scan(&usd); err != nil {
			return AISideSpendTotals{}, fmt.Errorf("scan ai side spend: %w", err)
		}
		totals.Calls++
		totals.Spent = totals.Spent.Add(decimal.NewFromFloat(usd))
	}
	if err := rows.Err(); err != nil {
		return AISideSpendTotals{}, fmt.Errorf("iterate ai side spend: %w", err)
	}
	return totals, nil
}
