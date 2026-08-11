package main

import (
	"context"
	"fmt"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// batchItemProcessor is the narrow subtitle.Pipeline surface the pipeline-mode
// batch runner drives (Rule 11 — enables a cmd/api test fake without a full
// Pipeline). *subtitle.Pipeline satisfies it.
type batchItemProcessor interface {
	ProcessItem(ctx context.Context, ref subtitle.MediaRef, opts subtitle.ProcessItemOptions) (*subtitle.ProcessOutcome, error)
}

// batchPoolGuard is the WorkerPool dedup bridge (CR sub-4-2 M4): the batch
// calls ProcessItem directly, so without sharing the pool's in-flight set a
// pool worker (FR12) and the batch could process the SAME media concurrently —
// two subtitle_runs rows, and the loser's terminal write clobbering the
// winner's media-row status. *subtitle.WorkerPool satisfies it.
type batchPoolGuard interface {
	TryReserve(ref subtitle.MediaRef) bool
	Release(ref subtitle.MediaRef)
}

// pipelineGenerationRunner bridges the consented batch (sub-4-2) to the D2
// pipeline. It lives HERE because the services package must not import
// subtitle (Rule 19 cycle point) — same placement rationale as
// pipelineASRAdapter / routePredictorAdapter in this directory.
//
// In pipeline mode the batch drives Pipeline.ProcessItem DIRECTLY (sequential,
// one item at a time) instead of the Route C transcription engine: an item
// with an extractable embedded subtitle takes the free extract→translate route
// and only no_text_source falls through to paid ASR (sub-3-1/3-2), so the F15
// per-route quote matches what actually runs. The WorkerPool QUEUE is not
// involved — the batch needs single-flight + a shared per-batch Budget +
// cancel + per-item progress, none of which the queue provides — but the
// pool's in-flight set IS shared via guard so batch and workers never process
// the same media concurrently.
//
// Error passthrough is contractual: the orchestrator classifies
// errors.Is(err, ai.ErrBudgetExceeded) as a PAUSE (budget_ceiling), and the
// pipeline's pauseASRItem returns that sentinel wrapped — this adapter must
// never swallow or rewrap the chain.
type pipelineGenerationRunner struct {
	pipeline batchItemProcessor
	// guard shares the WorkerPool's in-flight set (nil-safe for tests).
	guard batchPoolGuard
	// available is the pipeline capability gate (Claude key via the resolver —
	// the same subtitleCapabilityGate the FR12 endpoint and the pool use). An
	// ASR-less deployment stays available: extraction+translation still works,
	// and the ASR leg degrades per item inside the pipeline (such items come
	// back as skipped runs → ErrGenerationItemSkipped → fail_count, CR H1).
	available func() bool
}

// IsAvailable reports whether the pipeline can run at all (translation key).
func (r pipelineGenerationRunner) IsAvailable() bool {
	return r.pipeline != nil && r.available != nil && r.available()
}

// ExecuteGeneration runs one consented item through the D2 pipeline. filePath
// and mediaDir are unused — the pipeline loads media state itself via its
// MediaStore (movies, series, episodes).
//
// A (outcome, nil) return whose run ended SKIPPED is translated to
// services.ErrGenerationItemSkipped: the pipeline deliberately produced no
// subtitle (only non-target text tracks, or no_text_source with no live ASR),
// and crediting that as batch success would let a keyless deployment report
// N successes, $0 spent, zero subtitles (CR sub-4-2 H1).
func (r pipelineGenerationRunner) ExecuteGeneration(ctx context.Context, mediaID, mediaType, _, _ string) error {
	ref := subtitle.MediaRef{ID: mediaID, MediaType: mediaType}
	if r.guard != nil {
		if !r.guard.TryReserve(ref) {
			// The pool (FR12 or a worker) owns this media right now — same
			// classification as the Route C "user ran it mid-batch" case:
			// fail this item, continue the batch (CR sub-4-2 M4).
			return fmt.Errorf("media %s 已在管線佇列處理中: %w", mediaID, services.ErrTranscriptionInProgress)
		}
		defer r.guard.Release(ref)
	}
	outcome, err := r.pipeline.ProcessItem(ctx, ref, subtitle.ProcessItemOptions{})
	if err != nil {
		return err
	}
	if outcome != nil && outcome.Run != nil && outcome.Run.Status == models.SubtitleRunSkipped {
		return fmt.Errorf("media %s: %s: %w", mediaID, outcome.Run.ErrorMessage, services.ErrGenerationItemSkipped)
	}
	return nil
}
