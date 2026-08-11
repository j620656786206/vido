package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// Compile-time facts the wiring in main.go depends on: both runners satisfy
// services.GenerationRunner, and *subtitle.Pipeline satisfies the adapter's
// narrow port.
var (
	_ services.GenerationRunner = pipelineGenerationRunner{}
	_ services.GenerationRunner = services.RouteCGenerationRunner{}
	_ batchItemProcessor        = (*subtitle.Pipeline)(nil)
	_ batchPoolGuard            = (*subtitle.WorkerPool)(nil)
)

type fakeBatchItemProcessor struct {
	gotRef  subtitle.MediaRef
	gotOpts subtitle.ProcessItemOptions
	outcome *subtitle.ProcessOutcome
	err     error
}

func (f *fakeBatchItemProcessor) ProcessItem(_ context.Context, ref subtitle.MediaRef, opts subtitle.ProcessItemOptions) (*subtitle.ProcessOutcome, error) {
	f.gotRef = ref
	f.gotOpts = opts
	return f.outcome, f.err
}

// sub-4-2 AC #4: the pipeline-mode runner maps (mediaID, mediaType) onto
// subtitle.MediaRef verbatim and never forces — a consented batch runs the
// normal resume-aware pipeline path.
func TestPipelineGenerationRunner_MapsMediaRef(t *testing.T) {
	fake := &fakeBatchItemProcessor{}
	r := pipelineGenerationRunner{pipeline: fake, available: func() bool { return true }}

	if err := r.ExecuteGeneration(context.Background(), "ep-9", models.SubtitleRunMediaEpisode, "/unused", "/unused"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := subtitle.MediaRef{ID: "ep-9", MediaType: models.SubtitleRunMediaEpisode}
	if fake.gotRef != want {
		t.Fatalf("ref = %+v, want %+v", fake.gotRef, want)
	}
	if fake.gotOpts.Force {
		t.Fatal("a consented batch must not force — the pipeline's resume-aware path is the point")
	}
}

// sub-4-2 AC #6: the adapter must pass the error chain through untouched —
// the orchestrator classifies errors.Is(err, ai.ErrBudgetExceeded) as a
// budget_ceiling PAUSE, and the pipeline's pauseASRItem returns that sentinel
// wrapped.
func TestPipelineGenerationRunner_PreservesBudgetExceededChain(t *testing.T) {
	fake := &fakeBatchItemProcessor{err: fmt.Errorf("asr paused: %w", ai.ErrBudgetExceeded)}
	r := pipelineGenerationRunner{pipeline: fake, available: func() bool { return true }}

	err := r.ExecuteGeneration(context.Background(), "m-1", models.SubtitleRunMediaMovie, "", "")
	if !errors.Is(err, ai.ErrBudgetExceeded) {
		t.Fatalf("ErrBudgetExceeded chain lost: %v", err)
	}
}

func TestPipelineGenerationRunner_IsAvailable(t *testing.T) {
	if (pipelineGenerationRunner{}).IsAvailable() {
		t.Fatal("nil pipeline must report unavailable")
	}
	up := pipelineGenerationRunner{pipeline: &fakeBatchItemProcessor{}, available: func() bool { return true }}
	if !up.IsAvailable() {
		t.Fatal("configured runner must report available")
	}
	down := pipelineGenerationRunner{pipeline: &fakeBatchItemProcessor{}, available: func() bool { return false }}
	if down.IsAvailable() {
		t.Fatal("capability gate must govern availability")
	}
}

// CR H1: a (outcome, nil) return whose run ended SKIPPED must surface as
// services.ErrGenerationItemSkipped — the batch counts it as a failure, so a
// keyless deployment can never report N successes with zero subtitles.
func TestPipelineGenerationRunner_SkippedRunIsNotSuccess(t *testing.T) {
	fake := &fakeBatchItemProcessor{outcome: &subtitle.ProcessOutcome{
		Run: &models.SubtitleRun{Status: models.SubtitleRunSkipped, ErrorMessage: "no text source"},
	}}
	r := pipelineGenerationRunner{pipeline: fake, available: func() bool { return true }}

	err := r.ExecuteGeneration(context.Background(), "m-1", models.SubtitleRunMediaMovie, "", "")
	if !errors.Is(err, services.ErrGenerationItemSkipped) {
		t.Fatalf("skipped run must map to ErrGenerationItemSkipped, got: %v", err)
	}
}

func TestPipelineGenerationRunner_CompletedRunIsSuccess(t *testing.T) {
	fake := &fakeBatchItemProcessor{outcome: &subtitle.ProcessOutcome{
		Run: &models.SubtitleRun{Status: models.SubtitleRunCompleted},
	}}
	r := pipelineGenerationRunner{pipeline: fake, available: func() bool { return true }}

	if err := r.ExecuteGeneration(context.Background(), "m-1", models.SubtitleRunMediaMovie, "", ""); err != nil {
		t.Fatalf("completed run must be success, got: %v", err)
	}
}

// fakePoolGuard records reserve/release traffic (CR M4).
type fakePoolGuard struct {
	busy     bool
	reserved []subtitle.MediaRef
	released []subtitle.MediaRef
}

func (g *fakePoolGuard) TryReserve(ref subtitle.MediaRef) bool {
	if g.busy {
		return false
	}
	g.reserved = append(g.reserved, ref)
	return true
}
func (g *fakePoolGuard) Release(ref subtitle.MediaRef) { g.released = append(g.released, ref) }

// CR M4: when the pool already owns the media (FR12 / a worker), the batch
// item fails with the existing in-progress classification instead of racing
// the worker on the same media row.
func TestPipelineGenerationRunner_PoolOwnedItemIsRejected(t *testing.T) {
	fake := &fakeBatchItemProcessor{}
	guard := &fakePoolGuard{busy: true}
	r := pipelineGenerationRunner{pipeline: fake, guard: guard, available: func() bool { return true }}

	err := r.ExecuteGeneration(context.Background(), "m-1", models.SubtitleRunMediaMovie, "", "")
	if !errors.Is(err, services.ErrTranscriptionInProgress) {
		t.Fatalf("pool-owned item must classify as in-progress, got: %v", err)
	}
	if fake.gotRef.ID != "" {
		t.Fatal("ProcessItem must NOT run when the pool owns the item")
	}
}

func TestPipelineGenerationRunner_ReservesAndReleasesAroundProcessItem(t *testing.T) {
	fake := &fakeBatchItemProcessor{outcome: &subtitle.ProcessOutcome{
		Run: &models.SubtitleRun{Status: models.SubtitleRunCompleted},
	}}
	guard := &fakePoolGuard{}
	r := pipelineGenerationRunner{pipeline: fake, guard: guard, available: func() bool { return true }}

	want := subtitle.MediaRef{ID: "ep-1", MediaType: models.SubtitleRunMediaEpisode}
	if err := r.ExecuteGeneration(context.Background(), "ep-1", models.SubtitleRunMediaEpisode, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(guard.reserved) != 1 || guard.reserved[0] != want {
		t.Fatalf("reserve = %+v, want [%+v]", guard.reserved, want)
	}
	if len(guard.released) != 1 || guard.released[0] != want {
		t.Fatalf("release = %+v, want [%+v] (a leaked reservation deadlocks FR12 for this media)", guard.released, want)
	}
}
