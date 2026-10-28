package subtitle

// disc-2026-10-solo-generation-cancel-api CR L2/L3.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
)

// ctxCheckingSegmentCache refuses writes on a cancelled ctx.
type ctxCheckingSegmentCache struct{ *memorySegmentCache }

func (c ctxCheckingSegmentCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.memorySegmentCache.Set(ctx, key, value, ttl)
}

// CR L2: the translator answered (paid), then the stop landed — the cues are
// still cached, and the run, whose file is placed, completes.
func TestProcessItem_PaidCuesAreCachedDespiteAStopAfterTheCall(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning.", "Good night."))
	h.pipeline.cache = ctxCheckingSegmentCache{h.cache}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.trans.fn = func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		out := make(map[int]string, len(blocks))
		for _, b := range blocks {
			out[b.Index] = "早安"
		}
		cancel()
		return out, ai.CompletionUsage{}, nil
	}

	_, _ = h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{Regenerate: true, RestoreMediaOnCancel: true})

	assert.Len(t, h.cache.writes, 2, "both paid cues are in the segment cache")
}

// notifyingHub runs onTerminal when the solo job's terminal event goes out.
type notifyingHub struct {
	recordingHub
	once       sync.Once
	onTerminal func()
}

func (h *notifyingHub) Broadcast(e sse.Event) {
	h.recordingHub.Broadcast(e)
	if e.Type == services.EventTranscriptionComplete || e.Type == services.EventTranscriptionFailed {
		if d, ok := e.Data.(map[string]interface{}); ok && d["job_id"] != nil {
			h.once.Do(h.onTerminal)
		}
	}
}

// CR L3: between the terminal event and the release of the claim, the user's
// own job is still the owner — a Cancel there never answers "someone else".
func TestSoloRunner_CancelBetweenTerminalAndReleaseStillSeesOurJob(t *testing.T) {
	h := newItemHarness(t, translateDecision("Hi"))
	guard := newFakeSoloGuard()
	hub := &notifyingHub{}
	var runner *SoloRunner
	var got SoloCancelResult
	hub.onTerminal = func() { got = runner.Cancel(h.ref) }
	runner = NewSoloRunner(h.pipeline, guard, hub, nil)

	_, err := runner.Start(context.Background(), h.ref, "")
	require.NoError(t, err)
	runner.waitIdle()

	assert.Equal(t, SoloOwnerSolo, got.Owner, "never owner=other for the user's own job")
	assert.Empty(t, runner.ActiveSoloJobs())
	assert.False(t, guard.IsInFlight(h.ref))
	assert.Equal(t, models.SubtitleRunCompleted, h.runs.lastUpdate(t).Status)
}
