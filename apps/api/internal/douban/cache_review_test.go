package douban

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReviewCache(t *testing.T) (*Cache, func()) {
	t.Helper()
	db := setupTestDB(t)
	config := DefaultCacheConfig()
	config.CleanupInterval = 0 // disable background cleanup in tests
	cache := NewCache(db, config, nil)
	return cache, func() {
		cache.Close()
		db.Close()
	}
}

// TestCache_SetAndGetReviewSummary round-trips a review summary through the
// review_summary_json column (Story 12-6 Task 2.3 / AC #6).
func TestCache_SetAndGetReviewSummary(t *testing.T) {
	cache, cleanup := newReviewCache(t)
	defer cleanup()
	ctx := context.Background()

	summary := &ReviewSummaryResult{
		ID:            "1292052",
		TotalComments: 152340,
		TopComments: []ReviewComment{
			{Author: "甲", Rating: 5, Text: "這部電影太棒了"},
			{Author: "乙", Rating: 4, Text: "敘事流暢"},
		},
	}
	require.NoError(t, cache.SetReviewSummary(ctx, "1292052", summary))

	got, err := cache.GetReviewSummary(ctx, "1292052")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "1292052", got.ID)
	assert.Equal(t, 152340, got.TotalComments)
	require.Len(t, got.TopComments, 2)
	assert.Equal(t, "甲", got.TopComments[0].Author)
	assert.Equal(t, 5, got.TopComments[0].Rating)
	assert.Equal(t, "這部電影太棒了", got.TopComments[0].Text)
}

// TestCache_GetReviewSummary_Miss returns (nil, nil) for an unknown subject.
func TestCache_GetReviewSummary_Miss(t *testing.T) {
	cache, cleanup := newReviewCache(t)
	defer cleanup()

	got, err := cache.GetReviewSummary(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestCache_GetReviewSummary_Expired treats an expired row as a miss.
func TestCache_GetReviewSummary_Expired(t *testing.T) {
	cache, cleanup := newReviewCache(t)
	defer cleanup()
	ctx := context.Background()

	require.NoError(t, cache.SetReviewSummary(ctx, "expired", &ReviewSummaryResult{
		ID: "expired", TotalComments: 1, TopComments: []ReviewComment{{Author: "甲", Rating: 5, Text: "好"}},
	}))
	// Force the row to be expired, using SQLite's datetime() so the stored text
	// format matches CURRENT_TIMESTAMP (mirrors TestCache_GetExpired).
	_, err := cache.db.ExecContext(ctx, `UPDATE douban_cache SET expires_at = datetime('now', '-1 hour') WHERE douban_id = 'expired'`)
	require.NoError(t, err)

	got, err := cache.GetReviewSummary(ctx, "expired")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestCache_ReviewSummaryDoesNotClobberDetail confirms the two writers share the
// douban_id row without overwriting each other (AC #6) — a detail re-scrape keeps
// the review summary, and the review-summary write keeps the detail.
func TestCache_ReviewSummaryDoesNotClobberDetail(t *testing.T) {
	cache, cleanup := newReviewCache(t)
	defer cleanup()
	ctx := context.Background()

	require.NoError(t, cache.SetReviewSummary(ctx, "1292052", &ReviewSummaryResult{
		ID: "1292052", TotalComments: 10, TopComments: []ReviewComment{{Author: "甲", Rating: 5, Text: "好"}},
	}))
	require.NoError(t, cache.Set(ctx, &DetailResult{
		ID: "1292052", Title: "肖申克的救赎", Type: MediaTypeMovie, ScrapedAt: time.Now(),
	}))

	// Detail row is readable...
	detail, err := cache.Get(ctx, "1292052")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "肖申克的救赎", detail.Title)

	// ...and the review summary survived the detail write.
	rs, err := cache.GetReviewSummary(ctx, "1292052")
	require.NoError(t, err)
	require.NotNil(t, rs)
	assert.Equal(t, 10, rs.TotalComments)
	require.Len(t, rs.TopComments, 1)
	assert.Equal(t, "甲", rs.TopComments[0].Author)
}
