package subtitle

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/vido/api/internal/services"
)

// RouteKind classifies what the pipeline should do with a media item.
// [@contract-v1] — consumed by sub-1-5a (RouteTranslate), sub-1-5b (delivery
// of RouteDeliverDirect/RouteConvertThenDeliver + terminal status writes),
// sub-1-6 (orchestration). Changing kinds/fields = Rule 20 bump + stale-mark.
//
// Deliberately NOT models.SubtitleStatus: routing is a decision, persistence is
// a side effect. The orchestrator maps RouteSkip → SubtitleStatusSkipped and
// RouteNoTextSource → SubtitleStatusNoTextSource when it writes (1.5b). Keeping
// the enums separate stops the DB vocabulary leaking into pure functions.
type RouteKind string

const (
	RouteDeliverDirect      RouteKind = "deliver_direct"       // FR7 — content is already zh-Hant
	RouteConvertThenDeliver RouteKind = "convert_then_deliver" // FR8 — content is zh-Hans → OpenCC (caller runs it)
	RouteTranslate          RouteKind = "translate"            // FR10 — English → LLM
	RouteSkip               RouteKind = "skip"                 // FR9/P0 — no Chinese- or eng/en-tagged text track
	RouteNoTextSource       RouteKind = "no_text_source"       // FR5 — no usable text source at all
)

// RouteDecision is the front-half verdict handed to the orchestrator.
type RouteDecision struct {
	Kind            RouteKind
	Track           *ExtractedTrack // nil for RouteSkip / RouteNoTextSource
	DetectedVariant string          // detector result for the chosen track ("zh-Hant"/"zh-Hans"/"zh"/"und")
	Reason          string          // human-readable, for logs + the orchestrator's status message
}

// ExtractedTrack is the chosen embedded track after extraction and filtering.
type ExtractedTrack struct {
	StreamIndex int             // absolute ffmpeg stream index
	Language    string          // the ffprobe tag that admitted it ("chi"/"zho"/…/"eng"/"en")
	Codec       string          // source codec (subrip/ass/mov_text/…)
	Path        string          // extracted .srt in the caller-owned temp dir
	Blocks      []SubtitleBlock // parsed + SDH-filtered cues (original numbering — P7)
}

// TechProber is the narrow port the router needs from services.FFprobeService,
// so unit tests can inject a fake. subtitle → services is a legal Rule 19
// direction (the engine.go precedent).
type TechProber interface {
	Probe(ctx context.Context, filePath string) (*services.MediaTechInfo, error)
}

// TrackExtractor is the narrow port the router needs from *Extractor.
type TrackExtractor interface {
	Extract(ctx context.Context, mediaPath, tmpDir string, streamIndexes []int) (map[int]string, error)
}

// Router turns a media file into a routing verdict: probe → extract →
// SDH-filter → detect. It is PURE with respect to the outside world — it writes
// nothing to the DB, broadcasts no SSE, and never calls converter.go or
// placer.go. The verdict tells the caller to.
type Router struct {
	prober    TechProber
	extractor TrackExtractor
	logger    *slog.Logger
}

// NewRouter wires the router to its two ports.
func NewRouter(prober TechProber, extractor TrackExtractor, logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{
		prober:    prober,
		extractor: extractor,
		logger:    logger.With("component", "subtitle_router"),
	}
}

// SelectAndRoute probes fresh, extracts every candidate track in ONE ffmpeg
// pass, SDH-filters, detects the CJK variant, and returns the routing verdict.
//
// The probe is deliberately re-run at run time rather than reading the persisted
// subtitle_tracks JSON: files move or change after a scan, pre-existing rows
// carry no stream index, and `-map 0:{n}` needs the index to be true NOW. The
// persisted column stays the scan-time signal; this probe is the run-time truth.
func (r *Router) SelectAndRoute(ctx context.Context, mediaPath, tmpDir string) (RouteDecision, error) {
	info, err := r.prober.Probe(ctx, mediaPath)
	if err != nil {
		return RouteDecision{}, fmt.Errorf("subtitle route: probe %s: %w", mediaPath, err)
	}

	var tracks []services.SubtitleTrack
	if info != nil {
		tracks = info.SubtitleTracks
	}

	candidates := SelectCandidates(tracks)
	if len(candidates) == 0 {
		return r.verdictWithoutTrack(tracks), nil
	}

	indexes := make([]int, 0, len(candidates))
	for _, c := range candidates {
		indexes = append(indexes, c.StreamIndex)
	}

	outputs, err := r.extractor.Extract(ctx, mediaPath, tmpDir, indexes)
	if err != nil {
		return RouteDecision{}, fmt.Errorf("subtitle route: extract %s: %w", mediaPath, err)
	}

	best, variant, ok := r.pickBestCandidate(candidates, outputs)
	if !ok {
		return RouteDecision{}, fmt.Errorf(
			"%w: no candidate track of %s could be parsed (%d attempted)",
			ErrSubtitleExtractFailed, mediaPath, len(candidates))
	}

	if len(best.Blocks) == 0 {
		// FR5's word is *usable*: a track whose every cue was an SDH annotation
		// carries no dialogue to translate or deliver.
		return RouteDecision{
			Kind:   RouteNoTextSource,
			Reason: fmt.Sprintf("stream %d parsed but no cue survived SDH filtering", best.StreamIndex),
		}, nil
	}

	kind, reason := routeForVariant(variant, best.StreamIndex, best.Language)

	r.logger.Debug("subtitle route decided",
		"media", mediaPath,
		"stream_index", best.StreamIndex,
		"language_tag", best.Language,
		"detected_variant", variant,
		"cue_count", len(best.Blocks),
		"route", string(kind),
	)

	track := best
	return RouteDecision{
		Kind:            kind,
		Track:           &track,
		DetectedVariant: variant,
		Reason:          reason,
	}, nil
}

// verdictWithoutTrack distinguishes FR9 (a text track exists but M1 refuses to
// guess its language) from FR5 (there is no usable text source at all). Only the
// former is a deliberate skip; the latter is what P2's ASR can later recover.
func (r *Router) verdictWithoutTrack(tracks []services.SubtitleTrack) RouteDecision {
	textTracks, imageTracks := 0, 0
	for _, t := range tracks {
		if t.External {
			continue // M1 is embedded-only; sidecars stay the search engine's domain
		}
		switch {
		case IsTextSubtitleCodec(t.Format):
			textTracks++
		case IsImageSubtitleCodec(t.Format):
			imageTracks++
		}
	}

	if textTracks > 0 {
		return RouteDecision{
			Kind: RouteSkip,
			Reason: fmt.Sprintf(
				"%d embedded text track(s) present but none tagged Chinese or eng/en — M1 never treats und as English",
				textTracks),
		}
	}

	return RouteDecision{
		Kind: RouteNoTextSource,
		Reason: fmt.Sprintf(
			"no usable embedded text subtitle track (%d image-codec track(s), 0 text)", imageTracks),
	}
}

// pickBestCandidate parses and SDH-filters every extracted candidate, then picks
// the one with the highest surviving cue count — forced-narrative tracks have
// few cues, and SDH variants converge with full tracks once filtered. Equal cue
// counts break on the content variant (an already-Traditional track beats one
// that would need converting), and finally on the lowest stream index so the
// choice is deterministic. It returns the winner's detected variant so
// SelectAndRoute does not re-run detection.
//
// A candidate that produced no file, cannot be read, or does not parse is logged
// and skipped (Rule 13 — the error informs the fallback, it is never swallowed
// silently). ok is false only when EVERY candidate failed.
func (r *Router) pickBestCandidate(candidates []services.SubtitleTrack, outputs map[int]string) (ExtractedTrack, string, bool) {
	var best ExtractedTrack
	bestVariant := ""
	found := false

	for _, c := range candidates {
		// Defensive against the TrackExtractor port contract: the production
		// Extractor errors out when ANY output file is missing (AC #3.5), so
		// with it this branch is unreachable — it guards alternative
		// implementations that return a partial map instead.
		path, ok := outputs[c.StreamIndex]
		if !ok {
			r.logger.Warn("subtitle candidate skipped", "stream_index", c.StreamIndex, "reason", "no extracted output")
			continue
		}

		raw, err := os.ReadFile(path) //nolint:gosec // path is the extractor's own temp output
		if err != nil {
			r.logger.Warn("subtitle candidate skipped", "stream_index", c.StreamIndex, "reason", "read failed", "error", err)
			continue
		}

		blocks, err := ParseSRT(string(raw))
		if err != nil {
			r.logger.Warn("subtitle candidate skipped", "stream_index", c.StreamIndex, "reason", "parse failed", "error", err)
			continue
		}
		if len(blocks) == 0 {
			r.logger.Warn("subtitle candidate skipped", "stream_index", c.StreamIndex, "reason", "no cues parsed")
			continue
		}

		kept, removed := FilterSDH(blocks)
		r.logger.Debug("subtitle candidate filtered",
			"stream_index", c.StreamIndex, "cues_parsed", len(blocks), "cues_removed", removed, "cues_kept", len(kept))

		candidate := ExtractedTrack{
			StreamIndex: c.StreamIndex,
			Language:    c.Language,
			Codec:       c.Format,
			Path:        path,
			Blocks:      kept,
		}
		variant := Detect([]byte(cueText(kept))).Language

		if !found || betterCandidate(candidate, variant, best, bestVariant) {
			best = candidate
			bestVariant = variant
			found = true
		}
	}

	return best, bestVariant, found
}

// variantRank orders content variants by how much work (and how much loss) the
// delivery still costs: already-Traditional is free, Simplified/mixed pays an
// OpenCC round-trip, and no-CJK pays a whole LLM translation.
func variantRank(variant string) int {
	switch variant {
	case LangTraditional:
		return 0
	case LangSimplified, LangAmbiguous:
		return 1
	default: // LangUndetermined — English (or an unusable track)
		return 2
	}
}

// betterCandidate implements the selection heuristic: more surviving cues wins;
// on a tie the cheaper/lossless variant wins; on a further tie the lower stream
// index wins.
//
// Cue count stays the PRIMARY key on purpose — a forced-narrative track can be
// genuinely Traditional yet carry 20 cues, and it must never beat a full 400-cue
// track. The variant tie-break exists for the case live-observed on the owner's
// NAS (2026-07-31): Apple TV+ files ship Simplified, Traditional and Cantonese
// tracks with IDENTICAL cue counts, so the old cue-count/stream-index pair
// always landed on the lowest index — Simplified — and paid a needless s2twp
// conversion while the official Traditional track sat one stream away.
func betterCandidate(candidate ExtractedTrack, candidateVariant string, current ExtractedTrack, currentVariant string) bool {
	if len(candidate.Blocks) != len(current.Blocks) {
		return len(candidate.Blocks) > len(current.Blocks)
	}
	if r1, r2 := variantRank(candidateVariant), variantRank(currentVariant); r1 != r2 {
		return r1 < r2
	}
	return candidate.StreamIndex < current.StreamIndex
}

// routeForVariant maps the CONTENT-detected variant to a verdict. The ffprobe
// language tag never decides this (FR6) — that is the Bazarr mislabel bug the
// detector exists to fix, which is why detection runs AFTER extraction.
func routeForVariant(variant string, streamIndex int, languageTag string) (RouteKind, string) {
	switch variant {
	case LangTraditional:
		return RouteDeliverDirect, fmt.Sprintf(
			"stream %d tagged %s but its content is Traditional Chinese — deliver as-is", streamIndex, languageTag)
	case LangSimplified:
		return RouteConvertThenDeliver, fmt.Sprintf(
			"stream %d tagged %s but its content is Simplified Chinese — convert then deliver", streamIndex, languageTag)
	case LangAmbiguous:
		// s2twp is idempotent on already-Traditional text, so converting is the
		// safe branch for the 30–70% band (§9b co-production precedent).
		return RouteConvertThenDeliver, fmt.Sprintf(
			"stream %d tagged %s has mixed Chinese content — converting is the safe branch", streamIndex, languageTag)
	case LangUndetermined:
		return RouteTranslate, fmt.Sprintf(
			"stream %d tagged %s carries no CJK content — queue for translation", streamIndex, languageTag)
	default:
		// Detect only returns the four Lang* constants today; if it ever grows a
		// new value, an unknown variant must NOT silently reach the LLM. P0's
		// philosophy is to fail closed rather than mistranslate.
		return RouteSkip, fmt.Sprintf(
			"stream %d tagged %s: detector returned unrecognized variant %q — failing closed (P0)",
			streamIndex, languageTag, variant)
	}
}

// cueText joins the cue text for content detection. Only the dialogue matters:
// indexes and timestamps are ASCII and would tell the detector nothing.
func cueText(blocks []SubtitleBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		sb.WriteString(b.Text)
		sb.WriteByte('\n')
	}
	return sb.String()
}
