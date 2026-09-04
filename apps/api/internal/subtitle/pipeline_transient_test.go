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
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
)

// ─── sub-6-2 AC #3/#4/#5c: a transient-exhausted chunk keeps English ──────

// exhaustedTimeout is what the ai layer returns after three timeouts: the
// timeout sentinel wrapped with the exhausted verdict, itself wrapped by the
// translation service's chunk context — the exact chain the pipeline sees.
func exhaustedTimeout() error {
	return fmt.Errorf("translate chunk [1-10]: %w", fmt.Errorf("%w: %w after 3 attempts", ai.ErrAITimeout, ai.ErrAIRetriesExhausted))
}

func numberedCues(n int) []SubtitleBlock {
	texts := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		texts = append(texts, fmt.Sprintf("Line %d.", i))
	}
	return cues(texts...)
}

func translateAll(blocks []prompts.SubtitleTranslatorBlock) map[int]string {
	out := make(map[int]string, len(blocks))
	for _, b := range blocks {
		out[b.Index] = fmt.Sprintf("第%d句", b.Index)
	}
	return out
}

// failCallsWith returns a translator that fails the listed 1-based calls with
// err and answers every other call cleanly.
func failCallsWith(err error, failing ...int) *fakeTranslator {
	fails := make(map[int]struct{}, len(failing))
	for _, c := range failing {
		fails[c] = struct{}{}
	}
	return &fakeTranslator{
		fn: func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			if _, ok := fails[call]; ok {
				return nil, ai.CompletionUsage{}, err
			}
			return translateAll(blocks), ai.CompletionUsage{InputTokens: 10}, nil
		},
	}
}

func TestTranslateTrack_TransientExhaustedChunkKeepsEnglishAndContinues(t *testing.T) {
	// 60 cues = 6 chunks; chunk 2 (cues 11-20) times out through every retry.
	// 10/60 = 16.7% English — under the 20% transport ceiling.
	source := numberedCues(60)
	tr := failCallsWith(exhaustedTimeout(), 2)

	scope := &processScope{ref: MediaRef{ID: "m1", MediaType: "movie"}}
	ctx := withProcessScope(context.Background(), scope)

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(ctx, trackOf(source), TranslateContext{})
	require.NoError(t, err, "one flaky chunk must not kill a run the other five chunks already paid for")

	require.Len(t, tr.calls, 6, "the chunks after the failed one are still sent — no second transport loop, no abort")
	assert.Equal(t, 10, res.StubbornCues)
	assert.Equal(t, 10, res.TransientCues)
	for i := 11; i <= 20; i++ {
		assert.Equal(t, fmt.Sprintf("Line %d.", i), res.Blocks[i-1].Text, "cue %d keeps its English original (NFR-R1)", i)
		_, noted := scope.stubbornIndexes[i]
		assert.True(t, noted, "cue %d is kept OUT of the segment cache so the next run retries it for real", i)
	}
	assert.Equal(t, "第10句", res.Blocks[9].Text)
	assert.Equal(t, "第21句", res.Blocks[20].Text, "the chunk after the outage translates normally")
	assert.Len(t, scope.stubbornIndexes, 10)
}

func TestTranslateTrack_TransientOverCeilingFailsAsSoonAsItIsCertain(t *testing.T) {
	// 40 cues = 4 chunks. Ceiling is 20% → 8 cues. Chunk 1 fails (10 English
	// > 8) — the run must stop THERE, not send chunks 2-4 into the same outage.
	source := numberedCues(40)
	tr := failCallsWith(exhaustedTimeout(), 1, 2, 3, 4)

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
	assert.Contains(t, err.Error(), "20% ceiling")
	assert.Nil(t, res, "a track that would ship one-fifth English ships nothing")
	assert.Len(t, tr.calls, 1, "fail fast: no further chunk is sent once the ceiling is certain")
}

func TestTranslateTrack_TransientCeilingCountsQualityStubbornToo(t *testing.T) {
	// 20 cues = 2 chunks. Chunk 1 answers cleanly except cue 1, which fails the
	// gate through every quality retry (1/20 = 5%, exactly the FR16 ceiling).
	// Chunk 2 then times out (10 more English). 11/20 = 55% — over 20%.
	source := numberedCues(20)
	tr := &fakeTranslator{
		fn: func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			if blocks[0].Index == 11 {
				return nil, ai.CompletionUsage{}, exhaustedTimeout()
			}
			out := translateAll(blocks)
			delete(out, 1) // missing → gate rejects → quality retry → stubborn
			return out, ai.CompletionUsage{}, nil
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
	assert.Contains(t, err.Error(), "11 of 20 cues")
	assert.Nil(t, res)
}

func TestTranslateTrack_TransientDuringQualityRetryCountsInBothPools(t *testing.T) {
	// 20 cues, two chunks. Call 1 answers all of chunk 1 but cue 3 (gate
	// rejects it); the quality retry for cue 3 (call 2) then times out through
	// every attempt. Chunk 2 is clean. Cue 3 is BOTH a gate failure (it was
	// rejected once and never got its retry — FR16's 5% pool, 1/20 = exactly
	// the ceiling) and a transport failure (CR M3: the two pools are index
	// sets, so it is counted once in the 20% union). Only cue 3 ships English.
	source := numberedCues(20)
	tr := &fakeTranslator{
		fn: func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			switch call {
			case 1:
				out := translateAll(blocks)
				delete(out, 3)
				return out, ai.CompletionUsage{}, nil
			case 2:
				return nil, ai.CompletionUsage{}, exhaustedTimeout()
			default:
				return translateAll(blocks), ai.CompletionUsage{}, nil
			}
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err)

	require.Len(t, tr.calls, 3, "the retry that timed out is not retried again here (D8); chunk 2 still runs")
	assert.Equal(t, []int{3}, tr.calls[1].indexes())
	assert.Equal(t, 1, res.StubbornCues, "one English cue, not two — the pools dedupe")
	assert.Equal(t, 1, res.TransientCues)
	assert.Equal(t, "Line 3.", res.Blocks[2].Text)
	assert.Equal(t, "第4句", res.Blocks[3].Text)
}

func TestTranslateTrack_TransientDuringQualityRetryStillTripsTheQualityCeiling(t *testing.T) {
	// CR M3: the reviewer's scenario. A gate-rejected cue whose retry times out
	// must NOT escape FR16 by being reclassified as "merely transient" — 1 of
	// 10 is 10%, over the 5% quality ceiling, whatever killed its retry.
	source := numberedCues(10)
	tr := &fakeTranslator{
		fn: func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			if call == 1 {
				out := translateAll(blocks)
				delete(out, 3)
				return out, ai.CompletionUsage{}, nil
			}
			return nil, ai.CompletionUsage{}, exhaustedTimeout()
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
	assert.Contains(t, err.Error(), "quality gate")
	assert.Nil(t, res)
}

// ─── CR H1: the wall-clock circuit breaker ─────────────────────────────────

func TestTranslateTrack_TwoConsecutiveDeadChunksStopTheRun(t *testing.T) {
	// 200 cues = 20 chunks; the 20% ceiling would tolerate 40 English cues, so
	// a dead provider could burn four chunks × ~5 minutes of retries before the
	// ceiling spoke. Two whole chunks failing back-to-back is the breaker.
	source := numberedCues(200)
	tr := failCallsWith(exhaustedTimeout(), 1, 2, 3, 4, 5)

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
	assert.Contains(t, err.Error(), "provider unreachable")
	assert.ErrorIs(t, err, ai.ErrAITimeout, "the cause stays in the chain (Rule 13)")
	assert.Nil(t, res)
	assert.Len(t, tr.calls, maxConsecutiveTransientChunks, "no third chunk is sent into a dead provider")
}

func TestTranslateTrack_ASuccessBetweenDeadChunksResetsTheBreaker(t *testing.T) {
	// Chunks 1 and 3 die, chunk 2 succeeds: two flaky chunks, not a dead
	// provider. 20/100 = 20% — not OVER the ceiling — so the run completes.
	source := numberedCues(100)
	tr := failCallsWith(exhaustedTimeout(), 1, 3)

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err)

	assert.Len(t, tr.calls, 10)
	assert.Equal(t, 20, res.TransientCues)
	assert.Equal(t, "Line 1.", res.Blocks[0].Text)
	assert.Equal(t, "第11句", res.Blocks[10].Text)
	assert.Equal(t, "Line 21.", res.Blocks[20].Text)
}

// ─── CR M5: a stubborn chunk does not poison the next chunk's context ──────

func TestTranslateTrack_ContextWindowSkipsUntranslatedCues(t *testing.T) {
	// Chunk 2 (cues 11-20) dies. Chunk 3's read-only context must be the last
	// five TRANSLATED cues — 6-10 from chunk 1 — not the English of 16-20,
	// which the prompt would present as "how the previous lines were
	// translated".
	source := numberedCues(60) // 10/60 English stays under the 20% ceiling
	tr := failCallsWith(exhaustedTimeout(), 2)

	_, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err)

	require.Len(t, tr.calls, 6)
	ctxIndexes := make([]int, 0, 5)
	for _, b := range tr.calls[2].contextBlocks {
		ctxIndexes = append(ctxIndexes, b.Index)
		assert.NotContains(t, b.Text, "Line ", "context never carries an untranslated cue")
	}
	assert.Equal(t, []int{6, 7, 8, 9, 10}, ctxIndexes, "the window reaches back past the English chunk, in prompt order")
}

func TestContextWindow_EmptyWhenNothingBehindIsTranslated(t *testing.T) {
	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil)
	source := numberedCues(12)

	assert.Nil(t, p.contextWindow(source, 0, map[int]string{}), "chunk 1 has no context")
	assert.Nil(t, p.contextWindow(source, 10, map[int]string{}), "an all-English chunk behind us is no context at all")

	final := map[int]string{2: "第2句", 5: "第5句", 9: "第9句"}
	got := p.contextWindow(source, 10, final)
	require.Len(t, got, 3)
	assert.Equal(t, []prompts.SubtitleTranslatorBlock{{Index: 2, Text: "第2句"}, {Index: 5, Text: "第5句"}, {Index: 9, Text: "第9句"}}, got)
}

// ─── AC #3: only transient-exhausted degrades; everything else still stops ─

func TestTranslateTrack_PermanentErrorsStillFailTheRunImmediately(t *testing.T) {
	cases := map[string]error{
		"401 unauthorized": fmt.Errorf("translate chunk: %w: %w: status 401", ai.ErrAIProviderError, ai.ErrAIUnauthorized),
		"404 model":        fmt.Errorf("translate chunk: %w: %w: status 404", ai.ErrAIProviderError, ai.ErrAIModelNotFound),
		"budget ceiling":   fmt.Errorf("translate chunk: %w", ai.ErrBudgetExceeded),
		"provider quota (429 through every retry)": fmt.Errorf("translate chunk: %w: %w after 3 attempts", ai.ErrAIQuotaExceeded, ai.ErrAIRetriesExhausted),
		"malformed body": fmt.Errorf("translate chunk: %w", ai.ErrAIInvalidResponse),
		"single timeout without exhaustion (ctx ended the window)": fmt.Errorf("translate chunk: %w", ai.ErrAITimeout),
	}
	for name, cause := range cases {
		t.Run(name, func(t *testing.T) {
			tr := failCallsWith(cause, 1)

			res, err := NewPipeline(tr, &recordingConverter{}, nil).
				TranslateTrack(context.Background(), trackOf(numberedCues(30)), TranslateContext{})

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
			assert.ErrorIs(t, err, cause)
			assert.Nil(t, res)
			assert.Len(t, tr.calls, 1, "a rejection that would repeat must not be sent again chunk after chunk")
		})
	}
}

func TestTranslateTrack_BudgetExceededNeverDegradesEvenIfWrappedAsExhausted(t *testing.T) {
	// Defensive: governed() short-circuits ABOVE the retry loop today, so the
	// budget sentinel never carries the exhausted wrapper — but if a future
	// change ever did wrap it, "stop spending" must still win over "keep going".
	cause := fmt.Errorf("%w: %w", ai.ErrBudgetExceeded, ai.ErrAIRetriesExhausted)
	tr := failCallsWith(cause, 1)

	_, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(numberedCues(30)), TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded)
	assert.Len(t, tr.calls, 1)
}

func TestTranslateTrack_CancelledCtxNeverDegrades(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tr := &fakeTranslator{
		fn: func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			cancel() // shutdown lands mid-request; the ai layer reports exhaustion
			return nil, ai.CompletionUsage{}, exhaustedTimeout()
		},
	}

	_, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(ctx, trackOf(numberedCues(30)), TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
	assert.Len(t, tr.calls, 1, "a shutdown is the caller's decision — no English chunk, no next chunk")
}

// ─── AC #4: the retry is narrated on the progress stream ───────────────────

func TestTranslateTrack_NarratesTransportRetriesAndTheEnglishVerdict(t *testing.T) {
	var messages []string
	progress := func(ref MediaRef, stage PipelineStage, message string) {
		assert.Equal(t, StageTranslating, stage)
		assert.Equal(t, "m1", ref.ID, "the narration carries the item identity the SSE hook keys on")
		messages = append(messages, message)
	}

	// The fake plays the ai layer: two retry notices through the ctx observer
	// (exactly as retryTransient would), then the exhausted verdict.
	tr := &fakeTranslator{}
	tr.fn = func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		if call == 2 {
			return nil, ai.CompletionUsage{}, exhaustedTimeout()
		}
		return translateAll(blocks), ai.CompletionUsage{}, nil
	}
	narrating := &narratingTranslator{inner: tr, notices: []ai.RetryNotice{
		{Op: "claude.messages", Attempt: 1, MaxAttempts: 3, Err: ai.ErrAITimeout},
		{Op: "claude.messages", Attempt: 2, MaxAttempts: 3, Err: fmt.Errorf("%w: status 503", ai.ErrAIProviderError)},
	}}

	scope := &processScope{ref: MediaRef{ID: "m1", MediaType: "movie"}}
	ctx := withProcessScope(context.Background(), scope)

	_, err := NewPipeline(narrating, &recordingConverter{}, nil, WithProgress(progress)).
		TranslateTrack(ctx, trackOf(numberedCues(60)), TranslateContext{})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"translating chunk 1/6",
		"chunk 2/6 timed out, retrying 2/3",
		"chunk 2/6 failed transiently, retrying 3/3",
		"chunk 2/6 kept English after transport retries",
		"translating chunk 3/6",
		"translating chunk 4/6",
		"translating chunk 5/6",
		"translating chunk 6/6",
	}, messages)
}

// narratingTranslator invokes the ctx-attached ai.RetryObserver with the
// scripted notices before delegating — a fake provider narrating a retry the
// way the real retryTransient does — for calls that fail.
type narratingTranslator struct {
	inner   *fakeTranslator
	notices []ai.RetryNotice
}

func (n *narratingTranslator) TranslateChunk(ctx context.Context, sys []ai.SystemBlock, contextBlocks, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, map[string]string, ai.CompletionUsage, error) {
	got, terms, usage, err := n.inner.TranslateChunk(ctx, sys, contextBlocks, blocks)
	if err != nil {
		if observer := ai.RetryObserverFrom(ctx); observer != nil {
			for _, notice := range n.notices {
				observer(notice)
			}
		}
	}
	return got, terms, usage, err
}

func TestIsTransientExhausted(t *testing.T) {
	exhausted := exhaustedTimeout()
	assert.True(t, isTransientExhausted(context.Background(), exhausted))
	assert.False(t, isTransientExhausted(context.Background(), errors.New("plain")))
	assert.False(t, isTransientExhausted(context.Background(), fmt.Errorf("%w: %w", ai.ErrBudgetExceeded, ai.ErrAIRetriesExhausted)))
	assert.False(t, isTransientExhausted(context.Background(), fmt.Errorf("%w: %w", ai.ErrAIQuotaExceeded, ai.ErrAIRetriesExhausted)),
		"a provider-side quota is the provider's budget ceiling — stop, do not degrade (CR M4)")

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	assert.False(t, isTransientExhausted(cancelled, exhausted))
}

// ─── CR H2 / M8: the item flow stamps the counts and keeps a partial retryable ─

func TestProcessItem_StampsEnglishCueCountsOnTheRun(t *testing.T) {
	texts := longTrack(40)
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: cues(texts...)},
		DetectedVariant: LangUndetermined,
	})
	h.trans.fn = echoingTranslator(texts[1]) // 1 quality-stubborn cue of 40

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	run := h.runs.updated[len(h.runs.updated)-1]
	require.Equal(t, models.SubtitleRunCompleted, run.Status)
	require.NotNil(t, run.StubbornCount, "AC #3: the completed run records its English cues")
	assert.Equal(t, 1, *run.StubbornCount)
	require.NotNil(t, run.TransientCount)
	assert.Equal(t, 0, *run.TransientCount, "a quality-stubborn cue is not a transport failure")
	assert.Equal(t, models.SubtitleStatusFound, h.media.writes[len(h.media.writes)-1].status,
		"a quality-stubborn delivery is final, exactly as before sub-6-2")
}

func TestProcessItem_TransientDeliveryStaysQueuedAndRetriesOnlyTheEnglish(t *testing.T) {
	texts := longTrack(60) // 6 chunks; chunk 2 (cues 11-20) dies → 10/60 English, under the 20% ceiling
	source := cues(texts...)
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	})
	h.media.item.SubtitleStatus = models.SubtitleStatusNotFound
	h.media.item.SubtitlePath = ""
	healthy := func(blocks []prompts.SubtitleTranslatorBlock) map[int]string {
		out := make(map[int]string, len(blocks))
		for _, b := range blocks {
			out[b.Index] = "早安"
		}
		return out
	}
	h.trans.fn = func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		if call == 2 {
			return nil, ai.CompletionUsage{}, exhaustedTimeout()
		}
		return healthy(blocks), ai.CompletionUsage{CacheCreationInputTokens: 5000}, nil
	}

	out, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err, "one dead chunk of six must not fail the item")
	require.NotNil(t, out)

	// The file IS delivered, English lines included (NFR-R1 fail-soft) …
	require.Len(t, h.placer.requests, 1)
	assert.Contains(t, string(h.placer.requests[0].SubtitleData), texts[15])
	assert.Contains(t, string(h.placer.requests[0].SubtitleData), "早安")

	// … the run says how much of it is English and why …
	run := h.runs.updated[len(h.runs.updated)-1]
	require.Equal(t, models.SubtitleRunCompleted, run.Status)
	require.NotNil(t, run.StubbornCount)
	require.NotNil(t, run.TransientCount)
	assert.Equal(t, 10, *run.StubbornCount)
	assert.Equal(t, 10, *run.TransientCount)

	// … the media row goes back where Load found it, NOT `found` (CR H2):
	// `found` + zh-Hant would drop the item out of every enumeration, and the
	// English would be frozen until someone forced a re-run.
	last := h.media.writes[len(h.media.writes)-1]
	assert.Equal(t, models.SubtitleStatusNotFound, last.status, "a partial delivery keeps the item a candidate")
	assert.Contains(t, h.progress, "complete")

	// … and the translated cues are cached while the English ones are not.
	version := h.pipeline.runVersion(richContext())
	assert.Equal(t, "早安", h.cache.writes[segmentKey(source[0].Text, version)])
	_, cached := h.cache.writes[segmentKey(source[15].Text, version)]
	assert.False(t, cached, "a transport-stubborn cue must not be frozen into the cache")

	// ── The retry: sidecar on disk + a version-matched completed run that
	// says transient_count=10. The P5 pre-flight would early-exit on the
	// sidecar alone; it must let this one through and re-send ONLY the
	// English cues (the rest are cache hits).
	require.NoError(t, os.WriteFile(ExpectedSidecarPath(h.mediaPath), h.placer.requests[0].SubtitleData, 0o600))
	completed := run
	h.runs.completed = &completed
	h.trans.calls = nil
	h.trans.fn = func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		return healthy(blocks), ai.CompletionUsage{CacheReadInputTokens: 5000}, nil
	}

	out, err = h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	require.NotNil(t, out.Run, "the pre-flight must NOT early-exit a partial delivery")

	require.Len(t, h.trans.calls, 1, "only the English cues are re-sent; everything else is a cache hit")
	assert.Equal(t, []int{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, h.trans.calls[0].indexes())
	retried := h.runs.updated[len(h.runs.updated)-1]
	require.Equal(t, models.SubtitleRunCompleted, retried.Status)
	require.NotNil(t, retried.TransientCount)
	assert.Equal(t, 0, *retried.TransientCount)
	assert.Equal(t, models.SubtitleStatusFound, h.media.writes[len(h.media.writes)-1].status,
		"a complete delivery is final")
}

func TestProcessItem_PreflightStillSkipsAFullDelivery(t *testing.T) {
	// The partial-delivery gate must not weaken P5 for the normal case: a
	// version-matched run with transient_count=0 (or NULL, pre-034) plus an
	// acceptable sidecar is still an early-exit.
	for name, transient := range map[string]*int{"zero": intPtr(0), "pre-034 NULL": nil} {
		t.Run(name, func(t *testing.T) {
			h := newItemHarness(t, translateDecision("Good morning."))
			require.NoError(t, os.WriteFile(ExpectedSidecarPath(h.mediaPath), []byte(oneCueSRT), 0o600))
			version := h.pipeline.runVersion(richContext())
			h.runs.completed = &models.SubtitleRun{
				ID: "run-prev", MediaID: h.ref.ID, MediaType: h.ref.MediaType, Status: models.SubtitleRunCompleted,
				MetadataHash: version.MetadataHash, GlossaryVersion: version.GlossaryVersion,
				PromptVersion: version.PromptVersion, ModelID: version.ModelID,
				TransientCount: transient,
			}

			out, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
			require.NoError(t, err)
			assert.Nil(t, out.Run, "early-exit writes no provenance row")
			assert.Empty(t, h.trans.calls)
		})
	}
}

func intPtr(n int) *int { return &n }
