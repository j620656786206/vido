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
		select {
		case <-p.release:
		case <-ctx.Done():
			return nil, ctx.Err()
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
