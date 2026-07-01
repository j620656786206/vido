package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/repository"
)

// mockCacheRepo is a minimal hand-rolled CacheRepositoryInterface mock with a call-capturing
// ClearExpired. It avoids testify time-based flakiness — the sweep path is asserted directly.
type mockCacheRepo struct {
	mu              sync.Mutex
	clearExpiredN   int
	clearExpiredRet int64
	clearExpiredErr error
	signal          chan struct{} // optional: non-blocking send on each ClearExpired call
}

func (m *mockCacheRepo) Get(ctx context.Context, key string) (*repository.CacheEntry, error) {
	return nil, nil
}
func (m *mockCacheRepo) Set(ctx context.Context, key, value, cacheType string, ttl time.Duration) error {
	return nil
}
func (m *mockCacheRepo) Delete(ctx context.Context, key string) error { return nil }
func (m *mockCacheRepo) Clear(ctx context.Context) error              { return nil }
func (m *mockCacheRepo) ClearByType(ctx context.Context, cacheType string) (int64, error) {
	return 0, nil
}
func (m *mockCacheRepo) ClearExpired(ctx context.Context) (int64, error) {
	m.mu.Lock()
	m.clearExpiredN++
	m.mu.Unlock()
	if m.signal != nil {
		select {
		case m.signal <- struct{}{}:
		default:
		}
	}
	return m.clearExpiredRet, m.clearExpiredErr
}
func (m *mockCacheRepo) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clearExpiredN
}

// Compile-time check that the mock satisfies the interface the scheduler depends on.
var _ repository.CacheRepositoryInterface = (*mockCacheRepo)(nil)

// fakeTarget is a minimal ClearExpired-shaped sweep spy for the multi-target tests. It counts calls
// and can be configured to return an error or panic, so the per-target isolation in sweepOne can be
// asserted directly (adapted into a target via SweepFunc("name", f.clear)).
type fakeTarget struct {
	mu      sync.Mutex
	calls   int
	err     error
	panicOn bool
}

func (f *fakeTarget) clear(_ context.Context) (int64, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.panicOn {
		panic("boom")
	}
	return 0, f.err
}

func (f *fakeTarget) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestCacheSweepScheduler_ResolveInterval(t *testing.T) {
	ctx := context.Background()
	key := settingsKeyCacheSweepInterval

	t.Run("unset/unreadable defaults to 45 minutes", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", ctx, key).Return(0, assert.AnError)
		s := NewCacheSweepScheduler(nil, settingsRepo)

		interval, enabled := s.resolveInterval(ctx)
		assert.True(t, enabled)
		assert.Equal(t, 45*time.Minute, interval)
	})

	t.Run("configured value is honored", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", ctx, key).Return(120, nil)
		s := NewCacheSweepScheduler(nil, settingsRepo)

		interval, enabled := s.resolveInterval(ctx)
		assert.True(t, enabled)
		assert.Equal(t, 120*time.Minute, interval)
	})

	t.Run("zero disables the sweep", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", ctx, key).Return(0, nil)
		s := NewCacheSweepScheduler(nil, settingsRepo)

		interval, enabled := s.resolveInterval(ctx)
		assert.False(t, enabled)
		assert.Equal(t, time.Duration(0), interval)
	})

	t.Run("negative disables the sweep", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", ctx, key).Return(-1, nil)
		s := NewCacheSweepScheduler(nil, settingsRepo)

		_, enabled := s.resolveInterval(ctx)
		assert.False(t, enabled)
	})

	t.Run("below floor is clamped up to 5 minutes", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", ctx, key).Return(2, nil)
		s := NewCacheSweepScheduler(nil, settingsRepo)

		interval, enabled := s.resolveInterval(ctx)
		assert.True(t, enabled)
		assert.Equal(t, 5*time.Minute, interval)
	})

	t.Run("exactly the floor is honored", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", ctx, key).Return(5, nil)
		s := NewCacheSweepScheduler(nil, settingsRepo)

		interval, enabled := s.resolveInterval(ctx)
		assert.True(t, enabled)
		assert.Equal(t, 5*time.Minute, interval)
	})

	t.Run("above ceiling is clamped down to 7 days", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", ctx, key).Return(99_999_999, nil)
		s := NewCacheSweepScheduler(nil, settingsRepo)

		interval, enabled := s.resolveInterval(ctx)
		assert.True(t, enabled)
		assert.Equal(t, time.Duration(maxCacheSweepIntervalMinutes)*time.Minute, interval)
	})

	t.Run("overflow-prone max int does not panic and is clamped (CR H1)", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", ctx, key).Return(int(^uint(0)>>1), nil) // math.MaxInt
		s := NewCacheSweepScheduler(nil, settingsRepo)

		assert.NotPanics(t, func() {
			interval, enabled := s.resolveInterval(ctx)
			assert.True(t, enabled)
			assert.Equal(t, time.Duration(maxCacheSweepIntervalMinutes)*time.Minute, interval)
			assert.Positive(t, interval) // never negative → time.NewTicker(interval) is safe
		})
	})
}

func TestCacheSweepScheduler_Stop(t *testing.T) {
	t.Run("stop is idempotent (never started)", func(t *testing.T) {
		s := NewCacheSweepScheduler(nil, nil)
		s.Stop()
		s.Stop() // Should not panic
	})
}

func TestCacheSweepScheduler_Sweep(t *testing.T) {
	ctx := context.Background()

	t.Run("invokes ClearExpired exactly once on success", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{clearExpiredRet: 3}
		s := NewCacheSweepScheduler(cacheRepo, nil)

		s.sweep(ctx)

		assert.Equal(t, 1, cacheRepo.callCount())
	})

	t.Run("swallows ClearExpired error and does not panic", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{clearExpiredErr: assert.AnError}
		s := NewCacheSweepScheduler(cacheRepo, nil)

		assert.NotPanics(t, func() { s.sweep(ctx) })
		assert.Equal(t, 1, cacheRepo.callCount())
	})

	t.Run("context.Canceled is not treated as a sweep failure (CR L2)", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{clearExpiredErr: context.Canceled}
		s := NewCacheSweepScheduler(cacheRepo, nil)

		assert.NotPanics(t, func() { s.sweep(ctx) })
		assert.Equal(t, 1, cacheRepo.callCount())
	})
}

func TestCacheSweepScheduler_Run(t *testing.T) {
	t.Run("runs an immediate cold-start sweep before the first tick (CR M3)", func(t *testing.T) {
		sig := make(chan struct{}, 4)
		cacheRepo := &mockCacheRepo{signal: sig}
		s := NewCacheSweepScheduler(cacheRepo, nil)

		// A 10-minute interval cannot tick within the test window, so the only sweep that can
		// fire is the immediate cold-start one.
		go s.run(context.Background(), 10*time.Minute)
		defer s.Stop()

		select {
		case <-sig:
		case <-time.After(2 * time.Second):
			t.Fatal("no immediate cold-start sweep fired")
		}
		assert.Equal(t, 1, cacheRepo.callCount())
	})

	t.Run("ticker drives subsequent sweeps (CR M2 — exercises case <-ticker.C)", func(t *testing.T) {
		sig := make(chan struct{}, 8)
		cacheRepo := &mockCacheRepo{signal: sig}
		s := NewCacheSweepScheduler(cacheRepo, nil)

		go s.run(context.Background(), 5*time.Millisecond)
		defer s.Stop()

		// 1st signal = immediate cold-start sweep; 2nd MUST come from a ticker tick, proving the
		// `case <-ticker.C: s.sweep(ctx)` path actually runs (deleting it would hang this test).
		for i := 0; i < 2; i++ {
			select {
			case <-sig:
			case <-time.After(2 * time.Second):
				t.Fatalf("expected sweep #%d (immediate + ticker), got none", i+1)
			}
		}
		assert.GreaterOrEqual(t, cacheRepo.callCount(), 2)
	})

	t.Run("does not sweep when stopped before first sweep", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{}
		s := NewCacheSweepScheduler(cacheRepo, nil)
		s.Stop() // close stopCh before run

		done := make(chan struct{})
		go func() {
			s.run(context.Background(), 5*time.Millisecond)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("run did not return promptly when stopped before first sweep")
		}
		assert.Equal(t, 0, cacheRepo.callCount())
	})
}

func TestCacheSweepScheduler_Start(t *testing.T) {
	t.Run("disabled interval returns immediately without sweeping", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", context.Background(), settingsKeyCacheSweepInterval).Return(0, nil)
		cacheRepo := &mockCacheRepo{}
		s := NewCacheSweepScheduler(cacheRepo, settingsRepo)

		done := make(chan struct{})
		go func() {
			s.Start(context.Background())
			close(done)
		}()

		select {
		case <-done:
			// returned promptly because the sweep is disabled
		case <-time.After(2 * time.Second):
			t.Fatal("Start did not return for a disabled interval")
		}
		assert.Equal(t, 0, cacheRepo.callCount())
	})

	t.Run("returns when context is cancelled", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", mock.Anything, settingsKeyCacheSweepInterval).Return(10, nil)
		s := NewCacheSweepScheduler(&mockCacheRepo{}, settingsRepo)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			s.Start(ctx)
			close(done)
		}()

		cancel()
		select {
		case <-done:
			// returned on context cancellation (no 10-min tick wait)
		case <-time.After(2 * time.Second):
			t.Fatal("Start did not return on context cancellation")
		}
	})

	t.Run("returns when Stop is called", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		settingsRepo.On("GetInt", mock.Anything, settingsKeyCacheSweepInterval).Return(10, nil)
		s := NewCacheSweepScheduler(&mockCacheRepo{}, settingsRepo)

		done := make(chan struct{})
		go func() {
			s.Start(context.Background())
			close(done)
		}()

		// Stop closes stopCh permanently, so Start returns whether or not it has
		// reached its select loop yet — no sleep needed.
		s.Stop()
		select {
		case <-done:
			// returned on stop signal
		case <-time.After(2 * time.Second):
			t.Fatal("Start did not return after Stop")
		}
	})
}

func TestCacheSweepScheduler_MultiTarget(t *testing.T) {
	ctx := context.Background()

	t.Run("a tick sweeps all registered targets exactly once (AC1)", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{}
		aiCache := &fakeTarget{}
		offlineCache := &fakeTarget{}
		s := NewCacheSweepScheduler(cacheRepo, nil,
			SweepFunc("ai_cache", aiCache.clear),
			SweepFunc("offline_cache", offlineCache.clear),
		)
		assert.Len(t, s.targets, 3) // cache_entries + ai_cache + offline_cache

		s.sweep(ctx)

		assert.Equal(t, 1, cacheRepo.callCount())
		assert.Equal(t, 1, aiCache.callCount())
		assert.Equal(t, 1, offlineCache.callCount())
	})

	t.Run("one target erroring still sweeps the others (error isolation, AC2)", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{}
		mid := &fakeTarget{err: assert.AnError}
		last := &fakeTarget{}
		s := NewCacheSweepScheduler(cacheRepo, nil,
			SweepFunc("mid", mid.clear),
			SweepFunc("last", last.clear),
		)

		assert.NotPanics(t, func() { s.sweep(ctx) })
		assert.Equal(t, 1, cacheRepo.callCount())
		assert.Equal(t, 1, mid.callCount())
		assert.Equal(t, 1, last.callCount()) // reached despite the mid target's error
	})

	t.Run("one target panicking still sweeps the others and does not crash (panic isolation, AC2)", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{}
		boom := &fakeTarget{panicOn: true}
		last := &fakeTarget{}
		s := NewCacheSweepScheduler(cacheRepo, nil,
			SweepFunc("boom", boom.clear),
			SweepFunc("last", last.clear),
		)

		assert.NotPanics(t, func() { s.sweep(ctx) })
		assert.Equal(t, 1, cacheRepo.callCount())
		assert.Equal(t, 1, boom.callCount())
		assert.Equal(t, 1, last.callCount()) // reached despite the boom target's panic
	})

	t.Run("nil sweep func/cache is skipped at construction (AC3)", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{}
		real := &fakeTarget{}
		// SweepFunc with a nil fn AND SweepTarget with a nil ExpirableCache both yield a nil sweep,
		// which the constructor must drop so no nil-func call/panic ever happens on a tick.
		var nilCache ExpirableCache
		s := NewCacheSweepScheduler(cacheRepo, nil,
			SweepFunc("nil-func", nil),
			SweepTarget("nil-cache", nilCache),
			SweepFunc("real", real.clear),
		)
		assert.Len(t, s.targets, 2) // only cache_entries + real survive construction

		assert.NotPanics(t, func() { s.sweep(ctx) })
		assert.Equal(t, 1, cacheRepo.callCount())
		assert.Equal(t, 1, real.callCount())
	})

	t.Run("cache_entries alone is swept when zero extra targets supplied (AC4 backward-compat)", func(t *testing.T) {
		cacheRepo := &mockCacheRepo{}
		s := NewCacheSweepScheduler(cacheRepo, nil)
		assert.Len(t, s.targets, 1)

		s.sweep(ctx)

		assert.Equal(t, 1, cacheRepo.callCount())
	})
}
