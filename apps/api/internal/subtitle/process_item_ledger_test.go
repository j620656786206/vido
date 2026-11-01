package subtitle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// sub-7-6a — the run row is a ledger now: which lane ran, how much the cache
// served, which batch it belonged to, and ONE receipt per run.

func TestProcessItem_LedgerRecordsRouteCacheAndBatch(t *testing.T) {
	texts := longTrack(10)
	source := cues(texts...)
	var receipts []models.SubtitleRun
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	}, WithRunReceipt(func(run *models.SubtitleRun) { receipts = append(receipts, *run) }))

	// 4 of 10 cues are already in the segment cache.
	version := h.pipeline.runVersion(context.Background(), richContext())
	for i := 0; i < 4; i++ {
		h.cache.entries[segmentKey(source[i].Text, version)] = "快取譯文"
	}

	ctx := services.WithGenerationBatchID(context.Background(), "batch-7")
	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunCompleted, final.Status)
	assert.Equal(t, models.SubtitleRunRouteTranslate, final.Route, "the lane is ledger data, not a log line")
	assert.Equal(t, "batch-7", final.BatchID, "the consent batch rides the ctx, like the Budget")
	require.NotNil(t, final.CacheHitCues, "a translate lane measures its cache split")
	assert.Equal(t, 4, *final.CacheHitCues)
	assert.Equal(t, 10, final.CueCount)

	require.Len(t, receipts, 1, "exactly one receipt per run")
	assert.Equal(t, final.ID, receipts[0].ID)
	assert.Equal(t, models.SubtitleRunCompleted, receipts[0].Status)
	payload := receipts[0].ReceiptPayload()
	assert.Equal(t, 4, payload["cache_hit_cues"])
	assert.Equal(t, "batch-7", payload["batch_id"])
}

func TestProcessItem_LedgerColdCacheIsZeroNotNull(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning.", "Good night."))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	final := h.runs.lastUpdate(t)
	require.NotNil(t, final.CacheHitCues, "the split happened — 0 hits is a measurement")
	assert.Equal(t, 0, *final.CacheHitCues)
	assert.Equal(t, "", final.BatchID, "a pool / solo run belongs to no batch")
}

func TestProcessItem_LedgerSkippedRunStillGetsAReceipt(t *testing.T) {
	var receipts []models.SubtitleRun
	h := newItemHarness(t, RouteDecision{
		Kind:   RouteSkip,
		Reason: "only a Japanese text track — nothing to translate",
	}, WithRunReceipt(func(run *models.SubtitleRun) { receipts = append(receipts, *run) }))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunSkipped, final.Status)
	assert.Equal(t, models.SubtitleRunRouteSkip, final.Route)
	assert.Nil(t, final.CacheHitCues, "no translate lane ran — cache hits are NOT measured, not 0")
	require.Len(t, receipts, 1)
	assert.Equal(t, models.SubtitleRunSkipped, receipts[0].Status)
	assert.NotContains(t, receipts[0].ReceiptPayload(), "cache_hit_cues")
}

func TestProcessItem_LedgerFailedRunGetsAReceiptToo(t *testing.T) {
	var receipts []models.SubtitleRun
	h := newItemHarness(t, translateDecision("Good morning."),
		WithRunReceipt(func(run *models.SubtitleRun) { receipts = append(receipts, *run) }))
	h.placer.err = assert.AnError

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)

	require.Len(t, receipts, 1, "a failed run is still money spent — it gets its receipt")
	assert.Equal(t, models.SubtitleRunFailed, receipts[0].Status)
	assert.Equal(t, models.SubtitleRunRouteTranslate, receipts[0].Route)
}

// infra-optin-usage-report-a1 — the ledger records who started the run.

func TestProcessItem_LedgerDefaultsToManualTrigger(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunTriggeredManual, final.TriggeredBy,
		"a caller that does not say Automatic is a person — it must never count as 'produced on its own'")
}

func TestProcessItem_LedgerRecordsAutomaticTrigger(t *testing.T) {
	h := newItemHarness(t, RouteDecision{
		Kind:  RouteDeliverDirect,
		Track: &ExtractedTrack{StreamIndex: 3, Language: "chi", Blocks: cues("早安。")},
	})

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{FreeOnly: true, Automatic: true})
	require.NoError(t, err)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunCompleted, final.Status)
	assert.Equal(t, models.SubtitleRunTriggeredAuto, final.TriggeredBy)
}

// ─── disc-2026-11-spend-history-a-ledger ────────────────────────────────────

type fakeRunEstimator struct {
	usd   float64
	err   error
	calls []string
}

func (f *fakeRunEstimator) EstimateRunUSD(_ context.Context, mediaType, mediaID string) (float64, error) {
	f.calls = append(f.calls, mediaType+":"+mediaID)
	return f.usd, f.err
}

// AC #2: the quote is on the row from the moment it is created, and it is the
// estimator's figure for THIS item.
func TestProcessItem_LedgerRecordsRunStartEstimate(t *testing.T) {
	est := &fakeRunEstimator{usd: 0.61}
	var receipts []models.SubtitleRun
	h := newItemHarness(t, translateDecision("Good morning."), WithRunEstimator(est),
		WithRunReceipt(func(run *models.SubtitleRun) { receipts = append(receipts, *run) }))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	assert.Equal(t, []string{"episode:ep-1"}, est.calls, "one estimate, for this item")
	require.NotEmpty(t, h.runs.created)
	require.NotNil(t, h.runs.created[0].EstimateUSD, "stamped at Create, before any spend")
	assert.InDelta(t, 0.61, *h.runs.created[0].EstimateUSD, 1e-9)
	final := h.runs.lastUpdate(t)
	require.NotNil(t, final.EstimateUSD)
	assert.InDelta(t, 0.61, *final.EstimateUSD, 1e-9)
	require.Len(t, receipts, 1)
	assert.InDelta(t, 0.61, receipts[0].ReceiptPayload()["estimate_usd"], 1e-9, "AC #6: recorded → in the receipt")
}

// AC #2: a failing or missing estimator leaves NULL and the run completes.
func TestProcessItem_LedgerEstimateFailureIsNullNotAFailure(t *testing.T) {
	for name, opts := range map[string][]PipelineOption{
		"estimator errors": {WithRunEstimator(&fakeRunEstimator{err: errors.New("ffprobe timed out")})},
		"no estimator":     nil,
	} {
		t.Run(name, func(t *testing.T) {
			var receipts []models.SubtitleRun
			opts = append(opts, WithRunReceipt(func(run *models.SubtitleRun) { receipts = append(receipts, *run) }))
			h := newItemHarness(t, translateDecision("Good morning."), opts...)

			_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
			require.NoError(t, err)

			final := h.runs.lastUpdate(t)
			assert.Equal(t, models.SubtitleRunCompleted, final.Status)
			assert.Nil(t, final.EstimateUSD, "no figure is NULL, never a $0 quote")
			require.Len(t, receipts, 1)
			assert.NotContains(t, receipts[0].ReceiptPayload(), "estimate_usd")
		})
	}
}

// The FreeOnly lane defers every paid route; it is not quoted (and so does not
// fire the estimate's probes for every scanned item).
func TestProcessItem_LedgerFreeOnlyLaneIsNotQuoted(t *testing.T) {
	est := &fakeRunEstimator{usd: 0.5}
	h := newItemHarness(t, RouteDecision{
		Kind:  RouteDeliverDirect,
		Track: &ExtractedTrack{StreamIndex: 3, Language: "chi", Blocks: cues("早安。")},
	}, WithRunEstimator(est))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{FreeOnly: true, Automatic: true})
	require.NoError(t, err)

	assert.Empty(t, est.calls)
	assert.Nil(t, h.runs.lastUpdate(t).EstimateUSD)
}

// AC #3: the source column, per routing verdict.
func TestProcessItem_LedgerRecordsSource(t *testing.T) {
	cases := []struct {
		name     string
		decision RouteDecision
		want     string
	}{
		{"embedded English track", RouteDecision{
			Kind: RouteTranslate, Track: &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: cues("Good morning.")},
		}, models.SubtitleRunSourceEmbeddedText},
		{"embedded Chinese track delivered", RouteDecision{
			Kind: RouteDeliverDirect, Track: &ExtractedTrack{StreamIndex: 3, Language: "chi", Blocks: cues("早安。")},
		}, models.SubtitleRunSourceEmbeddedText},
		{"English sidecar beside the file", RouteDecision{
			Kind: RouteTranslate, Track: &ExtractedTrack{StreamIndex: -1, Language: "eng", Blocks: cues("Good morning.")},
		}, models.SubtitleRunSourceSidecar},
		{"skip", RouteDecision{Kind: RouteSkip, Reason: "only a Japanese text track"}, ""},
		{"no text source, no ASR", noTextDecision(), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newItemHarness(t, tc.decision)

			_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
			require.NoError(t, err)

			final := h.runs.lastUpdate(t)
			assert.Equal(t, tc.want, final.Source)
			assert.Equal(t, string(tc.decision.Kind), final.Route, "route is unchanged by the source column")
			assert.Equal(t, "", final.ASRProvider, "AC #4: no ASR ran")
			assert.Equal(t, "", final.ASRModel)
		})
	}
}

// AC #3/#4: the ASR fallback records source=audio and the engine the service
// named on the pipeline's own row; route still flips to "asr" as before.
func TestProcessItem_LedgerASRFallbackRecordsAudioAndEngine(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true, engine: services.ASRProviderElevenLabs, engineModel: "scribe_v2"}
	h := asrHarness(t, noTextDecision(), asr)
	zhPath := ExpectedSidecarPath(h.mediaPath)
	asr.onRun = func() {
		require.NoError(t, os.WriteFile(zhPath, []byte("1\n00:00:01,000 --> 00:00:02,000\n早安\n\n"), 0o644))
	}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunCompleted, final.Status)
	assert.Equal(t, models.SubtitleRunRouteASR, final.Route)
	assert.Equal(t, models.SubtitleRunSourceAudio, final.Source)
	assert.Equal(t, services.ASRProviderElevenLabs, final.ASRProvider)
	assert.Equal(t, "scribe_v2", final.ASRModel)
}

// AC #3/#4: a budget pause and a mid-ASR failure still ran speech recognition.
func TestProcessItem_LedgerASRFallbackPauseAndFailureKeepEngine(t *testing.T) {
	for name, runErr := range map[string]error{
		"budget pause": fmt.Errorf("asr: %w", ai.ErrBudgetExceeded),
		"asr failed":   errors.New("scribe 500"),
	} {
		t.Run(name, func(t *testing.T) {
			asr := &fakeSpeechTranscriber{available: true, err: runErr, engine: services.ASRProviderOpenAI}
			h := asrHarness(t, noTextDecision(), asr)

			_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
			require.Error(t, err)

			final := h.runs.lastUpdate(t)
			assert.Equal(t, models.SubtitleRunFailed, final.Status)
			assert.Equal(t, models.SubtitleRunSourceAudio, final.Source)
			assert.Equal(t, services.ASRProviderOpenAI, final.ASRProvider)
			assert.Equal(t, "", final.ASRModel, "an unknowable model stays NULL")
		})
	}
}

// AC #3/#4: the service declining (no ASR key) degrades to no_text_source —
// nothing came from the audio and no engine ran.
func TestProcessItem_LedgerASRDeclinedLeavesSourceAndEngineNull(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true, err: services.ErrTranscriptionDisabled, engine: services.ASRProviderOpenAI}
	h := asrHarness(t, noTextDecision(), asr)

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunRouteNoTextSource, final.Route)
	assert.Equal(t, "", final.Source)
	assert.Equal(t, "", final.ASRProvider)
}
