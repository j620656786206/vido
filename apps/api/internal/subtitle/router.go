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
// [@contract-v2] — consumed by sub-1-5a (RouteTranslate), sub-1-5b (delivery
// of RouteDeliverDirect/RouteConvertThenDeliver + terminal status writes),
// sub-1-6 (orchestration). Changing kinds/fields = Rule 20 bump + stale-mark.
// v1→v2 (disc-2026-10-english-track-selection): on RouteTranslate,
// ExtractedTrack.Blocks may also carry the file's forced-track cues, numbered
// above the main track's highest Index — types unchanged; Blocks is no longer
// guaranteed to come from the single stream at Path.
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
	Blocks      []SubtitleBlock // parsed + SDH-filtered cues (original numbering — P7); on RouteTranslate also merged forced cues, Index above the main track's max, time-ordered
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

	parsed := r.parseCandidates(candidates, outputs)
	if len(parsed) == 0 {
		return RouteDecision{}, fmt.Errorf(
			"%w: no candidate track of %s could be parsed (%d attempted)",
			ErrSubtitleExtractFailed, mediaPath, len(candidates))
	}
	var durationSeconds float64
	if info != nil {
		durationSeconds = info.DurationSeconds
	}
	main, qualified := r.chooseMain(mediaPath, parsed, durationSeconds)
	best, variant := main.track, main.variant

	if len(best.Blocks) == 0 {
		// FR5's word is *usable*: a track whose every cue was an SDH annotation
		// carries no dialogue to translate or deliver.
		return RouteDecision{
			Kind:   RouteNoTextSource,
			Reason: fmt.Sprintf("stream %d parsed but no cue survived SDH filtering", best.StreamIndex),
		}, nil
	}

	kind, reason := routeForVariant(variant, best.StreamIndex, best.Language)

	// The forced track rides along only on the English translate route: a
	// Chinese forced track may be Simplified under a Traditional main track,
	// and mixing the two needs its own conversion rules.
	// A fallback main that is itself the forced track never merges: there is
	// no full track to fold it into.
	if kind == RouteTranslate && main.kind != trackForced {
		var merged, dropped int
		var skipped []int
		best.Blocks, merged, dropped, skipped = mergeForcedCues(best.Blocks, best.StreamIndex, parsed)
		if merged > 0 || dropped > 0 {
			r.logger.Info("forced subtitle cues merged into the translate track",
				"media", mediaPath, "stream_index", best.StreamIndex,
				"forced_merged", merged, "forced_duplicates_dropped", dropped)
		}
		if len(skipped) > 0 {
			r.logger.Warn("forced-flagged track is as long as the main track — not merged (a full track marked forced)",
				"media", mediaPath, "stream_index", best.StreamIndex, "skipped_streams", skipped)
		}
	}

	r.logger.Debug("subtitle route decided",
		"media", mediaPath,
		"stream_index", best.StreamIndex,
		"track_kind", main.kind.String(),
		"qualified_main", qualified,
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

// parsedCandidate is one extracted track after parsing, SDH filtering and
// content detection, plus what the muxer says the track is.
type parsedCandidate struct {
	track   ExtractedTrack
	variant string
	kind    trackKind
}

// parseCandidates parses, SDH-filters and content-detects every extracted
// candidate, in probe order.
//
// A candidate that produced no file, cannot be read, or does not parse is logged
// and skipped (Rule 13 — the error informs the fallback, it is never swallowed
// silently). The result is empty only when EVERY candidate failed.
func (r *Router) parseCandidates(candidates []services.SubtitleTrack, outputs map[int]string) []parsedCandidate {
	parsed := make([]parsedCandidate, 0, len(candidates))

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
		kind := classifyTrack(c)
		r.logger.Debug("subtitle candidate filtered",
			"stream_index", c.StreamIndex, "kind", kind.String(), "title", c.Title,
			"cues_parsed", len(blocks), "cues_removed", removed, "cues_kept", len(kept))

		parsed = append(parsed, parsedCandidate{
			track: ExtractedTrack{
				StreamIndex: c.StreamIndex,
				Language:    c.Language,
				Codec:       c.Format,
				Path:        path,
				Blocks:      kept,
			},
			variant: Detect([]byte(cueText(kept))).Language,
			kind:    kind,
		})
	}

	return parsed
}

// chooseMain picks the track to deliver or translate (disc-2026-10-english-
// track-selection). A track may be the main one only when it is not forced,
// keeps at least half the cues of the fullest candidate (an unlabelled forced
// track is still a forced track), and — when the file's length is known —
// runs past half of it. Among those, a regular track beats an SDH one, then
// the original cue-count / variant / stream-index order decides.
//
// The cue-count-first order alone (sub-1-4 AC #5) assumed SDH tracks
// "converge with full tracks once filtered"; they do not — SDH splits lines,
// so a filtered SDH track can out-count the regular one and win. And it threw
// away the forced track, which is where on-screen text lives (See S01E02:
// "We are not alone" exists nowhere else in the file).
//
// qualified is false when no candidate passed and the original pick was used
// instead: the verdict is then never worse than before this rule existed.
func (r *Router) chooseMain(mediaPath string, parsed []parsedCandidate, durationSeconds float64) (main parsedCandidate, qualified bool) {
	maxCues := 0
	for _, p := range parsed {
		maxCues = max(maxCues, len(p.track.Blocks))
	}

	var eligible []parsedCandidate
	var rejected []string
	if maxCues > 0 {
		for _, p := range parsed {
			switch {
			case p.kind == trackForced:
				rejected = append(rejected, fmt.Sprintf("stream %d: forced", p.track.StreamIndex))
			case len(p.track.Blocks)*2 < maxCues:
				rejected = append(rejected, fmt.Sprintf("stream %d: %d of %d cues", p.track.StreamIndex, len(p.track.Blocks), maxCues))
			case !coversHalf(p.track.Blocks, durationSeconds):
				rejected = append(rejected, fmt.Sprintf("stream %d: ends before half of %.0fs", p.track.StreamIndex, durationSeconds))
			default:
				eligible = append(eligible, p)
			}
		}
	}

	if len(eligible) == 0 {
		if maxCues > 0 {
			r.logger.Warn("no subtitle track qualifies as the main track — falling back to the cue-count pick",
				"media", mediaPath, "candidates", len(parsed), "rejected", strings.Join(rejected, "; "))
		}
		best := parsed[0]
		for _, p := range parsed[1:] {
			if betterCandidate(p.track, p.variant, best.track, best.variant) {
				best = p
			}
		}
		return best, false
	}

	best := eligible[0]
	for _, p := range eligible[1:] {
		// Regular-over-SDH is an English-tier rule. Between Chinese tracks the
		// script decides first (the 2026-07-31 Apple TV+ ruling): a Traditional
		// SDH track still beats a Simplified regular one that needs converting.
		if p.kind != best.kind && p.variant == LangUndetermined && best.variant == LangUndetermined {
			if p.kind < best.kind {
				best = p
			}
			continue
		}
		if betterCandidate(p.track, p.variant, best.track, best.variant) {
			best = p
		}
	}
	return best, true
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
