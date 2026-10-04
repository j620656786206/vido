package services

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
)

// SubtitleRunLedger is the narrow port over repository.SubtitleRunRepository
// the transcription engine needs to record its OWN runs (sub-7-6a). Before
// this, a solo「生成字幕」click (StartTranscription) or a legacy-mode batch
// item (RouteCGenerationRunner) left no subtitle_runs row at all — the
// monthly ASR spend could only ever see pipeline-mode fallbacks.
type SubtitleRunLedger interface {
	Create(ctx context.Context, run *models.SubtitleRun) error
	Update(ctx context.Context, run *models.SubtitleRun) error
}

// SetRunLedger wires the ledger. Optional / nil-safe: unwired means no rows,
// the pre-sub-7-6a behaviour.
func (s *TranscriptionService) SetRunLedger(l SubtitleRunLedger) {
	s.runLedger = l
}

// WithRunRecordedByCaller tells the engine that the CALLER owns the run row.
// The D2 pipeline's ASR fallback (cmd/api/asr_adapter.go) passes it: that
// item already has a pipeline run row, and a second row for the same work
// would double-count it on the spend page.
func WithRunRecordedByCaller() TranscriptionOption {
	return func(cfg *transcriptionConfig) { cfg.recordedByCaller = true }
}

type ledgerOwnershipKey struct{}
type ledgerRunKey struct{}

// withLedgerOwnership marks ctx when the caller records the run itself, so
// runPipeline — which does not see the options — knows to open no row.
func withLedgerOwnership(ctx context.Context, recordedByCaller bool) context.Context {
	if !recordedByCaller {
		return ctx
	}
	return context.WithValue(ctx, ledgerOwnershipKey{}, true)
}

func ledgerOwnedByCaller(ctx context.Context) bool {
	v, _ := ctx.Value(ledgerOwnershipKey{}).(bool)
	return v
}

// withLedgerRun / ledgerRunFrom hand the open row down to translateSRT, which
// is where the version tuple, the cue count and the cache split become known.
func withLedgerRun(ctx context.Context, run *models.SubtitleRun) context.Context {
	if run == nil {
		return ctx
	}
	return context.WithValue(ctx, ledgerRunKey{}, run)
}

func ledgerRunFrom(ctx context.Context) *models.SubtitleRun {
	run, _ := ctx.Value(ledgerRunKey{}).(*models.SubtitleRun)
	return run
}

// openLedgerRun creates the `running` row for a run this engine owns and
// returns a ctx carrying it. nil when no ledger is wired, when the caller owns
// the row, or when the write fails — the run itself never fails over its
// bookkeeping (a missing receipt is a Warn, a missing subtitle is not).
func (s *TranscriptionService) openLedgerRun(ctx context.Context, mediaType, mediaID string) (*models.SubtitleRun, context.Context) {
	if s.runLedger == nil || ledgerOwnedByCaller(ctx) {
		return nil, ctx
	}
	run := &models.SubtitleRun{
		MediaID:   mediaID,
		MediaType: mediaType,
		Status:    models.SubtitleRunRunning,
		Route:     models.SubtitleRunRouteASR,
		BatchID:   GenerationBatchIDFromContext(ctx),
		StartedAt: time.Now().UTC(),
		// Route C transcription is only ever started by a person (a dialog or
		// a consent batch); the unattended lanes never reach paid ASR.
		TriggeredBy: models.SubtitleRunTriggeredManual,
	}
	if s.translationService != nil {
		run.ModelID = s.translationService.EffectiveModelID(ctx)
	}
	if err := s.runLedger.Create(ctx, run); err != nil {
		s.logger.Warn("subtitle run ledger: create failed — this run will have no receipt",
			"media_id", mediaID, "media_type", mediaType, "error", err)
		return nil, ctx
	}
	return run, withLedgerRun(ctx, run)
}

// stampLedgerVersion records what translateSRT learned: the RunVersion tuple
// (so the row is comparable with pipeline rows) and the routed cue count.
func stampLedgerVersion(ctx context.Context, v models.RunVersion, cueCount int) {
	run := ledgerRunFrom(ctx)
	if run == nil {
		return
	}
	run.MetadataHash = v.MetadataHash
	run.GlossaryVersion = v.GlossaryVersion
	run.PromptVersion = v.PromptVersion
	if v.ModelID != "" {
		run.ModelID = v.ModelID
	}
	run.CueCount = cueCount
}

// stampLedgerCache records the segment-cache split ONLY when a store was
// wired — without one every cue went to the model and "0 hits" would be a
// measurement that never happened (absent is not 0).
func stampLedgerCache(ctx context.Context, cached int, storeWired bool) {
	run := ledgerRunFrom(ctx)
	if run == nil || !storeWired {
		return
	}
	hits := cached
	run.CacheHitCues = &hits
}

// closeLedgerRun writes the terminal row — completed or failed — with this
// run's own spend (the Budget delta since the row opened: a legacy-mode batch
// shares one Budget across items, exactly like the pipeline's scope) — and
// emits the receipt event. runErr == nil is a completion.
func (s *TranscriptionService) closeLedgerRun(ctx context.Context, run *models.SubtitleRun, budget *ai.Budget, spentAtStart decimal.Decimal, outputPath string, runErr error) {
	if run == nil || s.runLedger == nil {
		return
	}
	completedAt := time.Now().UTC()
	run.CompletedAt = &completedAt
	run.OutputPath = outputPath
	if runErr != nil {
		run.Status = models.SubtitleRunFailed
		run.ErrorMessage = truncateLedgerMessage(runErr.Error())
	} else {
		run.Status = models.SubtitleRunCompleted
	}
	if budget != nil {
		snap := budget.Snapshot()
		delta := snap.Spent.Sub(spentAtStart)
		if delta.IsNegative() {
			delta = decimal.Zero
		}
		spent := delta.InexactFloat64()
		ceiling := snap.BudgetUSD
		run.SpentUSD = &spent
		run.BudgetUSD = &ceiling
	}
	// WithoutCancel: the row must close even when the failure WAS the
	// cancellation (the pipeline's failItem makes the same choice).
	if err := s.runLedger.Update(context.WithoutCancel(ctx), run); err != nil {
		s.logger.Warn("subtitle run ledger: terminal write failed",
			"run_id", run.ID, "media_id", run.MediaID, "error", err)
		return
	}
	s.broadcastEvent(sse.EventSubtitleRunReceipt, run.ReceiptPayload())
}

// truncateLedgerMessage keeps error_message bounded (the pipeline truncates
// at the same size; see subtitle.truncateErrorMessage).
func truncateLedgerMessage(msg string) string {
	const max = 1024
	if len(msg) <= max {
		return msg
	}
	return msg[:max]
}
