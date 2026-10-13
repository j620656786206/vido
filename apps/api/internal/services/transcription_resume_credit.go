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
// already stored and would be served back for free. It is the one-film case of
// resumedASRMinutesMany — the single estimate and the consent list share one
// answer, so the two screens can never disagree about the same film.
//
// The estimate does not cut the audio, so it trusts the cut points to repeat
// (same file, same silence detection). If they move, the run pays for the
// moved chunks — never more than the full price this replaces.
func (s *TranscriptionService) resumedASRMinutes(ctx context.Context, mediaID, filePath string, runtimeMinutes float64) float64 {
	return s.resumedASRMinutesMany(ctx, []resumeQuery{{mediaID: mediaID, filePath: filePath, runtimeMinutes: runtimeMinutes}})[mediaID]
}

// resumeQuery names one film whose stored speech recognition is asked about.
type resumeQuery struct {
	mediaID        string
	filePath       string
	runtimeMinutes float64
}

// resumedASRMinutesMany answers resumedASRMinutes for many films in two reads
// (disc-2026-09-generation-resume-c2): every manifest in one GetMany, then
// every chunk those manifests list in one more — the consent list asks about
// a whole library, and two queries per film would be thousands of round
// trips. It reads each film's manifest, then checks that the chunks it lists
// still exist, because the manifest and its chunks expire separately. Capped
// at runtimeMinutes: the last chunk is usually shorter than the grid.
//
// Films with nothing stored are absent from the map (a zero read). Any failed
// read counts as nothing done for every film it covered.
func (s *TranscriptionService) resumedASRMinutesMany(ctx context.Context, queries []resumeQuery) map[string]float64 {
	out := make(map[string]float64)
	if s.chunkStore == nil || len(queries) == 0 {
		return out
	}
	endpoint := asrEndpointOf(s.asr)

	type film struct {
		q           resumeQuery
		size        int64
		mtime       int64
		manifestKey string
		manifest    asrManifest
		chunkKeys   []string
	}
	films := make([]*film, 0, len(queries))
	manifestKeys := make([]string, 0, len(queries))
	for _, q := range queries {
		if ctx.Err() != nil {
			return out
		}
		if q.filePath == "" || q.runtimeMinutes <= 0 {
			continue
		}
		size, mtime, err := osFileIdentity(q.filePath)
		if err != nil {
			continue
		}
		f := &film{q: q, size: size, mtime: mtime, manifestKey: asrManifestKey(q.mediaID, size, mtime, endpoint)}
		films = append(films, f)
		manifestKeys = append(manifestKeys, f.manifestKey)
	}
	if len(films) == 0 {
		return out
	}
	manifests, err := s.chunkStore.GetMany(ctx, manifestKeys)
	if err != nil {
		s.logger.Debug("resume credit: manifest read failed — quoting the full price",
			"films", len(films), "error", err)
		return out
	}

	var chunkKeys []string
	withWork := films[:0]
	for _, f := range films {
		m, ok := decodeASRManifest(manifests[f.manifestKey])
		if !ok || len(m.Done) == 0 {
			continue
		}
		f.manifest = m
		f.chunkKeys = make([]string, len(m.Done))
		for i, start := range m.Done {
			f.chunkKeys[i] = asrChunkKey(ASRChunkIdentity{
				MediaID:      f.q.mediaID,
				FileSize:     f.size,
				FileMTime:    f.mtime,
				TrackIndex:   m.Track,
				Lang:         m.Lang,
				Start:        start,
				ChunkSeconds: m.ChunkSeconds,
				Endpoint:     endpoint,
			})
		}
		chunkKeys = append(chunkKeys, f.chunkKeys...)
		withWork = append(withWork, f)
	}
	if len(withWork) == 0 {
		return out
	}
	chunks, err := s.chunkStore.GetMany(ctx, chunkKeys)
	if err != nil {
		s.logger.Debug("resume credit: chunk read failed — quoting the full price",
			"films", len(withWork), "error", err)
		return out
	}

	for _, f := range withWork {
		present := 0
		for _, key := range f.chunkKeys {
			if _, ok := decodeASRChunkValue(chunks[key]); ok {
				present++
			}
		}
		if present == 0 {
			continue
		}
		if f.manifest.ChunkSeconds == 0 {
			// A short file is one chunk covering the whole audio.
			out[f.q.mediaID] = f.q.runtimeMinutes
			continue
		}
		out[f.q.mediaID] = min(float64(present*f.manifest.ChunkSeconds)/60, f.q.runtimeMinutes)
	}
	return out
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
