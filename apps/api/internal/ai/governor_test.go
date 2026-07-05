package ai

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGovernor_ConcurrencyCap(t *testing.T) {
	g := NewGovernor(2, 0, 0) // cap 2, no rate limit
	var inFlight, maxInFlight atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := g.Acquire(context.Background())
			require.NoError(t, err)
			defer release()
			cur := inFlight.Add(1)
			for {
				m := maxInFlight.Load()
				if cur <= m || maxInFlight.CompareAndSwap(m, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			inFlight.Add(-1)
		}()
	}
	wg.Wait()
	assert.LessOrEqual(t, maxInFlight.Load(), int32(2), "concurrency must never exceed the cap")
}

func TestGovernor_RateLimit(t *testing.T) {
	// 50 req/s, burst 1 → 5 calls take at least ~4 intervals (~80ms).
	g := NewGovernor(0, 50, 1)
	start := time.Now()
	for i := 0; i < 5; i++ {
		release, err := g.Acquire(context.Background())
		require.NoError(t, err)
		release()
	}
	assert.GreaterOrEqual(t, time.Since(start), 60*time.Millisecond, "rate limiter must space out calls")
}

func TestGovernor_NilIsNoOp(t *testing.T) {
	var g *Governor
	release, err := g.Acquire(context.Background())
	require.NoError(t, err)
	release() // must not panic
}

func TestGovernor_ContextCancelled(t *testing.T) {
	g := NewGovernor(1, 0, 0)
	// Hold the only slot.
	release, err := g.Acquire(context.Background())
	require.NoError(t, err)
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = g.Acquire(ctx)
	require.Error(t, err, "acquire on a full governor with a cancelled ctx must error")
}

func TestGoverned_BudgetCutoff(t *testing.T) {
	// A budget already at its ceiling short-circuits before the fn runs.
	b := NewBudget(1.0)
	b.RecordLLM("claude-opus-4-8", 1_000_000, 1_000_000) // $5+$25 → over $1
	require.True(t, b.Exceeded())

	ctx := WithBudget(context.Background(), b)
	called := false
	_, err := governed(ctx, nil, "test.op", func() (int, error) {
		called = true
		return 1, nil
	})
	require.ErrorIs(t, err, ErrBudgetExceeded)
	assert.False(t, called, "budget cutoff must prevent the call")
}

func TestGoverned_RunsWhenUnderBudget(t *testing.T) {
	b := NewBudget(100.0)
	ctx := WithBudget(context.Background(), b)
	got, err := governed(ctx, NewGovernor(2, 0, 0), "test.op", func() (int, error) {
		return 42, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 42, got)
}
