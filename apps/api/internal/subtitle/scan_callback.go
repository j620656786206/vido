package subtitle

import "context"

// EnqueueObserver is notified of one sweep's outcome: how many items the pool
// accepted, and the joined enumeration error (nil when both queries succeeded).
// main.go passes a slog line; tests pass an assertion.
type EnqueueObserver func(queued int, err error)

// ComposeScanCallback builds the FR13 auto-trigger (AC #2) by WRAPPING the
// scanner's existing scan-complete callback rather than replacing it.
//
// `ScannerService.SetOnScanComplete` holds exactly ONE function, and main.go
// already occupies it with post-scan enrichment. Wiring FR13 by calling the
// setter a second time would compile, pass every test in this package, and
// silently stop enriching newly-scanned media — which is why this story edits
// ZERO lines of scanner_service.go (AC #10's delta-tree correction) and does
// the composition here, where it is testable.
//
// A nil pool IS legacy mode (main.go only builds the pool when
// VIDO_SUBTITLE_PIPELINE_MODE=pipeline), and then prev is returned untouched:
// no wrapper, no behavioural difference, nothing to reason about.
//
// The sweep runs INLINE, not in a goroutine: it is two indexed queries plus
// non-blocking channel sends (WorkerPool.Enqueue never blocks), and running it
// inline is what makes "enrichment first, then enqueue" an assertable ordering
// instead of a race. The long work happens on the pool's workers.
func ComposeScanCallback(prev func(), pool *WorkerPool, observe EnqueueObserver) func() {
	if pool == nil {
		if prev == nil {
			return func() {}
		}
		return prev
	}

	return func() {
		if prev != nil {
			prev()
		}
		queued, err := pool.EnqueueMissing(context.Background())
		if observe != nil {
			observe(queued, err)
		}
	}
}
