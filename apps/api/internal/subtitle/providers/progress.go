package providers

import (
	"context"
	"fmt"
	"io"
	"time"

	"golang.org/x/time/rate"
)

// ProgressFunc receives how far a download has got, 0–1. Called from the
// downloading goroutine; must not block.
type ProgressFunc func(fraction float64)

type progressKey struct{}

// WithProgress attaches a progress callback to ctx for a provider's Download
// (disc-2026-11-online-sub-download-progress). A provider that never reports
// leaves the caller's own stage events as the only signal.
func WithProgress(ctx context.Context, fn ProgressFunc) context.Context {
	return context.WithValue(ctx, progressKey{}, fn)
}

// ReportProgress calls the ctx's progress callback, if any, clamped to 0–1.
// Providers call it from Download; a ctx without a callback ignores it.
func ReportProgress(ctx context.Context, fraction float64) {
	fn, _ := ctx.Value(progressKey{}).(ProgressFunc)
	if fn == nil {
		return
	}
	fn(min(max(fraction, 0), 1))
}

// progressReader reports bytes read as a fraction between from and to of the
// whole download, when the body's size is known.
type progressReader struct {
	r        io.Reader
	ctx      context.Context
	total    int64
	read     int64
	from, to float64
	lastStep int
}

func (pr *progressReader) Read(b []byte) (int, error) {
	n, err := pr.r.Read(b)
	pr.read += int64(n)
	if pr.total > 0 {
		// Report in 5% steps of this span so a fast body doesn't flood SSE.
		step := int(float64(pr.read) / float64(pr.total) * 20)
		if step != pr.lastStep {
			pr.lastStep = step
			ReportProgress(pr.ctx, pr.from+(pr.to-pr.from)*float64(pr.read)/float64(pr.total))
		}
	}
	return n, err
}

// waitWithProgress waits for a rate-limiter slot like Limiter.Wait, reporting
// from→to as the wait elapses. Assrt allows 5 calls a minute, so a season
// pack spends most of its ~30 s here; without this the ring would sit still
// for 13 s at a time and read as stuck.
func waitWithProgress(ctx context.Context, lim *rate.Limiter, from, to float64) error {
	r := lim.Reserve()
	if !r.OK() {
		return fmt.Errorf("rate limiter: request exceeds burst")
	}
	delay := r.Delay()
	if delay <= 0 {
		ReportProgress(ctx, to)
		return nil
	}
	start := time.Now()
	timer := time.NewTimer(delay)
	defer timer.Stop()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			r.Cancel()
			return ctx.Err()
		case <-timer.C:
			ReportProgress(ctx, to)
			return nil
		case <-tick.C:
			ReportProgress(ctx, from+(to-from)*float64(time.Since(start))/float64(delay))
		}
	}
}
