package services

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// usageReportCheckEvery is how often the scheduler asks the service whether a
// report is due. The service enforces the 7-day cadence; checking hourly only
// bounds how late the first report after turning it on can be (≤ 1 hour).
const usageReportCheckEvery = time.Hour

// usageReportTicker is the slice of UsageReportService the loop drives.
type usageReportTicker interface {
	Tick(ctx context.Context)
}

// UsageReportScheduler runs the weekly usage report check in the background
// (infra-optin-usage-report-a2). Rule 14: it takes a ctx, honors cancellation,
// and Stop waits for the loop to exit so main can close the DB after it.
type UsageReportScheduler struct {
	svc      usageReportTicker
	interval time.Duration

	mu      sync.Mutex
	stopCh  chan struct{}
	done    chan struct{}
	stopped bool
}

// NewUsageReportScheduler builds the scheduler for svc.
func NewUsageReportScheduler(svc usageReportTicker) *UsageReportScheduler {
	return &UsageReportScheduler{
		svc:      svc,
		interval: usageReportCheckEvery,
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start checks once right away, then every interval, until ctx is cancelled
// or Stop is called. Run it in its own goroutine.
func (s *UsageReportScheduler) Start(ctx context.Context) {
	defer close(s.done)
	slog.Info("Usage report scheduler started")
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.svc.Tick(ctx)
	for {
		select {
		case <-ctx.Done():
			slog.Info("Usage report scheduler stopped (context cancelled)")
			return
		case <-s.stopCh:
			slog.Info("Usage report scheduler stopped (stop signal)")
			return
		case <-ticker.C:
			s.svc.Tick(ctx)
		}
	}
}

// Stop signals the loop and waits until it has returned. Safe to call more
// than once, and safe if Start was never called.
func (s *UsageReportScheduler) Stop() {
	s.mu.Lock()
	if !s.stopped {
		s.stopped = true
		close(s.stopCh)
	}
	s.mu.Unlock()
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
	}
}
