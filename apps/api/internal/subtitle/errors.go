package subtitle

import "errors"

// Package-level sentinel errors for the generation pipeline (story sub-1-3).
//
// Wire format per project-context.md Rule 7: {SOURCE}_{ERROR_TYPE} under the
// EXISTING SUBTITLE_ prefix (architecture D7 — no new prefix; the authoritative
// prefix count stays 16). The leading code is what sub-1-6 lifts into
// ApiResponse.error.code, so each literal is a wire value even though the
// sentinel itself is Go-internal.
//
// These are sentinels rather than an AppError sub-type because the consumers are
// pipeline-internal (extractor / router / quality gate) and classify with
// errors.Is; the Rule 3 HTTP envelope — message and suggestion, in zh-TW — is
// composed once at the handler, which sub-1-6 owns. Same layering as the ai
// package's ErrAI* family (ai/types.go).
var (
	// ErrSubtitleExtractFailed — ffmpeg could not produce a usable subtitle file
	// from any candidate track. Consumer: sub-1-4.
	ErrSubtitleExtractFailed = errors.New("SUBTITLE_EXTRACT_FAILED: ffmpeg subtitle extraction failed")

	// ErrSubtitleNoTextSource — the file carries no usable text subtitle track
	// (image-only tracks, or none) — FR5. Recoverable by P2 ASR, unlike a
	// deliberately-skipped track. Consumer: sub-1-4.
	ErrSubtitleNoTextSource = errors.New("SUBTITLE_NO_TEXT_SOURCE: no usable text subtitle track")

	// ErrSubtitleTranslateFailed — LLM translation failed, or too many cues
	// remained untranslatable after the quality gate's retries. Consumer: sub-1-5a.
	ErrSubtitleTranslateFailed = errors.New("SUBTITLE_TRANSLATE_FAILED: LLM subtitle translation failed")

	// ErrSubtitleTimestampMismatch — the FR17 invariant broke: translated cue
	// count or per-cue timings diverge from the source track. Consumer: sub-1-5a.
	ErrSubtitleTimestampMismatch = errors.New("SUBTITLE_TIMESTAMP_MISMATCH: translated cue timestamps diverge from source")

	// ErrSubtitleTargetNotWritable — the media file's directory refused a
	// write probe, so the placer would fail AFTER the paid work. Raised by the
	// sub-6-1 pre-flight before any ffprobe, extract or LLM call. Consumer:
	// the item flow and the consent-list analysis (which reports it as
	// `writable=false` on the candidate instead of failing).
	ErrSubtitleTargetNotWritable = errors.New("SUBTITLE_TARGET_NOT_WRITABLE: target directory is not writable")

	// ErrSubtitleExtractWaitAborted — the item's ctx ended while it was QUEUED
	// behind another extraction on the ExtractGate (sub-6-3), before its own
	// ffmpeg ever started. Chained with the ctx error (Rule 13). It says
	// nothing about the file — the item flow treats it like a cancellation
	// (CancelledRunPrefix) so it never counts toward the free lane's
	// three-strikes parking; a deadline that fires INSIDE ffmpeg still does.
	ErrSubtitleExtractWaitAborted = errors.New("SUBTITLE_EXTRACT_WAIT_ABORTED: gave up waiting for the extraction slot")
)
