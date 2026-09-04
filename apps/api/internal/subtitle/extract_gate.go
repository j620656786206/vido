package subtitle

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
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

// Acquire takes the slot, blocking while another extraction holds it, and
// reports how long it had to wait. A nil gate is a no-op; a ctx that ends
// while waiting returns ctx.Err() immediately.
//
// onWait (optional) is the queue narrator, and it is called at most TWICE:
//   - once with ahead >= 1 when the slot is busy at entry — the holder plus
//     everyone already queued, so the caller can say「等待抽軌（前方 N 件）」;
//   - once with ahead == 0 the moment the slot is taken, so the caller can
//     replace that message. Without the second call the queue notice would
//     stay on screen for the whole extraction that followed it — the stalled
//     bubble this narration exists to remove (CR M2). It also covers the
//     race where the holder releases between the entry probe and the queue
//     join: onWait(1) has already gone out even though nothing was waited
//     for, and the 0 retracts it.
//
// The returned release is idempotent and must be called once the subprocess
// has exited.
func (g *ExtractGate) Acquire(ctx context.Context, onWait func(ahead int)) (release func(), waited time.Duration, err error) {
	if g == nil {
		return func() {}, 0, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	select {
	case g.slot <- struct{}{}:
		return g.releaser(), 0, nil
	default:
	}

	// Busy: join the queue and report what is ahead — the holder plus every
	// waiter that was already queued, i.e. the new queue depth exactly.
	ahead := int(g.waiting.Add(1))
	defer g.waiting.Add(-1)
	if onWait != nil {
		onWait(ahead)
	}

	start := time.Now()
	select {
	case g.slot <- struct{}{}:
		if onWait != nil {
			onWait(0)
		}
		return g.releaser(), time.Since(start), nil
	case <-ctx.Done():
		return nil, time.Since(start), ctx.Err()
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
