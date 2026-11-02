package providers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// disc-2026-11-online-sub-download-progress: a quota wait moves the ring as
// it elapses instead of freezing it, and a cancelled wait gives its slot back.
func TestWaitWithProgress(t *testing.T) {
	lim := rate.NewLimiter(rate.Every(2500*time.Millisecond), 1)
	require.True(t, lim.Allow()) // spend the burst so the next call waits

	var seen []float64
	ctx := WithProgress(context.Background(), func(f float64) { seen = append(seen, f) })
	require.NoError(t, waitWithProgress(ctx, lim, 0.25, 0.35))

	require.GreaterOrEqual(t, len(seen), 2, "at least one tick plus the end: %v", seen)
	for i := 1; i < len(seen); i++ {
		assert.GreaterOrEqual(t, seen[i], seen[i-1])
	}
	assert.Greater(t, seen[0], 0.25)
	assert.InDelta(t, 0.35, seen[len(seen)-1], 1e-9)

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, waitWithProgress(ctx2, lim, 0, 1), context.Canceled)
}
