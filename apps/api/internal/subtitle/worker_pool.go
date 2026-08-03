package subtitle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/vido/api/internal/models"
)

const (
	// PipelineConcurrencyM1 is the fixed worker count for M1 (AD #5, NFR-P3).
	//
	// Deliberately a constant, not a setting: the deployment target is a NAS
	// (DS920+, 4-core J4125) that is also serving media, and AC #7(a) commits
	// to a pipeline-attributable bound of ≤1 core. Two concurrent translates
	// already sit near that line; a knob would let an operator break the very
	// bar the M1 pilot exists to validate.
	PipelineConcurrencyM1 = 2

	// defaultQueueCapacity bounds the enqueue buffer. A full library scan of a
	// large collection can enumerate more than this; overflow DROPS rather
	// than blocks (see Enqueue) because the next scan re-enumerates and the P5
	// pre-flight makes re-enumeration free for completed items.
	defaultQueueCapacity = 1024
)

// MovieGenerationFinder enumerates movies needing zh-Hant generation.
// *repository.MovieRepository satisfies it (Story 9R-16 AC 4's query).
type MovieGenerationFinder interface {
	FindMissingZhHantSubtitle(ctx context.Context) ([]models.Movie, error)
}

// EpisodeGenerationFinder is the episode-grain counterpart.
//
// It did NOT exist before this story: 9R-16 shipped movies-only (its
// generationCandidateFinder takes only movie methods, and EpisodeRepository had
// no missing-subtitle query at all). Task 2's decision tree therefore fired its
// "absent ⇒ expand scope with a new AC" branch — see AC #11 and the Discovery
// Triage, Rule 24 lane ①.
type EpisodeGenerationFinder interface {
	FindMissingZhHantSubtitle(ctx context.Context) ([]models.Episode, error)
}

// WorkerPool runs ProcessItem over an enqueued work list at fixed concurrency
// (AC #3). One pool per process, owned by main.go and shut down with it.
//
// It is deliberately a GENERIC executor: it knows nothing about shows, prompt
// caches or routes. Per-show serialization is the Pipeline's D10 latch
// (sub-1-5b AC #5), which is an orchestrator concern.
type WorkerPool struct {
	item   ItemProcessor
	logger *slog.Logger

	// configured is the FR23 capability gate (AC #5). nil = always allowed.
	// It short-circuits the whole enqueue rather than each item, so an
	// unconfigured install logs once per scan instead of once per movie.
	configured func() bool

	movies   MovieGenerationFinder
	episodes EpisodeGenerationFinder

	queue chan MediaRef

	mu sync.Mutex
	// inFlight is the dedup set: an item queued OR running is not re-queued.
	// Bounded by construction (Rule 14) — at most queue capacity + worker
	// count entries can exist, because an entry is added only alongside a
	// successful queue send and removed when the item finishes.
	inFlight map[MediaRef]struct{}
	running  bool
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// WorkerPoolOption configures optional pool dependencies.
type WorkerPoolOption func(*WorkerPool)

// WithCandidateFinders wires the AC #2 enumeration sources. Either may be nil.
func WithCandidateFinders(movies MovieGenerationFinder, episodes EpisodeGenerationFinder) WorkerPoolOption {
	return func(p *WorkerPool) {
		p.movies = movies
		p.episodes = episodes
	}
}

// WithCapabilityGate installs the FR23 predicate (AC #5). Without a configured
// translation key the pipeline can only produce failed runs, so the gate stops
// the work before it is queued.
func WithCapabilityGate(configured func() bool) WorkerPoolOption {
	return func(p *WorkerPool) { p.configured = configured }
}

// WithQueueCapacity overrides the buffer size. Test-only in practice — the
// overflow path is unreachable in a unit test at the shipped 1024.
func WithQueueCapacity(capacity int) WorkerPoolOption {
	return func(p *WorkerPool) {
		if capacity > 0 {
			p.queue = make(chan MediaRef, capacity)
		}
	}
}

// NewWorkerPool builds the pool. logger may be nil.
func NewWorkerPool(item ItemProcessor, logger *slog.Logger, opts ...WorkerPoolOption) *WorkerPool {
	if logger == nil {
		logger = slog.Default()
	}
	p := &WorkerPool{
		item:     item,
		logger:   logger.With("component", "subtitle_pipeline_pool"),
		queue:    make(chan MediaRef, defaultQueueCapacity),
		inFlight: make(map[MediaRef]struct{}),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Start launches the workers. Idempotent; safe to call before any enqueue.
func (p *WorkerPool) Start(ctx context.Context) error {
	if p.item == nil {
		return errors.New("subtitle worker pool: no item processor wired")
	}

	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.stopCh = make(chan struct{})
	stopCh := p.stopCh
	p.mu.Unlock()

	for i := 0; i < PipelineConcurrencyM1; i++ {
		p.wg.Add(1)
		go func(worker int) {
			defer p.wg.Done()
			p.run(ctx, worker, stopCh)
		}(i)
	}

	p.logger.Info("subtitle pipeline pool started",
		"workers", PipelineConcurrencyM1, "queue_capacity", cap(p.queue))
	return nil
}

// Stop drains the workers and blocks until they have returned. Idempotent —
// main.go's graceful-shutdown block and a ctx cancellation can both reach it.
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	close(p.stopCh)
	p.mu.Unlock()

	p.wg.Wait()
	p.logger.Info("subtitle pipeline pool stopped")
}

// IsRunning reports whether the workers are live.
func (p *WorkerPool) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// QueueDepth reports buffered (not yet picked up) items.
func (p *WorkerPool) QueueDepth() int { return len(p.queue) }

// InFlightCount reports queued-or-running items — the Rule 14 bound assertion.
func (p *WorkerPool) InFlightCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.inFlight)
}

// Enqueue offers one item to the pool. It returns false when the item was NOT
// accepted, which happens for three distinct reasons, all fail-soft:
//
//   - the pool is stopped (nothing would ever drain the queue),
//   - the item is already queued or running (dedup, Rule 14),
//   - the queue is full (drop-and-warn — blocking here would stall the scanner
//     callback that called us, and the next scan re-enqueues).
func (p *WorkerPool) Enqueue(ref MediaRef) bool {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return false
	}
	if _, dup := p.inFlight[ref]; dup {
		p.mu.Unlock()
		return false
	}
	// Reserve BEFORE the send so two concurrent enqueues of the same ref cannot
	// both win the dedup check.
	p.inFlight[ref] = struct{}{}
	p.mu.Unlock()

	select {
	case p.queue <- ref:
		return true
	default:
		// Release the reservation: an item that never made it into the queue
		// must stay re-enqueueable, or a single overflow would lose it
		// permanently.
		p.mu.Lock()
		delete(p.inFlight, ref)
		p.mu.Unlock()

		p.logger.Warn("subtitle pipeline queue full — dropping item, the next scan will re-enqueue it",
			"media_id", ref.ID, "media_type", ref.MediaType, "queue_capacity", cap(p.queue))
		return false
	}
}

// EnqueueMissing enumerates every item eligible for generation and offers it to
// the pool (AC #2). Returns how many were accepted.
//
// Over-enumeration is SAFE and deliberate: ProcessItem's P5 pre-flight is the
// authoritative gate (sub-1-5b AC #2), so an item that already has an
// acceptable sidecar costs one stat + parse and writes no provenance row. That
// is why this uses the broad "no zh-Hant on record" predicate rather than
// trying to be clever about status values.
func (p *WorkerPool) EnqueueMissing(ctx context.Context) (int, error) {
	// FR23 gate BEFORE enumeration (AC #5): one log line for the whole scan,
	// not one per item, and no queued work that could only ever fail.
	if p.configured != nil && !p.configured() {
		p.logger.Info("subtitle pipeline enqueue skipped — no translation key configured")
		return 0, nil
	}

	var (
		queued int
		errs   []error
	)

	if p.movies != nil {
		movies, err := p.movies.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("enumerate movies missing zh-Hant subtitle: %w", err))
		}
		for _, m := range movies {
			if p.Enqueue(MediaRef{ID: m.ID, MediaType: models.SubtitleRunMediaMovie}) {
				queued++
			}
		}
	}

	// A failing movie query must not cost the episodes their run, and vice
	// versa — the two are independent enumerations over independent tables.
	if p.episodes != nil {
		episodes, err := p.episodes.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("enumerate episodes missing zh-Hant subtitle: %w", err))
		}
		for _, e := range episodes {
			if p.Enqueue(MediaRef{ID: e.ID, MediaType: models.SubtitleRunMediaEpisode}) {
				queued++
			}
		}
	}

	p.logger.Info("subtitle pipeline enqueue sweep complete", "queued", queued, "errors", len(errs))
	return queued, errors.Join(errs...)
}

// run is one worker's loop.
func (p *WorkerPool) run(ctx context.Context, worker int, stopCh <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			p.stopFromWorker()
			return
		case <-stopCh:
			return
		case ref := <-p.queue:
			p.process(ctx, worker, ref)
		}
	}
}

// process runs one item and always releases its dedup reservation.
func (p *WorkerPool) process(ctx context.Context, worker int, ref MediaRef) {
	defer func() {
		p.mu.Lock()
		delete(p.inFlight, ref)
		p.mu.Unlock()
	}()

	outcome, err := p.item.ProcessItem(ctx, ref, ProcessItemOptions{})
	if err != nil {
		// Rule 13 case 2: logged and halted FOR THIS ITEM. ProcessItem has
		// already recorded the failed run row and reverted the media row, so
		// the pool has nothing to add beyond visibility — and one poisoned
		// item must not take the worker down with it.
		p.logger.Error("subtitle pipeline item failed",
			"worker", worker, "media_id", ref.ID, "media_type", ref.MediaType, "error", err)
		return
	}
	p.logger.Debug("subtitle pipeline item done",
		"worker", worker, "media_id", ref.ID, "media_type", ref.MediaType,
		"kind", string(outcome.Kind), "subtitle_path", outcome.SubtitlePath)
}

// stopFromWorker flips the running flag when the pool is torn down by context
// cancellation rather than an explicit Stop, so IsRunning tells the truth and a
// later Stop is a no-op instead of a double-close panic.
func (p *WorkerPool) stopFromWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		p.running = false
		close(p.stopCh)
	}
}
