package subtitle

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
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

func TestTranslateTrack_TransientDuringQualityRetryDegradesOnlyThePendingCues(t *testing.T) {
	// 10 cues, one chunk. Call 1 answers all but cue 3 (gate rejects it); the
	// quality retry for cue 3 (call 2) then times out through every attempt.
	// Only cue 3 ships English; the 9 accepted cues are untouched.
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
	require.NoError(t, err, "1/10 = 10%: over FR16's 5% quality ceiling, but this cue is TRANSPORT-stubborn and the 20% ceiling applies")

	require.Len(t, tr.calls, 2, "the retry that timed out is not retried again here (D8)")
	assert.Equal(t, []int{3}, tr.calls[1].indexes())
	assert.Equal(t, 1, res.StubbornCues)
	assert.Equal(t, 1, res.TransientCues)
	assert.Equal(t, "Line 3.", res.Blocks[2].Text)
	assert.Equal(t, "第4句", res.Blocks[3].Text)
}

// ─── AC #3: only transient-exhausted degrades; everything else still stops ─

func TestTranslateTrack_PermanentErrorsStillFailTheRunImmediately(t *testing.T) {
	cases := map[string]error{
		"401 unauthorized": fmt.Errorf("translate chunk: %w: %w: status 401", ai.ErrAIProviderError, ai.ErrAIUnauthorized),
		"404 model":        fmt.Errorf("translate chunk: %w: %w: status 404", ai.ErrAIProviderError, ai.ErrAIModelNotFound),
		"budget ceiling":   fmt.Errorf("translate chunk: %w", ai.ErrBudgetExceeded),
		"malformed body":   fmt.Errorf("translate chunk: %w", ai.ErrAIInvalidResponse),
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

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	assert.False(t, isTransientExhausted(cancelled, exhausted))
}
