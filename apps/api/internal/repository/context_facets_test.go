package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// disc-2026-10-production-countries-never-filled. All of these run on the
// migrated schema (setupGlossaryDB runs every migration), because the bug this
// story fixes was invisible to tests that only looked at the in-memory model.

type hasCountries interface {
	GetProductionCountries() ([]models.ProductionCountry, error)
}

func countriesOf(t *testing.T, row hasCountries) []string {
	t.Helper()
	c, err := row.GetProductionCountries()
	require.NoError(t, err)
	out := make([]string, 0, len(c))
	for _, x := range c {
		out = append(out, x.ISO3166_1)
	}
	return out
}

// AC #1: the enrichment writer leaves the column alone, so the match path's
// own writer is what must reach the database — and survive a later
// UpdateEnrichedMetadata.
func TestUpdateProductionCountries_ReachesTheColumnAndSurvivesEnrichmentWrites(t *testing.T) {
	db := setupGlossaryDB(t)
	ctx := context.Background()
	movies := NewMovieRepository(db)
	series := NewSeriesRepository(db)

	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "m1", Title: "長安的荔枝", TMDbID: models.NewNullInt64(1)}))
	var m models.Movie
	require.NoError(t, m.SetProductionCountries([]models.ProductionCountry{{ISO3166_1: "CN", Name: "China"}}))
	require.NoError(t, movies.UpdateProductionCountries(ctx, "m1", m.ProductionCountriesJSON))

	got, err := movies.FindByID(ctx, "m1")
	require.NoError(t, err)
	assert.Equal(t, []string{"CN"}, countriesOf(t, got))

	got.Title = "長安的荔枝（重新比對）"
	require.NoError(t, movies.UpdateEnrichedMetadata(ctx, got))
	again, err := movies.FindByID(ctx, "m1")
	require.NoError(t, err)
	assert.Equal(t, []string{"CN"}, countriesOf(t, again), "the enrichment writer must not clear it")

	require.NoError(t, series.Create(ctx, &models.Series{ID: "s1", Title: "凡人修仙傳", TMDbID: models.NewNullInt64(106449)}))
	var s models.Series
	require.NoError(t, s.SetProductionCountries([]models.ProductionCountry{{ISO3166_1: "CN"}}))
	require.NoError(t, series.UpdateProductionCountries(ctx, "s1", s.ProductionCountriesJSON))
	gotS, err := series.FindByID(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, []string{"CN"}, countriesOf(t, gotS))
}

// AC #2: NULL ("TMDb gave none this time") keeps the stored answer; a value
// replaces it (a re-match is the authority); unknown ids are errors.
func TestUpdateProductionCountries_NullKeepsValueReplaces(t *testing.T) {
	db := setupGlossaryDB(t)
	ctx := context.Background()
	movies := NewMovieRepository(db)

	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "m1", Title: "A", TMDbID: models.NewNullInt64(1)}))
	require.NoError(t, movies.UpdateProductionCountries(ctx, "m1", models.NewNullString(`[{"iso_3166_1":"HK","name":"Hong Kong"}]`)))
	require.NoError(t, movies.UpdateProductionCountries(ctx, "m1", models.NullString{}))
	got, err := movies.FindByID(ctx, "m1")
	require.NoError(t, err)
	assert.Equal(t, []string{"HK"}, countriesOf(t, got))

	require.NoError(t, movies.UpdateProductionCountries(ctx, "m1", models.NewNullString(`[{"iso_3166_1":"US","name":"United States of America"}]`)))
	got, err = movies.FindByID(ctx, "m1")
	require.NoError(t, err)
	assert.Equal(t, []string{"US"}, countriesOf(t, got))

	assert.Error(t, movies.UpdateProductionCountries(ctx, "nope", models.NullString{}))
	assert.Error(t, movies.UpdateProductionCountries(ctx, "", models.NullString{}))
}

// AC #3: the backfill writer fills only empty columns, one column at a time,
// never overwrites, and leaves updated_at alone.
func TestFillContextFacets_FillsOnlyEmptyColumns(t *testing.T) {
	db := setupGlossaryDB(t)
	ctx := context.Background()
	movies := NewMovieRepository(db)
	series := NewSeriesRepository(db)
	cn := models.NewNullString(`[{"iso_3166_1":"CN","name":"China"}]`)

	// Both empty → both filled.
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "both", Title: "A", TMDbID: models.NewNullInt64(1)}))
	before, err := movies.FindByID(ctx, "both")
	require.NoError(t, err)
	written, err := movies.FillContextFacets(ctx, "both", `["劇情"]`, cn)
	require.NoError(t, err)
	assert.True(t, written)
	after, err := movies.FindByID(ctx, "both")
	require.NoError(t, err)
	assert.Equal(t, []string{"劇情"}, after.Genres)
	assert.Equal(t, []string{"CN"}, countriesOf(t, after))
	assert.Equal(t, before.UpdatedAt, after.UpdatedAt, "a background fill is not an edit of the title")

	// Genres already set (user or match) → only countries filled; genres untouched.
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "has-genres", Title: "B", TMDbID: models.NewNullInt64(2), Genres: []string{"動作"}}))
	written, err = movies.FillContextFacets(ctx, "has-genres", `["劇情"]`, cn)
	require.NoError(t, err)
	assert.True(t, written)
	got, err := movies.FindByID(ctx, "has-genres")
	require.NoError(t, err)
	assert.Equal(t, []string{"動作"}, got.Genres)
	assert.Equal(t, []string{"CN"}, countriesOf(t, got))

	// Countries already set → not overwritten; nothing else to fill → written=false.
	written, err = movies.FillContextFacets(ctx, "has-genres", `["劇情"]`, models.NewNullString(`[{"iso_3166_1":"US"}]`))
	require.NoError(t, err)
	assert.False(t, written)
	got, err = movies.FindByID(ctx, "has-genres")
	require.NoError(t, err)
	assert.Equal(t, []string{"CN"}, countriesOf(t, got))

	// Nothing given → no write.
	require.NoError(t, series.Create(ctx, &models.Series{ID: "s", Title: "C", TMDbID: models.NewNullInt64(3)}))
	written, err = series.FillContextFacets(ctx, "s", "[]", models.NullString{})
	require.NoError(t, err)
	assert.False(t, written)

	// Series: only genres given → genres filled, countries stay NULL.
	written, err = series.FillContextFacets(ctx, "s", `["動畫"]`, models.NullString{})
	require.NoError(t, err)
	assert.True(t, written)
	gotS, err := series.FindByID(ctx, "s")
	require.NoError(t, err)
	assert.Equal(t, []string{"動畫"}, gotS.Genres)
	assert.False(t, gotS.ProductionCountriesJSON.Valid)

	_, err = movies.FillContextFacets(ctx, "", `["x"]`, cn)
	assert.Error(t, err)

	// CR M1: a user-owned row's empty genre list may be deliberate — only its
	// countries are filled.
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "manual", Title: "M", TMDbID: models.NewNullInt64(9),
		MetadataSource: models.NewNullString(string(models.MetadataSourceManual))}))
	written, err = movies.FillContextFacets(ctx, "manual", `["劇情"]`, cn)
	require.NoError(t, err)
	assert.True(t, written)
	gotM, err := movies.FindByID(ctx, "manual")
	require.NoError(t, err)
	assert.Empty(t, gotM.Genres)
	assert.Equal(t, []string{"CN"}, countriesOf(t, gotM))
	written, err = movies.FillContextFacets(ctx, "manual", `["劇情"]`, models.NullString{})
	require.NoError(t, err)
	assert.False(t, written, "nothing fillable left on a user-owned row")
}

func TestFindMissingContextFacets_ListsMatchedRowsMissingEitherFacet(t *testing.T) {
	db := setupGlossaryDB(t)
	ctx := context.Background()
	movies := NewMovieRepository(db)
	series := NewSeriesRepository(db)

	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "a-both-empty", Title: "A", TMDbID: models.NewNullInt64(1)}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "b-no-countries", Title: "B", TMDbID: models.NewNullInt64(2), Genres: []string{"劇情"}}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "c-complete", Title: "C", TMDbID: models.NewNullInt64(3), Genres: []string{"劇情"}}))
	require.NoError(t, movies.UpdateProductionCountries(ctx, "c-complete", models.NewNullString(`[{"iso_3166_1":"US"}]`)))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "d-no-genres", Title: "D", TMDbID: models.NewNullInt64(4)}))
	require.NoError(t, movies.UpdateProductionCountries(ctx, "d-no-genres", models.NewNullString(`[{"iso_3166_1":"US"}]`)))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "e-unmatched", Title: "芳華"}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "f-removed", Title: "F", TMDbID: models.NewNullInt64(6), IsRemoved: true}))
	manual := models.NewNullString(string(models.MetadataSourceManual))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "g-manual-empty-genres", Title: "G", TMDbID: models.NewNullInt64(7), MetadataSource: manual}))
	require.NoError(t, movies.UpdateProductionCountries(ctx, "g-manual-empty-genres", models.NewNullString(`[{"iso_3166_1":"US"}]`)))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "h-manual-no-countries", Title: "H", TMDbID: models.NewNullInt64(8), MetadataSource: manual}))

	got, err := movies.FindMissingContextFacets(ctx, "", 50)
	require.NoError(t, err)
	ids := []string{}
	for _, m := range got {
		ids = append(ids, m.ID)
	}
	assert.Equal(t, []string{"a-both-empty", "b-no-countries", "d-no-genres", "h-manual-no-countries"}, ids,
		"a manual row is listed for its countries, never for its genres")

	// Paged by id.
	page2, err := movies.FindMissingContextFacets(ctx, "a-both-empty", 1)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, "b-no-countries", page2[0].ID)

	require.NoError(t, series.Create(ctx, &models.Series{ID: "see", Title: "末日光明", TMDbID: models.NewNullInt64(80752)}))
	require.NoError(t, series.Create(ctx, &models.Series{ID: "unmatched", Title: "X"}))
	gotS, err := series.FindMissingContextFacets(ctx, "", 50)
	require.NoError(t, err)
	require.Len(t, gotS, 1)
	assert.Equal(t, "see", gotS[0].ID)

	_, err = series.FillContextFacets(ctx, "see", `["劇情"]`, models.NewNullString(`[{"iso_3166_1":"US"}]`))
	require.NoError(t, err)
	gotS, err = series.FindMissingContextFacets(ctx, "", 50)
	require.NoError(t, err)
	assert.Empty(t, gotS, "once filled, the row leaves the list")
}
