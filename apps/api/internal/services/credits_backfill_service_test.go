package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

type backfillMovieRepo struct {
	rows    []models.Movie
	written map[string]*models.Credits
	listErr error
}

func (r *backfillMovieRepo) FindMissingCredits(_ context.Context, _ int) ([]models.Movie, error) {
	return r.rows, r.listErr
}
func (r *backfillMovieRepo) UpdateCredits(_ context.Context, id string, c *models.Credits) error {
	if r.written == nil {
		r.written = map[string]*models.Credits{}
	}
	r.written[id] = c
	return nil
}

type backfillSeriesRepo struct {
	rows     []models.Series
	written  map[string]*models.Credits
	writeErr error
}

func (r *backfillSeriesRepo) FindMissingCredits(_ context.Context, _ int) ([]models.Series, error) {
	return r.rows, nil
}
func (r *backfillSeriesRepo) UpdateCredits(_ context.Context, id string, c *models.Credits) error {
	if r.writeErr != nil {
		return r.writeErr
	}
	if r.written == nil {
		r.written = map[string]*models.Credits{}
	}
	r.written[id] = c
	return nil
}

// backfillFetcher answers per tmdb id; records every ask.
type backfillFetcher struct {
	byID  map[int64]*models.Credits
	err   map[int64]error
	asked []int64
}

func (f *backfillFetcher) FetchCredits(_ context.Context, _ string, tmdbID int64) (*models.Credits, []CastPair, error) {
	f.asked = append(f.asked, tmdbID)
	if err, ok := f.err[tmdbID]; ok {
		return nil, nil, err
	}
	return f.byID[tmdbID], nil, nil
}

func cast(names ...string) *models.Credits {
	c := &models.Credits{}
	for _, n := range names {
		c.Cast = append(c.Cast, models.CastMember{Name: n, Character: n + " role"})
	}
	return c
}

// The NAS shape: every matched row has no cast; one pass fills the ones TMDb
// knows, remembers the ones TMDb has nothing for, and keeps going past a
// failure.
func TestCreditsBackfill_Run(t *testing.T) {
	movies := &backfillMovieRepo{rows: []models.Movie{
		{ID: "mv-1", TMDbID: models.NewNullInt64(101)},
		{ID: "mv-2", TMDbID: models.NewNullInt64(102)}, // TMDb has no cast
		{ID: "mv-3", TMDbID: models.NewNullInt64(103)}, // fetch fails
	}}
	series := &backfillSeriesRepo{rows: []models.Series{
		{ID: "see", TMDbID: models.NewNullInt64(80752)},
	}}
	fetch := &backfillFetcher{
		byID: map[int64]*models.Credits{101: cast("Tom Hanks"), 102: {}, 80752: cast("Jason Momoa", "Sylvia Hoeks")},
		err:  map[int64]error{103: errors.New("tmdb 500")},
	}
	svc := NewCreditsBackfillService(movies, series, fetch, nil)

	res := svc.Run(context.Background())

	assert.Equal(t, CreditsBackfillResult{Movies: 1, Series: 1, Empty: 1, Failed: 1}, res)
	require.Contains(t, movies.written, "mv-1")
	assert.Equal(t, "Tom Hanks", movies.written["mv-1"].Cast[0].Name)
	require.Contains(t, series.written, "see")
	assert.Len(t, series.written["see"].Cast, 2)
	assert.NotContains(t, movies.written, "mv-2", "an empty cast is not written (UpdateCredits would store NULL anyway)")

	// Second pass in the same process: the empty one is not asked again, the
	// failed one is.
	fetch.asked = nil
	svc.Run(context.Background())
	assert.NotContains(t, fetch.asked, int64(102))
	assert.Contains(t, fetch.asked, int64(103))
}

func TestCreditsBackfill_FailSoft(t *testing.T) {
	t.Run("no fetcher → nothing happens", func(t *testing.T) {
		movies := &backfillMovieRepo{rows: []models.Movie{{ID: "mv-1", TMDbID: models.NewNullInt64(1)}}}
		res := NewCreditsBackfillService(movies, nil, nil, nil).Run(context.Background())
		assert.Equal(t, CreditsBackfillResult{}, res)
		assert.Empty(t, movies.written)
	})
	t.Run("listing error is logged, the other table still runs", func(t *testing.T) {
		movies := &backfillMovieRepo{listErr: errors.New("db locked")}
		series := &backfillSeriesRepo{rows: []models.Series{{ID: "s", TMDbID: models.NewNullInt64(7)}}}
		fetch := &backfillFetcher{byID: map[int64]*models.Credits{7: cast("A")}}
		res := NewCreditsBackfillService(movies, series, fetch, nil).Run(context.Background())
		assert.Equal(t, 1, res.Series)
	})
	t.Run("a write error counts as failed and is retried next pass", func(t *testing.T) {
		series := &backfillSeriesRepo{rows: []models.Series{{ID: "s", TMDbID: models.NewNullInt64(7)}}, writeErr: errors.New("disk full")}
		fetch := &backfillFetcher{byID: map[int64]*models.Credits{7: cast("A")}}
		svc := NewCreditsBackfillService(nil, series, fetch, nil)
		assert.Equal(t, 1, svc.Run(context.Background()).Failed)
		assert.Equal(t, 1, svc.Run(context.Background()).Failed, "not remembered as done")
	})
	t.Run("a cancelled ctx stops the walk", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		movies := &backfillMovieRepo{rows: []models.Movie{{ID: "mv-1", TMDbID: models.NewNullInt64(1)}}}
		fetch := &backfillFetcher{byID: map[int64]*models.Credits{1: cast("A")}}
		res := NewCreditsBackfillService(movies, nil, fetch, nil).Run(ctx)
		assert.Equal(t, CreditsBackfillResult{}, res)
		assert.Empty(t, fetch.asked)
	})
}
