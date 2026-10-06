package subtitle

// disc-2026-10-single-generate-ignores-embedded-english-a — the two additive
// ProcessItemOptions the detail-page 生成字幕 click needs, plus the pool's
// in-flight query the click's 409 and status endpoint read.

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// ─── AC #7: Regenerate bypasses the pre-flight but KEEPS cache reads ───────

func TestProcessItem_RegenerateBypassesPreflightButKeepsCacheReads(t *testing.T) {
	source := cues("Good morning.")
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	})
	require.NoError(t, os.WriteFile(ExpectedSidecarPath(h.mediaPath), []byte(oneCueSRT), 0o600))

	version := h.pipeline.runVersion(context.Background(), richContext())
	h.cache.entries[segmentKey(source[0].Text, version)] = "早安（快取）"

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{Regenerate: true})
	require.NoError(t, err)

	require.NotNil(t, outcome.Run, "regenerate runs the item for real")
	assert.Equal(t, 1, h.router.calls, "regenerate bypasses the P5 early-exit")
	assert.NotEmpty(t, h.cache.reads, "regenerate KEEPS segment-cache reads — a redo must not re-pay translated cues")
	assert.Empty(t, h.trans.calls, "the cached cue is served from the cache, not re-translated")
	assert.Contains(t, string(h.placer.requests[0].SubtitleData), "早安（快取）")
}

func TestProcessItem_WithoutRegenerateOrForceThePreflightStillSkips(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	require.NoError(t, os.WriteFile(ExpectedSidecarPath(h.mediaPath), []byte(oneCueSRT), 0o600))

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	assert.Nil(t, outcome.Run)
	assert.Zero(t, h.router.calls)
}

// ─── AC #8: TranscribeWhenSkipped turns RouteSkip into the ASR leg ─────────

func skipDecision() RouteDecision {
	return RouteDecision{
		Kind:   RouteSkip,
		Reason: "1 embedded text track(s) present but none tagged Chinese or eng/en — M1 never treats und as English",
	}
}

func TestProcessItem_TranscribeWhenSkippedRunsTheASRLeg(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true}
	h := asrHarness(t, skipDecision(), asr)

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{TranscribeWhenSkipped: true})
	require.NoError(t, err)
	require.NotNil(t, outcome)

	require.Len(t, asr.calls, 1, "the user asked for a subtitle — an und-tagged text track is not a reason to refuse")
	assert.Equal(t, h.ref, asr.calls[0].ref)
	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunCompleted, final.Status)
	assert.Equal(t, models.SubtitleRunRouteASR, final.Route)
}

func TestProcessItem_TranscribeWhenSkippedWithoutASRStillRecordsSkipped(t *testing.T) {
	// No ASR port at all: the verdict stays the honest `skipped`, not
	// `no_text_source` — a text track DID exist, it was declined.
	h := asrHarness(t, skipDecision(), nil)

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{TranscribeWhenSkipped: true})
	require.NoError(t, err)

	assert.Equal(t, models.SubtitleRunSkipped, h.runs.lastUpdate(t).Status)
	assert.Equal(t,
		[]models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusSkipped},
		h.media.statuses())
}

func TestProcessItem_RouteSkipWithoutTheOptionStillNeverTouchesASR(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true}
	h := asrHarness(t, skipDecision(), asr)

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	assert.Empty(t, asr.calls, "batch / auto behaviour is unchanged: skipped stays skipped")
	assert.Equal(t, models.SubtitleRunSkipped, h.runs.lastUpdate(t).Status)
}

// ─── AC #9 (guard half): the pool answers "is this item in flight?" ────────

func TestWorkerPool_IsInFlightFollowsReserveAndRelease(t *testing.T) {
	pool := NewWorkerPool(nil, nil)
	ref := MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaMovie}

	assert.False(t, pool.IsInFlight(ref))
	require.True(t, pool.TryReserve(ref))
	assert.True(t, pool.IsInFlight(ref))
	assert.False(t, pool.IsInFlight(MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaEpisode}),
		"the key is the whole ref — a movie and an episode sharing an id are different items")
	pool.Release(ref)
	assert.False(t, pool.IsInFlight(ref))
}
