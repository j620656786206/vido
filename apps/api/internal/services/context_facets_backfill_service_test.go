package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// disc-2026-10-production-countries-never-filled

type facetsWrite struct {
	genres    string
	countries models.NullString
}

// fillRule mimics the repository: a column is filled only while empty.
type facetsRow struct {
	genresSet, countriesSet bool
}

func fillLikeRepo(rows map[string]*facetsRow, writes map[string]facetsWrite, id, genres string, countries models.NullString) bool {
	writes[id] = facetsWrite{genres: genres, countries: countries}
	r := rows[id]
	if r == nil {
		return false
	}
	wrote := false
	if genres != "[]" && genres != "" && !r.genresSet {
		r.genresSet, wrote = true, true
	}
	if countries.Valid && !r.countriesSet {
		r.countriesSet, wrote = true, true
	}
	return wrote
}

type facetsMovieRepo struct {
	list     []models.Movie
	rows     map[string]*facetsRow
	writes   map[string]facetsWrite
	listErr  error
	afterIDs []string
}

func (r *facetsMovieRepo) FindMissingContextFacets(_ context.Context, after string, limit int) ([]models.Movie, error) {
	r.afterIDs = append(r.afterIDs, after)
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []models.Movie
	for _, m := range r.list {
		if m.ID > after && len(out) < limit {
			out = append(out, m)
		}
	}
	return out, nil
}
func (r *facetsMovieRepo) FillContextFacets(_ context.Context, id, genres string, countries models.NullString) (bool, error) {
	if r.writes == nil {
		r.writes = map[string]facetsWrite{}
	}
	return fillLikeRepo(r.rows, r.writes, id, genres, countries), nil
}

type facetsSeriesRepo struct {
	list     []models.Series
	rows     map[string]*facetsRow
	writes   map[string]facetsWrite
	writeErr error
}

func (r *facetsSeriesRepo) FindMissingContextFacets(_ context.Context, after string, _ int) ([]models.Series, error) {
	var out []models.Series
	for _, s := range r.list {
		if s.ID > after {
			out = append(out, s)
		}
	}
	return out, nil
}
func (r *facetsSeriesRepo) FillContextFacets(_ context.Context, id, genres string, countries models.NullString) (bool, error) {
	if r.writeErr != nil {
		return false, r.writeErr
	}
	if r.writes == nil {
		r.writes = map[string]facetsWrite{}
	}
	return fillLikeRepo(r.rows, r.writes, id, genres, countries), nil
}

type facetsFetcher struct {
	movies map[int]*tmdb.MovieDetails
	tv     map[int]*tmdb.TVShowDetails
	errs   map[int]error
	asked  []int
}

func (f *facetsFetcher) GetMovieDetails(_ context.Context, id int) (*tmdb.MovieDetails, error) {
	f.asked = append(f.asked, id)
	if err := f.errs[id]; err != nil {
		return nil, err
	}
	return f.movies[id], nil
}
func (f *facetsFetcher) GetTVShowDetails(_ context.Context, id int) (*tmdb.TVShowDetails, error) {
	f.asked = append(f.asked, id)
	if err := f.errs[id]; err != nil {
		return nil, err
	}
	return f.tv[id], nil
}

func TestContextFacetsBackfill_FillsCountriesAndGenres(t *testing.T) {
	movies := &facetsMovieRepo{
		list: []models.Movie{
			{ID: "m-lychee", TMDbID: models.NewNullInt64(1)},
			{ID: "m-unmatched"}, // no tmdb id → never asked
		},
		rows: map[string]*facetsRow{"m-lychee": {}},
	}
	series := &facetsSeriesRepo{
		list: []models.Series{{ID: "s-fanren", TMDbID: models.NewNullInt64(106449)}},
		rows: map[string]*facetsRow{"s-fanren": {genresSet: true}}, // user's genres stay
	}
	fetch := &facetsFetcher{
		movies: map[int]*tmdb.MovieDetails{1: {
			Genres:              []tmdb.Genre{{Name: "劇情"}, {Name: "歷史"}},
			ProductionCountries: []tmdb.Country{{ISO31661: "CN", Name: "China"}},
		}},
		// Most CN dramas: empty production_countries, origin_country set.
		tv: map[int]*tmdb.TVShowDetails{106449: {
			TVShow: tmdb.TVShow{OriginCountry: []string{"CN"}},
			Genres: []tmdb.Genre{{Name: "動畫"}},
		}},
	}

	res := NewContextFacetsBackfillService(movies, series, fetch, nil).Run(context.Background())

	assert.Equal(t, ContextFacetsBackfillResult{Movies: 1, Series: 1}, res)
	assert.Equal(t, []int{1, 106449}, fetch.asked)
	assert.Equal(t, `["劇情","歷史"]`, movies.writes["m-lychee"].genres)
	assert.JSONEq(t, `[{"iso_3166_1":"CN","name":"China"}]`, movies.writes["m-lychee"].countries.String)
	assert.JSONEq(t, `[{"iso_3166_1":"CN","name":""}]`, series.writes["s-fanren"].countries.String, "origin_country fallback")
	assert.True(t, series.rows["s-fanren"].countriesSet)
}

// TMDb has nothing for the empty column → asked once per process; a failed
// fetch is retried on the next pass; a failed write counts as failed.
func TestContextFacetsBackfill_EmptyFailedAndRetry(t *testing.T) {
	movies := &facetsMovieRepo{
		list: []models.Movie{{ID: "m-none", TMDbID: models.NewNullInt64(7)}, {ID: "m-flaky", TMDbID: models.NewNullInt64(8)}},
		rows: map[string]*facetsRow{"m-none": {genresSet: true}, "m-flaky": {}},
	}
	series := &facetsSeriesRepo{
		list:     []models.Series{{ID: "s-x", TMDbID: models.NewNullInt64(9)}},
		rows:     map[string]*facetsRow{"s-x": {}},
		writeErr: errors.New("database is locked"),
	}
	fetch := &facetsFetcher{
		movies: map[int]*tmdb.MovieDetails{7: {Genres: []tmdb.Genre{{Name: "劇情"}}}}, // no countries; genres already set
		tv:     map[int]*tmdb.TVShowDetails{9: {Genres: []tmdb.Genre{{Name: "劇情"}}}},
		errs:   map[int]error{8: errors.New("TMDB_RATE_LIMIT")},
	}
	svc := NewContextFacetsBackfillService(movies, series, fetch, nil)

	first := svc.Run(context.Background())
	assert.Equal(t, ContextFacetsBackfillResult{Empty: 1, Failed: 2}, first)

	delete(fetch.errs, 8)
	fetch.movies[8] = &tmdb.MovieDetails{ProductionCountries: []tmdb.Country{{ISO31661: "HK", Name: "Hong Kong"}}}
	series.writeErr = nil
	fetch.asked = nil
	second := svc.Run(context.Background())
	assert.Equal(t, ContextFacetsBackfillResult{Movies: 1, Series: 1}, second)
	assert.Equal(t, []int{8, 9}, fetch.asked, "m-none was tried already and is not asked again")
}

func TestContextFacetsBackfill_PagesAndStopsOnCancel(t *testing.T) {
	list := make([]models.Movie, 0, contextFacetsBatch+3)
	rows := map[string]*facetsRow{}
	fetch := &facetsFetcher{movies: map[int]*tmdb.MovieDetails{}}
	for i := 0; i < contextFacetsBatch+3; i++ {
		id := "m" + string(rune('a'+i/26/26%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i%26))
		list = append(list, models.Movie{ID: id, TMDbID: models.NewNullInt64(int64(i + 1))})
		rows[id] = &facetsRow{}
		fetch.movies[i+1] = &tmdb.MovieDetails{Genres: []tmdb.Genre{{Name: "劇情"}}}
	}
	movies := &facetsMovieRepo{list: list, rows: rows}

	res := NewContextFacetsBackfillService(movies, nil, fetch, nil).Run(context.Background())
	assert.Equal(t, contextFacetsBatch+3, res.Movies)
	require.Len(t, movies.afterIDs, 2)
	assert.Equal(t, list[contextFacetsBatch-1].ID, movies.afterIDs[1])

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fetch.asked = nil
	NewContextFacetsBackfillService(&facetsMovieRepo{list: list, rows: map[string]*facetsRow{}}, nil, fetch, nil).Run(ctx)
	assert.Empty(t, fetch.asked)

	// No fetcher → disabled.
	assert.Equal(t, ContextFacetsBackfillResult{}, NewContextFacetsBackfillService(movies, nil, nil, nil).Run(context.Background()))
	// Listing failure is fail-soft.
	assert.Equal(t, ContextFacetsBackfillResult{Failed: 1},
		NewContextFacetsBackfillService(&facetsMovieRepo{listErr: errors.New("boom")}, nil, fetch, nil).Run(context.Background()))
}

// CR L2: stop cancels a pass that has not started and waits for the goroutine.
func TestContextFacetsBackfill_RunAfterStop(t *testing.T) {
	fetch := &facetsFetcher{}
	movies := &facetsMovieRepo{list: []models.Movie{{ID: "m", TMDbID: models.NewNullInt64(1)}}, rows: map[string]*facetsRow{"m": {}}}
	stop := NewContextFacetsBackfillService(movies, nil, fetch, nil).RunAfter(context.Background(), time.Hour)
	stop() // returns promptly, without waiting an hour
	assert.Empty(t, fetch.asked)
	assert.Empty(t, movies.afterIDs)
}
