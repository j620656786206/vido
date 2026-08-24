package repository

// bugfix-series-popularity-never-persisted — popularity must round-trip.
//
// Before this story: series never persisted it at all, and movies WROTE it
// (wide Update, UpdateEnrichedMetadata) but never SELECTed it back — so any
// load-then-wide-update zeroed the value enrichment had just written. The
// same lost-write family as #263–#267, one column at a time.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// AC #1: create → read → wide-update → read keeps the value; and the motivating
// regression: enrichment writes popularity, a wide Update from a freshly loaded
// model must NOT zero it.
func TestMoviePopularity_RoundTripsAndSurvivesWideUpdate(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewMovieRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Movie{
		ID: "mv-pop", Title: "熱門片", Popularity: models.NewNullFloat64(123.4),
		VoteCount: models.NewNullInt64(4321),
	}))
	got, err := repo.FindByID(ctx, "mv-pop")
	require.NoError(t, err)
	assert.Equal(t, 123.4, got.Popularity.Float64, "Create must persist popularity and FindByID must scan it back")
	// Review observation, absorbed (Rule 24 lane ①): movie Create/BulkCreate
	// also omitted vote_count — the converter sets it, so a first insert lost
	// it until the next enrichment. Same family, two-line fix.
	assert.Equal(t, int64(4321), got.VoteCount.Int64, "Create must persist vote_count too")

	// Enrichment refreshes it…
	got.Popularity = models.NewNullFloat64(456.7)
	require.NoError(t, repo.UpdateEnrichedMetadata(ctx, got))

	// …then a user metadata edit loads the row and wide-updates it.
	loaded, err := repo.FindByID(ctx, "mv-pop")
	require.NoError(t, err)
	loaded.Title = "改名了"
	require.NoError(t, repo.Update(ctx, loaded))

	final, err := repo.FindByID(ctx, "mv-pop")
	require.NoError(t, err)
	assert.Equal(t, 456.7, final.Popularity.Float64,
		"a load-then-wide-update must not zero the popularity enrichment just wrote — the SELECT list omission this story fixes")

	// BulkCreate (the scanner's insert path) persists it too.
	require.NoError(t, repo.BulkCreate(ctx, []*models.Movie{{
		ID: "mv-pop-bulk", Title: "批次", Popularity: models.NewNullFloat64(7.7),
	}}))
	bulk, err := repo.FindByID(ctx, "mv-pop-bulk")
	require.NoError(t, err)
	assert.Equal(t, 7.7, bulk.Popularity.Float64)
}

// AC #2: series round-trip via Create, wide Update, and the Upsert re-match
// refresh (popularity is TMDb-owned under #269's ownership contract).
func TestSeriesPopularity_RoundTripsAndUpsertRefreshes(t *testing.T) {
	db := setupSeriesTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Series{
		ID: "sr-pop", Title: "熱門劇", TMDbID: models.NewNullInt64(1399), Popularity: models.NewNullFloat64(88.8),
	}))
	got, err := repo.FindByID(ctx, "sr-pop")
	require.NoError(t, err)
	assert.Equal(t, 88.8, got.Popularity.Float64)

	got.Popularity = models.NewNullFloat64(99.9)
	require.NoError(t, repo.Update(ctx, got))
	got, err = repo.FindByID(ctx, "sr-pop")
	require.NoError(t, err)
	assert.Equal(t, 99.9, got.Popularity.Float64)

	// Re-match: the fresh TMDb model carries the new popularity — it refreshes.
	require.NoError(t, repo.Upsert(ctx, &models.Series{
		Title: "熱門劇", TMDbID: models.NewNullInt64(1399), ParseStatus: models.ParseStatusSuccess,
		Popularity: models.NewNullFloat64(111.1),
	}))
	got, err = repo.FindByID(ctx, "sr-pop")
	require.NoError(t, err)
	assert.Equal(t, 111.1, got.Popularity.Float64, "popularity is TMDb-owned: a re-match refreshes, never preserves")
}
