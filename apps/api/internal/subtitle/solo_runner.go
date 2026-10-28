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
	// sidecarUsable answers "might routing translate the English subtitle
	// beside the file?" — the router tries it before speech recognition, so
	// the gate must not refuse such a run either. nil = no sidecar lane.
	sidecarUsable func(ctx context.Context, mediaPath string) bool
	// title names the job for the Activity row and the events. Defaults to the
	// loaded item's TranslateContext.Title.
	title        func(ctx context.Context, ref MediaRef, item *MediaItem) string
	runBudgetUSD float64
	// spend publishes each running job's Budget by media id, so the events
	// this runner does not send itself — the pipeline's subtitle_progress
	// stages and the ASR leg's transcription_* — carry what the job has spent
	// so far (disc-2026-10-solo-live-spend). nil = not published.
	spend *services.SoloSpendBoard
	now   func() time.Time

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
	// QueuedAt is when the run began waiting for the extraction slot; zero
	// while it is not waiting. Guarded by SoloRunner.mu.
	QueuedAt time.Time
	// cancel stops this job's run ctx (disc-2026-10-solo-generation-cancel-api).
	// Set before the job is published in SoloRunner.jobs, so Cancel never
	// finds a job without one.
	cancel context.CancelFunc
	// Cancelling is true once Cancel reached this job; the job stays in
	// SoloRunner.jobs (and holds the single-flight claim) until its run has
	// wound down and its terminal event is out. Guarded by SoloRunner.mu.
	Cancelling bool
}

// ErrSoloGenerationCancelling is Start's refusal while the previous click on
// the same item is still stopping (disc-2026-10-solo-generation-cancel-api
// AC #6). It wraps services.ErrTranscriptionInProgress, so every existing
// errors.Is check still answers 409; the handler only words it differently.
var ErrSoloGenerationCancelling = fmt.Errorf("SUBTITLE_SOLO_CANCELLING: the previous run is still stopping: %w",
	services.ErrTranscriptionInProgress)

// Who holds an item's single-flight claim, as Cancel reports it.
const (
	// SoloOwnerSolo: a detail-page click — this runner's job, which Cancel stops.
	SoloOwnerSolo = "solo"
	// SoloOwnerOther: the consent batch, a pool worker or the free lane. Cancel
	// does not touch it; the batch has its own cancel.
	SoloOwnerOther = "other"
)

// SoloCancelResult is Cancel's answer — the cancel endpoints' payload.
type SoloCancelResult struct {
	// Cancelled: a solo job for the item now has a stop in effect (also on a
	// repeated call while it winds down — Cancel is idempotent).
	Cancelled bool
	// Running: something is still generating the item after the call,
	// including a just-cancelled job that has not finished stopping.
	Running bool
	// Owner names the claim holder while Running (SoloOwnerSolo /
	// SoloOwnerOther); empty when nothing runs.
	Owner string
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

// WithSoloSidecarCheck wires the English-sidecar answer. It must not wait for
// ffmpeg: it runs inside the click's HTTP request.
func WithSoloSidecarCheck(fn func(ctx context.Context, mediaPath string) bool) SoloRunnerOption {
	return func(s *SoloRunner) { s.sidecarUsable = fn }
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

// WithSoloSpendBoard publishes each running job's Budget on board, which the
// pipeline's stage events and the ASR leg read (disc-2026-10-solo-live-spend).
func WithSoloSpendBoard(board *services.SoloSpendBoard) SoloRunnerOption {
	return func(s *SoloRunner) { s.spend = board }
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
//     recognition and none is configured (no translate-only resume, and no
//     English sidecar the router could translate instead): 503;
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
		if s.Cancelling(ref) {
			return "", ErrSoloGenerationCancelling
		}
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
	// rides the start event as predicted_route (CR L1). A file with no usable
	// embedded track still routes to the English subtitle beside it first
	// (-b2), so one the stored check calls aligned — or has not settled — is
	// not refused either (disc-2026-10-solo-gate-en-sidecar).
	predicted := ""
	if s.predict != nil {
		route, perr := s.predict(ctx, item.FilePath)
		switch {
		case perr != nil:
			s.logger.Debug("solo run: route prediction failed — proceeding to routing",
				"media_id", ref.ID, "media_type", ref.MediaType, "error", perr)
		case route != PredictExtract && !s.asrOK() && !s.resumable(ctx, ref) &&
			!s.sidecarOK(ctx, item.FilePath):
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
	// Detached from the request ctx (the StartTranscription precedent): the
	// job outlives the HTTP call. The per-click Budget rides the ctx so the
	// pipeline keeps it rather than attaching its own, and the ASR leg's
	// resolveBudget shares it too — one ceiling across ASR and LLM. Built
	// HERE, before the job is published, so Cancel always finds its cancel
	// (disc-2026-10-solo-generation-cancel-api).
	runCtx, cancel := context.WithCancel(ai.WithBudget(context.Background(), job.Budget))
	job.cancel = cancel
	s.mu.Lock()
	s.jobs[ref] = job
	s.mu.Unlock()
	s.spend.Track(ref.ID, job.Budget)

	s.emitStart(ref, job, predicted)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer cancel()
		// The Activity row says 排隊中 while the run waits for the extraction
		// slot (subtitle demux or audio extraction), and its step otherwise.
		runCtx = services.WithExtractWaitObserver(runCtx, func(waiting bool) { s.setQueued(job, waiting) })

		outcome, runErr := s.pipeline.ProcessItem(runCtx, ref, ProcessItemOptions{
			Regenerate:            true,
			TranscribeWhenSkipped: true,
			ModelID:               modelID,
			// A stopped click must not erase the row's subtitle — a
			// re-generation leaves the old file on disk, and the ASR leg's
			// resume key must survive (disc-2026-10-solo-generation-cancel-api).
			RestoreMediaOnCancel: true,
		})
		// Read before the deferred cancel: non-nil only when Cancel stopped it.
		stopped := runCtx.Err()

		// Order matters (CR M2): stop publishing the live spend (the terminal
		// carries the final figure itself, PR #935), emit the terminal while
		// the claim is still ours — so "the job's last event" cannot be
		// overtaken by a second click's start event — then forget THIS job
		// (never a newer one), and only then let go of the in-flight claim.
		// Forgetting AFTER the terminal (disc-2026-10-solo-generation-cancel-api
		// CR L3): until the claim is released, Cancel and Start still see the
		// user's own job — never "someone else holds it".
		s.spend.Untrack(ref.ID, job.Budget)
		s.emitTerminal(ref, job, outcome, runErr, stopped)
		s.mu.Lock()
		if cur, ok := s.jobs[ref]; ok && cur == job {
			delete(s.jobs, ref)
		}
		s.mu.Unlock()
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

// Cancel stops the solo job generating ref, if there is one — the
// …/transcribe/cancel endpoints (disc-2026-10-solo-generation-cancel-api).
// It only cancels the job's ctx: the run winds down through the pipeline's
// own failure path (run row failed with the cancelled prefix, spend stamped,
// media row settled), then the job's terminal transcription_failed event goes
// out and the single-flight claim is released — the same order as any other
// end. Idempotent. An item held by the batch, a pool worker or the free lane
// is left alone (Owner SoloOwnerOther).
func (s *SoloRunner) Cancel(ref MediaRef) SoloCancelResult {
	s.mu.Lock()
	job, ok := s.jobs[ref]
	if ok {
		job.Cancelling = true
		job.cancel()
	}
	s.mu.Unlock()
	switch {
	case ok:
		s.logger.Info("solo subtitle run cancelled by the user",
			"job_id", job.ID, "media_id", ref.ID, "media_type", ref.MediaType)
		return SoloCancelResult{Cancelled: true, Running: true, Owner: SoloOwnerSolo}
	case s.guard.IsInFlight(ref):
		return SoloCancelResult{Running: true, Owner: SoloOwnerOther}
	default:
		return SoloCancelResult{}
	}
}

// Cancelling reports whether ref's solo job has been told to stop and has not
// finished stopping yet (the status route's `cancelling`).
func (s *SoloRunner) Cancelling(ref MediaRef) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[ref]
	return ok && job.Cancelling
}

// ActiveSoloJobs is the Activity page's 進行中 list for solo clicks — one
// snapshot per running job (disc-2026-10-activity-solo-jobs-collapsed), so
// eight clicks read as eight rows instead of one. The ASR leg of a pipeline
// run registers in TranscriptionService as Solo=false, so main.go can
// concatenate this with that service's solo jobs without counting twice.
// Unordered: the Activity service owns the row order.
func (s *SoloRunner) ActiveSoloJobs() []services.SoloJobSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]services.SoloJobSnapshot, 0, len(s.jobs))
	for ref, job := range s.jobs {
		// The job's Budget exists from acceptance, so a running job always
		// has a figure — 0 until something is paid (disc-2026-10-solo-live-spend).
		spent := job.Budget.SpentUSD()
		out = append(out, services.SoloJobSnapshot{
			MediaID:   ref.ID,
			Title:     job.Title,
			StartedAt: job.StartedAt,
			Queued:    !job.QueuedAt.IsZero(),
			QueuedAt:  job.QueuedAt,
			SpentUSD:  &spent,
		})
	}
	return out
}

// setQueued is the job's extraction-wait observer (services.WithExtractWaitObserver):
// true when its run starts waiting for the extraction slot, false once it
// holds it. A repeated true keeps the original QueuedAt — the wait did not
// restart.
func (s *SoloRunner) setQueued(job *soloJob, waiting bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case !waiting:
		job.QueuedAt = time.Time{}
	case job.QueuedAt.IsZero():
		job.QueuedAt = s.now().UTC()
	}
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

func (s *SoloRunner) sidecarOK(ctx context.Context, mediaPath string) bool {
	return s.sidecarUsable != nil && s.sidecarUsable(ctx, mediaPath)
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
	// The same writer the ASR leg and the pipeline stages use: spent always,
	// the ceiling only when there is one — absent means "no ceiling", never 0.
	services.AddRunCost(data, job.Budget)
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

// soloStoppedMessage is the terminal line of a run the user stopped — F3s's
// first line (pen-prompts-solo-progress-cancel.md 提示詞 5). Not a failure
// sentence: the client words the rest from failure_reason=cancelled.
const soloStoppedMessage = "已停止生成"

// emitTerminal composes the job's LAST event from what the pipeline produced.
// stopped is the run ctx's Err() when ProcessItem returned: non-nil only when
// Cancel stopped the job.
func (s *SoloRunner) emitTerminal(ref MediaRef, job *soloJob, outcome *ProcessOutcome, runErr, stopped error) {
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
	reason := soloFailureReason(run, runErr)
	if reason == models.SubtitleRunFailureCancelled && stopped != nil {
		message = soloStoppedMessage
	}
	data["message"] = message
	data["error"] = detail
	// The reason CODE the activity record gives this same run (failure_reason,
	// disc-2026-10-live-activity-failure-reasons-a AC #1): the client words it
	// from the code; `error` stays for the dialog's diagnostic line.
	data["failure_reason"] = reason
	s.broadcast(services.EventTranscriptionFailed, data)
	s.logger.Warn("solo subtitle run did not produce a subtitle",
		"job_id", job.ID, "media_id", ref.ID, "media_type", ref.MediaType, "message", message, "error", detail)
}

// soloFailureReason is the failure_reason code of a solo run's
// transcription_failed event — the same code the activity record derives from
// the run row (models.SubtitleRunFailureReason(run.ErrorMessage)), so the live
// log and the activity page name the same cause
// (disc-2026-10-live-activity-failure-reasons-a AC #2).
//
// The row is the record when the pipeline hands it back. A SKIPPED row is read
// the same way: when transcribeFallback could not reach speech recognition it
// wrote the no-text-source prefix (NoTextSourceRunPrefix), and the code is
// no_text_source (disc-2026-10-no-text-source-skip-a, live-a AC #1
// [@contract-v2]); any other skipped row (its text is the router's reason)
// comes out other. When the run FAILED,
// failItem returns no outcome — only the error — and the stored message may
// carry a prefix the error does not (failItem adds it from the error TYPE, and
// a killed subprocess says "signal: killed", not "context canceled"). So the
// error is asked the same type questions failItem asks, through the same
// predicates, in the same order, before its text is classified — and the text
// is cut exactly as failItem stores it (truncateErrorMessage): a long ffmpeg
// stderr can push the only telling words past the cut, and then the record
// says other, so the live log must too. (The credit and budget checks come
// from the error's type; their codes sit within the first few hundred bytes
// of any real message, so the stored text names them as well.)
//
// A run the user STOPPED is read from the error alone: failItem decides
// "cancelled" from the cause and the ctx at failure time, and when it does it
// puts context.Canceled into the returned error — so a ffmpeg the stop killed
// ("signal: killed") is cancelled here too, and a stop that arrives only
// after the run already failed for another reason cannot relabel it
// (disc-2026-10-solo-generation-cancel-api AC #4, CR H1).
func soloFailureReason(run *models.SubtitleRun, runErr error) string {
	if run != nil && run.ErrorMessage != "" {
		return models.SubtitleRunFailureReason(run.ErrorMessage)
	}
	if runErr == nil {
		return models.SubtitleRunFailureOther
	}
	if isCallerCancellation(runErr, nil) {
		return models.SubtitleRunFailureCancelled
	}
	if isQueueTimeout(runErr) {
		// No deadline on a solo run today, so unreachable — kept so the rule
		// cannot drift from failItem's when one is added.
		return models.SubtitleRunFailureQueueTimeout
	}
	var credit *ai.CreditExhaustedError
	if errors.As(runErr, &credit) {
		return models.SubtitleRunFailureCreditExhausted
	}
	if errors.Is(runErr, ai.ErrBudgetExceeded) {
		return models.SubtitleRunFailureBudgetExceeded
	}
	return models.SubtitleRunFailureReason(truncateErrorMessage(runErr.Error()))
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
	// An empty provider account: say whose, and where to top it up — the raw
	// error used to be the provider's whole JSON body
	// (disc-2026-09-asr-quota-429-treated-as-transient). The detail is a
	// sentence (it starts in CJK, so the panel shows it as prose); the live
	// log reads failure_reason (soloFailureReason), not this text.
	var credit *ai.CreditExhaustedError
	if runErr != nil && errors.As(runErr, &credit) {
		name := credit.DisplayName()
		return name + " 帳號沒有額度了，儲值後再按一次生成",
			name + " 帳號沒有額度了，請到 " + credit.BillingURL + " 儲值後再按一次生成"
	}
	switch {
	case runErr != nil && errors.Is(runErr, ai.ErrBudgetExceeded):
		snap := budget.Snapshot()
		return fmt.Sprintf("達到單次費用上限 $%.2f，已停在半路；已翻好的部分有保留", snap.BudgetUSD), runErr.Error()
	case runErr != nil:
		return "字幕生成失敗", runErr.Error()
	case run != nil && run.Status == models.SubtitleRunSkipped:
		// A solo run reaches a skipped row only through transcribeFallback
		// (the click sets TranscribeWhenSkipped), so it IS "no usable source
		// and no speech recognition"; its failure_reason is no_text_source
		// when the row carries the no-text-source prefix (soloFailureReason).
		// The detail is the stored row text, prefix included.
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
