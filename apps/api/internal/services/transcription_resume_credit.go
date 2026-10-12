package services

import (
	"context"
	"os"

	"github.com/vido/api/internal/segkey"
)

// ─── Resume credit (disc-2026-09-generation-resume-c-estimate-deduction) ───
//
// A run stopped by the money ceiling leaves paid-for work behind: stored ASR
// chunks (story B) and cached translated cues (story A). The next run reuses
// both, so the quote on 生成字幕 must not charge for them again — a full price
// on a half-finished film reads as "this costs $1.57" when the click will
// spend a fraction of it, and the user declines.
//
// Both answers are best-effort and only ever LOWER the quote: any read that
// fails, or anything that does not decode, counts as "nothing done" and the
// quote stays the full price (Rule 13 case 3 — the safe direction).

// resumedASRMinutes is how many minutes of this file's speech recognition are
// already stored and would be served back for free. It reads the film's
// manifest, then checks that the chunks it lists still exist (one GetMany),
// because the manifest and its chunks expire separately. Capped at
// runtimeMinutes: the last chunk is usually shorter than the grid.
//
// The estimate does not cut the audio, so it trusts the cut points to repeat
// (same file, same silence detection). If they move, the run pays for the
// moved chunks — never more than the full price this replaces.
func (s *TranscriptionService) resumedASRMinutes(ctx context.Context, mediaID, filePath string, runtimeMinutes float64) float64 {
	if s.chunkStore == nil || filePath == "" || runtimeMinutes <= 0 {
		return 0
	}
	size, mtime, err := osFileIdentity(filePath)
	if err != nil {
		return 0
	}
	endpoint := asrEndpointOf(s.asr)
	manifestKey := asrManifestKey(mediaID, size, mtime, endpoint)
	values, err := s.chunkStore.GetMany(ctx, []string{manifestKey})
	if err != nil {
		s.logger.Debug("resume credit: manifest read failed — quoting the full price",
			"media_id", mediaID, "error", err)
		return 0
	}
	m, ok := decodeASRManifest(values[manifestKey])
	if !ok || len(m.Done) == 0 {
		return 0
	}
	keys := make([]string, len(m.Done))
	for i, start := range m.Done {
		keys[i] = asrChunkKey(ASRChunkIdentity{
			MediaID:      mediaID,
			FileSize:     size,
			FileMTime:    mtime,
			TrackIndex:   m.Track,
			Lang:         m.Lang,
			Start:        start,
			ChunkSeconds: m.ChunkSeconds,
			Endpoint:     endpoint,
		})
	}
	chunks, err := s.chunkStore.GetMany(ctx, keys)
	if err != nil {
		s.logger.Debug("resume credit: chunk read failed — quoting the full price",
			"media_id", mediaID, "error", err)
		return 0
	}
	present := 0
	for _, key := range keys {
		if _, ok := decodeASRChunkValue(chunks[key]); ok {
			present++
		}
	}
	if present == 0 {
		return 0
	}
	if m.ChunkSeconds == 0 {
		// A short file is one chunk covering the whole audio.
		return runtimeMinutes
	}
	return min(float64(present*m.ChunkSeconds)/60, runtimeMinutes)
}

// cachedTranslationShare is the fraction (0–1) of the resume English SRT's
// cues whose translation is already cached under the key the translate run
// will look up. It is only meaningful for a translate-only resume: the file it
// reads is the one tryTranslateOnlyResume hands to translateSRT, and the keys
// come from translationInputsFor — the run's own builder — so the estimate
// cannot count a cue the run would miss (AC #4).
func (s *TranscriptionService) cachedTranslationShare(ctx context.Context, mediaType, mediaID string) float64 {
	if s.segmentStore == nil || s.translationService == nil {
		return 0
	}
	path, ok := s.resumeSource(ctx, mediaType, mediaID)
	if !ok {
		return 0
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	blocks, err := ParseSRTToTranslationBlocks(string(content))
	if err != nil || len(blocks) == 0 {
		return 0
	}
	keys := translationCacheKeys(blocks, s.translationInputsFor(ctx, mediaType, mediaID))
	values, err := s.segmentStore.GetMany(ctx, keys)
	if err != nil {
		s.logger.Debug("resume credit: segment cache read failed — quoting the full price",
			"media_id", mediaID, "error", err)
		return 0
	}
	hits := 0
	for _, key := range keys {
		if _, ok := values[key]; ok {
			hits++
		}
	}
	return float64(hits) / float64(len(keys))
}

// translationCacheKeys is the per-cue key list splitCachedBlocks looks up for
// these blocks under in's run version.
func translationCacheKeys(blocks []TranslationBlock, in translationInputs) []string {
	keys := make([]string, len(blocks))
	for i, b := range blocks {
		keys[i] = segkey.SegmentKey(b.Text, in.version)
	}
	return keys
}
