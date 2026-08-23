package subtitle

// COST-CONSENT GUARD — the free lane (Story 9R-10b AC #5, AC #6).
//
// SIBLING TO `apps/api/internal/cost_consent_test.go`, which guards the other
// half of the same invariant. That file scans production source and fails if
// the library-wide PAID sweep (`WorkerPool.EnqueueMissing`) regains a caller.
// It is unchanged by 9R-10b and must stay that way: the invariant it protects
// did not change, and its header instruction to "delete this test" is addressed
// to whoever re-enables the PAID sweep. That is not this story.
//
// What this file adds is the assertion that source scanning cannot make: that
// the auto-trigger, driven end to end through the REAL Pipeline rather than a
// stub processor, reaches neither the LLM nor speech recognition.
//
// THE INCIDENT (2026-08-07). The first production run of
// VIDO_SUBTITLE_PIPELINE_MODE=pipeline wired the sweep into scan-complete. One
// press of 掃描媒體庫 enqueued 1026 items, roughly two thirds onto paid speech
// recognition (whisper-1 at $0.006/min, ~US$200 for the library), and no number
// was ever shown. The NAS had to be reverted to legacy mode to stop the spend.
//
// THE RULINGS. 2026-08-07 (Alexyu): scanning updates metadata and nothing else;
// paid generation is chosen explicitly on a screen that shows the estimate
// first. 2026-08-19 (Alexyu): 「9R-10b 花錢須同意」 — automation may run the
// zero-cost work, and must stop at the threshold of anything that bills.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// autoOverPipeline drives the REAL Pipeline through the auto-trigger for a
// single movie candidate, and hands back the fakes that would have been billed.
func autoOverPipeline(t *testing.T, decision RouteDecision, asr *fakeSpeechTranscriber) *itemHarness {
	t.Helper()

	var h *itemHarness
	if asr != nil {
		h = asrHarness(t, decision, asr)
	} else {
		h = newItemHarness(t, decision)
		h.ref = MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaMovie}
		h.media.item.ShowKey = ""
	}
	h.media.item.SubtitleStatus = models.SubtitleStatusNotSearched

	gen := NewAutoGenerator(
		h.pipeline,
		&autoFakeLibraryPolicy{enabled: autoEnabledSet("lib-a")},
		nil,
		WithAutoCandidateFinders(
			&autoFakeMovieFinder{movies: []models.Movie{
				autoMovieIn(h.ref.ID, "lib-a", models.SubtitleStatusNotSearched),
			}},
			nil,
		),
	)
	gen.Run(context.Background())
	return h
}

// TestFreeLane_NeverReachesPaidPorts is the assertion that matters most in this
// repository: not "the status is right" but "nothing was billed".
func TestFreeLane_NeverReachesPaidPorts(t *testing.T) {
	t.Run("an English track is NOT sent to the LLM", func(t *testing.T) {
		h := autoOverPipeline(t, translateDecision("Good morning", "How are you"), nil)

		assert.Empty(t, h.trans.calls,
			"the auto-trigger reached the translator — this is the 2026-08-19 ruling broken, and every call here is money")
	})

	t.Run("a track-less file is NOT sent to speech recognition", func(t *testing.T) {
		asr := &fakeSpeechTranscriber{available: true}
		h := autoOverPipeline(t, noTextDecision(), asr)

		assert.Empty(t, asr.calls,
			"the auto-trigger reached speech recognition — this is the exact 2026-08-07 spend path")
		assert.Empty(t, h.trans.calls)
	})
}

// TestFreeLane_FreeRoutesStillComplete is the other side of the guard: a lane
// that refuses everything would also pass the assertions above while shipping
// nothing. The zero-cost routes must actually finish.
func TestFreeLane_FreeRoutesStillComplete(t *testing.T) {
	for name, decision := range map[string]RouteDecision{
		"embedded Traditional track delivered as-is": deliverDirectDecision(),
		"embedded Simplified track converted local":  convertThenDeliverDecision(),
	} {
		t.Run(name, func(t *testing.T) {
			h := autoOverPipeline(t, decision, nil)

			assert.Empty(t, h.trans.calls, "the free routes must not touch the LLM either")
			require.NotEmpty(t, h.placer.requests, "a free route must actually place a sidecar")
			assert.Equal(t, models.SubtitleStatusFound, h.media.statuses()[len(h.media.statuses())-1])
		})
	}
}

// TestFreeLane_DeferredItemStaysOnTheConsentList is AC #5 end to end.
//
// The consent screen is not a queue — GenerationCandidateService recomputes it
// from the repository predicate every time it opens, and that predicate keys on
// subtitle_language (missingZhHantSubtitleWhere, movie_repository.go:898).
// So "the deferred item still shows up with its estimate" reduces to: the auto
// lane must not stamp a language or a path on it. If it ever did, the item
// would vanish from the estimate list AND from the badge's honest vocabulary —
// the user would simply never be offered the choice again.
func TestFreeLane_DeferredItemStaysOnTheConsentList(t *testing.T) {
	for name, tc := range map[string]struct {
		decision RouteDecision
		asr      *fakeSpeechTranscriber
	}{
		"translate":      {decision: translateDecision("Good morning")},
		"no_text_source": {decision: noTextDecision(), asr: &fakeSpeechTranscriber{available: true}},
	} {
		t.Run(name, func(t *testing.T) {
			h := autoOverPipeline(t, tc.decision, tc.asr)

			require.NotEmpty(t, h.media.writes)
			for _, w := range h.media.writes {
				assert.NotEqual(t, "zh-Hant", w.language,
					"stamping zh-Hant removes the item from missingZhHantSubtitleWhere — it would silently leave the consent list")
				assert.Empty(t, w.path, "a deferred item has no sidecar to point at")
			}
			assert.Empty(t, h.placer.requests, "nothing may be written to the media folder for a deferred item")
		})
	}
}

// ─── CR-249: the deferral marker's writer and reader must agree ───────────

// TestDeferredMarker_WriterAndReaderAgree closes a silent-failure gap.
//
// DeferredPaidRunPrefix is written by Pipeline.deferPaidItem and read back by
// AutoGenerator.excludedMediaIDs. Nothing else connects them: if the writer's
// message format drifts, the reader simply stops matching, the exclusion set
// goes empty, and the H1 starvation bug returns — silently, with every existing
// test still green, because every other test stubs one side or the other.
//
// This is the one test that exercises the REAL writer and the REAL reader
// against each other.
func TestDeferredMarker_WriterAndReaderAgree(t *testing.T) {
	h := autoOverPipeline(t, translateDecision("Good morning"), nil)

	require.NotEmpty(t, h.runs.updated, "the deferral must record a run row")
	written := h.runs.updated[len(h.runs.updated)-1]
	require.Equal(t, models.SubtitleRunSkipped, written.Status)

	// Feed exactly what the writer produced into the reader's own predicate.
	gen := NewAutoGenerator(&autoFakeItemProcessor{}, &autoFakeLibraryPolicy{enabled: autoEnabledSet("lib-a")}, nil,
		WithAutoDeferredRuns(&autoFakeRunLister{runs: []models.SubtitleRun{{
			MediaID:      h.ref.ID,
			Status:       written.Status,
			ErrorMessage: written.ErrorMessage,
			// The writer stamps the real clock; the reader ignores rows from
			// before FreeLaneEpoch (bugfix-auto-exclusion-never-expires D2).
			StartedAt: written.StartedAt,
		}}}),
	)
	require.True(t, written.StartedAt.After(freeLaneEpoch),
		"the harness clock predates FreeLaneEpoch — the reader would treat a fresh deferral as stale")
	got, err := gen.excludedMediaIDs(context.Background())
	require.NoError(t, err)

	assert.Contains(t, got, h.ref.ID,
		"the reader failed to recognise a marker the writer just produced — the exclusion set would go empty and paid items would re-occupy the budget every scan")
}
