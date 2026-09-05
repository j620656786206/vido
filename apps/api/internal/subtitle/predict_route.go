package subtitle

import (
	"context"
	"fmt"

	"github.com/vido/api/internal/services"
)

// RoutePrediction is the PROBE-ONLY, cost-relevant classification of a media
// file (story sub-4-1 AC #3). It is deliberately NOT RouteKind: RouteKind names
// post-extraction verdicts (sub-1-4), three of which — deliver_direct,
// convert_then_deliver, translate — cannot be told apart until the track has
// actually been extracted and its content sniffed. This type answers only the
// question the cost screen needs: does this file need PAID speech recognition?
type RoutePrediction string

const (
	// PredictExtract — a usable embedded text track exists, so the expensive
	// ASR leg is skipped. Note this is "no ASR", NOT "free": an English track
	// still pays for LLM translation.
	PredictExtract RoutePrediction = "extract"
	// PredictASR — no usable embedded text track, so speech recognition is the
	// only way to produce a subtitle. This is the paid class.
	PredictASR RoutePrediction = "asr"
	// PredictSkip — embedded text tracks exist but none qualifies (an `und` or
	// other non-Chinese, non-eng/en tag; P0 never treats `und` as English).
	// The pipeline declines these outright: they get neither extraction nor
	// ASR, so they are not actionable by generation and must not be quoted.
	PredictSkip RoutePrediction = "skip"
)

// PredictRoute classifies mediaPath WITHOUT extracting anything.
//
// SelectAndRoute (the run-time path) always follows its probe with an ffmpeg
// extraction pass whenever candidates exist. Running that over a whole library
// just to price it would cost more than the thing being priced, so this entry
// point stops at the probe and reuses the two pure pieces the verdict is built
// from — SelectCandidates and the no-usable-track verdict.
//
// A probe failure is returned, never swallowed: classifying an unreadable file
// as PredictASR would quote the user for a paid route on a file we cannot read.
func (r *Router) PredictRoute(ctx context.Context, mediaPath string) (RoutePrediction, error) {
	route, _, err := r.PredictRouteWithDuration(ctx, mediaPath)
	return route, err
}

// PredictRouteWithDuration is PredictRoute plus the container duration the
// SAME probe already measured (sub-6-10a AC #1).
//
// It exists because episodes have no tech-info columns: the candidate sweep
// probes each of them for tracks and then threw the duration away, so every
// episode was priced at the 45-minute assumption while ffprobe had just
// reported its real length. Returning it costs nothing — no second probe, no
// second ffmpeg process — and PredictRoute above delegates here so there is
// exactly one probe path, not two that can drift.
//
// The duration is 0 when the container has no duration header. Callers must
// treat 0 as "unknown", never as a length.
func (r *Router) PredictRouteWithDuration(ctx context.Context, mediaPath string) (RoutePrediction, float64, error) {
	info, err := r.prober.Probe(ctx, mediaPath)
	if err != nil {
		return "", 0, fmt.Errorf("predict route for %s: %w", mediaPath, err)
	}
	if info == nil {
		return "", 0, fmt.Errorf("predict route for %s: probe returned no tech info", mediaPath)
	}
	return PredictFromTracks(info.SubtitleTracks), info.DurationSeconds, nil
}

// PredictFromTracks is the pure classifier over an already-known track list.
//
// It exists separately because movies carry their probe result in the
// `subtitle_tracks` column from post-scan enrichment — re-probing those would
// be wasted work. Episodes have no such column and unenriched movies may have
// none either, which is what PredictRoute above is for.
//
// The tiering mirrors the run-time router exactly: SelectCandidates decides
// what is usable (embedded + text codec + Chinese-or-eng/en tag), and anything
// it rejects falls through to the same text-vs-image reasoning
// verdictWithoutTrack applies.
func PredictFromTracks(tracks []services.SubtitleTrack) RoutePrediction {
	if len(SelectCandidates(tracks)) > 0 {
		return PredictExtract
	}

	for _, t := range tracks {
		if t.External {
			continue // sidecars are the search engine's domain, not generation's
		}
		if IsTextSubtitleCodec(t.Format) {
			// A text track the tiering declined — an `und` or other
			// unqualified tag. The pipeline skips it; ASR never runs.
			return PredictSkip
		}
	}
	return PredictASR
}
