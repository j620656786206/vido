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

	release1, waited, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)
	assert.Zero(t, waited, "an idle gate is entered instantly")

	acquired := make(chan struct{})
	var notices []int
	var waitedBehind time.Duration
	go func() {
		release2, w, err := g.Acquire(context.Background(), func(ahead int) { notices = append(notices, ahead) })
		require.NoError(t, err)
		waitedBehind = w
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
	assert.Equal(t, []int{1, 0}, notices,
		"one extraction was ahead (the holder), then a 0 retracts the notice once the slot is taken")
	assert.Positive(t, waitedBehind, "the wait is reported so the caller can tell a starved budget from a slow file")
	assert.Equal(t, 0, g.Waiting())
}

func TestExtractGate_ReportsQueueDepthAhead(t *testing.T) {
	g := NewExtractGate()
	release, _, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)

	var mu sync.Mutex
	var aheads []int
	var wg sync.WaitGroup
	started := make(chan struct{}, 3)
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, _, err := g.Acquire(context.Background(), func(ahead int) {
				if ahead == 0 {
					return // the retraction; this test is about the depth
				}
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

// TestExtractGate_RetractsTheQueueNoticeOnEntry locks CR M2: without the
// closing onWait(0) the「等待抽軌」bubble would stay on screen for the whole
// extraction that followed it — the stalled bubble the narration replaces.
func TestExtractGate_RetractsTheQueueNoticeOnEntry(t *testing.T) {
	g := NewExtractGate()
	release, _, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)

	entered := make(chan struct{})
	var notices []int
	go func() {
		r, _, err := g.Acquire(context.Background(), func(ahead int) { notices = append(notices, ahead) })
		require.NoError(t, err)
		close(entered)
		r()
	}()

	require.Eventually(t, func() bool { return g.Waiting() == 1 }, time.Second, 5*time.Millisecond)
	release()
	<-entered

	require.Len(t, notices, 2)
	assert.Equal(t, 1, notices[0], "queued behind the holder")
	assert.Equal(t, 0, notices[1], "…and told when it got in")
}

func TestExtractGate_CancelledWhileWaitingReturnsAtOnce(t *testing.T) {
	g := NewExtractGate()
	release, _, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	type result struct {
		waited time.Duration
		err    error
	}
	done := make(chan result, 1)
	go func() {
		_, waited, err := g.Acquire(ctx, nil)
		done <- result{waited, err}
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case got := <-done:
		assert.ErrorIs(t, got.err, context.Canceled)
		assert.Positive(t, got.waited, "the abandoned wait is still reported")
	case <-time.After(time.Second):
		t.Fatal("a cancelled waiter must not stay queued behind a long extraction")
	}
	assert.Equal(t, 0, g.Waiting(), "a cancelled waiter leaves the queue")
}

func TestExtractGate_AlreadyEndedCtxNeverEntersTheQueue(t *testing.T) {
	g := NewExtractGate()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := g.Acquire(ctx, func(int) { t.Fatal("no wait notice for a ctx that already ended") })
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, g.Waiting())
}

func TestExtractGate_ReleaseIsIdempotentAndNilGateIsNoOp(t *testing.T) {
	g := NewExtractGate()
	release, _, err := g.Acquire(context.Background(), nil)
	require.NoError(t, err)
	release()
	release() // a second call must not free a slot nobody holds

	again, _, err := g.Acquire(context.Background(), func(int) { t.Fatal("slot must be free after release") })
	require.NoError(t, err)
	again()

	var none *ExtractGate
	r, waited, err := none.Acquire(context.Background(), nil)
	require.NoError(t, err)
	assert.Zero(t, waited)
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
