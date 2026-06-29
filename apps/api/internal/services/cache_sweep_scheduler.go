package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/vido/api/internal/repository"
)

const (
	// settingsKeyCacheSweepInterval is the settings key holding the sweep interval in minutes.
	settingsKeyCacheSweepInterval = "cache_sweep_interval_minutes"
	// defaultCacheSweepIntervalMinutes is used when the setting is unset/unreadable (aligned with the 1h cache TTL).
	defaultCacheSweepIntervalMinutes = 45
	// minCacheSweepIntervalMinutes is the safety floor; a positive value below it is clamped up
	// to prevent a misconfigured 1-min hammer on the SQLite writer lock.
	minCacheSweepIntervalMinutes = 5
	// maxCacheSweepIntervalMinutes is the safety ceiling (7 days). A value beyond this is clamped
	// down: without it, an absurd setting would overflow time.Duration's int64 nanoseconds
	// (~292 years) into a NEGATIVE duration, which makes time.NewTicker PANIC — and that panic
	// happens in Start (outside sweep's recover), so it would crash the whole process. 7 days is
	// far longer than any useful cadence for a 1h-TTL cache. (CR H1, 2026-06-29.)
	maxCacheSweepIntervalMinutes = 7 * 24 * 60
)

// CacheSweepSchedulerInterface defines the contract for the cache_entries expiry sweep scheduler.
type CacheSweepSchedulerInterface interface {
	Start(ctx context.Context)
	Stop()
}

// CacheSweepScheduler periodically deletes expired rows from the cache_entries SQLite table
// by calling CacheRepository.ClearExpired on a recurring ticker. It keeps the TMDb response
// cache from growing unbounded (especially once the Discover facet-counts fan-out write-amplifies
// it) without manual intervention. It targets cache_entries ONLY and never runs VACUUM.
type CacheSweepScheduler struct {
	cacheRepo    repository.CacheRepositoryInterface
	settingsRepo repository.SettingsRepositoryInterface
	mu           sync.Mutex
	stopCh       chan struct{}
	stopped      bool
}

// Compile-time interface verification
var _ CacheSweepSchedulerInterface = (*CacheSweepScheduler)(nil)

// NewCacheSweepScheduler creates a new CacheSweepScheduler.
func NewCacheSweepScheduler(
	cacheRepo repository.CacheRepositoryInterface,
	settingsRepo repository.SettingsRepositoryInterface,
) *CacheSweepScheduler {
	return &CacheSweepScheduler{
		cacheRepo:    cacheRepo,
		settingsRepo: settingsRepo,
		stopCh:       make(chan struct{}),
	}
}

// resolveInterval reads cache_sweep_interval_minutes from settings and returns the sweep
// interval plus whether the sweep is enabled:
//   - unset/unreadable → default (45 min), enabled
//   - <= 0             → disabled (0, false)
//   - 0 < v < floor    → clamped up to the safety floor (5 min), enabled
//   - v > ceiling      → clamped down to the safety ceiling (7 days), enabled
func (s *CacheSweepScheduler) resolveInterval(ctx context.Context) (time.Duration, bool) {
	mins, err := s.settingsRepo.GetInt(ctx, settingsKeyCacheSweepInterval)
	if err != nil {
		// Unset is the COMMON case (nothing writes this key yet) and a real DB error is rare and
		// surfaces elsewhere; the underlying Get does not wrap sql.ErrNoRows, so the two are not
		// cheaply distinguishable. Either way we fall back to the default — logged at DEBUG so the
		// fallback stays observable without spamming a line on every boot. (CR M2′.)
		slog.Debug("cache_sweep_interval_minutes unreadable, using default",
			"error", err, "default_minutes", defaultCacheSweepIntervalMinutes)
		mins = defaultCacheSweepIntervalMinutes
	}
	if mins <= 0 {
		return 0, false
	}
	if mins < minCacheSweepIntervalMinutes {
		// Don't silently override the operator's value. (CR L1.)
		slog.Warn("cache_sweep_interval_minutes below floor, clamping up",
			"configured_minutes", mins, "floor_minutes", minCacheSweepIntervalMinutes)
		mins = minCacheSweepIntervalMinutes
	}
	if mins > maxCacheSweepIntervalMinutes {
		slog.Warn("cache_sweep_interval_minutes above ceiling, clamping down",
			"configured_minutes", mins, "ceiling_minutes", maxCacheSweepIntervalMinutes)
		mins = maxCacheSweepIntervalMinutes
	}
	return time.Duration(mins) * time.Minute, true
}

// Start resolves the configured interval and runs the sweep loop until ctx is cancelled or
// Stop() is called. When the interval is disabled (<= 0) it logs and returns without sweeping.
// This method blocks; callers run it in a dedicated goroutine.
func (s *CacheSweepScheduler) Start(ctx context.Context) {
	interval, enabled := s.resolveInterval(ctx)
	if !enabled {
		slog.Info("Cache sweep scheduler disabled (interval <= 0)")
		return
	}

	slog.Info("Cache sweep scheduler started", "interval", interval)
	s.run(ctx, interval)
}

// run owns the cold-start sweep + ticker loop. It is split out from Start so unit tests can drive
// the ticker path directly with a short injected interval (the public Start path floors the
// interval at 5 min, which is impractical for a deterministic test).
//
// Like the sibling BackupScheduler this assumes a SINGLE Start per instance — main.go wires exactly
// one `go Start(ctx)` plus one Stop() — so it deliberately carries no restart/double-start guard,
// mirroring that sanctioned precedent. (CR L3: accepted, precedent-aligned.)
func (s *CacheSweepScheduler) run(ctx context.Context, interval time.Duration) {
	// Cold-start sweep: drain any accumulated backlog immediately. This story exists precisely to
	// clear a bloated cache_entries table; time.NewTicker would otherwise not fire for a FULL
	// interval, and a process that restarts more often than the interval would never sweep at all.
	// Skip if we are already shutting down. (CR M3.)
	select {
	case <-ctx.Done():
		slog.Info("Cache sweep scheduler stopped before first sweep (context cancelled)")
		return
	case <-s.stopCh:
		slog.Info("Cache sweep scheduler stopped before first sweep (stop signal)")
		return
	default:
		s.sweep(ctx)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Cache sweep scheduler stopped (context cancelled)")
			return
		case <-s.stopCh:
			slog.Info("Cache sweep scheduler stopped (stop signal)")
			return
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

// sweep performs a single expiry sweep. It deletes only expired rows via the existing
// CacheRepository.ClearExpired (which logs the deleted-row count when > 0) and NEVER runs
// VACUUM (AC #6). A ClearExpired error is logged and swallowed so the loop continues to the
// next tick (AC #4); a deferred recover guards against any unexpected panic so one bad tick
// can never kill the scheduler goroutine.
func (s *CacheSweepScheduler) sweep(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Cache sweep panicked", "recover", r)
		}
	}()

	if _, err := s.cacheRepo.ClearExpired(ctx); err != nil {
		// A cancelled/expired context during shutdown is expected, not a failure — log it quietly
		// so a tick that races graceful shutdown does not pollute the logs with a false ERROR.
		// (CR L2.)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			slog.Debug("Cache sweep interrupted by context", "error", err)
			return
		}
		slog.Error("Cache sweep failed", "error", err)
	}
}

// Stop gracefully stops the scheduler. It is idempotent — safe to call when the scheduler was
// never started or when called twice.
func (s *CacheSweepScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.stopped {
		s.stopped = true
		close(s.stopCh)
	}
}
