package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

type scopeSeriesFinder struct{ rows map[string]*models.Series }

func (f *scopeSeriesFinder) FindByID(_ context.Context, id string) (*models.Series, error) {
	if s, ok := f.rows[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("series with id %s not found: %w", id, sql.ErrNoRows)
}

type scopeMovieFinder struct{ rows map[string]*models.Movie }

func (f *scopeMovieFinder) FindByID(_ context.Context, id string) (*models.Movie, error) {
	if m, ok := f.rows[id]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("movie with id %s not found: %w", id, sql.ErrNoRows)
}

type scopeEpisodeFinder struct{ rows map[string]*models.Episode }

func (f *scopeEpisodeFinder) FindByID(_ context.Context, id string) (*models.Episode, error) {
	if e, ok := f.rows[id]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("episode with id %s: %w", id, repository.ErrEpisodeNotFound)
}

type brokenSeriesFinder struct{}

func (brokenSeriesFinder) FindByID(context.Context, string) (*models.Series, error) {
	return nil, errors.New("disk on fire")
}

type recordingMover struct {
	calls [][2]string
	err   error
}

func (m *recordingMover) MigrateScope(_ context.Context, from, to string) (int64, int64, error) {
	m.calls = append(m.calls, [2]string{from, to})
	return 1, 0, m.err
}

func newResolverFixture(mover glossaryScopeMover) *GlossaryScopeResolver {
	series := &scopeSeriesFinder{rows: map[string]*models.Series{
		"s-matched":   {ID: "s-matched", TMDbID: models.NewNullInt64(66732)},
		"s-unmatched": {ID: "s-unmatched"},
	}}
	movies := &scopeMovieFinder{rows: map[string]*models.Movie{
		"m-matched":   {ID: "m-matched", TMDbID: models.NewNullInt64(27205)},
		"m-unmatched": {ID: "m-unmatched"},
	}}
	episodes := &scopeEpisodeFinder{rows: map[string]*models.Episode{
		"e-of-matched":   {ID: "e-of-matched", SeriesID: "s-matched"},
		"e-of-unmatched": {ID: "e-of-unmatched", SeriesID: "s-unmatched"},
		"e-orphan":       {ID: "e-orphan", SeriesID: "s-gone"},
	}}
	return NewGlossaryScopeResolver(series, movies, episodes, mover, slog.Default())
}

// AC #2 / AC #5(b): the three media kinds and an id nothing knows.
func TestGlossaryScopeResolver_Resolve(t *testing.T) {
	r := newResolverFixture(nil)
	ctx := context.Background()
	cases := []struct{ id, want string }{
		{"s-matched", "tmdb:tv:66732"},
		{"s-unmatched", "local:s-unmatched"},
		{"m-matched", "tmdb:movie:27205"},
		{"m-unmatched", "local:m-unmatched"},
		{"e-of-matched", "tmdb:tv:66732"},
		{"e-of-unmatched", "local:s-unmatched"},
		{"e-orphan", "local:s-gone"},
		{"never-heard-of", "local:never-heard-of"},
		{"  s-matched  ", "tmdb:tv:66732"},
	}
	for _, c := range cases {
		got, err := r.Resolve(ctx, c.id)
		require.NoError(t, err, c.id)
		assert.Equal(t, c.want, got, c.id)
	}
}

func TestGlossaryScopeResolver_EmptyIDIsAValidationError(t *testing.T) {
	_, err := newResolverFixture(nil).Resolve(context.Background(), "   ")
	var verr *models.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "media_id", verr.Field)
}

func TestGlossaryScopeResolver_RepositoryErrorsAreReturnedNotSwallowed(t *testing.T) {
	r := NewGlossaryScopeResolver(brokenSeriesFinder{}, nil, nil, nil, nil)
	_, err := r.Resolve(context.Background(), "anything")
	require.Error(t, err, "a real DB failure must reach the caller's fail-soft log, not become local:")
	assert.Contains(t, err.Error(), "disk on fire")
}

// AC #3 / AC #5(c): resolving to a shared scope sweeps the local drawer(s) into it.
func TestGlossaryScopeResolver_UpgradesLocalDrawerIntoSharedScope(t *testing.T) {
	mover := &recordingMover{}
	r := newResolverFixture(mover)
	ctx := context.Background()

	_, err := r.Resolve(ctx, "s-matched")
	require.NoError(t, err)
	assert.Equal(t, [][2]string{{"local:s-matched", "tmdb:tv:66732"}}, mover.calls)

	mover.calls = nil
	_, err = r.Resolve(ctx, "e-of-matched")
	require.NoError(t, err)
	assert.Equal(t, [][2]string{
		{"local:s-matched", "tmdb:tv:66732"},    // the show's drawer
		{"local:e-of-matched", "tmdb:tv:66732"}, // and the drawer a pre-ShowKey caller keyed by the episode
	}, mover.calls)

	mover.calls = nil
	_, err = r.Resolve(ctx, "s-unmatched")
	require.NoError(t, err)
	assert.Empty(t, mover.calls, "a local answer has nothing to upgrade")
}

func TestGlossaryScopeResolver_UpgradeFailureIsFailSoft(t *testing.T) {
	mover := &recordingMover{err: errors.New("locked")}
	r := newResolverFixture(mover)
	got, err := r.Resolve(context.Background(), "m-matched")
	require.NoError(t, err, "the shared scope is still the right answer")
	assert.Equal(t, "tmdb:movie:27205", got)
	assert.Len(t, mover.calls, 1)
}

func TestGlossaryScopeResolver_NoCache_EveryResolveAsksAgain(t *testing.T) {
	// A match that lands AFTER the first resolve must be seen by the second —
	// the reason the resolver deliberately does not cache.
	series := &scopeSeriesFinder{rows: map[string]*models.Series{"s1": {ID: "s1"}}}
	r := NewGlossaryScopeResolver(series, nil, nil, nil, nil)
	ctx := context.Background()
	first, err := r.Resolve(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, "local:s1", first)

	series.rows["s1"] = &models.Series{ID: "s1", TMDbID: models.NewNullInt64(99)}
	second, err := r.Resolve(ctx, "s1")
	require.NoError(t, err)
	assert.Equal(t, "tmdb:tv:99", second)
}
