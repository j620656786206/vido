package subtitle

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vido/api/internal/models"
)

// ProcessItem runs one media item end to end: pre-flight, route, translate (or
// convert, or deliver as-is), place, record provenance.
//
// [@contract-v1] — consumed by sub-1-6, which owns the flag seam, the endpoint
// and the scanner enqueue. Changing the signature, the outcome shape, or the
// status/provenance semantics is a Rule 20 bump plus a downstream stale-mark.
//
// The step order is load-bearing and every step is a test seam:
//
//  1. pre-flight (P5)  — may exit before any ffprobe or LLM spend
//  2. run row pending→running, media row → extracting
//  3. SelectAndRoute   — verdict decides the rest
//  4. deliver → provenance → terminal media status, IN THAT ORDER (P9)
//  5. any failure      — run `failed`, media back to `not_searched` (retryable)
func (p *Pipeline) ProcessItem(ctx context.Context, ref MediaRef, opts ProcessItemOptions) (*ProcessOutcome, error) {
	if err := p.requireItemPorts(); err != nil {
		return nil, err
	}

	item, err := p.media.Load(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("subtitle pipeline: load %s %s: %w", ref.MediaType, ref.ID, err)
	}
	if item == nil || item.FilePath == "" {
		return nil, fmt.Errorf("subtitle pipeline: %s %s has no media file path", ref.MediaType, ref.ID)
	}

	version := p.runVersion(item.Context)

	// ── Step 1: pre-flight (AC #2) ──────────────────────────────────────────
	// Deliberately BEFORE the run row: an early-exit must leave no provenance
	// behind, or every scan would append a row per already-done item.
	if skip, reason := p.preflightSkip(ctx, ref, item.FilePath, version, opts); skip {
		return &ProcessOutcome{SubtitlePath: ExpectedSidecarPath(item.FilePath)}, nil
	} else if reason != "" {
		p.logger.Debug("subtitle pre-flight proceeding", "media_id", ref.ID, "reason", reason)
	}

	// The per-item scope rides the context so the chunk loop inside
	// TranslateTrack can attribute progress and the prompt-cache verdict
	// without changing sub-1-5a's stamped signature.
	scope := &processScope{ref: ref, showKey: item.ShowKey}
	ctx = withProcessScope(ctx, scope)

	// ── Step 2: run row + media status ──────────────────────────────────────
	run := &models.SubtitleRun{
		MediaID:         ref.ID,
		MediaType:       ref.MediaType,
		TMDbID:          item.TMDbID,
		MetadataHash:    version.MetadataHash,
		GlossaryVersion: version.GlossaryVersion,
		PromptVersion:   version.PromptVersion,
		ModelID:         version.ModelID,
		Status:          models.SubtitleRunPending,
		StartedAt:       p.now().UTC(),
	}
	if err := p.runs.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("subtitle pipeline: create run for %s %s: %w", ref.MediaType, ref.ID, err)
	}

	// pending → running is a separate write on purpose: a process killed
	// mid-item leaves an honest `running` row, distinguishable from one that
	// never started.
	run.Status = models.SubtitleRunRunning
	if err := p.runs.Update(ctx, run); err != nil {
		return p.failItem(ctx, ref, run, "mark run running", err)
	}

	p.emitProgress(ref, StageProbing, "probing embedded subtitle tracks")
	// The media row goes straight to `extracting`: the probe is milliseconds
	// and extraction is the long half, so `probing` stays an SSE-only
	// granularity rather than a persisted state nobody would ever observe.
	if err := p.setMediaStatus(ctx, ref, models.SubtitleStatusExtracting, "", ""); err != nil {
		return p.failItem(ctx, ref, run, "mark media extracting", err)
	}

	// ── Step 3: route ───────────────────────────────────────────────────────
	tmpDir, err := os.MkdirTemp("", "vido-subtitle-*")
	if err != nil {
		return p.failItem(ctx, ref, run, "create temp dir", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			p.logger.Warn("failed to remove subtitle temp dir", "dir", tmpDir, "error", err)
		}
	}()

	p.emitProgress(ref, StageExtracting, "extracting embedded subtitle track")
	decision, err := p.router.SelectAndRoute(ctx, item.FilePath, tmpDir)
	if err != nil {
		return p.failItem(ctx, ref, run, "route", err)
	}

	switch decision.Kind {
	case RouteNoTextSource:
		return p.recordSkip(ctx, ref, run, decision, models.SubtitleStatusNoTextSource)
	case RouteSkip:
		return p.recordSkip(ctx, ref, run, decision, models.SubtitleStatusSkipped)
	case RouteDeliverDirect, RouteConvertThenDeliver, RouteTranslate:
		// handled below
	default:
		// Fail closed. sub-1-4's routeForVariant already defaults to RouteSkip
		// for an unknown detector variant, so reaching here means the verdict
		// enum grew a member this switch was never taught — better a loud
		// failed run than silently delivering the wrong thing.
		return p.failItem(ctx, ref, run, "route",
			fmt.Errorf("unrecognized route verdict %q", decision.Kind))
	}

	if decision.Track == nil || len(decision.Track.Blocks) == 0 {
		return p.failItem(ctx, ref, run, "route",
			fmt.Errorf("%w: verdict %q carried no cues", ErrSubtitleExtractFailed, decision.Kind))
	}

	// ── Step 3b: produce the deliverable ────────────────────────────────────
	payload, cueCount, err := p.deliverable(ctx, ref, decision, item, version, opts)
	if err != nil {
		return p.failItem(ctx, ref, run, string(decision.Kind), err)
	}

	// ── Step 4: deliver → provenance → terminal status (P9) ─────────────────
	p.emitProgress(ref, StagePlacing, "placing subtitle file")
	placed, err := p.placer.Place(PlaceRequest{
		MediaFilePath: item.FilePath,
		SubtitleData:  payload,
		Language:      deliveredLanguage,
		Format:        deliveredFormat,
		// Score stays 0 so the repository writes NULL: this file was generated,
		// not scored against provider results (AC #6.3).
		Score: 0,
	})
	if err != nil {
		return p.failItem(ctx, ref, run, "place", err)
	}

	// Provenance strictly AFTER a confirmed place (P9). The reverse order
	// leaves a `completed` row pointing at a file a crash never wrote.
	completedAt := p.now().UTC()
	run.Status = models.SubtitleRunCompleted
	run.OutputPath = placed.SubtitlePath
	run.CueCount = cueCount
	run.SourceLanguage = sourceLanguageOf(decision)
	run.CacheEnabled = scope.cacheEnabled
	run.CompletedAt = &completedAt
	if err := p.runs.Update(ctx, run); err != nil {
		// The sidecar IS on disk. Reverting the media row to `not_searched`
		// keeps the item retryable, and the next run's P5 pre-flight will see
		// the acceptable sidecar and early-exit — so the inconsistency
		// self-heals rather than costing another translation.
		return p.failItem(ctx, ref, run, "record provenance", err)
	}

	if err := p.setMediaStatus(ctx, ref, models.SubtitleStatusFound, placed.SubtitlePath, deliveredLanguage); err != nil {
		return p.failItem(ctx, ref, run, "mark media found", err)
	}

	p.emitProgress(ref, StageComplete, "subtitle generated")
	p.logger.Info("subtitle pipeline completed",
		"media_id", ref.ID,
		"media_type", ref.MediaType,
		"route", string(decision.Kind),
		"cue_count", cueCount,
		"cache_enabled", run.CacheEnabled,
		"output_path", placed.SubtitlePath,
	)

	return &ProcessOutcome{Run: run, Kind: decision.Kind, SubtitlePath: placed.SubtitlePath}, nil
}

// deliverable turns a verdict into the exact bytes placer.Place will write,
// plus the cue count recorded as provenance.
func (p *Pipeline) deliverable(
	ctx context.Context,
	ref MediaRef,
	decision RouteDecision,
	item *MediaItem,
	version models.RunVersion,
	opts ProcessItemOptions,
) ([]byte, int, error) {
	source := decision.Track.Blocks

	switch decision.Kind {
	case RouteDeliverDirect:
		// FR7 — the content is already Traditional; the only thing wrong with
		// it was the ffprobe language tag.
		return []byte(SerializeSRT(source)), len(source), nil

	case RouteConvertThenDeliver:
		// FR8 — one deterministic s2twp pass over the serialized track. Unlike
		// the translate path (where the quality gate has already guaranteed no
		// Simplified leak and OpenCC is polish), conversion IS the deliverable
		// here: shipping unconverted text under a .zh-Hant.srt name would be a
		// lie, so a converter failure fails the item.
		converted, err := p.converter.ConvertS2TWP([]byte(SerializeSRT(source)))
		if err != nil {
			return nil, 0, fmt.Errorf("opencc s2twp on the routed track: %w", err)
		}
		return converted, len(source), nil

	case RouteTranslate:
		blocks, err := p.translateWithCache(ctx, ref, decision.Track, item.Context, version, opts)
		if err != nil {
			return nil, 0, err
		}
		return []byte(SerializeSRT(blocks)), len(blocks), nil

	default:
		return nil, 0, fmt.Errorf("unreachable verdict %q", decision.Kind)
	}
}

// translateWithCache is the FR10 path: split the track against the segment
// cache, translate only the misses, write them back, and merge into the full
// track.
func (p *Pipeline) translateWithCache(
	ctx context.Context,
	ref MediaRef,
	track *ExtractedTrack,
	tctx TranslateContext,
	version models.RunVersion,
	opts ProcessItemOptions,
) ([]SubtitleBlock, error) {
	if err := p.setMediaStatus(ctx, ref, models.SubtitleStatusTranslating, "", ""); err != nil {
		return nil, fmt.Errorf("mark media translating: %w", err)
	}

	source := track.Blocks
	hits, misses := p.splitCachedCues(ctx, source, version, opts.Force)
	p.logger.Info("segment cache split",
		"media_id", ref.ID, "cache_hits", len(hits), "cache_misses", len(misses), "force", opts.Force)

	var translated []SubtitleBlock
	if len(misses) > 0 {
		// A REDUCED copy of the routed track. The stamped TranslateTrack
		// contract takes a track and translates every cue in it, so handing it
		// only the misses needs no contract change — and the cues keep their
		// original Index values, which is what lets the merge below reassemble
		// the full track (P1/P7).
		reduced := *track
		reduced.Blocks = misses

		res, err := p.TranslateTrack(ctx, &reduced, tctx)
		if err != nil {
			return nil, err
		}
		translated = res.Blocks

		final := make(map[int]string, len(res.Blocks))
		for _, b := range res.Blocks {
			final[b.Index] = b.Text
		}
		// Keyed on the SOURCE cue text, storing the FINAL post-OpenCC text — a
		// later hit must be byte-identical to a fresh translation.
		p.storeCachedCues(ctx, misses, final, version)

		if res.StubbornCues > 0 {
			p.logger.Warn("subtitle cues kept their English original",
				"media_id", ref.ID, "stubborn_cues", res.StubbornCues, "total_cues", len(source))
		}
	}

	merged := mergeCues(source, hits, translated)

	// The FR17 invariant over the FULL set, not just the translated subset: a
	// partial-cache run is exactly where a merge bug would desync timestamps.
	if err := checkTimestampInvariant(source, merged); err != nil {
		return nil, err
	}
	return merged, nil
}

// recordSkip writes the two terminal rows for a routed-out item: the run is
// `skipped` with the router's reason, and the media row takes the status that
// distinguishes "nothing to work with" (recoverable by P2 ASR) from "we
// deliberately declined" (P0).
func (p *Pipeline) recordSkip(
	ctx context.Context,
	ref MediaRef,
	run *models.SubtitleRun,
	decision RouteDecision,
	status models.SubtitleStatus,
) (*ProcessOutcome, error) {
	completedAt := p.now().UTC()
	run.Status = models.SubtitleRunSkipped
	// error_message is the only free-text column on the row; for a skipped run
	// it carries the routing reason, not a failure.
	run.ErrorMessage = decision.Reason
	run.CompletedAt = &completedAt

	if err := p.runs.Update(ctx, run); err != nil {
		return p.failItem(ctx, ref, run, "record skip", err)
	}
	if err := p.setMediaStatus(ctx, ref, status, "", ""); err != nil {
		return p.failItem(ctx, ref, run, "mark media "+string(status), err)
	}

	p.emitProgress(ref, StageSkipped, decision.Reason)
	p.logger.Info("subtitle pipeline skipped item",
		"media_id", ref.ID, "media_type", ref.MediaType,
		"route", string(decision.Kind), "reason", decision.Reason)

	return &ProcessOutcome{Run: run, Kind: decision.Kind}, nil
}

// failItem is the single failure policy (AC #1.5): the run row records what
// went wrong, and the media row reverts to `not_searched`.
//
// `not_searched` rather than a dedicated failed state: SubtitleStatus has no
// `failed` member, leaving the row at `translating` would strand a phantom
// in-flight state across a restart, and `not_found` would lie (nothing was
// searched). `not_searched` + the `failed` run row = retryable with a full
// audit trail. Enumeration is scan- or operator-triggered, so a permanently
// failing item cannot hot-loop.
func (p *Pipeline) failItem(ctx context.Context, ref MediaRef, run *models.SubtitleRun, stage string, cause error) (*ProcessOutcome, error) {
	err := fmt.Errorf("subtitle pipeline: %s %s: %s: %w", ref.MediaType, ref.ID, stage, cause)

	// The cleanup writes must survive the cancellation that may have caused the
	// failure — otherwise a shutdown strands every in-flight item at
	// `translating` with no `failed` run row explaining why.
	cleanupCtx := context.WithoutCancel(ctx)

	if run != nil && run.ID != "" {
		completedAt := p.now().UTC()
		run.Status = models.SubtitleRunFailed
		// Diagnostic English + the SUBTITLE_ sentinel. The zh-TW Rule 3
		// envelope is composed at the handler, which sub-1-6 owns.
		run.ErrorMessage = truncateErrorMessage(err.Error())
		run.CompletedAt = &completedAt
		if uerr := p.runs.Update(cleanupCtx, run); uerr != nil {
			p.logger.Error("failed to record the failed subtitle run",
				"media_id", ref.ID, "run_id", run.ID, "error", uerr)
		}
	}

	if serr := p.setMediaStatus(cleanupCtx, ref, models.SubtitleStatusNotSearched, "", ""); serr != nil {
		p.logger.Error("failed to revert media subtitle status after a failed run",
			"media_id", ref.ID, "media_type", ref.MediaType, "error", serr)
	}

	p.emitProgress(ref, StageFailed, err.Error())
	p.logger.Error("subtitle pipeline failed", "media_id", ref.ID, "media_type", ref.MediaType, "stage", stage, "error", err)
	return nil, err
}

// maxErrorMessage bounds what lands in the error_message column: a wrapped
// ffmpeg stderr tail can run long, and provenance is for diagnosis, not
// archival.
const maxErrorMessage = 1000

func truncateErrorMessage(msg string) string {
	if len(msg) <= maxErrorMessage {
		return msg
	}
	// Cut on a rune boundary so the column never stores a broken UTF-8 tail.
	cut := maxErrorMessage
	for cut > 0 && !isRuneStart(msg[cut]) {
		cut--
	}
	return msg[:cut] + "…"
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }

// setMediaStatus guards every media-row write with sub-1-2's IsValid: this is
// the first code path that writes the five new enum values, and a typo would
// otherwise reach a column with no DB CHECK behind it (sub-1-2 finding 1).
func (p *Pipeline) setMediaStatus(ctx context.Context, ref MediaRef, status models.SubtitleStatus, subtitlePath, language string) error {
	if !status.IsValid() {
		return fmt.Errorf("refusing to write unknown subtitle status %q for %s %s", status, ref.MediaType, ref.ID)
	}
	if err := p.media.SetSubtitleStatus(ctx, ref, status, subtitlePath, language); err != nil {
		return fmt.Errorf("set subtitle status %s: %w", status, err)
	}
	return nil
}

// sourceLanguageOf records what the SOURCE track actually was. For a translate
// run that is the ffprobe tag that admitted it (eng/en — the detector found no
// CJK, so the variant says nothing useful); for the two Chinese routes it is
// the CONTENT-detected variant, which is the whole point of FR6/FR7 — the tag
// on those tracks is usually a lie.
func sourceLanguageOf(decision RouteDecision) string {
	if decision.Track == nil {
		return ""
	}
	if decision.Kind == RouteTranslate {
		return decision.Track.Language
	}
	if decision.DetectedVariant != "" {
		return decision.DetectedVariant
	}
	return decision.Track.Language
}

// requireItemPorts fails fast and by NAME when sub-1-6's wiring is incomplete.
// A nil port discovered mid-run would panic a worker after the item had already
// been marked `extracting`.
func (p *Pipeline) requireItemPorts() error {
	var missing []string
	if p.media == nil {
		missing = append(missing, "MediaStore (WithMediaStore)")
	}
	if p.runs == nil {
		missing = append(missing, "RunStore (WithRunStore)")
	}
	if p.router == nil {
		missing = append(missing, "TrackRouter (WithRouter)")
	}
	if p.placer == nil {
		missing = append(missing, "SubtitlePlacer (WithPlacer)")
	}
	if p.cache == nil {
		missing = append(missing, "SegmentCache (WithSegmentCache)")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("subtitle pipeline: ProcessItem is not wired — missing %s", strings.Join(missing, ", "))
}
