package subtitle

// Story 9R-10b AC #3 — the FREE-ONLY gate.
//
// WHY THIS SUITE EXISTS. On 2026-08-07 the first production run of
// `VIDO_SUBTITLE_PIPELINE_MODE=pipeline` wired a library-wide sweep into the
// scan-complete callback: one press of 掃描媒體庫 enqueued 1026 items, roughly
// two thirds of them onto PAID speech recognition (~US$200 estimated), and the
// user was never shown a number. The ruling that followed (Alexyu, 2026-08-07)
// is that scanning updates metadata and nothing else.
//
// 9R-10b re-opens automation UNDERNEATH that ruling, not through it: the
// second ruling (Alexyu, 2026-08-19, 「9R-10b 花錢須同意」) allows the pipeline
// to run automatically ONLY as far as the work that costs nothing —
//
//	deliver_direct       an embedded Traditional-Chinese track, delivered as-is
//	convert_then_deliver an embedded Simplified track, converted by local OpenCC
//
// — and requires it to STOP, without calling anything, at the two routes that
// spend money:
//
//	translate            the LLM
//	no_text_source       speech recognition
//
// The assertions below are deliberately on CALL COUNTS of the paid ports, not
// on statuses alone: a status can be right while a request has already been
// billed.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func deliverDirectDecision() RouteDecision {
	return RouteDecision{
		Kind:            RouteDeliverDirect,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "chi", Codec: "subrip", Blocks: cues("早安")},
		DetectedVariant: LangTraditional,
		Reason:          "stream 2 tagged chi but its content is Traditional Chinese — deliver as-is",
	}
}

func convertThenDeliverDecision() RouteDecision {
	return RouteDecision{
		Kind:            RouteConvertThenDeliver,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "chi", Codec: "subrip", Blocks: cues("早安")},
		DetectedVariant: LangSimplified,
		Reason:          "stream 2 tagged chi but its content is Simplified Chinese — convert then deliver",
	}
}

// ─── AC #3 / AC #8: four routes × FreeOnly ∈ {true,false} = 8 cases ────────

func TestProcessItem_FreeOnlyGate(t *testing.T) {
	// startingStatus is what the media row carried BEFORE the run. The brake
	// must put it back exactly, which is why it is not `not_searched` here:
	// blanket-resetting would erase bugfix-j's `untranslated` verdict and make
	// the library claim less than it knows (9R-10b AC #7).
	const startingStatus = models.SubtitleStatusUntranslated

	cases := []struct {
		name     string
		decision RouteDecision
		asr      bool // build the harness with a SpeechTranscriber
		freeOnly bool

		wantTranslateCalls  int
		wantTranscribeCalls int
		wantTerminalStatus  models.SubtitleStatus
		wantRunStatus       models.SubtitleRunStatus
	}{
		{
			name:               "deliver_direct is free — runs under FreeOnly",
			decision:           deliverDirectDecision(),
			freeOnly:           true,
			wantTerminalStatus: models.SubtitleStatusFound,
			wantRunStatus:      models.SubtitleRunCompleted,
		},
		{
			name:               "deliver_direct unchanged when FreeOnly is off",
			decision:           deliverDirectDecision(),
			freeOnly:           false,
			wantTerminalStatus: models.SubtitleStatusFound,
			wantRunStatus:      models.SubtitleRunCompleted,
		},
		{
			name:               "convert_then_deliver is free — OpenCC is local",
			decision:           convertThenDeliverDecision(),
			freeOnly:           true,
			wantTerminalStatus: models.SubtitleStatusFound,
			wantRunStatus:      models.SubtitleRunCompleted,
		},
		{
			name:               "convert_then_deliver unchanged when FreeOnly is off",
			decision:           convertThenDeliverDecision(),
			freeOnly:           false,
			wantTerminalStatus: models.SubtitleStatusFound,
			wantRunStatus:      models.SubtitleRunCompleted,
		},
		{
			name:               "translate is PAID — FreeOnly stops before the LLM",
			decision:           translateDecision("Good morning"),
			freeOnly:           true,
			wantTranslateCalls: 0,
			wantTerminalStatus: startingStatus,
			wantRunStatus:      models.SubtitleRunSkipped,
		},
		{
			name:               "translate still runs when FreeOnly is off",
			decision:           translateDecision("Good morning"),
			freeOnly:           false,
			wantTranslateCalls: 1,
			wantTerminalStatus: models.SubtitleStatusFound,
			wantRunStatus:      models.SubtitleRunCompleted,
		},
		{
			name:                "no_text_source is PAID — FreeOnly stops before ASR",
			decision:            noTextDecision(),
			asr:                 true,
			freeOnly:            true,
			wantTranscribeCalls: 0,
			wantTerminalStatus:  startingStatus,
			wantRunStatus:       models.SubtitleRunSkipped,
		},
		{
			name:                "no_text_source still transcribes when FreeOnly is off",
			decision:            noTextDecision(),
			asr:                 true,
			freeOnly:            false,
			wantTranscribeCalls: 1,
			wantTerminalStatus:  models.SubtitleStatusFound,
			wantRunStatus:       models.SubtitleRunCompleted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var asr *fakeSpeechTranscriber
			var h *itemHarness
			if tc.asr {
				asr = &fakeSpeechTranscriber{available: true}
				h = asrHarness(t, tc.decision, asr)
				// The transcription SERVICE owns the media-row writeback on the
				// ASR leg (9R-16 AC 12), so the fake has to perform it or the
				// non-FreeOnly row would assert against a status nobody wrote.
				asr.onRun = func() {
					require.NoError(t, h.media.SetSubtitleStatus(
						context.Background(), h.ref, models.SubtitleStatusFound,
						ExpectedSidecarPath(h.mediaPath), "zh-Hant"))
				}
			} else {
				h = newItemHarness(t, tc.decision)
			}
			h.media.item.SubtitleStatus = startingStatus

			outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{FreeOnly: tc.freeOnly})
			require.NoError(t, err)
			require.NotNil(t, outcome)

			assert.Len(t, h.trans.calls, tc.wantTranslateCalls,
				"translator call count — a FreeOnly run must never reach the LLM")
			if asr != nil {
				assert.Len(t, asr.calls, tc.wantTranscribeCalls,
					"transcriber call count — a FreeOnly run must never reach speech recognition")
			}

			statuses := h.media.statuses()
			require.NotEmpty(t, statuses)
			assert.Equal(t, tc.wantTerminalStatus, statuses[len(statuses)-1],
				"final media status")

			require.NotNil(t, outcome.Run)
			assert.Equal(t, tc.wantRunStatus, outcome.Run.Status, "run row status")
		})
	}
}

// ─── AC #3: the braked item must stay enumerable ──────────────────────────

// TestProcessItem_FreeOnlyBrakeKeepsItemMissing pins the property the consent
// list depends on: a deferred item must NOT be given a zh-Hant subtitle_language
// and must NOT be parked in a status the auto-trigger's own filter excludes.
// Both halves matter — the repository predicate keys on LANGUAGE
// (missingZhHantSubtitleWhere, movie_repository.go:898), while 9R-10b AC #4's
// trigger filter keys on STATUS. Fail either and the item silently disappears
// from the F15 estimate list, which is worse than never automating at all.
func TestProcessItem_FreeOnlyBrakeKeepsItemMissing(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision RouteDecision
		asr      bool
	}{
		{name: "translate", decision: translateDecision("Good morning")},
		{name: "no_text_source", decision: noTextDecision(), asr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var h *itemHarness
			if tc.asr {
				h = asrHarness(t, tc.decision, &fakeSpeechTranscriber{available: true})
			} else {
				h = newItemHarness(t, tc.decision)
			}
			h.media.item.SubtitleStatus = models.SubtitleStatusNotSearched

			_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{FreeOnly: true})
			require.NoError(t, err)

			for _, w := range h.media.writes {
				assert.NotEqual(t, "zh-Hant", w.language,
					"a deferred item must never be stamped zh-Hant — that removes it from the consent list")
				assert.Empty(t, w.path,
					"a deferred item must never be given a subtitle path")
			}

			last := h.media.statuses()[len(h.media.statuses())-1]
			assert.Contains(t,
				[]models.SubtitleStatus{
					models.SubtitleStatusNotSearched,
					models.SubtitleStatusNotFound,
					models.SubtitleStatusUntranslated,
				},
				last,
				"final status must stay inside the auto-trigger's own eligible set (AC #4 filter 2)")
		})
	}
}

// ─── AC #3: skip is untouched by the gate ─────────────────────────────────

func TestProcessItem_FreeOnlyLeavesSkipUntouched(t *testing.T) {
	decision := RouteDecision{
		Kind:   RouteSkip,
		Reason: "stream 2 tagged und: detector returned unrecognized variant — failing closed (P0)",
	}
	for _, freeOnly := range []bool{false, true} {
		h := newItemHarness(t, decision)
		h.media.item.SubtitleStatus = models.SubtitleStatusNotSearched

		outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{FreeOnly: freeOnly})
		require.NoError(t, err)
		require.NotNil(t, outcome.Run)

		assert.Equal(t, models.SubtitleRunSkipped, outcome.Run.Status)
		assert.Equal(t, models.SubtitleStatusSkipped, h.media.statuses()[len(h.media.statuses())-1],
			"RouteSkip keeps its terminal `skipped` status regardless of FreeOnly — the pipeline DECLINED this item, it did not defer it")
	}
}

// ─── CR-249 H1: the brake must not erase a paid ASR's resume marker ───────

// TestProcessItem_FreeOnlyBrakePreservesResumeMarker is a COST regression pin,
// not a tidiness one.
//
// bugfix-j leaves an item at `untranslated` when speech recognition already ran
// and produced an English sidecar but the translate leg could not finish. That
// row's subtitle_path is the receipt for money already spent: it is what lets
// the next consented run translate instead of transcribing again.
//
// The shared writer maps an empty string to NULL unconditionally
// (repository/subtitle_generation_status.go:44-45), so handing setMediaStatus
// "" does not mean "leave this column alone" — it means ERASE it. The brake did
// exactly that, which would have made this feature charge the user a second
// time for the same audio.
func TestProcessItem_FreeOnlyBrakePreservesResumeMarker(t *testing.T) {
	const enSidecar = "/media/Show.S01E01.en.srt"

	for name, tc := range map[string]struct {
		decision RouteDecision
		asr      bool
	}{
		"translate":      {decision: translateDecision("Good morning")},
		"no_text_source": {decision: noTextDecision(), asr: true},
	} {
		t.Run(name, func(t *testing.T) {
			var h *itemHarness
			if tc.asr {
				h = asrHarness(t, tc.decision, &fakeSpeechTranscriber{available: true})
			} else {
				h = newItemHarness(t, tc.decision)
			}
			h.media.item.SubtitleStatus = models.SubtitleStatusUntranslated
			h.media.item.SubtitlePath = enSidecar
			h.media.item.SubtitleLanguage = "en"

			_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{FreeOnly: true})
			require.NoError(t, err)

			last := h.media.writes[len(h.media.writes)-1]
			assert.Equal(t, models.SubtitleStatusUntranslated, last.status)
			assert.Equal(t, enSidecar, last.path,
				"blanking subtitle_path destroys the receipt for an ASR run the user already paid for")
			assert.Equal(t, "en", last.language,
				"blanking subtitle_language loses the translate-only resume signal")
		})
	}
}

// TestProcessItem_FreeOnlyBrakeLeavesAnEmptyRowEmpty pins the other half: an
// item that genuinely has no sidecar must not gain a phantom one.
func TestProcessItem_FreeOnlyBrakeLeavesAnEmptyRowEmpty(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning"))
	h.media.item.SubtitleStatus = models.SubtitleStatusNotSearched

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{FreeOnly: true})
	require.NoError(t, err)

	last := h.media.writes[len(h.media.writes)-1]
	assert.Empty(t, last.path)
	assert.Empty(t, last.language)
}
