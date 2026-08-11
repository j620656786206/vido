package services

import "context"

// transcriptionGenerationSeam is the narrow TranscriptionService surface the
// legacy-mode batch runner rides (Rule 11 — enables an in-package fake that
// can assert which TranscriptionOptions were forwarded).
type transcriptionGenerationSeam interface {
	IsAvailable() bool
	RunTranscription(ctx context.Context, mediaID string, filePath string, mediaDir string, opts ...TranscriptionOption) error
}

// RouteCGenerationRunner adapts TranscriptionService to GenerationRunner for
// LEGACY pipeline mode (sub-4-2 AC #5): the batch keeps today's Route C
// transcribe→translate engine, but now forwards the item's media type so the
// sub-3-2 writeback/resume dispatch reaches the right table — a missing
// WithMediaType silently defaults to movie and would write an episode's
// subtitle status onto the movies table.
type RouteCGenerationRunner struct {
	TS transcriptionGenerationSeam
}

// NewRouteCGenerationRunner wires the legacy-mode batch runner.
func NewRouteCGenerationRunner(ts transcriptionGenerationSeam) RouteCGenerationRunner {
	return RouteCGenerationRunner{TS: ts}
}

// IsAvailable reports the Route C capability gate (ASR key + FFmpeg).
func (r RouteCGenerationRunner) IsAvailable() bool {
	return r.TS != nil && r.TS.IsAvailable()
}

// ExecuteGeneration runs one item through the synchronous 9R-16 seam.
// WithTranslation preserves the pre-sub-4-2 behavior byte-for-byte for movies;
// WithMediaType is the sub-4-2 addition that makes episodes write back to the
// episodes table.
func (r RouteCGenerationRunner) ExecuteGeneration(ctx context.Context, mediaID, mediaType, filePath, mediaDir string) error {
	return r.TS.RunTranscription(ctx, mediaID, filePath, mediaDir,
		WithTranslation(), WithMediaType(mediaType))
}
