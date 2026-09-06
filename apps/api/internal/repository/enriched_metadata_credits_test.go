package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// sub-7-3: the credits-only writer must actually reach the column — the
// enrichment writer deliberately excludes `credits`, and an in-memory
// SetCredits alone never lands (CR finding on the first cut of this story).
func TestMovieRepository_UpdateCredits_RoundTripsThroughSQLite(t *testing.T) {
	db := setupGlossaryDB(t)
	repo := NewMovieRepository(db)
	ctx := context.Background()

	movie := &models.Movie{ID: "m-credits", Title: "Fight Club", FilePath: models.NewNullString("/x/fc.mkv")}
	require.NoError(t, repo.Create(ctx, movie))

	credits := &models.Credits{
		Cast: []models.CastMember{{ID: 819, Name: "愛德華·諾頓", Character: "The Narrator", Order: 0}},
		Crew: []models.CrewMember{{ID: 7467, Name: "David Fincher", Job: "Director", Department: "Directing"}},
	}
	require.NoError(t, repo.UpdateCredits(ctx, "m-credits", credits))

	got, err := repo.FindByID(ctx, "m-credits")
	require.NoError(t, err)
	require.NotNil(t, got.Credits, "scanMovie exposes non-empty credits")
	require.Len(t, got.Credits.Cast, 1)
	assert.Equal(t, "愛德華·諾頓", got.Credits.Cast[0].Name)
	assert.Equal(t, "Director", got.Credits.Crew[0].Job)

	// The enrichment writer must not clobber it afterwards (it does not touch credits).
	got.Title = "鬥陣俱樂部"
	require.NoError(t, repo.UpdateEnrichedMetadata(ctx, got))
	again, err := repo.FindByID(ctx, "m-credits")
	require.NoError(t, err)
	require.NotNil(t, again.Credits)
	assert.Equal(t, "愛德華·諾頓", again.Credits.Cast[0].Name)

	// nil clears.
	require.NoError(t, repo.UpdateCredits(ctx, "m-credits", nil))
	cleared, err := repo.FindByID(ctx, "m-credits")
	require.NoError(t, err)
	assert.Nil(t, cleared.Credits)

	// Unknown id is an error, not a silent no-op.
	assert.Error(t, repo.UpdateCredits(ctx, "nope", credits))
}

func TestSeriesRepository_UpdateCredits_RoundTripsThroughSQLite(t *testing.T) {
	db := setupGlossaryDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	series := &models.Series{ID: "s-credits", Title: "Breaking Bad"}
	require.NoError(t, repo.Create(ctx, series))

	credits := &models.Credits{Cast: []models.CastMember{{ID: 17419, Name: "布萊恩·克蘭斯頓", Character: "華特·懷特 / 海森堡"}}}
	require.NoError(t, repo.UpdateCredits(ctx, "s-credits", credits))

	got, err := repo.FindByID(ctx, "s-credits")
	require.NoError(t, err)
	require.NotNil(t, got.Credits)
	assert.Equal(t, "華特·懷特 / 海森堡", got.Credits.Cast[0].Character)

	assert.Error(t, repo.UpdateCredits(ctx, "", credits))
}

func TestGlossaryRepository_HasSourceInScope(t *testing.T) {
	db := setupGlossaryDB(t)
	repo := NewGlossaryRepository(db)
	ctx := context.Background()

	has, err := repo.HasSourceInScope(ctx, "tmdb:tv:1396", models.GlossarySourceMetadata)
	require.NoError(t, err)
	assert.False(t, has, "empty drawer")

	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "s1", Scope: "tmdb:tv:1396", TermSrc: "Walter White", TermZh: "老白", Source: models.GlossarySourceSubtitle}))
	has, err = repo.HasSourceInScope(ctx, "tmdb:tv:1396", models.GlossarySourceMetadata)
	require.NoError(t, err)
	assert.False(t, has, "a subtitle-harvested term is not a seed")

	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "s1", Scope: "tmdb:tv:1396", TermSrc: "Jesse", TermZh: "傑西", Source: models.GlossarySourceMetadata}))
	has, err = repo.HasSourceInScope(ctx, "tmdb:tv:1396", models.GlossarySourceMetadata)
	require.NoError(t, err)
	assert.True(t, has)

	has, err = repo.HasSourceInScope(ctx, "tmdb:tv:1", models.GlossarySourceMetadata)
	require.NoError(t, err)
	assert.False(t, has, "scope is exact, not a prefix")
}
