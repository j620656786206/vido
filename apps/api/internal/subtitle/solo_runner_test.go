package subtitle

// disc-2026-10-single-generate-ignores-embedded-english-a — the detail-page
// 生成字幕 click as a job over Pipeline.ProcessItem.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
)

// fakeSoloGuard is the WorkerPool in-flight set the runner shares.
type fakeSoloGuard struct {
	mu       sync.Mutex
	inFlight map[MediaRef]bool
	reserves int
	releases int
}

func newFakeSoloGuard() *fakeSoloGuard { return &fakeSoloGuard{inFlight: map[MediaRef]bool{}} }

func (g *fakeSoloGuard) TryReserve(ref MediaRef) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.inFlight[ref] {
		return false
	}
	g.inFlight[ref] = true
	g.reserves++
	return true
}

func (g *fakeSoloGuard) Release(ref MediaRef) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.inFlight, ref)
	g.releases++
}

func (g *fakeSoloGuard) IsInFlight(ref MediaRef) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.inFlight[ref]
}

// recordingHub captures every SSE event in order.
type recordingHub struct {
	mu     sync.Mutex
	events []sse.Event
}

func (h *recordingHub) Broadcast(e sse.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, e)
}

func (h *recordingHub) ofType(t sse.EventType) []map[string]interface{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []map[string]interface{}
	for _, e := range h.events {
		if e.Type == t {
			out = append(out, e.Data.(map[string]interface{}))
		}
	}
	return out
}

func (h *recordingHub) last() sse.Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.events[len(h.events)-1]
}

type soloHarness struct {
	*itemHarness
	runner *SoloRunner
	guard  *fakeSoloGuard
	hub    *recordingHub
}

func newSoloHarness(t *testing.T, decision RouteDecision, pipelineOpts []PipelineOption, runnerOpts ...SoloRunnerOption) *soloHarness {
	t.Helper()
	h := newItemHarness(t, decision, pipelineOpts...)
	guard := newFakeSoloGuard()
	hub := &recordingHub{}
	runner := NewSoloRunner(h.pipeline, guard, hub, nil, runnerOpts...)
	return &soloHarness{itemHarness: h, runner: runner, guard: guard, hub: hub}
}

func startAndWait(t *testing.T, s *soloHarness, modelID string) string {
	t.Helper()
	jobID, err := s.runner.Start(context.Background(), s.ref, modelID)
	require.NoError(t, err)
	require.NotEmpty(t, jobID)
	s.runner.waitIdle()
	return jobID
}

// ─── AC #1: the click runs ProcessItem with the solo options ───────────────

func TestSoloRunner_RunsProcessItemWithRegenerateAndModel(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Good morning."), nil)
	// A sidecar is already on disk: only Regenerate gets past the pre-flight.
	require.NoError(t, os.WriteFile(ExpectedSidecarPath(s.mediaPath), []byte(oneCueSRT), 0o600))

	jobID := startAndWait(t, s, "claude-sonnet-5")

	assert.Equal(t, 1, s.router.calls, "Regenerate bypassed the pre-flight")
	assert.NotEmpty(t, s.cache.reads, "Regenerate keeps cache reads (Force would not)")
	require.NotEmpty(t, s.runs.created)
	assert.Equal(t, "claude-sonnet-5", s.runs.created[0].ModelID, "the per-run model reaches the run row")
	assert.Equal(t, 1, s.guard.reserves)
	assert.Equal(t, 1, s.guard.releases, "the in-flight claim is released when the job ends")
	inProgress, _ := s.runner.Status(s.ref)
	assert.False(t, inProgress)
	assert.Len(t, s.hub.ofType(services.EventTranscriptionComplete), 1)
	assert.Equal(t, jobID, s.hub.ofType(services.EventTranscriptionComplete)[0]["job_id"])
}

func TestSoloRunner_SkipVerdictGoesToASR(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true}
	s := newSoloHarness(t, skipDecision(), []PipelineOption{WithSpeechTranscriber(asr)})
	s.ref = MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaMovie}

	startAndWait(t, s, "")

	require.Len(t, asr.calls, 1, "TranscribeWhenSkipped is set for the click")
	done := s.hub.ofType(services.EventTranscriptionComplete)
	require.Len(t, done, 1)
	assert.Equal(t, models.SubtitleRunRouteASR, done[0]["route"])
}

// ─── AC #3: single flight ───────────────────────────────────────────────────

func TestSoloRunner_RefusesWhenTheItemIsAlreadyInFlight(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Hi"), nil)
	require.True(t, s.guard.TryReserve(s.ref), "someone else (batch, pool) owns the item")

	_, err := s.runner.Start(context.Background(), s.ref, "")

	assert.ErrorIs(t, err, services.ErrTranscriptionInProgress)
	assert.Zero(t, s.router.calls)
	assert.True(t, s.guard.IsInFlight(s.ref), "the OTHER owner's claim must not be released by our refusal")
	assert.Equal(t, 0, s.guard.releases)
}

// ─── AC #4: the capability gate ────────────────────────────────────────────

func TestSoloRunner_Gate(t *testing.T) {
	cases := []struct {
		name         string
		predicted    RoutePrediction
		predictErr   error
		asrAvailable bool
		resumable    bool
		wantErr      error
	}{
		{"asr route, no ASR, no resume → 503", PredictASR, nil, false, false, services.ErrTranscriptionDisabled},
		{"skip route, no ASR, no resume → 503", PredictSkip, nil, false, false, services.ErrTranscriptionDisabled},
		{"asr route, no ASR, resumable → proceeds", PredictASR, nil, false, true, nil},
		{"asr route, ASR present → proceeds", PredictASR, nil, true, false, nil},
		{"extract route, no ASR → proceeds (routing is free)", PredictExtract, nil, false, false, nil},
		{"probe failed → proceeds (never 503 on a read error)", "", errors.New("ffprobe: boom"), false, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSoloHarness(t, translateDecision("Hi"), nil,
				WithSoloRoutePredictor(func(context.Context, string) (RoutePrediction, error) { return tc.predicted, tc.predictErr }),
				WithSoloASRAvailability(func() bool { return tc.asrAvailable }),
				WithSoloResumeCheck(func(context.Context, MediaRef) bool { return tc.resumable }),
			)
			_, err := s.runner.Start(context.Background(), s.ref, "")
			s.runner.waitIdle()
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Zero(t, s.router.calls, "a refused click must not route")
				assert.False(t, s.guard.IsInFlight(s.ref), "a refused click releases its claim")
				assert.Empty(t, s.hub.events, "a refused click emits nothing")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 1, s.router.calls)
		})
	}
}

// ─── AC #5: the writable probe runs before anything is paid ────────────────

func TestSoloRunner_UnwritableFolderIsRefusedBeforeTheRun(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Hi"), []PipelineOption{
		WithWritableProbe(func(context.Context, string) error { return errors.New("read-only mount") }),
	})

	_, err := s.runner.Start(context.Background(), s.ref, "")

	assert.ErrorIs(t, err, ErrSubtitleTargetNotWritable)
	assert.Zero(t, s.router.calls)
	assert.Empty(t, s.trans.calls)
	assert.False(t, s.guard.IsInFlight(s.ref))
	inProgress, jobID := s.runner.Status(s.ref)
	assert.False(t, inProgress)
	assert.Empty(t, jobID)
}

// ─── AC #6: the job-level events ───────────────────────────────────────────

func TestSoloRunner_StartEventThenTerminalLast(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Good morning."), nil,
		WithSoloRoutePredictor(func(context.Context, string) (RoutePrediction, error) { return PredictExtract, nil }),
		WithSoloRunBudgetUSD(2.5),
	)

	jobID := startAndWait(t, s, "")

	first := s.hub.events[0]
	assert.Equal(t, services.EventTranscriptionExtracting, first.Type)
	start := first.Data.(map[string]interface{})
	assert.Equal(t, jobID, start["job_id"])
	assert.Equal(t, s.ref.ID, start["media_id"])
	assert.Equal(t, s.ref.MediaType, start["media_type"])
	assert.Equal(t, "怪奇物語", start["title"], "the title comes from the loaded item's context")
	assert.Equal(t, "extracting", start["phase"])
	assert.Equal(t, "extract", start["predicted_route"])
	assert.Equal(t, 0.0, start["spent_usd"])
	assert.Equal(t, 2.5, start["budget_usd"])
	assert.NotEmpty(t, start["message"])

	last := s.hub.last()
	require.Equal(t, services.EventTranscriptionComplete, last.Type, "the solo terminal is the job's LAST event")
	done := last.Data.(map[string]interface{})
	assert.Equal(t, jobID, done["job_id"])
	assert.Equal(t, "怪奇物語", done["title"])
	assert.Equal(t, "complete", done["phase"])
	assert.Equal(t, string(RouteTranslate), done["route"])
	assert.Equal(t, ExpectedSidecarPath(s.mediaPath), done["zh_srt_path"])
	assert.Equal(t, "翻譯完成（翻譯片內英文字幕）", done["message"])
	assert.Contains(t, done, "spent_usd")
	assert.Equal(t, 2.5, done["budget_usd"])
	assert.NotContains(t, done, "partial")
}

func TestSoloRunner_CompleteMessageNamesTheRoute(t *testing.T) {
	cases := []struct {
		name     string
		decision RouteDecision
		route    string
		message  string
	}{
		{
			"deliver_direct",
			RouteDecision{Kind: RouteDeliverDirect, DetectedVariant: LangTraditional,
				Track: &ExtractedTrack{StreamIndex: 3, Language: "chi", Blocks: cues("答案都在箱子裡。")}},
			string(RouteDeliverDirect),
			"字幕已生成（直接使用片內中文字幕，沒有花錢）",
		},
		{
			"convert_then_deliver",
			RouteDecision{Kind: RouteConvertThenDeliver, DetectedVariant: LangSimplified,
				Track: &ExtractedTrack{StreamIndex: 3, Language: "chi", Blocks: cues("答案都在箱子里。")}},
			string(RouteConvertThenDeliver),
			"字幕已生成（片內簡體字幕已轉成繁體，沒有花錢）",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSoloHarness(t, tc.decision, nil)
			startAndWait(t, s, "")
			done := s.hub.ofType(services.EventTranscriptionComplete)
			require.Len(t, done, 1)
			assert.Equal(t, tc.route, done[0]["route"])
			assert.Equal(t, tc.message, done[0]["message"])
			assert.Equal(t, 0.0, done[0]["spent_usd"], "a free route reports $0, not a missing key")
		})
	}
}

func TestSoloRunner_ASRRouteCompletesWithTranscriptionMessage(t *testing.T) {
	asr := &fakeSpeechTranscriber{available: true}
	s := newSoloHarness(t, noTextDecision(), []PipelineOption{WithSpeechTranscriber(asr)})
	s.ref = MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaMovie}

	startAndWait(t, s, "")

	done := s.hub.ofType(services.EventTranscriptionComplete)
	require.Len(t, done, 1)
	assert.Equal(t, models.SubtitleRunRouteASR, done[0]["route"])
	assert.Equal(t, "轉錄完成", done[0]["message"])
	assert.NotContains(t, done[0], "zh_srt_path", "the fake ASR wrote no sidecar, so no path is claimed")
}

func TestSoloRunner_PartialDeliveryIsDisclosed(t *testing.T) {
	// 60 cues = 6 chunks; chunk 2 times out through every retry and ships in
	// English (pipeline_transient_test.go precedent). The job terminal must
	// say so — the dialog's verdict line must not read 完成 off zh_srt_path.
	source := numberedCues(60)
	s := newSoloHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	}, nil)
	s.trans.fn = func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		if call == 2 {
			return nil, ai.CompletionUsage{}, exhaustedTimeout()
		}
		out := map[int]string{}
		for _, b := range blocks {
			out[b.Index] = fmt.Sprintf("第%d句", b.Index)
		}
		return out, ai.CompletionUsage{}, nil
	}

	startAndWait(t, s, "")

	last := s.hub.last()
	require.Equal(t, services.EventTranscriptionComplete, last.Type)
	done := last.Data.(map[string]interface{})
	assert.Equal(t, true, done["partial"])
	assert.Equal(t, 10, done["english_kept_blocks"])
	assert.Contains(t, done["message"], "10 句保留英文")
	assert.Equal(t, ExpectedSidecarPath(s.mediaPath), done["zh_srt_path"], "the file IS on disk and usable")
}

func TestSoloRunner_FailedTerminals(t *testing.T) {
	t.Run("no text source and no ASR → failed with the no-source message", func(t *testing.T) {
		s := newSoloHarness(t, noTextDecision(), nil)
		s.ref = MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaMovie}
		jobID := startAndWait(t, s, "")

		last := s.hub.last()
		require.Equal(t, services.EventTranscriptionFailed, last.Type)
		data := last.Data.(map[string]interface{})
		assert.Equal(t, jobID, data["job_id"])
		assert.Equal(t, "failed", data["phase"])
		assert.Equal(t, "這部影片沒有可用的字幕來源，也沒有設定語音辨識", data["message"])
		assert.Contains(t, data, "spent_usd")
		// disc-2026-10-no-text-source-skip-a: the skip now has its own code,
		// and it is the code this run's row classifies to.
		assert.Equal(t, models.SubtitleRunFailureNoTextSource, data["failure_reason"])
		row := s.runs.lastUpdate(t)
		require.Equal(t, models.SubtitleRunSkipped, row.Status)
		assert.Equal(t, models.SubtitleRunFailureReason(row.ErrorMessage), data["failure_reason"],
			"the live event and the run row name the same cause (row: %q)", row.ErrorMessage)
		assert.False(t, s.guard.IsInFlight(s.ref))
	})

	// disc-2026-10-no-text-source-skip-a AC #6: the other two ways a solo run
	// ends "no usable source and no speech recognition".
	noTextSourceCases := []struct {
		name     string
		decision RouteDecision
		opts     []PipelineOption
	}{
		{
			// The click opts into ASR for an und-tagged track; there is none.
			name:     "und track (RouteSkip) and no ASR → no_text_source",
			decision: skipDecision(),
		},
		{
			// The ASR key was removed mid-run: the service reports disabled.
			name:     "ASR wired but reports disabled → no_text_source",
			decision: noTextDecision(),
			opts: []PipelineOption{WithSpeechTranscriber(
				&fakeSpeechTranscriber{available: false, err: services.ErrTranscriptionDisabled})},
		},
	}
	for _, tc := range noTextSourceCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSoloHarness(t, tc.decision, tc.opts)
			s.ref = MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaMovie}
			startAndWait(t, s, "")

			last := s.hub.last()
			require.Equal(t, services.EventTranscriptionFailed, last.Type)
			data := last.Data.(map[string]interface{})
			assert.Equal(t, "failed", data["phase"])
			assert.Equal(t, "這部影片沒有可用的字幕來源，也沒有設定語音辨識", data["message"])
			assert.Equal(t, models.SubtitleRunFailureNoTextSource, data["failure_reason"])
			row := s.runs.lastUpdate(t)
			require.Equal(t, models.SubtitleRunSkipped, row.Status)
			assert.Equal(t, models.SubtitleRunFailureReason(row.ErrorMessage), data["failure_reason"])
		})
	}

	t.Run("budget ceiling → failed with the ceiling message", func(t *testing.T) {
		asr := &fakeSpeechTranscriber{available: true, err: fmt.Errorf("transcribe: %w", ai.ErrBudgetExceeded)}
		s := newSoloHarness(t, noTextDecision(), []PipelineOption{WithSpeechTranscriber(asr)}, WithSoloRunBudgetUSD(1.5))
		s.ref = MediaRef{ID: "mv-1", MediaType: models.SubtitleRunMediaMovie}
		startAndWait(t, s, "")

		last := s.hub.last()
		require.Equal(t, services.EventTranscriptionFailed, last.Type)
		data := last.Data.(map[string]interface{})
		assert.Contains(t, data["message"], "$1.50")
		assert.Contains(t, data["message"], "上限")
		assert.NotEmpty(t, data["error"])
	})

	t.Run("router error → failed with the error text", func(t *testing.T) {
		s := newSoloHarness(t, translateDecision("Hi"), nil)
		s.router.err = errors.New("ffmpeg exploded")
		startAndWait(t, s, "")

		last := s.hub.last()
		require.Equal(t, services.EventTranscriptionFailed, last.Type)
		data := last.Data.(map[string]interface{})
		assert.Contains(t, data["error"], "ffmpeg exploded")
	})
}

// killedByCancel is a cause whose TYPE says "the caller cancelled" while its
// text does not — the shape failItem's comment warns about (a subprocess
// killed by cancellation reads "signal: killed").
type killedByCancel struct{}

func (killedByCancel) Error() string { return "signal: killed" }
func (killedByCancel) Unwrap() error { return context.Canceled }

// stoppedByDeadline is the queue-timeout twin: the deadline ran out, the text
// does not say so.
type stoppedByDeadline struct{}

func (stoppedByDeadline) Error() string { return "ffmpeg stopped" }
func (stoppedByDeadline) Unwrap() error { return context.DeadlineExceeded }

// disc-2026-10-live-activity-failure-reasons-a AC #2 / #3: the live event's
// failure_reason is the code the activity record derives from THIS run's row
// (models.SubtitleRunFailureReason(error_message)) — for every way a solo run
// can fail, including the ones only the error's type gives away.
func TestSoloRunner_FailureReasonMatchesTheActivityRecord(t *testing.T) {
	cases := []struct {
		name  string
		setup func(s *soloHarness)
		want  string
	}{
		{
			// 正式機 和平使者 S01E01: translated, then the sidecar write failed.
			name: "placer permission denied after translating",
			setup: func(s *soloHarness) {
				s.placer.err = fmt.Errorf("placer: write failed: write temp file: open /media/tv/x.zh-Hant.srt.tmp: %w", fs.ErrPermission)
			},
			want: models.SubtitleRunFailureTargetNotWritable,
		},
		{
			name:  "cancelled: the cause is context.Canceled",
			setup: func(s *soloHarness) { s.router.err = fmt.Errorf("extract: %w", context.Canceled) },
			want:  models.SubtitleRunFailureCancelled,
		},
		{
			name:  "cancelled: the cause only SAYS signal: killed",
			setup: func(s *soloHarness) { s.router.err = fmt.Errorf("extract: %w", killedByCancel{}) },
			want:  models.SubtitleRunFailureCancelled,
		},
		{
			name: "cancelled while queued for the extraction slot",
			setup: func(s *soloHarness) {
				s.router.err = fmt.Errorf("extract: %w: %w", ErrSubtitleExtractWaitAborted, context.Canceled)
			},
			want: models.SubtitleRunFailureCancelled,
		},
		{
			name: "queue timeout: wait aborted on a deadline",
			setup: func(s *soloHarness) {
				s.router.err = fmt.Errorf("extract: %w: %w", ErrSubtitleExtractWaitAborted, context.DeadlineExceeded)
			},
			want: models.SubtitleRunFailureQueueTimeout,
		},
		{
			name: "queue timeout: wait aborted, the text never says deadline",
			setup: func(s *soloHarness) {
				s.router.err = fmt.Errorf("extract: %w: %w", ErrSubtitleExtractWaitAborted, stoppedByDeadline{})
			},
			want: models.SubtitleRunFailureQueueTimeout,
		},
		{
			name:  "budget ceiling",
			setup: func(s *soloHarness) { s.router.err = fmt.Errorf("translate: %w", ai.ErrBudgetExceeded) },
			want:  models.SubtitleRunFailureBudgetExceeded,
		},
		{
			name: "AI account with no credit",
			setup: func(s *soloHarness) {
				s.router.err = fmt.Errorf("transcribe: %w",
					&ai.CreditExhaustedError{Provider: "openai", Code: "insufficient_quota", BillingURL: ai.OpenAIBillingURL})
			},
			want: models.SubtitleRunFailureCreditExhausted,
		},
		{
			name:  "anything else",
			setup: func(s *soloHarness) { s.router.err = errors.New("ffmpeg exploded") },
			want:  models.SubtitleRunFailureOther,
		},
		{
			// The stored row is cut at maxErrorMessage; a long ffmpeg stderr
			// pushes the only telling words past the cut, so the record says
			// other — and the live event must not say 處理逾時 for it.
			name: "the telling words sit past the stored cut",
			setup: func(s *soloHarness) {
				s.router.err = fmt.Errorf("extract: SUBTITLE_EXTRACT_FAILED: ffmpeg stopped by the caller's deadline after 3m0s on x.mkv (ffmpeg stderr: %s): context deadline exceeded",
					strings.Repeat("size=34kB time=00:28:10.85 ", 40))
			},
			want: models.SubtitleRunFailureOther,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSoloHarness(t, translateDecision("Hi"), nil)
			tc.setup(s)
			startAndWait(t, s, "")

			last := s.hub.last()
			require.Equal(t, services.EventTranscriptionFailed, last.Type)
			data := last.Data.(map[string]interface{})

			row := s.runs.lastUpdate(t)
			require.Equal(t, models.SubtitleRunFailed, row.Status)
			assert.Equal(t, models.SubtitleRunFailureReason(row.ErrorMessage), data["failure_reason"],
				"the live event and the activity record name the same cause (row: %q)", row.ErrorMessage)
			assert.Equal(t, tc.want, data["failure_reason"])
			assert.NotEmpty(t, data["error"], "error stays for the dialog's diagnostic line")
		})
	}
}

// soloFailureReason reads the row when the pipeline hands one back (a skip
// verdict, or a run that ended without an error) and the error otherwise.
func TestSoloFailureReason_RowFirstThenError(t *testing.T) {
	skipped := &models.SubtitleRun{Status: models.SubtitleRunSkipped, ErrorMessage: "no usable embedded text subtitle track (0 image-codec track(s), 0 text)"}
	assert.Equal(t, models.SubtitleRunFailureOther, soloFailureReason(skipped, nil),
		"a skipped row without the no-text-source prefix names no known cause")
	prefixed := &models.SubtitleRun{Status: models.SubtitleRunSkipped,
		ErrorMessage: NoTextSourceRunPrefix + "no usable embedded text subtitle track (0 image-codec track(s), 0 text)"}
	assert.Equal(t, models.SubtitleRunFailureNoTextSource, soloFailureReason(prefixed, nil))

	stored := &models.SubtitleRun{Status: models.SubtitleRunFailed, ErrorMessage: "cancelled: subtitle pipeline: movie x: route: signal: killed"}
	assert.Equal(t, models.SubtitleRunFailureCancelled, soloFailureReason(stored, errors.New("signal: killed")),
		"the stored row wins when the pipeline hands it back")

	assert.Equal(t, models.SubtitleRunFailureOther, soloFailureReason(nil, nil))
	assert.Equal(t, models.SubtitleRunFailureNoAudioTrack,
		soloFailureReason(nil, errors.New("asr fallback: extract audio: no audio track found in media file")))
}

// ─── AC #9: Status ──────────────────────────────────────────────────────────

func TestSoloRunner_StatusWhileRunningAndAfter(t *testing.T) {
	release := make(chan struct{})
	s := newSoloHarness(t, translateDecision("Good morning."), nil)
	s.trans.fn = func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		<-release
		out := map[int]string{}
		for _, b := range blocks {
			out[b.Index] = "早安"
		}
		return out, ai.CompletionUsage{}, nil
	}

	jobID, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err)

	inProgress, gotJob := s.runner.Status(s.ref)
	assert.True(t, inProgress)
	assert.Equal(t, jobID, gotJob)

	_, err = s.runner.Start(context.Background(), s.ref, "")
	assert.ErrorIs(t, err, services.ErrTranscriptionInProgress, "a second click during the run is refused")

	close(release)
	s.runner.waitIdle()

	inProgress, gotJob = s.runner.Status(s.ref)
	assert.False(t, inProgress)
	assert.Empty(t, gotJob)
}

func TestSoloRunner_StatusSeesABatchOrPoolRunWithoutAJobID(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Hi"), nil)
	require.True(t, s.guard.TryReserve(s.ref))

	inProgress, jobID := s.runner.Status(s.ref)
	assert.True(t, inProgress)
	assert.Empty(t, jobID, "not our job — no id to attach to")
}

// ─── CR fixes ───────────────────────────────────────────────────────────────

func TestSoloRunner_PredictsTheRouteOnce(t *testing.T) {
	calls := 0
	s := newSoloHarness(t, translateDecision("Hi"), nil,
		WithSoloRoutePredictor(func(context.Context, string) (RoutePrediction, error) { calls++; return PredictExtract, nil }))

	startAndWait(t, s, "")

	assert.Equal(t, 1, calls, "the gate's probe is reused for predicted_route — ffprobe on a NAS share is not free")
	assert.Equal(t, "extract", s.hub.ofType(services.EventTranscriptionExtracting)[0]["predicted_route"])
}

func TestSoloRunner_TerminalIsEmittedBeforeTheClaimIsReleased(t *testing.T) {
	// A second click racing the first job's cleanup must not be able to put
	// its start event BEFORE the first job's terminal, and must not have its
	// own job entry deleted by the first job's cleanup.
	h := newItemHarness(t, translateDecision("Hi"))
	hub := &recordingHub{}
	releasedAt := -1
	guard := &releaseSpyGuard{fakeSoloGuard: newFakeSoloGuard(), hub: hub, onRelease: func(n int) { releasedAt = n }}
	runner := NewSoloRunner(h.pipeline, guard, hub, nil)

	_, err := runner.Start(context.Background(), h.ref, "")
	require.NoError(t, err)
	runner.waitIdle()

	require.Greater(t, len(hub.events), 1)
	assert.Equal(t, len(hub.events), releasedAt, "Release happens after the last event was broadcast")
	assert.Equal(t, services.EventTranscriptionComplete, hub.last().Type)
}

func TestSoloRunner_CleanupOnlyForgetsItsOwnJob(t *testing.T) {
	first := newSoloHarness(t, translateDecision("Hi"), nil)
	startAndWait(t, first, "")

	// A second job on the same ref, still running: its entry must survive.
	release := make(chan struct{})
	first.trans.fn = func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		<-release
		return map[int]string{blocks[0].Index: "嗨"}, ai.CompletionUsage{}, nil
	}
	jobID2, err := first.runner.Start(context.Background(), first.ref, "")
	require.NoError(t, err)
	inProgress, got := first.runner.Status(first.ref)
	assert.True(t, inProgress)
	assert.Equal(t, jobID2, got)
	close(release)
	first.runner.waitIdle()
}

func TestSoloRunner_ActivityAndUsageSeeRunningJobs(t *testing.T) {
	release := make(chan struct{})
	s := newSoloHarness(t, translateDecision("Good morning."), nil, WithSoloRunBudgetUSD(3))
	s.trans.fn = func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		<-release
		return map[int]string{blocks[0].Index: "早安"}, ai.CompletionUsage{}, nil
	}

	active, _, current, _, _ := s.runner.ActivityProgress()
	assert.False(t, active)
	assert.Zero(t, current)
	_, _, ok := s.runner.ActiveManualUsage()
	assert.False(t, ok, "nothing running → absent, never $0")

	_, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err)

	active, _, current, _, item := s.runner.ActivityProgress()
	assert.True(t, active)
	assert.Equal(t, 1, current)
	assert.Equal(t, "怪奇物語", item)
	// disc-2026-10-activity-generation-progress-detail: the one running item
	// is named for the Activity row's step.
	id, named := s.runner.ActiveMediaID()
	assert.True(t, named)
	assert.Equal(t, s.ref.ID, id)
	spent, budget, ok := s.runner.ActiveManualUsage()
	assert.True(t, ok)
	assert.Equal(t, 0.0, spent)
	assert.Equal(t, 3.0, budget)

	close(release)
	s.runner.waitIdle()
	active, _, _, _, _ = s.runner.ActivityProgress()
	assert.False(t, active)
	_, named = s.runner.ActiveMediaID()
	assert.False(t, named, "idle names nothing")
}

// releaseSpyGuard records how many events the hub held when Release ran.
type releaseSpyGuard struct {
	*fakeSoloGuard
	hub       *recordingHub
	onRelease func(eventsSoFar int)
}

func (g *releaseSpyGuard) Release(ref MediaRef) {
	g.hub.mu.Lock()
	n := len(g.hub.events)
	g.hub.mu.Unlock()
	g.onRelease(n)
	g.fakeSoloGuard.Release(ref)
}

// disc-2026-09-asr-quota-429-treated-as-transient — an empty provider account
// names the provider and the billing page, through every layer that wraps it.
func TestFailureMessage_EmptyAccountSaysWhereToTopUp(t *testing.T) {
	credit := &ai.CreditExhaustedError{Provider: "openai", Code: "insufficient_quota", BillingURL: ai.OpenAIBillingURL}
	runErr := fmt.Errorf("subtitle pipeline: movie m1: asr fallback: %w",
		fmt.Errorf("transcribe: %w", fmt.Errorf("transcribe chunk 1/6: %w", fmt.Errorf("%w: %w", ai.ErrWhisperAPIError, credit))))

	message, detail := failureMessage(nil, runErr, nil)
	assert.Equal(t, "OpenAI 帳號沒有額度了，儲值後再按一次生成", message)
	assert.Equal(t, "OpenAI 帳號沒有額度了，請到 "+ai.OpenAIBillingURL+" 儲值後再按一次生成", detail)
	assert.NotContains(t, detail, "{", "no JSON body")

	anthropic := &ai.CreditExhaustedError{Provider: "anthropic", BillingURL: ai.AnthropicBillingURL}
	message, _ = failureMessage(nil, fmt.Errorf("translate: %w", anthropic), nil)
	assert.Equal(t, "Anthropic 帳號沒有額度了，儲值後再按一次生成", message)
}
