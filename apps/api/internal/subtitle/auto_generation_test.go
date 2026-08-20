package subtitle

// Story 9R-10b AC #4 + AC #8 — the on-add auto-trigger's selection policy.
//
// The trigger runs UNDERNEATH the 2026-08-07 cost ruling: every item it touches
// is processed with FreeOnly, so the worst case of selecting too many items is
// local CPU time, never a charge. What selection still has to get right is the
// opposite failure — selecting an item whose library never opted in.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// ─── fakes ─────────────────────────────────────────────────────────────────

type autoFakeMovieFinder struct {
	movies []models.Movie
	err    error
	calls  int
}

func (f *autoFakeMovieFinder) FindMissingZhHantSubtitle(context.Context) ([]models.Movie, error) {
	f.calls++
	return f.movies, f.err
}

type autoFakeEpisodeFinder struct {
	episodes []models.Episode
	err      error
}

func (f *autoFakeEpisodeFinder) FindMissingZhHantSubtitle(context.Context) ([]models.Episode, error) {
	return f.episodes, f.err
}

type autoFakeSeriesResolver struct {
	byID map[string]*models.Series
	err  error
}

func (f *autoFakeSeriesResolver) FindByID(_ context.Context, id string) (*models.Series, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byID[id], nil
}

type autoFakeLibraryPolicy struct {
	enabled map[string]struct{}
	err     error
	calls   int
}

func (f *autoFakeLibraryPolicy) AutoSubtitleLibraryIDs(context.Context) (map[string]struct{}, error) {
	f.calls++
	return f.enabled, f.err
}

type autoProcessedCall struct {
	ref  MediaRef
	opts ProcessItemOptions
}

type autoFakeItemProcessor struct {
	calls   []autoProcessedCall
	outcome func(ref MediaRef) (*ProcessOutcome, error)
}

func (f *autoFakeItemProcessor) ProcessItem(_ context.Context, ref MediaRef, opts ProcessItemOptions) (*ProcessOutcome, error) {
	f.calls = append(f.calls, autoProcessedCall{ref: ref, opts: opts})
	if f.outcome != nil {
		return f.outcome(ref)
	}
	return &ProcessOutcome{Kind: RouteDeliverDirect}, nil
}

func (f *autoFakeItemProcessor) refIDs() []string {
	out := make([]string, 0, len(f.calls))
	for _, c := range f.calls {
		out = append(out, c.ref.ID)
	}
	return out
}

// ─── builders ──────────────────────────────────────────────────────────────

func autoMovieIn(id, libraryID string, status models.SubtitleStatus) models.Movie {
	m := models.Movie{ID: id, SubtitleStatus: status}
	if libraryID != "" {
		m.LibraryID = models.NewNullString(libraryID)
	}
	return m
}

func autoEpisodeIn(id, seriesID string, status models.SubtitleStatus) models.Episode {
	return models.Episode{ID: id, SeriesID: seriesID, SubtitleStatus: status}
}

func autoSeriesIn(id, libraryID string) *models.Series {
	s := &models.Series{ID: id}
	if libraryID != "" {
		s.LibraryID = models.NewNullString(libraryID)
	}
	return s
}

func autoEnabledSet(ids ...string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

type autoHarness struct {
	gen      *AutoGenerator
	movies   *autoFakeMovieFinder
	episodes *autoFakeEpisodeFinder
	series   *autoFakeSeriesResolver
	policy   *autoFakeLibraryPolicy
	item     *autoFakeItemProcessor
}

func newAutoHarness(t *testing.T, opts ...AutoGeneratorOption) *autoHarness {
	t.Helper()
	h := &autoHarness{
		movies:   &autoFakeMovieFinder{},
		episodes: &autoFakeEpisodeFinder{},
		series:   &autoFakeSeriesResolver{byID: map[string]*models.Series{}},
		policy:   &autoFakeLibraryPolicy{enabled: autoEnabledSet()},
		item:     &autoFakeItemProcessor{},
	}
	h.gen = NewAutoGenerator(h.item, h.policy, nil,
		append([]AutoGeneratorOption{
			WithAutoCandidateFinders(h.movies, h.episodes),
			WithAutoSeriesResolver(h.series),
		}, opts...)...)
	return h
}

// ─── AC #4: the policy matrix ──────────────────────────────────────────────

func TestAutoGenerator_SelectionPolicy(t *testing.T) {
	eligible := models.SubtitleStatusNotSearched

	cases := []struct {
		name     string
		enabled  []string
		movies   []models.Movie
		episodes []models.Episode
		series   map[string]*models.Series
		maxPer   int
		wantRefs []string
	}{
		{
			name:     "no library opted in — nothing runs",
			enabled:  nil,
			movies:   []models.Movie{autoMovieIn("m1", "lib-a", eligible)},
			wantRefs: []string{},
		},
		{
			name:     "movie in an enabled library is selected",
			enabled:  []string{"lib-a"},
			movies:   []models.Movie{autoMovieIn("m1", "lib-a", eligible)},
			wantRefs: []string{"m1"},
		},
		{
			name:     "movie in a DISABLED library is skipped",
			enabled:  []string{"lib-a"},
			movies:   []models.Movie{autoMovieIn("m1", "lib-b", eligible)},
			wantRefs: []string{},
		},
		{
			name:     "movie with no library at all is skipped — opt-in cannot be assumed",
			enabled:  []string{"lib-a"},
			movies:   []models.Movie{autoMovieIn("m1", "", eligible)},
			wantRefs: []string{},
		},
		{
			name:     "episode resolves its library through the SERIES row",
			enabled:  []string{"lib-a"},
			episodes: []models.Episode{autoEpisodeIn("e1", "s1", eligible)},
			series:   map[string]*models.Series{"s1": autoSeriesIn("s1", "lib-a")},
			wantRefs: []string{"e1"},
		},
		{
			name:     "episode whose series sits in a disabled library is skipped",
			enabled:  []string{"lib-a"},
			episodes: []models.Episode{autoEpisodeIn("e1", "s1", eligible)},
			series:   map[string]*models.Series{"s1": autoSeriesIn("s1", "lib-b")},
			wantRefs: []string{},
		},
		{
			name:     "episode with an unresolvable series is skipped, not guessed",
			enabled:  []string{"lib-a"},
			episodes: []models.Episode{autoEpisodeIn("e1", "ghost", eligible)},
			series:   map[string]*models.Series{},
			wantRefs: []string{},
		},
		{
			name:    "status filter — only not_searched / not_found / untranslated are eligible",
			enabled: []string{"lib-a"},
			movies: []models.Movie{
				autoMovieIn("m-not-searched", "lib-a", models.SubtitleStatusNotSearched),
				autoMovieIn("m-not-found", "lib-a", models.SubtitleStatusNotFound),
				autoMovieIn("m-untranslated", "lib-a", models.SubtitleStatusUntranslated),
				autoMovieIn("m-found", "lib-a", models.SubtitleStatusFound),
				autoMovieIn("m-skipped", "lib-a", models.SubtitleStatusSkipped),
				autoMovieIn("m-no-text", "lib-a", models.SubtitleStatusNoTextSource),
				autoMovieIn("m-translating", "lib-a", models.SubtitleStatusTranslating),
			},
			wantRefs: []string{"m-not-searched", "m-not-found", "m-untranslated"},
		},
		{
			name:    "under the per-run cap everything runs",
			enabled: []string{"lib-a"},
			movies: []models.Movie{
				autoMovieIn("m1", "lib-a", eligible), autoMovieIn("m2", "lib-a", eligible),
			},
			maxPer:   3,
			wantRefs: []string{"m1", "m2"},
		},
		{
			name:    "over the per-run cap the list is truncated in enumeration order",
			enabled: []string{"lib-a"},
			movies: []models.Movie{
				autoMovieIn("m1", "lib-a", eligible), autoMovieIn("m2", "lib-a", eligible),
				autoMovieIn("m3", "lib-a", eligible), autoMovieIn("m4", "lib-a", eligible),
			},
			maxPer:   2,
			wantRefs: []string{"m1", "m2"},
		},
		{
			name:    "the cap spans BOTH families — episodes do not get a fresh budget",
			enabled: []string{"lib-a"},
			movies: []models.Movie{
				autoMovieIn("m1", "lib-a", eligible), autoMovieIn("m2", "lib-a", eligible),
			},
			episodes: []models.Episode{autoEpisodeIn("e1", "s1", eligible)},
			series:   map[string]*models.Series{"s1": autoSeriesIn("s1", "lib-a")},
			maxPer:   2,
			wantRefs: []string{"m1", "m2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []AutoGeneratorOption{}
			if tc.maxPer > 0 {
				opts = append(opts, WithAutoMaxPerRun(tc.maxPer))
			}
			h := newAutoHarness(t, opts...)
			h.policy.enabled = autoEnabledSet(tc.enabled...)
			h.movies.movies = tc.movies
			h.episodes.episodes = tc.episodes
			if tc.series != nil {
				h.series.byID = tc.series
			}

			h.gen.Run(context.Background())

			assert.Equal(t, tc.wantRefs, h.item.refIDs())
		})
	}
}

// ─── AC #4: every processed item rides the free lane ──────────────────────

// TestAutoGenerator_AlwaysProcessesFreeOnly is the selection-side half of the
// cost guard. The pipeline-side half lives in process_item_freeonly_test.go;
// this one pins that the trigger cannot accidentally hand the pipeline a
// budget-bearing options struct.
func TestAutoGenerator_AlwaysProcessesFreeOnly(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m2", "lib-a", models.SubtitleStatusNotFound),
	}

	h.gen.Run(context.Background())

	require.Len(t, h.item.calls, 2)
	for _, c := range h.item.calls {
		assert.True(t, c.opts.FreeOnly,
			"the auto-trigger must NEVER call ProcessItem without FreeOnly — that is the 2026-08-19 ruling in one field")
		assert.False(t, c.opts.Force,
			"Force bypasses the pre-flight and would re-run completed items on every scan")
	}
	assert.Equal(t, models.SubtitleRunMediaMovie, h.item.calls[0].ref.MediaType)
}

func TestAutoGenerator_EpisodeRefCarriesEpisodeMediaType(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.episodes.episodes = []models.Episode{autoEpisodeIn("e1", "s1", models.SubtitleStatusNotSearched)}
	h.series.byID = map[string]*models.Series{"s1": autoSeriesIn("s1", "lib-a")}

	h.gen.Run(context.Background())

	require.Len(t, h.item.calls, 1)
	assert.Equal(t, models.SubtitleRunMediaEpisode, h.item.calls[0].ref.MediaType,
		"an episode processed as a movie would resolve the wrong file path (9R-10a 紅線 1)")
}

// ─── AC #4 + 9R-10a M1 lesson: a failed read aborts, it never half-runs ────

func TestAutoGenerator_AbortsOnPolicyReadFailure(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.err = errors.New("database is locked")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.calls,
		"a policy read failure must abort the round — a locked DB must not be read as 'nobody opted in' (9R-10a CR M1)")
	assert.Zero(t, h.movies.calls,
		"enumeration must not even be attempted once the policy is unknown")
}

func TestAutoGenerator_AbortsOnEnumerationFailure(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.err = errors.New("database is locked")

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.calls,
		"a half-enumerated library would silently process an arbitrary subset")
}

func TestAutoGenerator_ContinuesWhenOneItemFails(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m2", "lib-a", models.SubtitleStatusNotSearched),
	}
	h.item.outcome = func(ref MediaRef) (*ProcessOutcome, error) {
		if ref.ID == "m1" {
			return nil, errors.New("ffmpeg exploded")
		}
		return &ProcessOutcome{Kind: RouteDeliverDirect}, nil
	}

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1", "m2"}, h.item.refIDs(),
		"one bad file must not strand the rest of the round — per-item failure is already recorded on its run row")
}

// ─── AC #4: no work at all is a routine outcome, not an error ─────────────

func TestAutoGenerator_NoEligibleItemsIsANoOp(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.calls)
}

// ─── AC #4: nil ports degrade instead of panicking ────────────────────────

func TestAutoGenerator_NilPortsAreSafe(t *testing.T) {
	gen := NewAutoGenerator(&autoFakeItemProcessor{}, &autoFakeLibraryPolicy{enabled: autoEnabledSet("lib-a")}, nil)
	assert.NotPanics(t, func() { gen.Run(context.Background()) },
		"a boot without finders wired must idle, not crash the scan-complete callback")
}
