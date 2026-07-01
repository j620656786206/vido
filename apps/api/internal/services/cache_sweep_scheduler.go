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

// ExpirableCache is any DB-table cache that can delete its own expired rows. cache_entries
// (repository.CacheRepositoryInterface) and offline_cache (*cache.OfflineCache) satisfy it directly
// with the identical ClearExpired shape; ai_cache is adapted via SweepFunc (its wrapper method name
// differs). The scheduler sweeps each registered target on the shared ticker.
type ExpirableCache interface {
	ClearExpired(ctx context.Context) (int64, error)
}

// CacheSweepTarget is one named sweep bound into the scheduler. Its fields are unexported so a
// target can only be built via SweepTarget/SweepFunc; a nil cache/func yields a target whose sweep
// is nil, which NewCacheSweepScheduler drops at construction (AC3). The type itself stays exported
// so callers (main.go) can hold a []CacheSweepTarget without tripping the "exported func returns
// unexported type" lint on the helpers.
type CacheSweepTarget struct {
	name  string
	sweep func(context.Context) (int64, error)
}

// SweepTarget builds a target from anything satisfying ExpirableCache (e.g. cache_entries,
// offline_cache). An untyped-nil cache yields a target with a nil sweep, which the constructor skips
// (AC3) — the guard here also avoids panicking on the c.ClearExpired method-value expression for a
// nil interface. A non-nil interface wrapping a typed-nil pointer is the caller's responsibility
// (all call sites pass a live cache); were one ever passed, the eventual nil-receiver panic is
// contained by sweepOne's per-target recover, so it degrades to a logged error rather than a crash.
func SweepTarget(name string, c ExpirableCache) CacheSweepTarget {
	if c == nil {
		return CacheSweepTarget{name: name}
	}
	return CacheSweepTarget{name: name, sweep: c.ClearExpired}
}

// SweepFunc builds a target from a bare ClearExpired-shaped func. Required for
// AIService.ClearExpiredCache, whose method name differs from ClearExpired and so does not satisfy
// ExpirableCache directly (this avoids adding a rename/alias to the AIService API). A nil fn yields
// a target the constructor skips (AC3).
func SweepFunc(name string, fn func(context.Context) (int64, error)) CacheSweepTarget {
	return CacheSweepTarget{name: name, sweep: fn}
}

// CacheSweepSchedulerInterface defines the contract for the cache expiry sweep scheduler.
type CacheSweepSchedulerInterface interface {
	Start(ctx context.Context)
	Stop()
}

// CacheSweepScheduler periodically deletes expired rows from one or more DB-table caches
// (cache_entries plus, when wired, ai_cache and offline_cache) by calling each target's ClearExpired
// on a single recurring ticker. It keeps those SQLite caches from growing unbounded (especially once
// the Discover facet-counts fan-out write-amplifies cache_entries) without manual intervention.
// Targets are swept SEQUENTIALLY in one goroutine so they never contend the single SQLite writer
// lock, and it never runs VACUUM.
type CacheSweepScheduler struct {
	targets      []CacheSweepTarget
	settingsRepo repository.SettingsRepositoryInterface
	mu           sync.Mutex
	stopCh       chan struct{}
	stopped      bool
}

// Compile-time interface verification
var _ CacheSweepSchedulerInterface = (*CacheSweepScheduler)(nil)

// NewCacheSweepScheduler creates a new CacheSweepScheduler. cache_entries (from cacheRepo) is always
// the first target when cacheRepo is non-nil; extraTargets (e.g. offline_cache, ai_cache) are
// appended in order. Any target with a nil sweep (absent/nil cache) is skipped (AC3). The variadic
// tail keeps the signature source-compatible with the single-target #98 form.
func NewCacheSweepScheduler(
	cacheRepo repository.CacheRepositoryInterface,
	settingsRepo repository.SettingsRepositoryInterface,
	extraTargets ...CacheSweepTarget,
) *CacheSweepScheduler {
	var targets []CacheSweepTarget
	if cacheRepo != nil { // keeps NewCacheSweepScheduler(nil, settingsRepo) tests valid
		targets = append(targets, SweepTarget("cache_entries", cacheRepo))
	}
	for _, t := range extraTargets {
		if t.sweep != nil { // skip a nil/absent cache (AC3)
			targets = append(targets, t)
		}
	}
	return &CacheSweepScheduler{
		targets:      targets,
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

	slog.Info("Cache sweep scheduler started", "interval", interval, "targets", len(s.targets))
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

// sweep performs a single expiry sweep across ALL registered targets, SEQUENTIALLY (never
// concurrently — so the targets never contend the single SQLite writer lock, AC #6). It deletes only
// expired rows via each target's existing ClearExpired (which logs the deleted-row count when > 0)
// and NEVER runs VACUUM. Per-target isolation lives in sweepOne so one bad target can neither abort
// the others on this tick nor kill the scheduler goroutine (AC #2).
func (s *CacheSweepScheduler) sweep(ctx context.Context) {
	for _, t := range s.targets {
		s.sweepOne(ctx, t)
	}
}

// sweepOne sweeps a single target with its own recover + error handling so a panic or error in one
// target is contained to that target (AC #2). A ClearExpired error is logged and swallowed so the
// remaining targets — and the next tick — still run (AC #4); a cancelled/expired context during
// shutdown is expected and logged quietly so a tick racing graceful shutdown does not emit a false
// ERROR (CR L2).
func (s *CacheSweepScheduler) sweepOne(ctx context.Context, t CacheSweepTarget) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Cache sweep panicked", "target", t.name, "recover", r)
		}
	}()

	if _, err := t.sweep(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			slog.Debug("Cache sweep interrupted by context", "target", t.name, "error", err)
			return
		}
		slog.Error("Cache sweep failed", "target", t.name, "error", err)
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
