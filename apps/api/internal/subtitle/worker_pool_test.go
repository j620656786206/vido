package subtitle

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// ─── Fakes ─────────────────────────────────────────────────────────────────

// blockingProcessor lets a test hold workers inside ProcessItem so queue and
// in-flight state can be observed mid-run rather than inferred.
type blockingProcessor struct {
	mu      sync.Mutex
	seen    []MediaRef
	entered chan MediaRef
	// release is closed exactly once by releaseAll; a closed channel receives
	// immediately, so every parked AND every future call proceeds. The fields
	// are otherwise immutable after construction — mutating them mid-test
	// would race the workers reading them.
	release   chan struct{}
	releaseOn sync.Once
	err       error
	blocking  bool
	// ignoreCtxCancel keeps a parked call parked THROUGH a ctx cancellation —
	// it simulates the failItem cleanup writes that run on WithoutCancel and
	// must complete even when the pool ctx is torn down (CR M2's test needs it).
	ignoreCtxCancel bool
}

func newBlockingProcessor(capacity int) *blockingProcessor {
	return &blockingProcessor{
		entered:  make(chan MediaRef, capacity),
		release:  make(chan struct{}),
		blocking: true,
	}
}

// releaseAll unparks every worker, now and in future. Idempotent.
func (p *blockingProcessor) releaseAll() {
	p.releaseOn.Do(func() { close(p.release) })
}

func (p *blockingProcessor) ProcessItem(ctx context.Context, ref MediaRef, _ ProcessItemOptions) (*ProcessOutcome, error) {
	p.mu.Lock()
	p.seen = append(p.seen, ref)
	p.mu.Unlock()

	select {
	case p.entered <- ref:
	default:
	}

	if p.blocking {
		if p.ignoreCtxCancel {
			<-p.release
		} else {
			select {
			case <-p.release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	if p.err != nil {
		return nil, p.err
	}
	return &ProcessOutcome{Kind: RouteTranslate}, nil
}

func (p *blockingProcessor) snapshot() []MediaRef {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]MediaRef(nil), p.seen...)
}

type fakeMovieFinder struct {
	movies []models.Movie
	err    error
}

func (f *fakeMovieFinder) FindMissingZhHantSubtitle(context.Context) ([]models.Movie, error) {
	return f.movies, f.err
}

type fakeEpisodeFinder struct {
	episodes []models.Episode
	err      error
}

func (f *fakeEpisodeFinder) FindMissingZhHantSubtitle(context.Context) ([]models.Episode, error) {
	return f.episodes, f.err
}

// ─── AC #3: lifecycle, dedup, overflow ─────────────────────────────────────

func TestWorkerPool_ProcessesEnqueuedItems(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false
	pool := NewWorkerPool(proc, nil)

	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(pool.Stop)

	assert.True(t, pool.Enqueue(MediaRef{ID: "m1", MediaType: "movie"}))
	assert.True(t, pool.Enqueue(MediaRef{ID: "ep1", MediaType: "episode"}))

	require.Eventually(t, func() bool { return len(proc.snapshot()) == 2 },
		2*time.Second, 5*time.Millisecond, "both items must reach ProcessItem")
}

// TestWorkerPool_ConcurrencyIsFixedAtTwo — AD #5 / NFR-P3. A NAS box is the
// deployment target; a third concurrent translate would blow the ≤1-core bound
// AC #7(a) commits to.
func TestWorkerPool_ConcurrencyIsFixedAtTwo(t *testing.T) {
	assert.Equal(t, 2, PipelineConcurrencyM1, "M1 ships fixed concurrency 2")

	proc := newBlockingProcessor(8)
	pool := NewWorkerPool(proc, nil)
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(func() { proc.releaseAll(); pool.Stop() })

	for i := 0; i < 5; i++ {
		require.True(t, pool.Enqueue(MediaRef{ID: fmt.Sprintf("m%d", i), MediaType: "movie"}))
	}

	// Exactly PipelineConcurrencyM1 workers may be inside ProcessItem at once.
	for i := 0; i < PipelineConcurrencyM1; i++ {
		select {
		case <-proc.entered:
		case <-time.After(2 * time.Second):
			t.Fatalf("worker %d never started", i)
		}
	}
	select {
	case ref := <-proc.entered:
		t.Fatalf("a third item (%s) started while 2 workers were still busy", ref.ID)
	case <-time.After(100 * time.Millisecond):
	}
}

// TestWorkerPool_DedupsItemsAlreadyQueuedOrRunning — Rule 14. A scan that fires
// twice in a minute must not queue the same episode twice and pay for it twice.
func TestWorkerPool_DedupsItemsAlreadyQueuedOrRunning(t *testing.T) {
	proc := newBlockingProcessor(8)
	pool := NewWorkerPool(proc, nil)
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(func() { proc.releaseAll(); pool.Stop() })

	ref := MediaRef{ID: "ep-77", MediaType: "episode"}
	assert.True(t, pool.Enqueue(ref), "first enqueue is accepted")
	assert.False(t, pool.Enqueue(ref), "the same ref must not be queued twice")

	// Still deduped once a worker has PICKED IT UP — "already running" counts.
	select {
	case <-proc.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the item never started")
	}
	assert.False(t, pool.Enqueue(ref), "an in-flight item must not be re-queued either")

	// A DIFFERENT item is unaffected.
	assert.True(t, pool.Enqueue(MediaRef{ID: "ep-78", MediaType: "episode"}))
}

// TestWorkerPool_ReEnqueueableAfterCompletion — dedup is in-flight-scoped, not
// permanent: a later scan must be able to retry an item that failed.
func TestWorkerPool_ReEnqueueableAfterCompletion(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false
	proc.err = errors.New("provider 500")
	pool := NewWorkerPool(proc, nil)
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(pool.Stop)

	ref := MediaRef{ID: "m1", MediaType: "movie"}
	require.True(t, pool.Enqueue(ref))
	require.Eventually(t, func() bool { return len(proc.snapshot()) == 1 }, 2*time.Second, 5*time.Millisecond)

	require.Eventually(t, func() bool { return pool.Enqueue(ref) }, 2*time.Second, 5*time.Millisecond,
		"once an item finishes it must be enqueueable again — otherwise a failure is permanent")
}

// TestWorkerPool_OverflowDropsAndWarns — fail-soft (AC #3). The next scan
// re-enqueues, so a drop costs a delay, never an item. Mirrors the SSE hub's
// drop discipline rather than blocking the scanner callback.
func TestWorkerPool_OverflowDropsAndWarns(t *testing.T) {
	proc := newBlockingProcessor(8)
	pool := NewWorkerPool(proc, nil, WithQueueCapacity(2))
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(func() { proc.releaseAll(); pool.Stop() })

	// Both workers park inside ProcessItem, so nothing drains the buffer.
	for i := 0; i < PipelineConcurrencyM1; i++ {
		require.True(t, pool.Enqueue(MediaRef{ID: fmt.Sprintf("busy%d", i), MediaType: "movie"}))
		select {
		case <-proc.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("worker never picked up its item")
		}
	}

	// Now fill the 2-slot buffer, then overflow it.
	assert.True(t, pool.Enqueue(MediaRef{ID: "m1", MediaType: "movie"}))
	assert.True(t, pool.Enqueue(MediaRef{ID: "m2", MediaType: "movie"}))
	assert.False(t, pool.Enqueue(MediaRef{ID: "m3", MediaType: "movie"}),
		"an overflowing enqueue drops rather than blocking the scan callback")

	assert.Equal(t, 2, pool.QueueDepth())
	// The dropped item must NOT linger in the dedup map, or a single overflow
	// would make it permanently un-re-enqueueable — a silent, invisible loss.
	assert.Equal(t, PipelineConcurrencyM1+2, pool.InFlightCount(),
		"2 running + 2 buffered; the dropped item is not among them")

	// Proof it is genuinely re-offerable: free the workers and the same ref lands.
	proc.releaseAll()
	require.Eventually(t, func() bool { return pool.Enqueue(MediaRef{ID: "m3", MediaType: "movie"}) },
		2*time.Second, 10*time.Millisecond,
		"a dropped item must be accepted again once there is room")
}

// TestWorkerPool_StopIsGracefulAndLeaksNoGoroutines — AC #3 lifecycle.
func TestWorkerPool_StopIsGracefulAndLeaksNoGoroutines(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false
	pool := NewWorkerPool(proc, nil)

	require.NoError(t, pool.Start(context.Background()))
	require.True(t, pool.IsRunning())
	require.True(t, pool.Enqueue(MediaRef{ID: "m1", MediaType: "movie"}))

	done := make(chan struct{})
	go func() { pool.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop must return once the workers have drained — a hung shutdown blocks the whole process")
	}
	assert.False(t, pool.IsRunning())

	assert.NotPanics(t, func() { pool.Stop() }, "Stop must be idempotent — the shutdown block may call it twice")
	assert.False(t, pool.Enqueue(MediaRef{ID: "m2", MediaType: "movie"}),
		"a stopped pool must refuse work rather than accept it into a queue nobody drains")
}

func TestWorkerPool_ContextCancellationStopsWorkers(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false
	pool := NewWorkerPool(proc, nil)

	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, pool.Start(ctx))
	cancel()

	require.Eventually(t, func() bool { return !pool.IsRunning() }, 3*time.Second, 10*time.Millisecond,
		"ctx cancellation must tear the pool down without an explicit Stop")
	pool.Stop() // idempotent
}

// TestWorkerPool_StopAfterContextCancelStillWaitsForInFlightWork — CR M2.
// main.go's shutdown cancels the pool ctx and THEN calls Stop; the cancel can
// flip `running` via an idle worker's stopFromWorker before Stop runs. Stop
// must still wait for the busy worker — returning early would race that item's
// failItem cleanup writes against the rest of the shutdown sequence.
func TestWorkerPool_StopAfterContextCancelStillWaitsForInFlightWork(t *testing.T) {
	proc := newBlockingProcessor(8) // blocking: the worker parks inside ProcessItem
	proc.ignoreCtxCancel = true     // ...and stays parked through the cancel, like a WithoutCancel cleanup write
	pool := NewWorkerPool(proc, nil)

	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, pool.Start(ctx))

	require.True(t, pool.Enqueue(MediaRef{ID: "m1", MediaType: "movie"}))
	select {
	case <-proc.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the item never started")
	}

	// The IDLE worker observes ctx.Done and flips `running`; the busy worker
	// stays parked inside ProcessItem (ignoreCtxCancel — a cleanup write that
	// must complete).
	cancel()
	require.Eventually(t, func() bool { return !pool.IsRunning() }, 2*time.Second, 5*time.Millisecond)

	stopReturned := make(chan struct{})
	go func() { pool.Stop(); close(stopReturned) }()

	select {
	case <-stopReturned:
		t.Fatal("Stop returned while a worker was still inside ProcessItem — the wg.Wait was skipped")
	case <-time.After(150 * time.Millisecond):
	}

	proc.releaseAll()
	select {
	case <-stopReturned:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop never returned after the in-flight item finished")
	}
}

// TestWorkerPool_StopDoesNotDrainBufferedItems — CR M3. A Go select picks
// randomly among ready cases, so without loop-head stop priority a worker could
// keep pulling buffered items after Stop was signalled — one multi-minute
// translate per pull, with Stop's wg.Wait held open the whole time.
func TestWorkerPool_StopDoesNotDrainBufferedItems(t *testing.T) {
	proc := newBlockingProcessor(8)
	pool := NewWorkerPool(proc, nil, WithQueueCapacity(4))
	require.NoError(t, pool.Start(context.Background()))

	// Park both workers, then buffer two more items nobody has picked up.
	for i := 0; i < PipelineConcurrencyM1; i++ {
		require.True(t, pool.Enqueue(MediaRef{ID: fmt.Sprintf("busy%d", i), MediaType: "movie"}))
		select {
		case <-proc.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("worker never picked up its item")
		}
	}
	require.True(t, pool.Enqueue(MediaRef{ID: "buffered1", MediaType: "movie"}))
	require.True(t, pool.Enqueue(MediaRef{ID: "buffered2", MediaType: "movie"}))

	stopReturned := make(chan struct{})
	go func() { pool.Stop(); close(stopReturned) }()
	// Give Stop time to close stopCh so the workers' loop-head check sees it.
	time.Sleep(50 * time.Millisecond)

	proc.releaseAll()
	select {
	case <-stopReturned:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop never returned")
	}

	assert.Len(t, proc.snapshot(), PipelineConcurrencyM1,
		"only the in-flight items ran — a stopping worker must not take new work from the buffer")
	assert.Equal(t, 2, pool.QueueDepth(),
		"the buffered items stay queued for a later Start, not silently consumed mid-shutdown")
}

// ─── AC #2: EnqueueMissing ─────────────────────────────────────────────────

func TestWorkerPool_EnqueueMissingEnumeratesMoviesAndEpisodes(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false

	movies := &fakeMovieFinder{movies: []models.Movie{{ID: "m1"}, {ID: "m2"}}}
	episodes := &fakeEpisodeFinder{episodes: []models.Episode{{ID: "ep1"}, {ID: "ep2"}, {ID: "ep3"}}}

	pool := NewWorkerPool(proc, nil, WithCandidateFinders(movies, episodes))
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(pool.Stop)

	queued, err := pool.EnqueueMissing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 5, queued, "movies AND episodes are both enumerated")

	require.Eventually(t, func() bool { return len(proc.snapshot()) == 5 }, 2*time.Second, 5*time.Millisecond)

	seen := map[string]string{}
	for _, ref := range proc.snapshot() {
		seen[ref.ID] = ref.MediaType
	}
	assert.Equal(t, "movie", seen["m1"])
	assert.Equal(t, "episode", seen["ep1"],
		"an episode is enqueued under the EPISODE vocabulary, not `series` (sub-1-2 AC #1)")
}

// TestWorkerPool_EnqueueMissingSkipsTerminalVerdicts — CR H1. `no_text_source`
// and `skipped` rows keep subtitle_language NULL forever, so the broad
// enumeration re-returns them on every scan; the P5 pre-flight cannot gate them
// (no sidecar exists), so without this filter every sweep would re-probe the
// file and append a fresh run row per permanently-declined item, without bound.
func TestWorkerPool_EnqueueMissingSkipsTerminalVerdicts(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false

	movies := &fakeMovieFinder{movies: []models.Movie{
		{ID: "m-eligible"},
		{ID: "m-not-found", SubtitleStatus: models.SubtitleStatusNotFound},
		{ID: "m-no-text", SubtitleStatus: models.SubtitleStatusNoTextSource},
		{ID: "m-skipped", SubtitleStatus: models.SubtitleStatusSkipped},
	}}
	episodes := &fakeEpisodeFinder{episodes: []models.Episode{
		{ID: "ep-eligible", SubtitleStatus: models.SubtitleStatusNotSearched},
		{ID: "ep-no-text", SubtitleStatus: models.SubtitleStatusNoTextSource},
		{ID: "ep-skipped", SubtitleStatus: models.SubtitleStatusSkipped},
	}}

	pool := NewWorkerPool(proc, nil, WithCandidateFinders(movies, episodes))
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(pool.Stop)

	queued, err := pool.EnqueueMissing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, queued, "terminal pipeline verdicts are filtered; everything else stays in scope")

	require.Eventually(t, func() bool { return len(proc.snapshot()) == 3 }, 2*time.Second, 5*time.Millisecond)
	seen := map[string]bool{}
	for _, ref := range proc.snapshot() {
		seen[ref.ID] = true
	}
	assert.True(t, seen["m-eligible"])
	assert.True(t, seen["m-not-found"],
		"`not_found` is a SEARCH verdict — the item still lacks zh-Hant and stays in generation scope")
	assert.True(t, seen["ep-eligible"])
	assert.False(t, seen["m-no-text"], "a permanently-declined movie must not re-run every scan")
	assert.False(t, seen["m-skipped"])
	assert.False(t, seen["ep-no-text"])
	assert.False(t, seen["ep-skipped"])
}

// TestWorkerPool_EnqueueMissingSurvivesOneFinderFailing — a broken episode query
// must not cost the movies their run (Rule 13 case 2: logged and reported, work
// continues).
func TestWorkerPool_EnqueueMissingSurvivesOneFinderFailing(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false

	movies := &fakeMovieFinder{movies: []models.Movie{{ID: "m1"}}}
	episodes := &fakeEpisodeFinder{err: errors.New("no such column")}

	pool := NewWorkerPool(proc, nil, WithCandidateFinders(movies, episodes))
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(pool.Stop)

	queued, err := pool.EnqueueMissing(context.Background())
	require.Error(t, err, "the failure is reported, not swallowed")
	assert.Equal(t, 1, queued, "the movies still got enqueued")
}

func TestWorkerPool_EnqueueMissingWithNoFindersIsANoOp(t *testing.T) {
	pool := NewWorkerPool(newBlockingProcessor(1), nil)
	queued, err := pool.EnqueueMissing(context.Background())
	require.NoError(t, err)
	assert.Zero(t, queued)
}

// ─── AC #4: the endpoint's `force` has to survive the queue ────────────────

// optionsRecorder captures the ProcessItemOptions each item was run with.
type optionsRecorder struct {
	mu   sync.Mutex
	seen map[MediaRef]ProcessItemOptions
	done chan struct{}
}

func newOptionsRecorder() *optionsRecorder {
	return &optionsRecorder{seen: map[MediaRef]ProcessItemOptions{}, done: make(chan struct{}, 8)}
}

func (r *optionsRecorder) ProcessItem(_ context.Context, ref MediaRef, opts ProcessItemOptions) (*ProcessOutcome, error) {
	r.mu.Lock()
	r.seen[ref] = opts
	r.mu.Unlock()
	r.done <- struct{}{}
	return &ProcessOutcome{Kind: RouteTranslate}, nil
}

func (r *optionsRecorder) optionsFor(ref MediaRef) ProcessItemOptions {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seen[ref]
}

// TestWorkerPool_EnqueueItemCarriesOptionsToProcessItem — FR32's re-run switch
// is a request field, so it has to ride the queue. A pool that only carried
// MediaRef would accept `force: true` and silently run the cached path.
func TestWorkerPool_EnqueueItemCarriesOptionsToProcessItem(t *testing.T) {
	rec := newOptionsRecorder()
	pool := NewWorkerPool(rec, nil)
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(pool.Stop)

	forced := MediaRef{ID: "m-forced", MediaType: models.SubtitleRunMediaMovie}
	plain := MediaRef{ID: "m-plain", MediaType: models.SubtitleRunMediaMovie}

	require.Equal(t, EnqueueAccepted, pool.EnqueueItem(forced, ProcessItemOptions{Force: true}))
	require.True(t, pool.Enqueue(plain))

	for i := 0; i < 2; i++ {
		select {
		case <-rec.done:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for both items")
		}
	}

	assert.True(t, rec.optionsFor(forced).Force, "force must reach ProcessItem unchanged")
	assert.False(t, rec.optionsFor(plain).Force, "the scanner path must never force")
}

// TestWorkerPool_EnqueueItemDedupsOnRefNotOptions — two requests for the same
// item are the SAME work regardless of `force`; deduping on the pair would let
// a double-click queue the item twice and translate it twice.
func TestWorkerPool_EnqueueItemDedupsOnRefNotOptions(t *testing.T) {
	proc := newBlockingProcessor(4)
	pool := NewWorkerPool(proc, nil, WithQueueCapacity(4))
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(func() { proc.releaseAll(); pool.Stop() })

	ref := MediaRef{ID: "m1", MediaType: models.SubtitleRunMediaMovie}
	require.Equal(t, EnqueueAccepted, pool.EnqueueItem(ref, ProcessItemOptions{Force: true}))
	assert.Equal(t, EnqueueDuplicate, pool.EnqueueItem(ref, ProcessItemOptions{}),
		"already queued — the second offer is rejected on the ref alone")
}

// TestWorkerPool_EnqueueItemReportsDistinctOutcomes — CR M1. The three
// rejection reasons are different truths: only a genuine duplicate may be
// answered "already_queued" upstream; a drop and a stopped pool must be
// distinguishable so the endpoint never promises work that was discarded.
func TestWorkerPool_EnqueueItemReportsDistinctOutcomes(t *testing.T) {
	proc := newBlockingProcessor(4)
	pool := NewWorkerPool(proc, nil, WithQueueCapacity(1))

	ref := MediaRef{ID: "m1", MediaType: models.SubtitleRunMediaMovie}
	assert.Equal(t, EnqueueStopped, pool.EnqueueItem(ref, ProcessItemOptions{}),
		"a pool that was never started accepts nothing")

	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(func() { proc.releaseAll(); pool.Stop() })

	// Park both workers so the 1-slot buffer is the only capacity left.
	for i := 0; i < PipelineConcurrencyM1; i++ {
		require.True(t, pool.Enqueue(MediaRef{ID: fmt.Sprintf("busy%d", i), MediaType: "movie"}))
		select {
		case <-proc.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("worker never picked up its item")
		}
	}

	require.Equal(t, EnqueueAccepted, pool.EnqueueItem(ref, ProcessItemOptions{}))
	assert.Equal(t, EnqueueDuplicate, pool.EnqueueItem(ref, ProcessItemOptions{}))
	assert.Equal(t, EnqueueQueueFull, pool.EnqueueItem(MediaRef{ID: "m2", MediaType: "movie"}, ProcessItemOptions{}),
		"the buffer is full and both workers are busy — this item was dropped, not queued")
}

// ─── AC #5: the capability gate, scanner-enqueue entry point ───────────────

// spyFinder records whether it was consulted at all — the assertion that the
// gate short-circuits BEFORE enumeration, not after.
type spyFinder struct {
	called bool
	movies []models.Movie
}

func (s *spyFinder) FindMissingZhHantSubtitle(context.Context) ([]models.Movie, error) {
	s.called = true
	return s.movies, nil
}

func TestWorkerPool_EnqueueMissingShortCircuitsWhenUnconfigured(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false
	finder := &spyFinder{movies: []models.Movie{{ID: "m1"}, {ID: "m2"}}}

	pool := NewWorkerPool(proc, nil,
		WithCandidateFinders(finder, nil),
		WithCapabilityGate(func() bool { return false }))
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(pool.Stop)

	queued, err := pool.EnqueueMissing(context.Background())

	require.NoError(t, err, "an unconfigured install is not an error state — it is a gated one")
	assert.Zero(t, queued, "no queued work that can only fail")
	assert.False(t, finder.called,
		"the gate runs BEFORE enumeration: an unconfigured install pays no query per scan")
	assert.Zero(t, pool.InFlightCount())
}

func TestWorkerPool_EnqueueMissingRunsWhenConfigured(t *testing.T) {
	proc := newBlockingProcessor(8)
	proc.blocking = false
	finder := &spyFinder{movies: []models.Movie{{ID: "m1"}}}

	pool := NewWorkerPool(proc, nil,
		WithCandidateFinders(finder, nil),
		WithCapabilityGate(func() bool { return true }))
	require.NoError(t, pool.Start(context.Background()))
	t.Cleanup(pool.Stop)

	queued, err := pool.EnqueueMissing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, queued, "the gate is a gate, not an off switch")
	assert.True(t, finder.called)
}
