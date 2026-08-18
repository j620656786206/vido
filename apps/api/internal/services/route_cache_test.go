package services

// Story sub-5-4 — the route cache.
//
// The value proposition is falsifiable in one sentence: a repeat analysis of an
// untouched library must issue ZERO probes. Every test below either proves that
// or proves the invalidation that makes it safe — a cache that never misses
// when the file changed would be worse than no cache at all.

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// ─── Fakes ─────────────────────────────────────────────────────────────────

// fakeRouteCache counts its reads so the batch-shape assertion (AC #5) is
// about the QUERY COUNT, not about timings that would pass on a small fixture.
type fakeRouteCache struct {
	entries  map[string]string
	getKeys  [][]string
	getErr   error
	sets     map[string]string
	setCalls int
	setTTLs  []time.Duration
	setErr   error
}

func newFakeRouteCache() *fakeRouteCache {
	return &fakeRouteCache{entries: map[string]string{}, sets: map[string]string{}}
}

func (c *fakeRouteCache) GetMany(_ context.Context, keys []string) (map[string]string, error) {
	c.getKeys = append(c.getKeys, append([]string(nil), keys...))
	if c.getErr != nil {
		return nil, c.getErr
	}
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := c.entries[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func (c *fakeRouteCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.setCalls++
	c.setTTLs = append(c.setTTLs, ttl)
	if c.setErr != nil {
		return c.setErr
	}
	c.sets[key] = value
	c.entries[key] = value
	return nil
}

// getCalls is the number of GetMany round trips — one per Analyze is the
// contract (sub-1-5b CR M3: per-row Gets are the N+1 this cache must not
// reintroduce).
func (c *fakeRouteCache) getCalls() int { return len(c.getKeys) }

// stubFileIdentity is the injectable os.Stat. Tests own the size/mtime pair
// outright, so invalidation can be asserted without touching a real disk.
type stubFileIdentity struct {
	sizes  map[string]int64
	mtimes map[string]int64
	calls  int
	err    error
}

func (s *stubFileIdentity) identity(path string) (int64, int64, error) {
	s.calls++
	if s.err != nil {
		return 0, 0, s.err
	}
	return s.sizes[path], s.mtimes[path], nil
}

// routeCacheFixture wires one episode-only library (episodes are the probe-bound
// class — their table has no tech-info columns at all) against a fake cache.
type routeCacheFixture struct {
	svc   *GenerationCandidateService
	cache *fakeRouteCache
	pred  *stubPredictor
	stat  *stubFileIdentity
	logs  *bytes.Buffer
}

func newRouteCacheFixture(t *testing.T, episodes []models.Episode, route RoutePrediction) *routeCacheFixture {
	t.Helper()

	cache := newFakeRouteCache()
	pred := &stubPredictor{probeRoute: route}
	stat := &stubFileIdentity{sizes: map[string]int64{}, mtimes: map[string]int64{}}
	for _, e := range episodes {
		stat.sizes[e.FilePath.String] = 1000
		stat.mtimes[e.FilePath.String] = 1700000000
	}

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	svc := NewGenerationCandidateService(
		nil, &stubEpisodeFinder{episodes: episodes}, nil, pred, false, 0, logger)
	svc.SetRouteCache(cache)
	svc.fileIdentity = stat.identity

	return &routeCacheFixture{svc: svc, cache: cache, pred: pred, stat: stat, logs: &logs}
}

func routeCacheEpisodes(n int) []models.Episode {
	out := make([]models.Episode, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, episodeRow(fmt.Sprintf("e%d", i), 1, i, fmt.Sprintf("/tv/s01e%02d.mkv", i), 45))
	}
	return out
}

// ─── AC #1 / Task 1: the key ───────────────────────────────────────────────

func TestRouteCacheKey_EveryFieldParticipates(t *testing.T) {
	base := routeCacheKey(1, "e1", 1000, 1700000000)

	assert.Equal(t, "subtitle:route:v1:e1:1000:1700000000", base,
		"the key format is the [@contract-v1] surface — anything reading or writing this family depends on it")

	assert.NotEqual(t, base, routeCacheKey(2, "e1", 1000, 1700000000), "routeVersion must participate")
	assert.NotEqual(t, base, routeCacheKey(1, "e2", 1000, 1700000000), "mediaID must participate")
	assert.NotEqual(t, base, routeCacheKey(1, "e1", 1001, 1700000000), "fileSize must participate")
	assert.NotEqual(t, base, routeCacheKey(1, "e1", 1000, 1700000001), "mtime must participate")
}

// ─── AC #5: file change ⇒必然重探 ────────────────────────────────────────────

func TestRouteCache_UnchangedFileHits(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(1), RouteASR)
	f.cache.entries[routeCacheKey(routeVersion, "e1", 1000, 1700000000)] = string(RouteASR)

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Empty(t, f.pred.probed, "an untouched file must not be probed again")
	require.Len(t, res.Candidates, 1)
	assert.Equal(t, RouteASR, res.Candidates[0].Route)
}

func TestRouteCache_SizeChangeMisses(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(1), RouteExtract)
	f.cache.entries[routeCacheKey(routeVersion, "e1", 1000, 1700000000)] = string(RouteASR)
	f.stat.sizes["/tv/s01e01.mkv"] = 2000 // re-encoded in place

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"/tv/s01e01.mkv"}, f.pred.probed, "a resized file must be re-probed")
	require.Len(t, res.Candidates, 1)
	assert.Equal(t, RouteExtract, res.Candidates[0].Route, "the fresh verdict wins, not the stale entry")
}

func TestRouteCache_MtimeChangeMisses(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(1), RouteExtract)
	f.cache.entries[routeCacheKey(routeVersion, "e1", 1000, 1700000000)] = string(RouteASR)
	f.stat.mtimes["/tv/s01e01.mkv"] = 1700000001 // same bytes count, new content

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"/tv/s01e01.mkv"}, f.pred.probed, "a touched file must be re-probed")
	require.Len(t, res.Candidates, 1)
	assert.Equal(t, RouteExtract, res.Candidates[0].Route)
}

// ─── AC #5: routeVersion bump ⇒ 全庫失效 ─────────────────────────────────────

func TestRouteCache_RouteVersionBumpInvalidatesEveryEntry(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(1), RouteExtract)
	// Written by a build whose predicate was one version older: the verdict is
	// no longer trustworthy, so it must not be readable by this build.
	f.cache.entries[routeCacheKey(routeVersion-1, "e1", 1000, 1700000000)] = string(RouteASR)

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"/tv/s01e01.mkv"}, f.pred.probed,
		"a predicate-version bump must strand every entry written under the old semantics")
	require.Len(t, res.Candidates, 1)
	assert.Equal(t, RouteExtract, res.Candidates[0].Route)
}

// ─── AC #3: 探測失敗絕不快取 (the red line) ──────────────────────────────────

func TestRouteCache_ProbeFailureIsNeverCached(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(1), RouteASR)
	f.pred.probeErr = fmt.Errorf("i/o timeout")

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, res.Candidates, "an unreadable file is not quotable")
	assert.Empty(t, f.cache.sets, "freezing a transient I/O error would delete this file from the list for 30 days")

	// The next sweep must still try — that is the whole point of not caching it.
	f.pred.probeErr = nil
	f.pred.probed = nil
	res, err = f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"/tv/s01e01.mkv"}, f.pred.probed, "the failed item must be re-probed next time")
	require.Len(t, res.Candidates, 1)
}

func TestRouteCache_SkippedRouteIsCached(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(1), RouteSkipped)

	_, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	key := routeCacheKey(routeVersion, "e1", 1000, 1700000000)
	assert.Equal(t, string(RouteSkipped), f.cache.sets[key],
		"skip is a trustworthy verdict, and skipped items are exactly the ones worth never probing twice")
}

func TestRouteCache_CachedSkipHitStillCountsAsSkipped(t *testing.T) {
	// CR sub-5-4 M2: skip is the one hit branch that changes the SUMMARY the
	// user reads (SkippedCount, and absence from Candidates) — "hit and fresh
	// land in the same switch" must be pinned, not assumed.
	f := newRouteCacheFixture(t, routeCacheEpisodes(1), RouteASR)
	f.cache.entries[routeCacheKey(routeVersion, "e1", 1000, 1700000000)] = string(RouteSkipped)

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Empty(t, f.pred.probed, "a cached skip must not be re-probed")
	assert.Empty(t, res.Candidates, "skipped items are never quotable, cached or fresh")
	assert.Equal(t, 1, res.Summary.SkippedCount, "the numbers must still add up when the verdict came from the cache")
	assert.Zero(t, f.cache.setCalls, "a hit has nothing new to write")
}

// ─── AC #2 / #5: batch read, zero probes on a full hit ─────────────────────

func TestRouteCache_ReadsTheWholeBatchInOneQuery(t *testing.T) {
	const n = 50
	f := newRouteCacheFixture(t, routeCacheEpisodes(n), RouteASR)

	_, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	require.Equal(t, 1, f.cache.getCalls(), "one query for the sweep — per-row Gets are the sub-1-5b N+1")
	assert.Len(t, f.cache.getKeys[0], n, "every probe-bound row belongs to the single batch")
}

func TestRouteCache_FullHitIssuesZeroProbes(t *testing.T) {
	const n = 20
	episodes := routeCacheEpisodes(n)
	f := newRouteCacheFixture(t, episodes, RouteASR)
	for _, e := range episodes {
		f.cache.entries[routeCacheKey(routeVersion, e.ID, 1000, 1700000000)] = string(RouteExtract)
	}

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Empty(t, f.pred.probed, "the story's value proposition: a repeat sweep of an untouched library probes nothing")
	assert.Equal(t, n, res.Summary.ExtractCount, "cached verdicts must price identically to fresh ones")
	assert.Zero(t, f.cache.setCalls, "a hit has nothing new to write")
}

// ─── AC #2: the persisted-tracks fast path still outranks the cache ────────

func TestRouteCache_PersistedTracksBypassTheCacheEntirely(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow("m1", "Enriched", "/m/a.mkv", 120, `[{"language":"eng","format":"subrip","stream_index":2}]`),
	}}
	cache := newFakeRouteCache()
	pred := &stubPredictor{fromTracks: RouteExtract, probeRoute: RouteASR}

	svc := NewGenerationCandidateService(movies, nil, nil, pred, false, 0, nil)
	svc.SetRouteCache(cache)
	svc.fileIdentity = (&stubFileIdentity{
		sizes:  map[string]int64{"/m/a.mkv": 10},
		mtimes: map[string]int64{"/m/a.mkv": 20},
	}).identity

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Zero(t, cache.getCalls(), "a row that needs no disk access needs no cache lookup either")
	assert.Zero(t, cache.setCalls, "and nothing to store")
	assert.Empty(t, pred.probed)
	require.Len(t, res.Candidates, 1)
	assert.Equal(t, RouteExtract, res.Candidates[0].Route)
}

// ─── AC #2: F14's denominator is untouched ─────────────────────────────────

func TestRouteCache_ProgressStillReportsEveryRow(t *testing.T) {
	const n = 5
	episodes := routeCacheEpisodes(n)
	f := newRouteCacheFixture(t, episodes, RouteASR)
	for _, e := range episodes {
		f.cache.entries[routeCacheKey(routeVersion, e.ID, 1000, 1700000000)] = string(RouteASR)
	}

	var seen [][2]int
	_, err := f.svc.Analyze(context.Background(), func(done, total int) {
		seen = append(seen, [2]int{done, total})
	})
	require.NoError(t, err)

	require.Len(t, seen, n, "a hit is faster, not invisible — a shrinking denominator reads as lost items")
	assert.Equal(t, [2]int{1, n}, seen[0])
	assert.Equal(t, [2]int{n, n}, seen[n-1])
}

// ─── AC #5: storage failures are never the analysis's problem ──────────────

func TestRouteCache_WriteFailureDoesNotFailTheAnalysis(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(2), RouteASR)
	f.cache.setErr = fmt.Errorf("database is locked")

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err, "a cache write failure costs speed, never correctness")
	require.Len(t, res.Candidates, 2)
	assert.Equal(t, 2, res.Summary.ASRCount)
}

func TestRouteCache_ReadFailureFallsBackToProbing(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(2), RouteASR)
	f.cache.getErr = fmt.Errorf("database is locked")

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, f.pred.probed, 2, "an unreadable cache degrades to the old behaviour, it does not fail the sweep")
	require.Len(t, res.Candidates, 2)
}

func TestRouteCache_UnstattableFileSkipsTheCacheButStillProbes(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(1), RouteASR)
	f.stat.err = fmt.Errorf("permission denied")

	res, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Zero(t, f.cache.getCalls(), "no identity means no key — inventing one would make every unstattable file share an entry")
	assert.Zero(t, f.cache.setCalls)
	assert.Len(t, f.pred.probed, 1)
	require.Len(t, res.Candidates, 1)
}

// ─── AC #4: the hit rate is observable ─────────────────────────────────────

func TestRouteCache_LogsHitAndProbeCounts(t *testing.T) {
	episodes := routeCacheEpisodes(3)
	f := newRouteCacheFixture(t, episodes, RouteASR)
	f.cache.entries[routeCacheKey(routeVersion, "e1", 1000, 1700000000)] = string(RouteASR)
	f.cache.entries[routeCacheKey(routeVersion, "e2", 1000, 1700000000)] = string(RouteASR)

	_, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	out := f.logs.String()
	require.Contains(t, out, "generation candidate analysis complete")
	assert.Contains(t, out, "route_cache_hits=2", "without the counters, \"did the cache work?\" is unanswerable")
	assert.Contains(t, out, "route_probes=1")
	assert.True(t, strings.Contains(out, "candidates=3"), "the existing fields must survive untouched")
}

// ─── AC #1: the TTL actually reaches storage (CR sub-5-4 M1) ───────────────

func TestRouteCache_WritesCarryThePositiveTTL(t *testing.T) {
	// The real repository REJECTS ttl <= 0, and storeRoute fail-softs that
	// rejection down to a Debug log — a broken routeCacheTTL would disable the
	// whole cache silently, forever. This is the story's own named
	// worst-failure class (silent), so the constant's journey to Set is pinned.
	f := newRouteCacheFixture(t, routeCacheEpisodes(2), RouteASR)

	_, err := f.svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	require.Len(t, f.cache.setTTLs, 2)
	for _, ttl := range f.cache.setTTLs {
		assert.Equal(t, routeCacheTTL, ttl, "a zero or wrong TTL would be rejected by the real repository and silently disable the cache")
	}
	assert.Positive(t, routeCacheTTL)
}

// ─── L1: cancellation is honoured during the stat pass ─────────────────────

func TestRouteCache_CancelledContextStopsTheStatSweep(t *testing.T) {
	f := newRouteCacheFixture(t, routeCacheEpisodes(50), RouteASR)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.svc.Analyze(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, f.stat.calls, "os.Stat takes no ctx — the plan loop's own check is the only thing keeping cancel responsive here")
	assert.Empty(t, f.pred.probed)
}

// ─── Adapter: the repository bridge (CR sub-5-4 M1) ────────────────────────

// fakeCacheRepo implements the full repository.CacheRepositoryInterface so the
// shipped adapter — not just the port — is under test: the type tag is what
// ClearByType eviction keys on, and the value mapping is what a hit returns.
type fakeCacheRepo struct {
	entries   map[string]*repository.CacheEntry
	lastType  string
	lastTTL   time.Duration
	manyCalls int
}

func (r *fakeCacheRepo) Get(context.Context, string) (*repository.CacheEntry, error) { return nil, nil }

func (r *fakeCacheRepo) GetMany(_ context.Context, keys []string) (map[string]*repository.CacheEntry, error) {
	r.manyCalls++
	out := map[string]*repository.CacheEntry{}
	for _, k := range keys {
		if e, ok := r.entries[k]; ok {
			out[k] = e
		}
	}
	return out, nil
}

func (r *fakeCacheRepo) Set(_ context.Context, key, value, cacheType string, ttl time.Duration) error {
	if r.entries == nil {
		r.entries = map[string]*repository.CacheEntry{}
	}
	r.entries[key] = &repository.CacheEntry{Key: key, Value: value, Type: cacheType}
	r.lastType, r.lastTTL = cacheType, ttl
	return nil
}

func (r *fakeCacheRepo) Delete(context.Context, string) error               { return nil }
func (r *fakeCacheRepo) Clear(context.Context) error                        { return nil }
func (r *fakeCacheRepo) ClearExpired(context.Context) (int64, error)        { return 0, nil }
func (r *fakeCacheRepo) ClearByType(context.Context, string) (int64, error) { return 0, nil }

func TestRouteCacheRepository_TagsTheFamilyAndMapsValues(t *testing.T) {
	repo := &fakeCacheRepo{}
	cache := NewRouteCacheRepository(repo)

	require.NoError(t, cache.Set(context.Background(), "k1", "asr", routeCacheTTL))
	assert.Equal(t, routeCacheType, repo.lastType,
		"a wrong type tag would orphan the family from ClearByType eviction and audits")
	assert.Equal(t, routeCacheTTL, repo.lastTTL)

	values, err := cache.GetMany(context.Background(), []string{"k1", "k-missing"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"k1": "asr"}, values,
		"a miss is expressed by absence, and a hit returns the stored value verbatim")
}

// ─── Backward compatibility: no cache wired ⇒ exactly today's behaviour ────

func TestRouteCache_NilCacheKeepsProbingEveryRow(t *testing.T) {
	pred := &stubPredictor{probeRoute: RouteASR}
	svc := NewGenerationCandidateService(
		nil, &stubEpisodeFinder{episodes: routeCacheEpisodes(3)}, nil, pred, false, 0, nil)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, pred.probed, 3, "an unwired cache must not change a single verdict")
	require.Len(t, res.Candidates, 3)
}
