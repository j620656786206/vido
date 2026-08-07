package subtitle

// sub-3-1 — the ASR fallback leg. RouteNoTextSource stops being an
// unconditional dead end: when the OPTIONAL SpeechTranscriber port can serve
// the item, the pipeline falls through to speech recognition instead of
// recording `no_text_source`. AC #9's seven cases live here (a–f) and in
// worker_pool_test.go (g).

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// fakeSpeechTranscriber stands in for the services-side adapter over
// TranscriptionService (Rule 11 test-fake precedent:
// TranscriptionServiceInterface in transcription_handler.go). onRun simulates
// the SERVICE's own writeback — the transcription service, not the pipeline,
// is the sole writer of the ASR outcome on the media row (9R-16 AC 12).
type asrCall struct{ mediaID, filePath, mediaDir string }

type fakeSpeechTranscriber struct {
	available bool
	err       error
	calls     []asrCall
	onRun     func()
}

func (f *fakeSpeechTranscriber) Available() bool { return f.available }

func (f *fakeSpeechTranscriber) Transcribe(_ context.Context, mediaID, filePath, mediaDir string) error {
	f.calls = append(f.calls, asrCall{mediaID: mediaID, filePath: filePath, mediaDir: mediaDir})
	if f.err != nil {
		return f.err
	}
	if f.onRun != nil {
		f.onRun()
	}
	return nil
}

func noTextDecision() RouteDecision {
	return RouteDecision{
		Kind:   RouteNoTextSource,
		Reason: "no usable embedded text subtitle track (2 image-codec track(s), 0 text)",
	}
}

// asrHarness rebinds the shared item harness onto a MOVIE ref — the ASR leg is
// movie-scoped (TranscriptionService's writer/reader are movie-bound today).
func asrHarness(t *testing.T, decision RouteDecision, asr *fakeSpeechTranscriber) *itemHarness {
	t.Helper()
	opts := []PipelineOption{}
	if asr != nil {
		opts = append(opts, WithSpeechTranscriber(asr))
	}
	h := newItemHarness(t, decision, opts...)
	h.ref = MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaMovie}
	h.media.item.ShowKey = "" // movies carry no show key
	return h
}

// ─── AC #9 (a): available ASR → transcription invoked, status = ASR outcome ─

func TestProcessItem_ASRFallback_RunsTranscription(t *testing.T) {
	t.Run("zh-Hant success — media found, run records the sidecar", func(t *testing.T) {
		asr := &fakeSpeechTranscriber{available: true}
		h := asrHarness(t, noTextDecision(), asr)

		zhPath := ExpectedSidecarPath(h.mediaPath)
		asr.onRun = func() {
			// What the real TranscriptionService does on success: place the
			// zh-Hant sidecar and write found/zh-path/zh-Hant (9R-16 AC 12).
			require.NoError(t, os.WriteFile(zhPath,
				[]byte("1\n00:00:01,000 --> 00:00:02,000\n早安\n\n"), 0o644))
			require.NoError(t, h.media.SetSubtitleStatus(context.Background(), h.ref,
				models.SubtitleStatusFound, zhPath, "zh-Hant"))
		}

		outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
		require.NoError(t, err)
		require.NotNil(t, outcome)

		require.Len(t, asr.calls, 1, "the ASR leg runs exactly one transcription")
		assert.Equal(t, asrCall{
			mediaID:  "mv-1",
			filePath: h.mediaPath,
			mediaDir: filepath.Dir(h.mediaPath),
		}, asr.calls[0])

		assert.Equal(t, RouteNoTextSource, outcome.Kind)
		assert.Equal(t, zhPath, outcome.SubtitlePath)

		final := h.runs.lastUpdate(t)
		assert.Equal(t, models.SubtitleRunCompleted, final.Status, "the run reflects the ASR outcome, not a skip")
		assert.Equal(t, zhPath, final.OutputPath)
		assert.Equal(t, 1, final.CueCount)
		assert.Empty(t, final.ErrorMessage)
		require.NotNil(t, final.CompletedAt)

		assert.Equal(t,
			[]models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusFound},
			h.media.statuses(),
			"no no_text_source write anywhere — the service's own verdict is the record")
		assert.NotContains(t, h.progress, string(StageSkipped))
		assert.Contains(t, h.progress, string(StageComplete))
	})

	t.Run("en-only outcome — the service's untranslated verdict stands", func(t *testing.T) {
		asr := &fakeSpeechTranscriber{available: true}
		h := asrHarness(t, noTextDecision(), asr)

		asr.onRun = func() {
			// Translation expected but absent (no key): the service writes
			// untranslated + the EN path — the resume marker (sub-2-2a).
			require.NoError(t, h.media.SetSubtitleStatus(context.Background(), h.ref,
				models.SubtitleStatusUntranslated, h.mediaPath+".en.srt", "en"))
		}

		outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
		require.NoError(t, err)
		require.NotNil(t, outcome)

		final := h.runs.lastUpdate(t)
		assert.Equal(t, models.SubtitleRunCompleted, final.Status)
		assert.Empty(t, final.OutputPath, "no zh-Hant sidecar exists — the run does not claim one")
		assert.Empty(t, outcome.SubtitlePath)

		assert.Equal(t,
			[]models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusUntranslated},
			h.media.statuses(),
			"the pipeline must not overwrite the service's untranslated verdict")
	})
}

// ─── AC #9 (b) + (e): nil port → byte-identical to today ───────────────────

// A nil port is exactly what a legacy-flag deployment, an ASR-less wiring and
// sub-1-5a's translate-only construction all produce (main.go only wires the
// port under `pipeline` mode) — so this case IS the AC #5 legacy guarantee at
// the pipeline seam.
func TestProcessItem_ASRFallback_NilPortRecordsNoTextSourceAsToday(t *testing.T) {
	h := asrHarness(t, noTextDecision(), nil)

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	require.NotNil(t, outcome)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunSkipped, final.Status)
	assert.Equal(t, noTextDecision().Reason, final.ErrorMessage)
	assert.Equal(t,
		[]models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusNoTextSource},
		h.media.statuses())
	assert.Contains(t, h.progress, string(StageSkipped))
}

// ─── AC #9 (c): unavailable service → same recorded outcome as (b) ─────────

// The leg deliberately does NOT pre-probe Available() per item: the service's
// own entry gate is RICHER (it admits ASR-less translate-only resumes — CR
// sub-2-2a M2), so the leg delegates and maps ErrTranscriptionDisabled back to
// today's degrade path. No transcription work happens; the recorded outcome is
// byte-identical to the nil-port case.
func TestProcessItem_ASRFallback_UnavailableServiceDegradesLikeNilPort(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: false, err: services.ErrTranscriptionDisabled}
	h := asrHarness(t, noTextDecision(), asr)

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	require.NotNil(t, outcome)

	require.Len(t, asr.calls, 1, "the service's own gate is the availability decision")

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunSkipped, final.Status)
	assert.Equal(t,
		[]models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusNoTextSource},
		h.media.statuses())
	assert.Contains(t, h.progress, string(StageSkipped))
}

// ─── AC #9 (d): RouteSkip byte-unchanged even with a live port ─────────────

func TestProcessItem_ASRFallback_RouteSkipNeverTouchesASR(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true}
	h := asrHarness(t, RouteDecision{
		Kind:   RouteSkip,
		Reason: "1 embedded text track(s) present but none tagged eng/en",
	}, asr)

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	require.NotNil(t, outcome)

	assert.Empty(t, asr.calls, "skipped is a deliberate routing decision — ASR must not second-guess it")
	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunSkipped, final.Status)
	assert.Equal(t,
		[]models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusSkipped},
		h.media.statuses())
}

// ─── AC #9 (f): budget-exceeded propagates as a pause, not a failure ───────

func TestProcessItem_ASRFallback_BudgetExceededPropagatesAsPause(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true, err: fmt.Errorf("transcribe: %w", ai.ErrBudgetExceeded)}
	h := asrHarness(t, noTextDecision(), asr)

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)
	assert.Nil(t, outcome)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded,
		"the sentinel must survive wrapping so a batch caller can classify the pause (9R-16 precedent)")

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunFailed, final.Status, "the provenance row closes with the budget reason")
	assert.Contains(t, final.ErrorMessage, "AI_BUDGET_EXCEEDED")

	// The media row is deliberately NOT touched: no `no_text_source` (AC #6),
	// and no `not_searched` revert — that would destroy the `untranslated`
	// resume marker the service may have just written and re-pay the whole ASR
	// on resume (sub-2-2a's headline bug class).
	assert.Equal(t,
		[]models.SubtitleStatus{models.SubtitleStatusExtracting},
		h.media.statuses(),
		"no media write after the pause — the service's last write is the truth")
	assert.Contains(t, h.progress, string(StageFailed))
	assert.NotContains(t, h.progress, string(StageSkipped))
}

// ─── Non-budget ASR failures follow the single failure policy ──────────────

func TestProcessItem_ASRFallback_OrdinaryFailureFollowsFailurePolicy(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true, err: errors.New("extract audio: no audio track")}
	h := asrHarness(t, noTextDecision(), asr)

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)
	assert.Nil(t, outcome)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunFailed, final.Status)
	assert.Contains(t, final.ErrorMessage, "no audio track")
	assert.Equal(t,
		[]models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusNotSearched},
		h.media.statuses(),
		"an ordinary ASR failure reverts to not_searched — retryable, like every other stage failure")
}

// ─── Movie-only scope: episodes degrade exactly like a nil port ────────────

// TranscriptionService's status writer and resume reader are movie-repository-
// bound (SetSubtitleStatusWriter/SetSubtitleStateReader take repos.Movies), so
// an episode run would pay full ASR and then fail at the writeback. Episodes
// therefore keep today's behavior until backlog-episode-asr-fallback lands.
func TestProcessItem_ASRFallback_EpisodesDegradeToNoTextSource(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true}
	h := newItemHarness(t, noTextDecision(), WithSpeechTranscriber(asr)) // default ref is an episode

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	require.NotNil(t, outcome)

	assert.Empty(t, asr.calls, "no episode may reach the movie-bound transcription service")
	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunSkipped, final.Status)
	assert.Equal(t,
		[]models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusNoTextSource},
		h.media.statuses())
}
