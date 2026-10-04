package subtitle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
