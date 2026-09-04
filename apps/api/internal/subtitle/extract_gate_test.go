package subtitle

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── sub-6-3 AC #2/#4b/#4c: ExtractGate ─────────────────────────────────────

func TestExtractGate_SerializesHolders(t *testing.T) {
	g := NewExtractGate()

	release1, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)

	acquired := make(chan struct{})
	var waitedBehind int
	go func() {
		release2, err := g.Acquire(context.Background(), func(ahead int) { waitedBehind = ahead })
		require.NoError(t, err)
		close(acquired)
		release2()
	}()

	select {
	case <-acquired:
		t.Fatal("second Acquire must block while the first holder is inside ffmpeg")
	case <-time.After(50 * time.Millisecond):
	}
	assert.Equal(t, 1, g.Waiting())

	release1()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("release must hand the slot to the waiter")
	}
	assert.Equal(t, 1, waitedBehind, "one extraction was ahead: the holder")
	assert.Equal(t, 0, g.Waiting())
}

func TestExtractGate_ReportsQueueDepthAhead(t *testing.T) {
	g := NewExtractGate()
	release, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)

	var mu sync.Mutex
	var aheads []int
	var wg sync.WaitGroup
	started := make(chan struct{}, 3)
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := g.Acquire(context.Background(), func(ahead int) {
				mu.Lock()
				aheads = append(aheads, ahead)
				mu.Unlock()
				started <- struct{}{}
			})
			require.NoError(t, err)
			r()
		}()
		<-started // queue them one at a time so the depth is deterministic
	}

	mu.Lock()
	assert.Equal(t, []int{1, 2, 3}, aheads, "each newcomer sees the holder plus everyone already queued")
	mu.Unlock()

	release()
	wg.Wait()
	assert.Equal(t, 0, g.Waiting())
}

func TestExtractGate_CancelledWhileWaitingReturnsAtOnce(t *testing.T) {
	g := NewExtractGate()
	release, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := g.Acquire(ctx, nil)
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("a cancelled waiter must not stay queued behind a long extraction")
	}
	assert.Equal(t, 0, g.Waiting(), "a cancelled waiter leaves the queue")
}

func TestExtractGate_AlreadyEndedCtxNeverEntersTheQueue(t *testing.T) {
	g := NewExtractGate()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.Acquire(ctx, func(int) { t.Fatal("no wait notice for a ctx that already ended") })
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, g.Waiting())
}

func TestExtractGate_ReleaseIsIdempotentAndNilGateIsNoOp(t *testing.T) {
	g := NewExtractGate()
	release, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)
	release()
	release() // a second call must not free a slot nobody holds

	again, err := g.Acquire(context.Background(), func(int) { t.Fatal("slot must be free after release") })
	require.NoError(t, err)
	again()

	var none *ExtractGate
	r, err := none.Acquire(context.Background(), nil)
	require.NoError(t, err)
	r()
	assert.Equal(t, 0, none.Waiting())
}

func TestExtractWaitNotifier_RoundTripsThroughCtx(t *testing.T) {
	assert.Nil(t, extractWaitNotifierFrom(context.Background()))
	assert.Nil(t, extractWaitNotifierFrom(WithExtractWaitNotifier(context.Background(), nil)))

	var got int
	ctx := WithExtractWaitNotifier(context.Background(), func(ahead int) { got = ahead })
	extractWaitNotifierFrom(ctx)(4)
	assert.Equal(t, 4, got)
}
