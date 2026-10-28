package subtitle

// disc-2026-10-solo-generation-cancel-api: stopping a detail-page click in
// every phase — the run row, the media row, the terminal event, the Activity
// list and the single-flight claim.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// blockingTranscriber is the ASR leg mid-recognition: it records some paid
// audio on the job's Budget, then waits for the run ctx to end — the way the
// real leg's HTTP call returns when its ctx is cancelled.
type blockingTranscriber struct {
	entered chan struct{}
}

func (f *blockingTranscriber) Available() bool { return true }

func (f *blockingTranscriber) Transcribe(ctx context.Context, _ MediaRef, _, _ string) error {
	ai.BudgetFromContext(ctx).RecordASR(600) // ten paid minutes
	close(f.entered)
	<-ctx.Done()
	return fmt.Errorf("transcribe: chunk 3/12: %w", ctx.Err())
}

// assertStoppedCleanly is AC #3's common half: what every stopped phase must
// leave behind.
func assertStoppedCleanly(t *testing.T, s *soloHarness, jobID string) models.SubtitleRun {
	t.Helper()
	run := s.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunFailed, run.Status)
	assert.True(t, strings.HasPrefix(run.ErrorMessage, CancelledRunPrefix), run.ErrorMessage)
	assert.Equal(t, models.SubtitleRunFailureCancelled, models.SubtitleRunFailureReason(run.ErrorMessage),
		"the activity record says 中途停止")
	require.NotNil(t, run.SpentUSD, "the spend is stamped on the stopped run")

	last := s.hub.last()
	assert.Equal(t, services.EventTranscriptionFailed, last.Type, "no new event kind")
	data := last.Data.(map[string]interface{})
	assert.Equal(t, jobID, data["job_id"], "the job's own event is its last one")
	assert.Equal(t, models.SubtitleRunFailureCancelled, data["failure_reason"])
	assert.Equal(t, soloStoppedMessage, data["message"])
	assert.InDelta(t, *run.SpentUSD, data["spent_usd"], 1e-9, "event and run row name the same spend")

	assert.Empty(t, s.runner.ActiveSoloJobs(), "the Activity 進行中 row is gone")
	assert.False(t, s.guard.IsInFlight(s.ref), "the single-flight claim is released")
	assert.False(t, s.runner.Cancelling(s.ref))
	inProgress, _ := s.runner.Status(s.ref)
	assert.False(t, inProgress)
	return run
}

// AC #3a: queued at the extraction gate — stopped immediately, $0.
func TestSoloRunner_CancelWhileQueuedForTheExtractionSlot(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Good morning."), nil)
	entered := make(chan struct{})
	s.router.err = fmt.Errorf("%w: 2 ahead after 3s: %w", ErrSubtitleExtractWaitAborted, context.Canceled)
	s.router.onRoute = func(ctx context.Context) {
		close(entered)
		<-ctx.Done()
	}

	jobID, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err)
	<-entered
	assert.Equal(t, SoloCancelResult{Cancelled: true, Running: true, Owner: SoloOwnerSolo}, s.runner.Cancel(s.ref))
	s.runner.waitIdle()

	run := assertStoppedCleanly(t, s, jobID)
	assert.Zero(t, *run.SpentUSD, "nothing was paid while queued")
	assert.Empty(t, s.trans.calls, "the translator never ran")
}

// AC #3b + #4: the extraction subprocess killed by the stop reports only
// "signal: killed" — still cancelled, in the row AND the event.
func TestSoloRunner_CancelWhileExtractingKilledSubprocessIsStillCancelled(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Good morning."), nil)
	entered := make(chan struct{})
	s.router.err = errors.New("ffmpeg extract: signal: killed")
	s.router.onRoute = func(ctx context.Context) {
		close(entered)
		<-ctx.Done()
	}

	jobID, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err)
	<-entered
	s.runner.Cancel(s.ref)
	s.runner.waitIdle()

	assertStoppedCleanly(t, s, jobID)
}

// AC #3c: transcribing — the paid audio so far stays on the run row.
func TestSoloRunner_CancelWhileTranscribingKeepsTheSpend(t *testing.T) {
	asr := &blockingTranscriber{entered: make(chan struct{})}
	s := newSoloHarness(t, noTextDecision(), []PipelineOption{WithSpeechTranscriber(asr)})

	jobID, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err)
	<-asr.entered
	s.runner.Cancel(s.ref)
	s.runner.waitIdle()

	run := assertStoppedCleanly(t, s, jobID)
	assert.Greater(t, *run.SpentUSD, 0.0, "ten paid minutes are recorded, not refunded")
	assert.Equal(t, models.SubtitleRunRouteNoTextSource, run.Route)
}

// AC #3d + #5a + #6: translating a re-generation. While the stop winds down a
// new click is refused with the "still stopping" sentinel and the status
// reads cancelling; afterwards the row still points at the old subtitle and a
// new click is accepted.
func TestSoloRunner_CancelWhileTranslatingARegeneration(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Good morning."), nil)
	store := newLivingStore(s.itemHarness, models.SubtitleStatusFound, "/media/show/ep1.zh-Hant.srt", "zh-Hant")
	s.pipeline.media = store
	entered, release := make(chan struct{}), make(chan struct{})
	s.trans.fn = func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		close(entered)
		<-release
		return nil, ai.CompletionUsage{}, context.Canceled
	}

	jobID, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err)
	<-entered

	first := s.runner.Cancel(s.ref)
	second := s.runner.Cancel(s.ref)
	assert.Equal(t, SoloCancelResult{Cancelled: true, Running: true, Owner: SoloOwnerSolo}, first)
	assert.Equal(t, first, second, "cancel is idempotent")
	assert.True(t, s.runner.Cancelling(s.ref))
	_, err = s.runner.Start(context.Background(), s.ref, "")
	assert.ErrorIs(t, err, ErrSoloGenerationCancelling)
	assert.ErrorIs(t, err, services.ErrTranscriptionInProgress, "still the 409 every caller already maps")
	assert.Len(t, s.runner.ActiveSoloJobs(), 1, "the row stays until the run has wound down")

	close(release)
	s.runner.waitIdle()

	assertStoppedCleanly(t, s, jobID)
	assert.Equal(t, statusWrite{models.SubtitleStatusFound, "/media/show/ep1.zh-Hant.srt", "zh-Hant"}, store.row(),
		"這部片現有的字幕沒有被動到")

	s.trans.fn = echoingTranslator()
	jobID2, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err, "the claim is free again")
	assert.NotEqual(t, jobID, jobID2)
	s.runner.waitIdle()
}

// AC #3e: a stop that lands after the file was placed changes nothing — the
// run completed, and the terminal says so.
func TestSoloRunner_CancelAfterThePlaceCompletes(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Good morning."), nil)
	s.pipeline.runs = ctxRunStore{s.runs}
	s.pipeline.placer = cancelOnPlacePlacer{recordingPlacer: s.placer, cancel: func() { s.runner.Cancel(s.ref) }}

	_, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err)
	s.runner.waitIdle()

	assert.Equal(t, models.SubtitleRunCompleted, s.runs.lastUpdate(t).Status)
	assert.Equal(t, services.EventTranscriptionComplete, s.hub.last().Type)
	assert.False(t, s.guard.IsInFlight(s.ref))
}

// AC #1 + #2: nothing running, and an item owned by the batch / pool.
func TestSoloRunner_CancelLeavesOtherOwnersAlone(t *testing.T) {
	s := newSoloHarness(t, translateDecision("Good morning."), nil)

	assert.Equal(t, SoloCancelResult{}, s.runner.Cancel(s.ref), "nothing runs: cancelled=false, running=false")

	require.True(t, s.guard.TryReserve(s.ref), "a batch item holds the claim")
	assert.Equal(t, SoloCancelResult{Running: true, Owner: SoloOwnerOther}, s.runner.Cancel(s.ref))
	assert.True(t, s.guard.IsInFlight(s.ref), "the batch's claim is untouched")
	assert.False(t, s.runner.Cancelling(s.ref))
	_, err := s.runner.Start(context.Background(), s.ref, "")
	assert.ErrorIs(t, err, services.ErrTranscriptionInProgress)
	assert.NotErrorIs(t, err, ErrSoloGenerationCancelling, "only OUR stopping job is worded as stopping")
}

// CR H1: a stop that lands only AFTER the run already failed for another
// reason (here: inside failItem, after it recorded the row) must not relabel
// the live event as 中途停止 — the event and the activity record agree.
func TestSoloRunner_LateCancelDoesNotRelabelAnEarlierFailure(t *testing.T) {
	var runner *SoloRunner
	var ref MediaRef
	s := newSoloHarness(t, translateDecision("Good morning."), []PipelineOption{
		WithProgress(func(_ MediaRef, stage PipelineStage, _ string) {
			if stage == StageFailed {
				runner.Cancel(ref)
			}
		}),
	})
	runner, ref = s.runner, s.ref
	s.placer.err = errors.New("placer: write failed: permission denied")

	_, err := s.runner.Start(context.Background(), s.ref, "")
	require.NoError(t, err)
	s.runner.waitIdle()

	row := s.runs.lastUpdate(t)
	assert.False(t, strings.HasPrefix(row.ErrorMessage, CancelledRunPrefix), row.ErrorMessage)
	data := s.hub.last().Data.(map[string]interface{})
	assert.Equal(t, models.SubtitleRunFailureReason(row.ErrorMessage), data["failure_reason"])
	assert.Equal(t, models.SubtitleRunFailureTargetNotWritable, data["failure_reason"])
	assert.Equal(t, "字幕生成失敗", data["message"], "not worded as a stop")
}

// CR H1, the other half: failItem's cancelled verdict reaches the caller in
// the error itself, even when the cause says only "signal: killed".
func TestProcessItem_CancelledVerdictIsInTheReturnedError(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.router.err = errors.New("ffprobe: signal: killed")

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{})
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, models.SubtitleRunFailureCancelled, soloFailureReason(nil, err))
}
