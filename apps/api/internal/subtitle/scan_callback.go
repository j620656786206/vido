package subtitle

// ComposeScanCallback chains two scan-complete callbacks into the ONE slot the
// scanner provides.
//
// `ScannerService.SetOnScanComplete` holds exactly one function, and
// cmd/api/main.go already occupies it with post-scan metadata enrichment.
// Wiring a second concern by calling the setter again would compile, pass every
// test, and silently stop enriching newly-scanned media — a failure that would
// only surface weeks later as "my new files never get metadata". Composing here
// makes the ordering explicit and testable instead.
//
// HISTORY: an earlier version of this function shipped in sub-1-6 and was
// removed by sub-4-1 when the paid scan sweep was torn out (the comment at
// main.go:435 outlived it). 9R-10b restores the composition for the FREE lane
// only — the sweep it used to drive stays gone, and
// `internal/cost_consent_test.go` still enforces that.
//
// Composition is TRANSPARENT: it does not recover panics. A panic in `prev`
// today reaches the scanner's goroutine and takes the process down; wrapping it
// in a recover would change that, and would hide a real defect behind a feature
// whose whole point is that it costs nothing. `next` is therefore not reached
// if `prev` panics — the same outcome as today, where `prev` is the only
// callback there is.
func ComposeScanCallback(prev, next func()) func() {
	switch {
	case next == nil && prev == nil:
		return func() {}
	case next == nil:
		// No wrapper at all: the caller gets back exactly what it had, so a
		// build with the feature absent has nothing extra to reason about.
		return prev
	case prev == nil:
		return next
	}
	return func() {
		prev()
		next()
	}
}
