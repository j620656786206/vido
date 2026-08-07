package main

import (
	"context"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// pipelineASRAdapter bridges *services.TranscriptionService to the pipeline's
// subtitle.SpeechTranscriber port (sub-3-1). It lives HERE because the
// services package must not import subtitle (Rule 19 cycle point) and the
// subtitle package's port stays narrow — the same placement rationale as
// subtitlePlacerAdapter above it in this directory.
//
// Transcribe rides the SYNCHRONOUS 9R-16 seam — never StartTranscription,
// which detaches from the caller ctx and would sever both the shared
// ai.Budget and cancel propagation. WithTranslation is always requested: the
// pipeline's goal is a zh-Hant sidecar, and the service itself degrades an
// unconfigured translation key to the `untranslated` verdict (sub-2-2a).
type pipelineASRAdapter struct {
	ts *services.TranscriptionService
}

// Available is the sweep-side probe (worker-pool re-enumeration gate). It is
// the plain capability fact — key + FFmpeg. The per-item entry gate stays the
// service's own richer, resume-aware check inside RunTranscription.
func (a pipelineASRAdapter) Available() bool { return a.ts.IsAvailable() }

func (a pipelineASRAdapter) Transcribe(ctx context.Context, ref subtitle.MediaRef, filePath, mediaDir string) error {
	// WithMediaType routes the service's writeback + resume-read to the right
	// media table (sub-3-2) — movie stays the default for every other caller.
	return a.ts.RunTranscription(ctx, ref.ID, filePath, mediaDir,
		services.WithTranslation(), services.WithMediaType(ref.MediaType))
}
