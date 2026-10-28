package services

import (
	"context"
	"time"
)

// SoloJobSnapshot is one ad-hoc 生成字幕 click as the Activity page lists it
// (disc-2026-10-activity-solo-jobs-collapsed): one row per job, never one row
// for all of them. Both solo sources fill it — subtitle.SoloRunner (pipeline
// mode) and TranscriptionService's Solo jobs (legacy mode).
type SoloJobSnapshot struct {
	MediaID string
	Title   string
	// StartedAt is when the click was accepted.
	StartedAt time.Time
	// Queued is true while the job is blocked waiting for the process-wide
	// extraction slot (subtitle-track demux or audio extraction — see
	// WithExtractWaitObserver). QueuedAt is when that wait began; zero when
	// the job is not waiting.
	Queued   bool
	QueuedAt time.Time
	// SpentUSD is what the job has spent so far — its ai.Budget's recorded
	// figure, raw (disc-2026-10-solo-live-spend). nil = the job has no meter
	// yet (a legacy run before its Budget is attached); 0 = nothing spent.
	SpentUSD *float64
}

// extractWaitObserverKey carries a job's "am I waiting for the extraction
// slot?" observer down to the places that wait: the subtitle extractor's
// ExtractGate and the audio extractor's semaphore + gate. The job owner (the
// solo runner, a legacy solo transcription) attaches it to the job's ctx; the
// waiters know no job, only their ctx — the same shape as the subtitle
// package's WithExtractWaitNotifier narrator, but job-level and boolean.
type extractWaitObserverKey struct{}

// WithExtractWaitObserver attaches fn, called with true when work under this
// ctx starts waiting for an extraction slot and with false once it holds the
// slot or gave up. Calls may repeat (false while not waiting is harmless).
func WithExtractWaitObserver(ctx context.Context, fn func(waiting bool)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, extractWaitObserverKey{}, fn)
}

// ReportExtractWait tells ctx's observer, if any, whether its work is
// waiting for an extraction slot right now.
func ReportExtractWait(ctx context.Context, waiting bool) {
	if fn, ok := ctx.Value(extractWaitObserverKey{}).(func(waiting bool)); ok {
		fn(waiting)
	}
}

// ExtractWaitReporter adapts ctx's observer to the ExtractGate's onWait
// narrator shape (ahead >= 1: queued; 0: holds the slot). nil when ctx
// carries no observer, so the gate skips the call entirely.
func ExtractWaitReporter(ctx context.Context) func(ahead int) {
	fn, ok := ctx.Value(extractWaitObserverKey{}).(func(waiting bool))
	if !ok {
		return nil
	}
	return func(ahead int) { fn(ahead > 0) }
}
