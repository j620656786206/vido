package repository

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// backlog-ai-parse-reads-env-keys-only-a — the parse-path spend ledger, on
// the app's own driver so the stored time text is the production shape.

func TestAISideSpendRepository_CreateStampsTimeWhenUnset(t *testing.T) {
	db := setupSubtitleRunAppDriverDB(t)
	repo := NewAISideSpendRepository(db)
	ctx := context.Background()

	row := &models.AISideSpend{Kind: models.AISideSpendKindParse, Provider: "claude", ModelID: "claude-sonnet-5",
		InputTokens: 300, OutputTokens: 80, SpentUSD: 0.0021}
	require.NoError(t, repo.Create(ctx, row))
	assert.NotZero(t, row.ID)
	assert.False(t, row.CreatedAt.IsZero())

	var stored string
	require.NoError(t, db.QueryRow(`SELECT CAST(created_at AS TEXT) FROM ai_side_spend WHERE id = ?`, row.ID).Scan(&stored))
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{9}Z$`, stored, "the post-042 UTC text shape")
}

func TestAISideSpendRepository_TotalsBetweenIsHalfOpen(t *testing.T) {
	db := setupSubtitleRunAppDriverDB(t)
	repo := NewAISideSpendRepository(db)
	ctx := context.Background()
	from := time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC) // 2026-10-01 00:00 +08:00
	to := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)

	seed := func(usd float64, at time.Time) {
		require.NoError(t, repo.Create(ctx, &models.AISideSpend{Kind: models.AISideSpendKindParse,
			Provider: "gemini", ModelID: "gemini-2.5-flash-lite", SpentUSD: usd, CreatedAt: at}))
	}
	seed(0.1, from)                       // included (lower bound)
	seed(0.2, from.Add(10*24*time.Hour))  // included
	seed(0.4, from.Add(-time.Nanosecond)) // excluded
	seed(0.8, to)                         // excluded (upper bound)

	totals, err := repo.TotalsBetween(ctx, from, to)
	require.NoError(t, err)
	assert.Equal(t, 2, totals.Calls)
	assert.True(t, totals.Spent.Equal(decimal.RequireFromString("0.3")), "got %s", totals.Spent)
}

func TestAISideSpendRepository_TotalsBetweenEmpty(t *testing.T) {
	repo := NewAISideSpendRepository(setupSubtitleRunAppDriverDB(t))
	totals, err := repo.TotalsBetween(context.Background(), time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.Zero(t, totals.Calls)
	assert.True(t, totals.Spent.IsZero())
}
