package subtitle

import (
	"context"
	"sync"
	"sync/atomic"
)

// ExtractGate serializes ffmpeg subtitle extraction process-wide (sub-6-3
// AC #2): capacity ONE, built once in main.go and shared by every Extractor
// (Rule 14), held only around the ffmpeg subprocess — never around ffprobe
// (a header read) and never around translation, so the two pipeline workers
// still translate concurrently.
//
// This REVERSES the extractor's original "deliberately NO internal semaphore"
// ruling. That ruling assumed extraction was bounded by the orchestrator's
// fixed concurrency of 2 (NFR-P3) and that a second bound would only halve
// it. eval-1 finding 7 measured the opposite on the owner's NAS: two workers
// demuxing two 20 GB remuxes at once fought over the same spindle, each ran
// past the 10-minute ceiling and BOTH failed, while either alone finished in
// 3.5 minutes. Two concurrent extractions are not twice the throughput; they
// are zero. One at a time is the bound that actually reflects the hardware.
type ExtractGate struct {
	slot    chan struct{}
	waiting atomic.Int32
}

// NewExtractGate returns a capacity-one gate.
func NewExtractGate() *ExtractGate {
	return &ExtractGate{slot: make(chan struct{}, 1)}
}

// Acquire takes the slot, blocking while another extraction holds it. When
// the slot is busy at entry, onWait (if non-nil) is called ONCE with the
// number of extractions ahead of this one — the holder plus everyone already
// queued — so the caller can say「等待抽軌（前方 N 件）」. A ctx that ends while
// waiting returns ctx.Err() immediately; a nil gate is a no-op.
//
// The returned release is idempotent and must be called exactly once the
// subprocess has exited.
func (g *ExtractGate) Acquire(ctx context.Context, onWait func(ahead int)) (release func(), err error) {
	if g == nil {
		return func() {}, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	select {
	case g.slot <- struct{}{}:
		return g.releaser(), nil
	default:
	}

	// Busy: join the queue and report what is ahead — the holder plus every
	// waiter that was already queued, i.e. the new queue depth exactly.
	ahead := int(g.waiting.Add(1))
	defer g.waiting.Add(-1)
	if onWait != nil {
		onWait(ahead)
	}

	select {
	case g.slot <- struct{}{}:
		return g.releaser(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Waiting reports how many callers are currently blocked in Acquire.
func (g *ExtractGate) Waiting() int {
	if g == nil {
		return 0
	}
	return int(g.waiting.Load())
}

func (g *ExtractGate) releaser() func() {
	var once sync.Once
	return func() { once.Do(func() { <-g.slot }) }
}

// extractWaitNotifierKey carries the item flow's「等待抽軌」narrator down to
// Extract through the ctx: the Extractor is built once and knows no MediaRef,
// while ProcessItem knows the ref and owns the progress hook. Same shape as
// processScope — a ctx value the package reads back, never a wider signature
// on the stamped TrackExtractor port.
type extractWaitNotifierKey struct{}

// WithExtractWaitNotifier attaches fn, called with the number of extractions
// ahead when this ctx's Extract has to queue behind the ExtractGate.
func WithExtractWaitNotifier(ctx context.Context, fn func(ahead int)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, extractWaitNotifierKey{}, fn)
}

func extractWaitNotifierFrom(ctx context.Context) func(ahead int) {
	fn, _ := ctx.Value(extractWaitNotifierKey{}).(func(ahead int))
	return fn
}
