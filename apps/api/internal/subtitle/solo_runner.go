package subtitle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
)

// SoloRunner is the detail-page 生成字幕 click as a JOB over
// Pipeline.ProcessItem (disc-2026-10-single-generate-ignores-embedded-english-a).
//
// Before it, that click always went to TranscriptionService.StartTranscription
// — speech recognition + translation, whatever the file carried — while the
// batch already routed through the pipeline (Chinese track → deliver, English
// track → translate, nothing → ASR). See S01E02 showed the cost of the split:
// a file with an official English track was transcribed instead, with the
// timing, names and dropped lines that come from listening, plus $0.34 of ASR
// nobody needed to pay. Routing now lives in ONE place; this type only adds
// what a single click needs on top of the item flow:
//
//   - single flight shared with the pool and the batch (the pool's in-flight
//     set, through SoloGuard), so the click's 409 and the status endpoint
//     agree with everything else that may be generating the item;
//   - a per-click ai.Budget, so ASR and LLM spend share one ceiling and the
//     events carry the figure;
//   - a job id and a title, and the job-level transcription_* START and
//     TERMINAL events the dialog and the workspace already listen to. The
//     pipeline's own D6 subtitle_progress events (no job id, no path, no cost)
//     and the ASR leg's service events (a different job id) still fire in
//     between; the contract is that the event carrying THIS job's id is its
//     last one.
//
// It lives in this package because it needs the Pipeline's unexported
// writable probe and media store; handlers may import subtitle (Rule 19).
type SoloRunner struct {
	pipeline *Pipeline
	guard    SoloGuard
	hub      ProgressBroadcaster
	logger   *slog.Logger

	// predict is the probe-only route classifier (Router.PredictRoute) behind
	// the capability gate. nil = no gate: every click proceeds to routing.
	predict func(ctx context.Context, mediaPath string) (RoutePrediction, error)
	// asrAvailable is the speech-recognition capability fact. nil = unknown,
	// which never blocks (the item degrades inside the pipeline instead).
	asrAvailable func() bool
	// canResume answers "would the ASR leg resume translate-only?" — such a
	// run needs no ASR, so the gate must not refuse it (CR sub-2-2a M2).
	canResume func(ctx context.Context, ref MediaRef) bool
	// title names the job for the Activity row and the events. Defaults to the
	// loaded item's TranslateContext.Title.
	title        func(ctx context.Context, ref MediaRef, item *MediaItem) string
	runBudgetUSD float64
	now          func() time.Time

	mu   sync.Mutex
	jobs map[MediaRef]*soloJob
	wg   sync.WaitGroup
}

// SoloGuard is the slice of WorkerPool the runner shares: the in-flight set
// that makes "one generation per item at a time" true across the pool's
// workers, the consented batch and this runner.
type SoloGuard interface {
	TryReserve(ref MediaRef) bool
	Release(ref MediaRef)
	IsInFlight(ref MediaRef) bool
}

type soloJob struct {
	ID        string
	Title     string
	Budget    *ai.Budget
	StartedAt time.Time
}

// SoloRunnerOption configures one optional knob of NewSoloRunner.
type SoloRunnerOption func(*SoloRunner)

// WithSoloRoutePredictor wires the probe-only classifier behind the gate.
func WithSoloRoutePredictor(fn func(ctx context.Context, mediaPath string) (RoutePrediction, error)) SoloRunnerOption {
	return func(s *SoloRunner) { s.predict = fn }
}

// WithSoloASRAvailability wires the speech-recognition capability fact.
func WithSoloASRAvailability(fn func() bool) SoloRunnerOption {
	return func(s *SoloRunner) { s.asrAvailable = fn }
}

// WithSoloResumeCheck wires the translate-only resume answer.
func WithSoloResumeCheck(fn func(ctx context.Context, ref MediaRef) bool) SoloRunnerOption {
	return func(s *SoloRunner) { s.canResume = fn }
}

// WithSoloTitle overrides how a job is named (e.g. 「劇名 S01E02」 for episodes).
func WithSoloTitle(fn func(ctx context.Context, ref MediaRef, item *MediaItem) string) SoloRunnerOption {
	return func(s *SoloRunner) { s.title = fn }
}

// WithSoloRunBudgetUSD sets the per-click AI cost ceiling (AI_RUN_BUDGET_USD;
// 0 = unlimited) — the same envelope the pipeline applies to budget-less items.
func WithSoloRunBudgetUSD(usd float64) SoloRunnerOption {
	return func(s *SoloRunner) { s.runBudgetUSD = usd }
}

// WithSoloClock injects the clock (tests).
func WithSoloClock(now func() time.Time) SoloRunnerOption {
	return func(s *SoloRunner) {
		if now != nil {
			s.now = now
		}
	}
}

// NewSoloRunner wires the runner. pipeline and guard are mandatory; hub may be
// nil (no events, as every other hook in this package treats a nil hub).
func NewSoloRunner(pipeline *Pipeline, guard SoloGuard, hub ProgressBroadcaster, logger *slog.Logger, opts ...SoloRunnerOption) *SoloRunner {
	if pipeline == nil {
		panic("subtitle.NewSoloRunner: Pipeline must not be nil")
	}
	if guard == nil {
		panic("subtitle.NewSoloRunner: SoloGuard must not be nil — single flight is the whole point")
	}
	if logger == nil {
		logger = slog.Default()
	}
	s := &SoloRunner{
		pipeline: pipeline,
		guard:    guard,
		hub:      hub,
		logger:   logger.With("component", "subtitle_solo_runner"),
		title: func(_ context.Context, _ MediaRef, item *MediaItem) string {
			if item == nil {
				return ""
			}
			return item.Context.Title
		},
		now:  time.Now,
		jobs: make(map[MediaRef]*soloJob),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start begins one click. It returns the job id once the run is accepted, or
// one of the services sentinels the transcription handler already maps:
//
//   - services.ErrTranscriptionInProgress — the item is in flight elsewhere
//     (pool worker, batch, or an earlier click): 409;
//   - services.ErrTranscriptionDisabled — the file would need speech
//     recognition and none is configured (and no translate-only resume): 503;
//   - ErrSubtitleTargetNotWritable — the media folder refused the write probe,
//     before anything was paid: 409.
//
// Everything after acceptance happens on a detached goroutine; failures reach
// the client as the job's transcription_failed event, never as this error.
func (s *SoloRunner) Start(ctx context.Context, ref MediaRef, modelID string) (jobID string, err error) {
	if ref.MediaType != models.SubtitleRunMediaMovie && ref.MediaType != models.SubtitleRunMediaEpisode {
		return "", fmt.Errorf("subtitle solo run: %q is not a generatable media type", ref.MediaType)
	}
	if s.pipeline.media == nil {
		return "", errors.New("subtitle solo run: pipeline has no MediaStore (WithMediaStore)")
	}

	item, err := s.pipeline.media.Load(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("subtitle solo run: load %s %s: %w", ref.MediaType, ref.ID, err)
	}
	if item == nil || item.FilePath == "" {
		return "", fmt.Errorf("subtitle solo run: %s %s has no media file path", ref.MediaType, ref.ID)
	}

	// Single flight FIRST: every later refusal releases this claim, and a
	// refusal here must not touch a claim that belongs to someone else.
	if !s.guard.TryReserve(ref) {
		return "", services.ErrTranscriptionInProgress
	}
	defer func() {
		if err != nil {
			s.guard.Release(ref)
		}
	}()

	// Capability gate. Routing itself is free (ffprobe + a local extract), so
	// only a file that would END UP on speech recognition is refused when
	// there is none to run — the same 503 the click answered before this
	// story, for the same files. A probe that cannot be read never refuses:
	// "語音辨識尚未設定" would be the wrong diagnosis for an unreadable file,
	// and the pipeline reports the real failure. ONE probe: the same answer
	// rides the start event as predicted_route (CR L1).
	predicted := ""
	if s.predict != nil {
		route, perr := s.predict(ctx, item.FilePath)
		switch {
		case perr != nil:
			s.logger.Debug("solo run: route prediction failed — proceeding to routing",
				"media_id", ref.ID, "media_type", ref.MediaType, "error", perr)
		case route != PredictExtract && !s.asrOK() && !s.resumable(ctx, ref):
			return "", services.ErrTranscriptionDisabled
		default:
			predicted = string(route)
		}
	}

	// Writable probe BEFORE the paid legs, synchronously, so the click gets
	// its 409 instead of a failed job (disc-2026-09-solo-run-unwritable-folder-pays-asr).
	// The pipeline probes again after routing; that repeat is cheap.
	if s.pipeline.probeWritable != nil {
		if perr := s.pipeline.probeWritable(ctx, filepath.Dir(item.FilePath)); perr != nil {
			return "", fmt.Errorf("%w: %v", ErrSubtitleTargetNotWritable, perr)
		}
	}

	job := &soloJob{
		ID:        uuid.New().String(),
		Title:     s.title(ctx, ref, item),
		Budget:    ai.NewBudget(s.runBudgetUSD),
		StartedAt: s.now().UTC(),
	}
	s.mu.Lock()
	s.jobs[ref] = job
	s.mu.Unlock()

	s.emitStart(ref, job, predicted)

	// Detached from the request ctx (the StartTranscription precedent): the
	// job outlives the HTTP call. The per-click Budget rides the ctx so the
	// pipeline keeps it rather than attaching its own, and the ASR leg's
	// resolveBudget shares it too — one ceiling across ASR and LLM.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		runCtx, cancel := context.WithCancel(ai.WithBudget(context.Background(), job.Budget))
		defer cancel()

		outcome, runErr := s.pipeline.ProcessItem(runCtx, ref, ProcessItemOptions{
			Regenerate:            true,
			TranscribeWhenSkipped: true,
			ModelID:               modelID,
		})

		// Order matters (CR M2): forget THIS job (never a newer one that may
		// already own the ref), emit its terminal while the claim is still
		// ours — so "the job's last event" cannot be overtaken by a second
		// click's start event — and only then let go of the in-flight claim.
		s.mu.Lock()
		if cur, ok := s.jobs[ref]; ok && cur == job {
			delete(s.jobs, ref)
		}
		s.mu.Unlock()
		s.emitTerminal(ref, job, outcome, runErr)
		s.guard.Release(ref)
	}()

	return job.ID, nil
}

// Status answers GET …/transcribe/status: whether anything is generating ref
// right now, and — when it is one of OUR jobs — its id, so a reopened dialog
// can attach to the right terminal event. A pool worker or the batch owning
// the item reports in-progress with no id.
func (s *SoloRunner) Status(ref MediaRef) (inProgress bool, jobID string) {
	s.mu.Lock()
	job, ok := s.jobs[ref]
	s.mu.Unlock()
	if ok {
		return true, job.ID
	}
	return s.guard.IsInFlight(ref), ""
}

// ActivityProgress is the Activity page's in-flight row for solo clicks —
// the same primitive shape services.batchJobSource takes, so main.go can
// compose this runner with TranscriptionService (legacy solo runs, and the
// ASR leg of a pipeline run counts itself as Solo=false there, so nothing is
// counted twice). currentItem is one running job's title when several run.
func (s *SoloRunner) ActivityProgress() (active bool, percentDone, current, total int, currentItem string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range s.jobs {
		current++
		currentItem = job.Title
	}
	return current > 0, 0, current, 0, currentItem
}

// ActiveManualUsage is GET /ai/usage's live-spend figure for solo clicks
// (9R-17 AC #3): the summed spend and ceiling of every running job. ok is
// false when nothing is running — absent, never 0.
func (s *SoloRunner) ActiveManualUsage() (spentUSD, budgetUSD float64, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range s.jobs {
		snap := job.Budget.Snapshot()
		spentUSD += snap.SpentUSD
		budgetUSD += snap.BudgetUSD
		ok = true
	}
	return spentUSD, budgetUSD, ok
}

// waitIdle blocks until every started job has finished (tests).
func (s *SoloRunner) waitIdle() { s.wg.Wait() }

func (s *SoloRunner) asrOK() bool {
	// Unknown is treated as available: the gate exists to stop a pointless
	// paid attempt, never to refuse on a wiring gap it cannot judge.
	return s.asrAvailable == nil || s.asrAvailable()
}

func (s *SoloRunner) resumable(ctx context.Context, ref MediaRef) bool {
	return s.canResume != nil && s.canResume(ctx, ref)
}

// ─── Events ────────────────────────────────────────────────────────────────

// jobFields are the keys every event of one job carries.
func (s *SoloRunner) jobFields(ref MediaRef, job *soloJob, phase string) map[string]interface{} {
	data := map[string]interface{}{
		"job_id":     job.ID,
		"media_id":   ref.ID,
		"media_type": ref.MediaType,
		"title":      job.Title,
		"phase":      phase,
	}
	// Mirrors TranscriptionService.costFields: spent always, the ceiling only
	// when there is one — absent means "no ceiling", never 0.
	snap := job.Budget.Snapshot()
	data["spent_usd"] = snap.SpentUSD
	if snap.BudgetUSD > 0 {
		data["budget_usd"] = snap.BudgetUSD
	}
	return data
}

func (s *SoloRunner) emitStart(ref MediaRef, job *soloJob, predicted string) {
	data := s.jobFields(ref, job, "extracting")
	data["message"] = "正在檢查片內字幕…"
	if predicted != "" {
		data["predicted_route"] = predicted
	}
	s.broadcast(services.EventTranscriptionExtracting, data)
}

// emitTerminal composes the job's LAST event from what the pipeline produced.
func (s *SoloRunner) emitTerminal(ref MediaRef, job *soloJob, outcome *ProcessOutcome, runErr error) {
	var run *models.SubtitleRun
	if outcome != nil {
		run = outcome.Run
	}

	switch {
	case runErr == nil && run == nil && outcome != nil && outcome.SubtitlePath != "":
		// Pre-flight early exit. Regenerate makes this unreachable today, but a
		// future caller that drops it must still get an honest terminal.
		data := s.jobFields(ref, job, "complete")
		data["zh_srt_path"] = outcome.SubtitlePath
		data["message"] = "字幕已存在，沒有重新生成"
		s.broadcast(services.EventTranscriptionComplete, data)
		return

	case runErr == nil && run != nil && run.Status == models.SubtitleRunCompleted:
		data := s.jobFields(ref, job, "complete")
		data["route"] = run.Route
		if outcome.SubtitlePath != "" {
			data["zh_srt_path"] = outcome.SubtitlePath
		}
		if run.CueCount > 0 {
			data["cue_count"] = run.CueCount
		}
		msg := completeMessageForRoute(run.Route)
		if n := transientCuesOf(run); n > 0 {
			data["partial"] = true
			data["english_kept_blocks"] = n
			msg += fmt.Sprintf("（部分翻譯失敗，%d 句保留英文）", n)
		}
		data["message"] = msg
		s.broadcast(services.EventTranscriptionComplete, data)
		return
	}

	data := s.jobFields(ref, job, "failed")
	if run != nil && run.Route != "" {
		data["route"] = run.Route
	}
	message, detail := failureMessage(run, runErr, job.Budget)
	data["message"] = message
	data["error"] = detail
	s.broadcast(services.EventTranscriptionFailed, data)
	s.logger.Warn("solo subtitle run did not produce a subtitle",
		"job_id", job.ID, "media_id", ref.ID, "media_type", ref.MediaType, "message", message, "error", detail)
}

// completeMessageForRoute is the zh-TW verdict per lane. Rule 3: the words
// live at the edge, keyed on the ledger's route vocabulary.
func completeMessageForRoute(route string) string {
	switch route {
	case models.SubtitleRunRouteDeliverDirect:
		return "字幕已生成（直接使用片內中文字幕，沒有花錢）"
	case models.SubtitleRunRouteConvertThenDeliver:
		return "字幕已生成（片內簡體字幕已轉成繁體，沒有花錢）"
	case models.SubtitleRunRouteTranslate:
		return "翻譯完成（翻譯片內英文字幕）"
	case models.SubtitleRunRouteASR:
		return "轉錄完成"
	default:
		return "字幕已生成"
	}
}

// failureMessage turns the pipeline's outcome into the user-facing line plus
// the diagnostic detail the panel shows under it.
func failureMessage(run *models.SubtitleRun, runErr error, budget *ai.Budget) (message, detail string) {
	switch {
	case runErr != nil && errors.Is(runErr, ai.ErrBudgetExceeded):
		snap := budget.Snapshot()
		return fmt.Sprintf("達到單次費用上限 $%.2f，已停在半路；已翻好的部分有保留", snap.BudgetUSD), runErr.Error()
	case runErr != nil:
		return "字幕生成失敗", runErr.Error()
	case run != nil && run.Status == models.SubtitleRunSkipped:
		return "這部影片沒有可用的字幕來源，也沒有設定語音辨識", run.ErrorMessage
	case run != nil && run.Status == models.SubtitleRunFailed:
		return "字幕生成失敗", run.ErrorMessage
	default:
		return "字幕生成失敗", "the pipeline returned no run and no error"
	}
}

func (s *SoloRunner) broadcast(eventType sse.EventType, data map[string]interface{}) {
	if s.hub == nil {
		return
	}
	s.hub.Broadcast(sse.Event{ID: uuid.New().String(), Type: eventType, Data: data})
}
