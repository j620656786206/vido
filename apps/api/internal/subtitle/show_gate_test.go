package subtitle

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
)

// fixedClock is a hand-advanced clock: the warm window is the pipeline's only
// wall-clock read, and a deterministic clock is what makes "stale entry
// re-warms" assertable without sleeping.
type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFixedClock() *fixedClock {
	return &fixedClock{now: time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)}
}

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestShowGate_FirstRequestRunsAloneThenReleasesWaiters(t *testing.T) {
	clock := newFixedClock()
	gate := newShowGate(clock.Now)

	leaderRelease, err := gate.enter(context.Background(), "show-1")
	require.NoError(t, err)

	waiterEntered := make(chan struct{})
	go func() {
		release, err := gate.enter(context.Background(), "show-1")
		assert.NoError(t, err)
		release()
		close(waiterEntered)
	}()

	select {
	case <-waiterEntered:
		t.Fatal("the second item must NOT proceed while the show's first request is in flight — " +
			"a cache entry is only readable once the first response begins (architecture fact 6)")
	case <-time.After(50 * time.Millisecond):
	}

	leaderRelease()

	select {
	case <-waiterEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("waiters must be released as soon as the first response returns")
	}
}

func TestShowGate_WarmWindowSkipsTheGateEntirely(t *testing.T) {
	clock := newFixedClock()
	gate := newShowGate(clock.Now)

	release, err := gate.enter(context.Background(), "show-1")
	require.NoError(t, err)
	release()

	// Inside the warm window the prefix is live provider-side, so nothing needs
	// to be serialized: enter must return without any leader in flight.
	clock.advance(showWarmWindow - time.Minute)
	done := make(chan struct{})
	go func() {
		r, err := gate.enter(context.Background(), "show-1")
		assert.NoError(t, err)
		r()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a warm show must not block")
	}
}

func TestShowGate_StaleEntryMakesTheNextRequesterFirstAgain(t *testing.T) {
	clock := newFixedClock()
	gate := newShowGate(clock.Now)

	release, err := gate.enter(context.Background(), "show-1")
	require.NoError(t, err)
	release()

	// Past the warm window the provider-side entry may have expired (the window
	// is deliberately shorter than the 1h TTL), so someone must re-warm it.
	clock.advance(showWarmWindow + time.Minute)

	leaderRelease, err := gate.enter(context.Background(), "show-1")
	require.NoError(t, err)

	blocked := make(chan struct{})
	go func() {
		r, err := gate.enter(context.Background(), "show-1")
		assert.NoError(t, err)
		r()
		close(blocked)
	}()

	select {
	case <-blocked:
		t.Fatal("a stale entry must re-serialize: the next requester becomes 'first' again")
	case <-time.After(50 * time.Millisecond):
	}
	leaderRelease()
	<-blocked
}

func TestShowGate_MoviesBypass(t *testing.T) {
	gate := newShowGate(newFixedClock().Now)

	// An empty key is what a movie carries: nothing shares its prompt prefix, so
	// serializing it would only add latency.
	first, err := gate.enter(context.Background(), "")
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		r, err := gate.enter(context.Background(), "")
		assert.NoError(t, err)
		r()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("movies must never queue behind each other")
	}
	first()
}

func TestShowGate_CancellationWhileWaitingReturnsContextError(t *testing.T) {
	gate := newShowGate(newFixedClock().Now)

	leaderRelease, err := gate.enter(context.Background(), "show-1")
	require.NoError(t, err)
	defer leaderRelease()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := gate.enter(ctx, "show-1")
		result <- err
	}()

	// Let the waiter park on the latch before cancelling.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		assert.ErrorIs(t, err, context.Canceled, "a shutdown must not leave a worker parked on the gate")
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation must unpark the waiter")
	}
}

// TestShowGate_PrunesStaleEntries is the Rule 14 bound: one long-lived Pipeline
// serves an entire library, so the latch map must not grow one entry per show
// forever.
func TestShowGate_PrunesStaleEntries(t *testing.T) {
	clock := newFixedClock()
	gate := newShowGate(clock.Now)

	for i := 0; i < 50; i++ {
		release, err := gate.enter(context.Background(), fmt.Sprintf("show-%d", i))
		require.NoError(t, err)
		release()
	}
	assert.Equal(t, 50, gate.size())

	clock.advance(showEntryTTL + time.Minute)
	release, err := gate.enter(context.Background(), "show-fresh")
	require.NoError(t, err)
	release()

	assert.Equal(t, 1, gate.size(), "stale entries are pruned on access — only the live one survives")
}

// TestShowGate_ReleaseIsIdempotent guards the defer-plus-explicit-call shape:
// closing an already-closed latch would panic and take the worker with it.
func TestShowGate_ReleaseIsIdempotent(t *testing.T) {
	gate := newShowGate(newFixedClock().Now)
	release, err := gate.enter(context.Background(), "show-1")
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		release()
		release()
	})
}

// TestShowGate_ConcurrentShowsDoNotBlockEachOther — the gate is per show, not
// global; two different shows must translate in parallel.
func TestShowGate_ConcurrentShowsDoNotBlockEachOther(t *testing.T) {
	gate := newShowGate(newFixedClock().Now)

	releaseA, err := gate.enter(context.Background(), "show-a")
	require.NoError(t, err)
	defer releaseA()

	done := make(chan struct{})
	go func() {
		r, err := gate.enter(context.Background(), "show-b")
		assert.NoError(t, err)
		r()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("show-b must not wait behind show-a")
	}
}

// TestTranslateTrack_SerializesTheShowsFirstRequest is the wiring proof: the
// gate is useless unless the translate loop actually enters it around the
// item's FIRST LLM request. Two episodes of one show run concurrently and the
// second's first request must not reach the translator until the first returns.
func TestTranslateTrack_SerializesTheShowsFirstRequest(t *testing.T) {
	const showKey = "series-42"

	leaderInside := make(chan struct{})
	leaderMayReturn := make(chan struct{})
	var mu sync.Mutex
	var callOrder []string

	newTranslator := func(label string, block bool) *fakeTranslator {
		return &fakeTranslator{
			fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
				mu.Lock()
				callOrder = append(callOrder, label)
				mu.Unlock()
				if block {
					close(leaderInside)
					<-leaderMayReturn
				}
				out := make(map[int]string, len(blocks))
				for _, b := range blocks {
					out[b.Index] = "早安"
				}
				return out, ai.CompletionUsage{CacheCreationInputTokens: 5000}, nil
			},
		}
	}

	gate := newShowGate(newFixedClock().Now)

	run := func(label string, block bool) <-chan struct{} {
		done := make(chan struct{})
		go func() {
			defer close(done)
			p := NewPipeline(newTranslator(label, block), &recordingConverter{}, nil)
			p.gate = gate // one shared gate, exactly as one shared Pipeline would have
			ctx := withProcessScope(context.Background(), &processScope{
				ref:     MediaRef{ID: label, MediaType: models.SubtitleRunMediaEpisode},
				showKey: showKey,
			})
			_, err := p.TranslateTrack(ctx, trackOf(cues("Good morning.")), TranslateContext{})
			assert.NoError(t, err)
		}()
		return done
	}

	leaderDone := run("leader", true)
	<-leaderInside // the leader is inside the provider call, holding the gate

	waiterDone := run("waiter", false)
	select {
	case <-waiterDone:
		t.Fatal("the second episode's first request must wait for the show's prefix to be warmed")
	case <-time.After(50 * time.Millisecond):
	}

	mu.Lock()
	assert.Equal(t, []string{"leader"}, callOrder, "only the leader may have reached the provider so far")
	mu.Unlock()

	close(leaderMayReturn)
	<-leaderDone
	select {
	case <-waiterDone:
	case <-time.After(2 * time.Second):
		t.Fatal("the waiter must proceed once the first response has returned")
	}
}
