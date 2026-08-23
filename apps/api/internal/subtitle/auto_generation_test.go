package subtitle

// Story 9R-10b AC #4 + AC #8 — the on-add auto-trigger's selection policy.
//
// The trigger runs UNDERNEATH the 2026-08-07 cost ruling: every item it touches
// is processed with FreeOnly, so the worst case of selecting too many items is
// local CPU time, never a charge. What selection still has to get right is the
// opposite failure — selecting an item whose library never opted in.

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// autoFakeSeriesResolver mirrors the REAL SeriesRepository.FindByID contract.
//
// CR H2: it previously returned (nil, nil) for a missing id, which is NOT what
// the repository does — series_repository.go:108 returns an ERROR wrapping
// sql.ErrNoRows. The old fake made a test pass while asserting a behaviour the
// production code did not have. Any fake that is more forgiving than the thing
// it stands in for is worse than no fake at all.
type autoFakeSeriesResolver struct {
	byID map[string]*models.Series
	err  error
}

func (f *autoFakeSeriesResolver) FindByID(_ context.Context, id string) (*models.Series, error) {
	if f.err != nil {
		return nil, f.err
	}
	series, ok := f.byID[id]
	if !ok {
		return nil, fmt.Errorf("series with id %s not found: %w", id, sql.ErrNoRows)
	}
	return series, nil
}

// autoFakeRunLister stands in for the subtitle_runs reader behind CR H1.
type autoFakeRunLister struct {
	runs  []models.SubtitleRun
	err   error
	calls int
}

func (f *autoFakeRunLister) ListByStatus(_ context.Context, status models.SubtitleRunStatus, _ int) ([]models.SubtitleRun, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	var out []models.SubtitleRun
	for _, r := range f.runs {
		if r.Status == status {
			out = append(out, r)
		}
	}
	return out, nil
}

// autoNow is the fixed "run time" every fixture row carries — after
// FreeLaneEpoch, so the rows count (bugfix-auto-exclusion-never-expires AC #3).
var autoNow = freeLaneEpoch.Add(24 * time.Hour)

// autoOldFile is the default mtime the harness reports: older than every row,
// so a parked item STAYS parked unless a test says the file changed.
var autoOldFile = autoNow.Add(-time.Hour)

func deferredRun(mediaID string) models.SubtitleRun {
	return models.SubtitleRun{
		MediaID:      mediaID,
		Status:       models.SubtitleRunSkipped,
		ErrorMessage: DeferredPaidRunPrefix + "translate requires paid work",
		StartedAt:    autoNow,
	}
}

func withStartedAt(r models.SubtitleRun, t time.Time) models.SubtitleRun {
	r.StartedAt = t
	return r
}

// skippedRun is a `skipped` row that is NOT a deferral — a decline the pipeline
// reached on its own (fail-closed detection, an unusable stream).
func skippedRun(mediaID, reason string) models.SubtitleRun {
	return models.SubtitleRun{
		MediaID:      mediaID,
		Status:       models.SubtitleRunSkipped,
		ErrorMessage: reason,
		StartedAt:    autoNow,
	}
}

func failedRun(mediaID string) models.SubtitleRun {
	return models.SubtitleRun{
		MediaID:      mediaID,
		Status:       models.SubtitleRunFailed,
		ErrorMessage: "subtitle pipeline: movie " + mediaID + ": probe: ffprobe: no such file or directory",
		StartedAt:    autoNow,
	}
}

// cancelledRun is a `failed` row written by failItem under shutdown
// cancellation (bugfix-autogenerator-no-timeout-or-shutdown AC #5) — it must
// NOT count toward autoFailureAttemptLimit.
func cancelledRun(mediaID string) models.SubtitleRun {
	return models.SubtitleRun{
		MediaID:      mediaID,
		Status:       models.SubtitleRunFailed,
		ErrorMessage: CancelledRunPrefix + "subtitle pipeline: movie " + mediaID + ": extract: ffmpeg cancelled",
		StartedAt:    autoNow,
	}
}

func failedRuns(mediaID string, n int) []models.SubtitleRun {
	out := make([]models.SubtitleRun, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, failedRun(mediaID))
	}
	return out
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

// autoFakeItemProcessor is mutex-guarded because the single-flight test calls
// Run from two goroutines: a regression there must surface as a failed
// assertion on the recorded calls, not as a data race that only -race sees
// (補審 M7).
type autoFakeItemProcessor struct {
	mu      sync.Mutex
	calls   []autoProcessedCall
	outcome func(ctx context.Context, ref MediaRef) (*ProcessOutcome, error)
}

func (f *autoFakeItemProcessor) ProcessItem(ctx context.Context, ref MediaRef, opts ProcessItemOptions) (*ProcessOutcome, error) {
	f.mu.Lock()
	f.calls = append(f.calls, autoProcessedCall{ref: ref, opts: opts})
	outcome := f.outcome
	f.mu.Unlock()
	if outcome != nil {
		return outcome(ctx, ref)
	}
	return &ProcessOutcome{Kind: RouteDeliverDirect}, nil
}

func (f *autoFakeItemProcessor) recorded() []autoProcessedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]autoProcessedCall(nil), f.calls...)
}

func (f *autoFakeItemProcessor) refIDs() []string {
	calls := f.recorded()
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, c.ref.ID)
	}
	return out
}

// ─── builders ──────────────────────────────────────────────────────────────

func autoMovieIn(id, libraryID string, status models.SubtitleStatus) models.Movie {
	m := models.Movie{ID: id, SubtitleStatus: status, FilePath: models.NewNullString("/media/" + id + ".mkv")}
	if libraryID != "" {
		m.LibraryID = models.NewNullString(libraryID)
	}
	return m
}

func autoEpisodeIn(id, seriesID string, status models.SubtitleStatus) models.Episode {
	return models.Episode{ID: id, SeriesID: seriesID, SubtitleStatus: status, FilePath: models.NewNullString("/media/" + id + ".mkv")}
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
	// blockSecondOnCancel makes overlapHarness's 2nd ProcessItem call wait on
	// ctx.Done() (used by the Stop-drops-follow-up test).
	blockSecondOnCancel bool

	gen      *AutoGenerator
	movies   *autoFakeMovieFinder
	episodes *autoFakeEpisodeFinder
	series   *autoFakeSeriesResolver
	policy   *autoFakeLibraryPolicy
	item     *autoFakeItemProcessor
	runs     *autoFakeRunLister
	files    *autoFakeFiles
}

// autoFakeFiles is the mod-time port: per-path mtimes (default autoOldFile),
// an optional error, and a record of which paths were stat-ed (AC #5).
type autoFakeFiles struct {
	modTimes map[string]time.Time
	err      error
	stats    []string
}

func (f *autoFakeFiles) modTime(path string) (time.Time, error) {
	f.stats = append(f.stats, path)
	if f.err != nil {
		return time.Time{}, f.err
	}
	if t, ok := f.modTimes[path]; ok {
		return t, nil
	}
	return autoOldFile, nil
}

func (f *autoFakeFiles) replaced(id string, at time.Time) { f.modTimes["/media/"+id+".mkv"] = at }

func newAutoHarness(t *testing.T, opts ...AutoGeneratorOption) *autoHarness {
	t.Helper()
	h := &autoHarness{
		movies:   &autoFakeMovieFinder{},
		episodes: &autoFakeEpisodeFinder{},
		series:   &autoFakeSeriesResolver{byID: map[string]*models.Series{}},
		policy:   &autoFakeLibraryPolicy{enabled: autoEnabledSet()},
		item:     &autoFakeItemProcessor{},
		runs:     &autoFakeRunLister{},
		files:    &autoFakeFiles{modTimes: map[string]time.Time{}},
	}
	h.gen = NewAutoGenerator(h.item, h.policy, nil,
		append([]AutoGeneratorOption{
			WithAutoCandidateFinders(h.movies, h.episodes),
			WithAutoSeriesResolver(h.series),
			WithAutoDeferredRuns(h.runs),
			WithAutoFileModTime(h.files.modTime),
		}, opts...)...)
	// Releases the lifetime ctx, and gives every test free coverage that Stop
	// after a finished round returns (review L8).
	t.Cleanup(h.gen.Stop)
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
	h.item.outcome = func(_ context.Context, ref MediaRef) (*ProcessOutcome, error) {
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

// ─── CR H1: a deferred item must not re-occupy the budget every scan ──────

// TestAutoGenerator_ExcludesPreviouslyDeferredItems pins the fix for the
// starvation bug. The pipeline's pre-flight gate is "does an acceptable sidecar
// exist", and a deferred item never gets one — so without this exclusion the
// alphabetically-first items (which is what the enumeration order guarantees)
// would be re-extracted on every scan forever while the free items behind them
// were never reached.
func TestAutoGenerator_ExcludesPreviouslyDeferredItems(t *testing.T) {
	eligible := models.SubtitleStatusNotSearched
	h := newAutoHarness(t, WithAutoMaxPerRun(2))
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m-paid-1", "lib-a", eligible),
		autoMovieIn("m-paid-2", "lib-a", eligible),
		autoMovieIn("m-free-1", "lib-a", eligible),
		autoMovieIn("m-free-2", "lib-a", eligible),
	}
	h.runs.runs = []models.SubtitleRun{deferredRun("m-paid-1"), deferredRun("m-paid-2")}

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m-free-1", "m-free-2"}, h.item.refIDs(),
		"the budget must move PAST items already known to need paid work — otherwise the free items are never reached")
	assert.Equal(t, 2, h.runs.calls,
		"two run queries per round (skipped + failed, 補審 M1), not one per item")
}

func TestAutoGenerator_DeferredExclusionIgnoresOtherSkippedRuns(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}
	// A RouteSkip run is also `skipped`, but it is a DECLINE, not a deferral —
	// only the marker distinguishes them.
	h.runs.runs = []models.SubtitleRun{{
		MediaID:      "m1",
		Status:       models.SubtitleRunSkipped,
		ErrorMessage: "stream 2 tagged und: detector returned unrecognized variant — failing closed (P0)",
	}}

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1"}, h.item.refIDs(),
		"only runs carrying DeferredPaidRunPrefix may exclude an item")
}

func TestAutoGenerator_DeferredRunLookupFailureAborts(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}
	h.runs.err = errors.New("database is locked")

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.calls,
		"an unreadable run table must abort the round — proceeding would re-burn every deferred item")
}

// TestAutoGenerator_ExcludesDeferredEPISODES is the episode twin of the movie
// exclusion above (補審 M5). The two families run through separate loops with
// separate guards, and only the movie one was pinned — deleting the episode
// guard left the whole suite green, so H1 could come back for TV alone.
func TestAutoGenerator_ExcludesDeferredEpisodes(t *testing.T) {
	eligible := models.SubtitleStatusNotSearched
	h := newAutoHarness(t, WithAutoMaxPerRun(2))
	h.policy.enabled = autoEnabledSet("lib-a")
	h.series.byID["s1"] = autoSeriesIn("s1", "lib-a")
	h.episodes.episodes = []models.Episode{
		autoEpisodeIn("e-paid-1", "s1", eligible),
		autoEpisodeIn("e-paid-2", "s1", eligible),
		autoEpisodeIn("e-free-1", "s1", eligible),
		autoEpisodeIn("e-free-2", "s1", eligible),
	}
	h.runs.runs = []models.SubtitleRun{deferredRun("e-paid-1"), deferredRun("e-paid-2")}

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"e-free-1", "e-free-2"}, h.item.refIDs(),
		"the episode loop must skip parked items too — the movie guard alone leaves TV starving")
}

// ─── 補審 M1: an item that keeps FAILING must stop eating the budget ───────

// TestAutoGenerator_ExcludesItemsThatKeepFailing pins the second half of the
// starvation fix. A failed run puts the media row back at `not_searched`
// (failItem) and leaves no sidecar, so without this the same broken file is
// re-probed and re-recorded on every scan for the life of the library.
func TestAutoGenerator_ExcludesItemsThatKeepFailing(t *testing.T) {
	eligible := models.SubtitleStatusNotSearched
	h := newAutoHarness(t, WithAutoMaxPerRun(2))
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m-broken-1", "lib-a", eligible),
		autoMovieIn("m-broken-2", "lib-a", eligible),
		autoMovieIn("m-free-1", "lib-a", eligible),
		autoMovieIn("m-free-2", "lib-a", eligible),
	}
	h.runs.runs = append(
		failedRuns("m-broken-1", autoFailureAttemptLimit),
		failedRuns("m-broken-2", autoFailureAttemptLimit+2)...,
	)

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m-free-1", "m-free-2"}, h.item.refIDs(),
		"a file that cannot be processed must not re-occupy the budget on every scan")
}

// TestAutoGenerator_RetriesAnItemBelowTheFailureLimit is the other side of the
// boundary: one bad scan is usually the NAS, not the file, so the item must
// still be retried. Without this the limit could silently become 1 and nothing
// would notice.
func TestAutoGenerator_RetriesAnItemBelowTheFailureLimit(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}
	// A literal 2, deliberately not autoFailureAttemptLimit-1: expressed in
	// terms of the constant this test moves WITH the limit and could never
	// catch it being lowered, which is the regression it exists for.
	h.runs.runs = failedRuns("m1", 2)

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1"}, h.item.refIDs(),
		"below the limit the item is still retried — a transient share hiccup is not a verdict")
}

// ─── 補審 M3: the LATEST skipped run decides, as the comment always claimed ──

func TestAutoGenerator_LatestSkippedRunDecidesTheExclusion(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}
	// Newest first, the ListByStatus contract: the deferral is HISTORY, the
	// current verdict is the decline above it.
	h.runs.runs = []models.SubtitleRun{
		skippedRun("m1", "stream 2 tagged und: detector returned unrecognized variant — failing closed (P0)"),
		deferredRun("m1"),
	}

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1"}, h.item.refIDs(),
		"an older deferral must not outvote the latest skipped run — that guard was inert before 補審 M3")
}

func TestAutoGenerator_WithoutDeferredRunPortStillWorks(t *testing.T) {
	gen := NewAutoGenerator(&autoFakeItemProcessor{}, &autoFakeLibraryPolicy{enabled: autoEnabledSet("lib-a")}, nil,
		WithAutoCandidateFinders(&autoFakeMovieFinder{movies: []models.Movie{
			autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched),
		}}, nil))
	assert.NotPanics(t, func() { gen.Run(context.Background()) },
		"the deferred-run port is optional — without it the trigger degrades in cost, never in correctness")
}

// ─── CR H2: an orphan episode is skipped, the round survives ──────────────

// TestAutoGenerator_OrphanEpisodeDoesNotAbortTheRound is the regression pin for
// the fake-vs-reality gap. SeriesRepository.FindByID returns an ERROR wrapping
// sql.ErrNoRows when the parent row is gone; treating that as fatal meant ONE
// orphan episode discarded every movie already collected, on every scan.
func TestAutoGenerator_OrphanEpisodeDoesNotAbortTheRound(t *testing.T) {
	eligible := models.SubtitleStatusNotSearched
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", eligible)}
	h.episodes.episodes = []models.Episode{
		autoEpisodeIn("e-orphan", "series-gone", eligible),
		autoEpisodeIn("e-ok", "s1", eligible),
	}
	h.series.byID = map[string]*models.Series{"s1": autoSeriesIn("s1", "lib-a")}

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1", "e-ok"}, h.item.refIDs(),
		"the orphan is skipped; everything else in the round still runs")
}

func TestAutoGenerator_GenuineSeriesLookupFailureStillAborts(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.episodes.episodes = []models.Episode{autoEpisodeIn("e1", "s1", models.SubtitleStatusNotSearched)}
	h.series.err = errors.New("database is locked")

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.calls,
		"a locked DB is NOT an orphan — it must abort rather than silently skip every episode (9R-10a CR M1)")
}

// ─── CR M1: one round at a time ───────────────────────────────────────────

// ─── bugfix-autogenerator-dropped-round-not-deferred ───────────────────────
//
// overlapHarness blocks the FIRST ProcessItem call until release is closed and
// reports when it has been entered, so a test can land extra triggers mid-round.
// `fourth` closes when the 4th ProcessItem call lands — i.e. the follow-up
// round has reached its last item — so a test can wait for it without Stop
// (which would cancel the follow-up, AC #4) and without a sleep.
func overlapHarness(t *testing.T) (h *autoHarness, entered, release, fourth chan struct{}) {
	t.Helper()
	h = newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m2", "lib-a", models.SubtitleStatusNotSearched),
	}
	entered = make(chan struct{})
	release = make(chan struct{})
	fourth = make(chan struct{})
	var once, fourthOnce sync.Once
	var n atomic.Int32
	h.item.outcome = func(ctx context.Context, _ MediaRef) (*ProcessOutcome, error) {
		once.Do(func() {
			close(entered)
			<-release
		})
		if h.blockSecondOnCancel && n.Load() == 1 {
			// The 2nd call waits for the generator's cancel — which happens only
			// AFTER stopped=true/pending=false under mu — so the round provably
			// observes Stop before its deferred unlock runs.
			<-ctx.Done()
		}
		if n.Add(1) == 4 {
			fourthOnce.Do(func() { close(fourth) })
		}
		return &ProcessOutcome{Kind: RouteDeliverDirect}, nil
	}
	return h, entered, release, fourth
}

// AC #1 + #6: an overlapping trigger is queued, not dropped — and still never
// runs CONCURRENTLY with the round in flight (the CR M1 invariant).
//
// 補審 M7 note still applies: the guard must return for the second caller, not
// block it — a sync.Once-style block deadlocks the package and shows up as
// `panic: test timed out`.
func TestAutoGenerator_OverlappingTriggerIsDeferredNotDropped(t *testing.T) {
	h, entered, release, fourth := overlapHarness(t)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); h.gen.Run(context.Background()) }()

	<-entered
	h.gen.Run(context.Background()) // second scan lands mid-round; must return at once
	assert.Equal(t, []string{"m1"}, h.item.refIDs(), "the second trigger must not start a concurrent pass")
	close(release)
	wg.Wait()
	<-fourth     // the follow-up reached its last item
	h.gen.Stop() // and is drained (its goroutine returns before Stop does)

	assert.Equal(t, []string{"m1", "m2", "m1", "m2"}, h.item.refIDs(),
		"the overlapping trigger must produce ONE follow-up round after the first ends — a dropped trigger waits for the next scan that happens to add files")
	assert.Equal(t, 2, h.policy.calls, "AC #3: the follow-up must not re-trigger itself")
}

// AC #5: the queue and the follow-up are visible in the log.
func TestAutoGenerator_DeferredFollowUpIsLogged(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	movies := &autoFakeMovieFinder{movies: []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}}
	item := &autoFakeItemProcessor{}
	gen := NewAutoGenerator(item, &autoFakeLibraryPolicy{enabled: autoEnabledSet("lib-a")}, logger,
		WithAutoCandidateFinders(movies, nil))
	t.Cleanup(gen.Stop)

	entered, release, second := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var n atomic.Int32
	item.outcome = func(context.Context, MediaRef) (*ProcessOutcome, error) {
		switch n.Add(1) {
		case 1:
			close(entered)
			<-release
		case 2:
			close(second)
		}
		return &ProcessOutcome{Kind: RouteDeliverDirect}, nil
	}

	gen.ScanCallback()()
	<-entered
	gen.Run(context.Background())
	close(release)
	<-second
	gen.Stop()

	out := logs.String()
	assert.Contains(t, out, "follow-up round queued")
	assert.Contains(t, out, "starting deferred follow-up round")
	assert.Contains(t, out, "reason=deferred_trigger")
}

// AC #2: N triggers during one round merge into one follow-up.
func TestAutoGenerator_ManyOverlappingTriggersMergeIntoOneFollowUp(t *testing.T) {
	h, entered, release, fourth := overlapHarness(t)

	h.gen.ScanCallback()()
	<-entered
	// Direct Run calls, not ScanCallback: a `go`-spawned trigger that is only
	// scheduled AFTER the first round ends is legitimately a NEW trigger (and
	// would correctly queue a third round) — the test must land all three
	// while the round is provably in flight.
	h.gen.Run(context.Background())
	h.gen.Run(context.Background())
	h.gen.Run(context.Background())
	close(release)
	<-fourth
	h.gen.Stop()

	assert.Equal(t, []string{"m1", "m2", "m1", "m2"}, h.item.refIDs(),
		"each round re-enumerates the whole eligible set, so a second follow-up would find nothing the first did not")
	assert.Equal(t, 2, h.policy.calls)
}

// AC #4: Stop drops the pending follow-up.
func TestAutoGenerator_StopDropsThePendingFollowUp(t *testing.T) {
	h, entered, release, _ := overlapHarness(t)
	h.blockSecondOnCancel = true

	h.gen.ScanCallback()()
	<-entered
	h.gen.Run(context.Background()) // queued (synchronous trigger — see the merge test)
	stopped := make(chan struct{})
	go func() { h.gen.Stop(); close(stopped) }()
	close(release)
	<-stopped

	// Deterministic (review H1): m2 blocks until Stop's cancel, which is issued
	// only after stopped/pending were written under mu — so the round's defer
	// is guaranteed to see pending=false, whatever the scheduler does.
	assert.Equal(t, 1, h.policy.calls, "shutdown is not the time to start work — the next boot's first scan covers it")
	// Two legitimate endings, both "Stop won": the cancel landed before m1
	// returned (loop head breaks → [m1]) or after (m2 runs, waits for the
	// cancel → [m1 m2]). What must never appear is a third item — a follow-up.
	ids := h.item.refIDs()
	assert.LessOrEqual(t, len(ids), 2, "a follow-up round ran after Stop: %v", ids)
	assert.Equal(t, "m1", ids[0])
}

// TestAutoGenerator_InFlightFlagIsReleasedAfterTheRound pins the OTHER half of
// the single-flight guard (補審 M7): the release.
//
// Nothing covered it, so deleting the deferred unlock left the suite green
// while production lost the feature outright — the flag would stay true for the
// life of the process and every later scan would no-op, with one Debug line as
// the only trace.
func TestAutoGenerator_InFlightFlagIsReleasedAfterTheRound(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}

	h.gen.Run(context.Background())
	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1", "m1"}, h.item.refIDs(),
		"a finished round must free the slot — a stuck flag kills auto-generation until the container restarts")
}

// ─── bugfix-autogenerator-no-timeout-or-shutdown: lifecycle ────────────────
//
// AC #1: Stop cancels the in-flight round and returns only after Run has.
func TestAutoGenerator_StopCancelsTheInFlightRoundAndDrains(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m2", "lib-a", models.SubtitleStatusNotSearched),
	}

	entered := make(chan struct{})
	var observed error
	var itemReturned atomic.Bool
	h.item.outcome = func(ctx context.Context, ref MediaRef) (*ProcessOutcome, error) {
		close(entered)
		<-ctx.Done()
		observed = ctx.Err()
		itemReturned.Store(true)
		// The real failItem returns an error on cancellation — the fake must too
		// (9R-10b CR H2: fakes match the production contract).
		return nil, observed
	}

	h.gen.ScanCallback()()
	<-entered
	h.gen.Stop()

	// Stop's wg.Wait is the only thing that orders this flag before the check:
	// without the drain, Stop returns while the item is still blocked.
	assert.True(t, itemReturned.Load(),
		"Stop returned while the in-flight item was still running — failItem's cleanup would race db.Close")
	assert.True(t, errors.Is(observed, context.Canceled), "the item must see Canceled, got %v", observed)
	assert.Equal(t, []string{"m1"}, h.item.refIDs(), "AC #2: items after the cancellation point are not started")
}

// Review M1: the item that was mid-flight at cancellation is NOT a failure —
// its run row carries CancelledRunPrefix and is exempt from parking, so the
// round's own counters and log lines must say the same thing.
func TestAutoGenerator_CancelledItemIsNotCountedAsFailed(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	movies := &autoFakeMovieFinder{movies: []models.Movie{
		autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m2", "lib-a", models.SubtitleStatusNotSearched),
	}}
	item := &autoFakeItemProcessor{}
	gen := NewAutoGenerator(item, &autoFakeLibraryPolicy{enabled: autoEnabledSet("lib-a")}, logger,
		WithAutoCandidateFinders(movies, nil))
	t.Cleanup(gen.Stop)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	item.outcome = func(_ context.Context, ref MediaRef) (*ProcessOutcome, error) {
		cancel()
		return nil, context.Canceled
	}

	gen.Run(ctx)

	out := logs.String()
	assert.NotContains(t, out, "item failed", "a shutdown is not a file failure")
	assert.Contains(t, out, "item cancelled mid-flight")
	assert.Contains(t, out, "subtitle auto-generation cancelled")
	assert.Contains(t, out, "failed=0", "counters must agree with the CancelledRunPrefix exemption: %s", out)
	assert.Contains(t, out, "remaining=2", "the cancelled item is still owed, together with the one never started")
}

// AC #1: idempotent, and safe on a generator that never ran.
func TestAutoGenerator_StopIsIdempotent(t *testing.T) {
	h := newAutoHarness(t)
	assert.NotPanics(t, func() {
		h.gen.Stop()
		h.gen.Stop()
	})
}

// AC #2: cancellation landing mid-item stops the loop at that item.
func TestAutoGenerator_CancelledRoundStopsIterating(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m2", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m3", "lib-a", models.SubtitleStatusNotSearched),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.item.outcome = func(_ context.Context, ref MediaRef) (*ProcessOutcome, error) {
		if ref.ID == "m1" {
			cancel()
			return nil, context.Canceled
		}
		return &ProcessOutcome{Kind: RouteDeliverDirect}, nil
	}

	h.gen.Run(ctx)

	assert.Equal(t, []string{"m1"}, h.item.refIDs(),
		"after cancellation every remaining item would fail instantly and write a pointless failed row each")
}

// AC #2 (Task 1.5): a round that is already cancelled before it starts does not
// even read the policy — otherwise every shutdown logs an Error-level
// "cannot read library policy" line from a goroutine that raced Stop.
func TestAutoGenerator_AlreadyCancelledRoundDoesNotReadThePolicy(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	h.gen.Run(ctx)

	assert.Equal(t, 0, h.policy.calls, "policy must not be queried on a dead ctx")
	assert.Empty(t, h.item.refIDs())
}

// AC #3 (review L5): a DIRECT Run after Stop is refused too, not only the
// scan-callback path — a future re-run caller cannot escape the drain.
func TestAutoGenerator_RunAfterStopIsRefused(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}

	h.gen.Stop()
	h.gen.Run(context.Background())

	assert.Empty(t, h.item.refIDs())
	assert.Equal(t, 0, h.policy.calls)
}

// AC #3: no rounds after Stop.
func TestAutoGenerator_ScanCallbackAfterStopIsANoOp(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}

	h.gen.Stop()
	h.gen.ScanCallback()()
	h.gen.Stop() // would block forever if a goroutine had been spawned without wg bookkeeping

	assert.Empty(t, h.item.refIDs(), "a scan completing inside the shutdown window must not spawn a round behind Stop's back")
	assert.Equal(t, 0, h.policy.calls)
}

// AC #4: each item runs under its own deadline; one slow item fails on its own
// and the next one starts with a fresh clock.
func TestAutoGenerator_EachItemGetsItsOwnDeadline(t *testing.T) {
	h := newAutoHarness(t, WithAutoItemTimeout(50*time.Millisecond))
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m2", "lib-a", models.SubtitleStatusNotSearched),
	}
	var m1Err error
	var m1Deadline, m2Deadline time.Time
	var m2HasDeadline, m2Live bool
	h.item.outcome = func(ctx context.Context, ref MediaRef) (*ProcessOutcome, error) {
		switch ref.ID {
		case "m1":
			m1Deadline, _ = ctx.Deadline()
			<-ctx.Done() // waits on the deadline itself, never on a sleep
			m1Err = ctx.Err()
			return nil, m1Err
		default:
			m2Deadline, m2HasDeadline = ctx.Deadline()
			m2Live = ctx.Err() == nil
			return &ProcessOutcome{Kind: RouteDeliverDirect}, nil
		}
	}

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1", "m2"}, h.item.refIDs(), "a per-item deadline must not end the round")
	assert.True(t, errors.Is(m1Err, context.DeadlineExceeded), "got %v", m1Err)
	assert.True(t, m2HasDeadline && m2Live, "m2 must run under its own, unexpired deadline")
	// Load-independent (review L6): a fresh deadline is strictly LATER than the
	// one that already fired, whatever the scheduler did in between.
	assert.True(t, m2Deadline.After(m1Deadline), "m2 deadline %v must be after m1's %v", m2Deadline, m1Deadline)
}

// AC #4: the default is the package constant; a non-positive override is ignored
// (the WithAutoMaxPerRun shape).
func TestAutoGenerator_ItemTimeoutDefaultsToTheConstant(t *testing.T) {
	h := newAutoHarness(t, WithAutoItemTimeout(0))
	assert.Equal(t, AutoGenerationItemTimeout, h.gen.itemTimeout)
	assert.Equal(t, 15*time.Minute, AutoGenerationItemTimeout,
		"must stay above defaultExtractTimeout (10m) + ffprobe (10s) so the subprocess deadlines fire first")
}

// AC #5: a shutdown-cancelled failure does not count toward parking.
func TestAutoGenerator_CancelledFailuresDoNotCountTowardParking(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}
	// Pinned at the boundary (review L7): limit-1 genuine + 2 cancelled must
	// still enumerate — if cancelled rows were counted this would be limit+1.
	h.runs.runs = append(failedRuns("m1", autoFailureAttemptLimit-1), cancelledRun("m1"), cancelledRun("m1"))

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1"}, h.item.refIDs(),
		"restarts landing on the same long file must not park it — only genuine failures count")
}

// The other direction: cancelled rows do not RESCUE an item that has genuinely
// failed `autoFailureAttemptLimit` times.
func TestAutoGenerator_CancelledRowsDoNotRescueAParkedItem(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}
	h.runs.runs = append(failedRuns("m1", autoFailureAttemptLimit), cancelledRun("m1"), cancelledRun("m1"))

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.refIDs(), "genuine failures at the limit still park, whatever else is in the history")
}

// ─── bugfix-auto-exclusion-never-expires ───────────────────────────────────

func exclusionHarness(t *testing.T) *autoHarness {
	t.Helper()
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{autoMovieIn("m1", "lib-a", models.SubtitleStatusNotSearched)}
	return h
}

// AC #1: a deferred verdict holds only while the file is the one it was made on.
func TestAutoExclusion_ReplacedFileUnparksADeferredItem(t *testing.T) {
	h := exclusionHarness(t)
	h.runs.runs = []models.SubtitleRun{deferredRun("m1")}
	h.files.replaced("m1", autoNow.Add(time.Minute)) // newer than the run

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1"}, h.item.refIDs(),
		"a new remux may carry an embedded Chinese track — the paid verdict was about the OLD file")
}

func TestAutoExclusion_OlderFileKeepsADeferredItemParked(t *testing.T) {
	h := exclusionHarness(t)
	h.runs.runs = []models.SubtitleRun{deferredRun("m1")}
	h.files.replaced("m1", autoNow.Add(-time.Minute))

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.refIDs())
}

// AC #2: only failures AFTER the current file's mtime count.
func TestAutoExclusion_ReplacedFileResetsTheFailureCount(t *testing.T) {
	cases := []struct {
		name      string
		replaced  time.Duration // mtime relative to autoNow
		failedAts []time.Duration
		want      []string
	}{
		{"all failures predate the file", time.Minute, []time.Duration{-3 * time.Minute, -2 * time.Minute, -time.Minute}, []string{"m1"}},
		{"enough failures predate the file to drop below the limit", -90 * time.Second, []time.Duration{-3 * time.Minute, -2 * time.Minute, -time.Minute}, []string{"m1"}},
		{"the limit is reached after the file", -10 * time.Minute, []time.Duration{-3 * time.Minute, -2 * time.Minute, -time.Minute}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := exclusionHarness(t)
			for _, d := range tc.failedAts {
				h.runs.runs = append(h.runs.runs, withStartedAt(failedRun("m1"), autoNow.Add(d)))
			}
			h.files.replaced("m1", autoNow.Add(tc.replaced))

			h.gen.Run(context.Background())

			assert.ElementsMatch(t, tc.want, h.item.refIDs())
		})
	}
}

// AC #3: rows from before the free lane's current epoch are stale verdicts.
func TestAutoExclusion_RowsBeforeTheEpochDoNotCount(t *testing.T) {
	before := freeLaneEpoch.Add(-time.Hour)
	t.Run("deferred", func(t *testing.T) {
		h := exclusionHarness(t)
		h.runs.runs = []models.SubtitleRun{withStartedAt(deferredRun("m1"), before)}
		h.gen.Run(context.Background())
		assert.Equal(t, []string{"m1"}, h.item.refIDs(), "an upgrade may have made this item free")
	})
	t.Run("failed", func(t *testing.T) {
		h := exclusionHarness(t)
		for i := 0; i < autoFailureAttemptLimit; i++ {
			h.runs.runs = append(h.runs.runs, withStartedAt(failedRun("m1"), before))
		}
		h.gen.Run(context.Background())
		assert.Equal(t, []string{"m1"}, h.item.refIDs())
	})
	t.Run("rows after the epoch still count", func(t *testing.T) {
		h := exclusionHarness(t)
		h.runs.runs = []models.SubtitleRun{deferredRun("m1")}
		h.gen.Run(context.Background())
		assert.Empty(t, h.item.refIDs())
	})
}

// AC #4: cannot tell → stays parked (re-probing a vanished file every scan is
// H1's starvation again).
func TestAutoExclusion_StatFailureKeepsTheItemParked(t *testing.T) {
	h := exclusionHarness(t)
	h.runs.runs = []models.SubtitleRun{deferredRun("m1")}
	h.files.err = errors.New("stale NFS handle")

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.refIDs())
}

func TestAutoExclusion_EmptyPathKeepsTheItemParked(t *testing.T) {
	h := exclusionHarness(t)
	h.movies.movies[0].FilePath = models.NullString{}
	h.runs.runs = []models.SubtitleRun{deferredRun("m1")}

	h.gen.Run(context.Background())

	assert.Empty(t, h.item.refIDs())
	assert.Empty(t, h.files.stats, "nothing to stat")
}

// AC #5: the stat happens only for items that are parked AND would otherwise
// be selected — never for un-parked items, never for ineligible ones.
func TestAutoExclusion_StatIsBoundedToParkedEligibleItems(t *testing.T) {
	h := newAutoHarness(t)
	h.policy.enabled = autoEnabledSet("lib-a")
	h.movies.movies = []models.Movie{
		autoMovieIn("m-parked", "lib-a", models.SubtitleStatusNotSearched),
		autoMovieIn("m-parked-other-lib", "lib-b", models.SubtitleStatusNotSearched),
		autoMovieIn("m-parked-found", "lib-a", models.SubtitleStatusFound),
		autoMovieIn("m-free", "lib-a", models.SubtitleStatusNotSearched),
	}
	h.episodes.episodes = []models.Episode{
		autoEpisodeIn("e-parked", "s-a", models.SubtitleStatusNotSearched),
		autoEpisodeIn("e-parked-other-lib", "s-b", models.SubtitleStatusNotSearched),
	}
	h.series.byID["s-a"] = autoSeriesIn("s-a", "lib-a")
	h.series.byID["s-b"] = autoSeriesIn("s-b", "lib-b")
	h.runs.runs = []models.SubtitleRun{
		deferredRun("m-parked"), deferredRun("m-parked-other-lib"), deferredRun("m-parked-found"),
		deferredRun("e-parked"), deferredRun("e-parked-other-lib"),
		skippedRun("m-free", "fail-closed: unusable stream"),
	}

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"/media/m-parked.mkv", "/media/e-parked.mkv"}, h.files.stats,
		"episodes are stat-ed only after the series→library filter, like movies after theirs")
	assert.Equal(t, []string{"m-free"}, h.item.refIDs())
}

// Review H1: the production lookup must notice a replacement that PRESERVED
// mtime (rsync -a, cp -p, *arr imports) — ctime moves on every such operation.
func TestFileChangedAt_PrefersInodeChangeTimeOverAPreservedMtime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "movie.mkv")
	require.NoError(t, os.WriteFile(path, []byte("v2"), 0o644))
	old := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(path, old, old)) // "rsync -a" kept the old mtime

	changed, err := fileChangedAt(path)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.ModTime().Before(time.Now().Add(-47*time.Hour)), "fixture: mtime really is old")
	assert.True(t, changed.After(time.Now().Add(-time.Minute)),
		"a file written a moment ago must report a recent change time even though its mtime was set back: got %v", changed)
}

// Review M2: a stat that hangs (D-state mount) must not hold the round — it
// times out and the item stays parked.
func TestAutoExclusion_HungStatTimesOutAndKeepsTheItemParked(t *testing.T) {
	h := exclusionHarness(t)
	h.runs.runs = []models.SubtitleRun{deferredRun("m1")}
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	ctx, cancel := context.WithCancel(context.Background())
	var statted atomic.Bool
	// The "hung" stat cancels the round itself and then never returns — so
	// the round provably reached the stat, and the ctx ceiling (cheaper to
	// exercise than autoStatTimeout) is what frees it.
	h.gen.modTime = func(string) (time.Time, error) { statted.Store(true); cancel(); <-block; return autoNow, nil }

	h.gen.Run(ctx)

	assert.True(t, statted.Load())
	assert.Empty(t, h.item.refIDs())
}

// AC #6: #263's cancelled rows stay out of the count regardless of mtime.
func TestAutoExclusion_CancelledRowsStayExemptAfterReplacement(t *testing.T) {
	h := exclusionHarness(t)
	h.runs.runs = append(failedRuns("m1", autoFailureAttemptLimit-1), cancelledRun("m1"), cancelledRun("m1"))
	h.files.replaced("m1", autoNow.Add(-10*time.Minute))

	h.gen.Run(context.Background())

	assert.Equal(t, []string{"m1"}, h.item.refIDs())
}
