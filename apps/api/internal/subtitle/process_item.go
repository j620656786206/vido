package subtitle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
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

	// sub-5-5 AC #5: feed the per-show glossary BEFORE the version is computed
	// — GlossaryVersion hashes exactly what the prompt will carry, so cache key
	// and prompt content agree by construction. A lookup failure feeds empty
	// (Rule 13 case 3, the 9R-10 loadGlossary posture): a glossary miss costs
	// consistency, never the episode.
	p.feedGlossary(ctx, ref, item)

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

	// sub-5-1 AC #3: every paid call under this item runs against a Budget —
	// the translate leg's LLM spend was previously unmetered and uncapped on
	// the FR12/pool path. Mirrors TranscriptionService.resolveBudget: a ctx
	// that ALREADY carries a Budget keeps it (a consent batch's shared ceiling
	// — sub-4-2 — must NEVER be overridden by a per-item envelope); only a
	// budget-less ctx gets the per-item one. A ceiling hit surfaces as
	// ai.ErrBudgetExceeded out of governed() and fails the item (translate leg)
	// or pauses it (ASR leg, the existing transcribeFallback classification).
	//
	// KNOWN LIMITATION (sub-5-1 CR H1, tracked as
	// backlog-translate-budget-partial-progress): a translate-leg ceiling hit
	// discards the chunks already translated (the segment cache is written
	// only after the full track succeeds), so an item whose translation costs
	// more than the ceiling cannot complete via this path and each EXPLICIT
	// retry re-spends up to the ceiling from cue 1. Bounded by consent: since
	// sub-4-1 nothing auto-enqueues generation (scan is metadata-only,
	// repo-guarded), so every retry is a user-triggered FR12/consent action
	// under a visible ceiling — but the waste is real and the fix (partial
	// segment-cache write on budget failure) touches the stamped
	// TranslateTrack contract, hence a story of its own.
	if ai.BudgetFromContext(ctx) == nil {
		ctx = ai.WithBudget(ctx, ai.NewBudget(p.runBudgetUSD))
	}

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
		// sub-3-1: no longer an unconditional recordSkip — the ASR fallback
		// leg runs when the optional port can serve this item, and degrades to
		// today's `no_text_source` record when it cannot (AC #4).
		return p.transcribeFallback(ctx, ref, run, decision, item)
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
	// requests_sent disambiguates the two ways cache_enabled lands on false:
	// requests_sent > 0 means the prompt prefix was sent and silently failed to
	// cache (the AC #4.2 verdict), requests_sent == 0 means nothing was ever
	// sent — a fully segment-cached item, or a route with no LLM leg at all.
	// The column itself is a non-nullable bool and cannot hold that third state
	// (backlog-subtitle-run-cache-enabled-tristate), so the M1 pilot separates
	// the populations from this line.
	p.logger.Info("subtitle pipeline completed",
		"media_id", ref.ID,
		"media_type", ref.MediaType,
		"route", string(decision.Kind),
		"cue_count", cueCount,
		"cache_enabled", run.CacheEnabled,
		"requests_sent", scope.requestsSent,
		// sub-5-5 AC #6: glossary_fed = pairs injected into the prompt;
		// harvested_terms = NEW terms this run wrote back (dedupes excluded).
		"glossary_fed", len(item.Context.Glossary),
		"harvested_terms", scope.harvestedTerms,
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

	// FR16's stubborn ceiling is a property of the DELIVERED track. TranslateTrack
	// only ever sees the miss subset, so without this the ceiling would tighten as
	// the cache warms and one flaky cue would fail an otherwise-complete episode.
	scope := processScopeFrom(ctx)
	if scope != nil {
		scope.fullTrackCues = len(source)
	}

	var translated []SubtitleBlock
	var harvest map[string]string
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
		harvest = res.HarvestedTerms

		final := make(map[int]string, len(res.Blocks))
		for _, b := range res.Blocks {
			final[b.Index] = b.Text
		}

		// A stubborn cue ships its ENGLISH original (NFR-R1 fail-soft). Caching
		// that would freeze a transient failure for the full 30-day TTL — and
		// because the key is cue content + show metadata, the same English line
		// would then be served to every other episode of the show, with no
		// retry ever firing again short of --force or a version bump. Drop them
		// from the write set; they simply cost tokens next run.
		if scope != nil {
			for index := range scope.stubbornIndexes {
				delete(final, index)
			}
		}

		// Keyed on the SOURCE cue text, storing the FINAL post-OpenCC text — a
		// later hit must be byte-identical to a fresh translation.
		p.storeCachedCues(ctx, misses, final, version)

		if res.StubbornCues > 0 {
			p.logger.Warn("subtitle cues kept their English original — excluded from the segment cache",
				"media_id", ref.ID, "stubborn_cues", res.StubbornCues, "total_cues", len(source))
		}
	}

	merged, err := mergeCues(source, hits, translated)
	if err != nil {
		return nil, err
	}

	// The FR17 invariant over the FULL set, not just the translated subset: a
	// partial-cache run is exactly where a merge bug would desync timestamps.
	if err := checkTimestampInvariant(source, merged); err != nil {
		return nil, err
	}

	// sub-5-5 AC #4 回程: the translate stage has fully succeeded — feed the
	// trailer yield back. Cache-hit cues sent no LLM request and yield nothing;
	// that is the DEFINITION of opportunistic harvest, not a gap to fill.
	if harvest != nil {
		p.harvestTerms(ctx, ref, scope, harvest)
	}
	return merged, nil
}

// transcribeFallback is the sub-3-1 ASR leg: a RouteNoTextSource verdict falls
// through to speech recognition instead of dead-ending, finally delivering
// what verdictWithoutTrack's "what P2's ASR can later recover" comment always
// intended.
//
// Degradations (AC #4) — each records `no_text_source` exactly as the pre-M2
// pipeline did, via the same recordSkip path (`RouteSkip` handling is a
// different call site and stays byte-unchanged):
//
//   - port not wired: translate-only construction (sub-1-5a) or an ASR-less
//     deployment — main.go only wires the port in `pipeline` mode, so the
//     legacy flag mode never grows this leg (AC #5);
//   - the service declines (ErrTranscriptionDisabled): no ASR key / no FFmpeg
//     and no translate-only resume. The leg deliberately does NOT pre-probe
//     Available() per item — the service's own entry gate is RICHER (it admits
//     ASR-less translate-only resumes, CR sub-2-2a M2) and must stay
//     authoritative;
//   - a series (or unknown) ref: a series row is a container, not a media
//     file — only movie and episode refs carry transcribable audio. Episodes
//     joined the leg in sub-3-2 (media-type-aware TranscriptionService).
//
// On success the SERVICE is the sole writer of the media-row outcome (found +
// zh sidecar, or untranslated + the EN SRT — 9R-16 AC 12 / sub-2-2a); the leg
// only records its own provenance row, so the two writers can never disagree.
//
// ai.ErrBudgetExceeded is a PAUSE, not a failure (AC #6): the sentinel
// propagates for the caller to classify (the generation-batch precedent), the
// run row closes with the budget reason, and the media row is left exactly as
// the service's last write — reverting it to `not_searched` would destroy the
// `untranslated` resume marker and re-pay the whole ASR on resume (sub-2-2a's
// headline bug class). The ctx-attached ai.Budget threads straight through:
// RunTranscription's resolveBudget reuses a ctx budget when one is present and
// creates the per-run envelope otherwise, so ASR and LLM always share one
// ceiling.
func (p *Pipeline) transcribeFallback(
	ctx context.Context,
	ref MediaRef,
	run *models.SubtitleRun,
	decision RouteDecision,
	item *MediaItem,
) (*ProcessOutcome, error) {
	transcribable := ref.MediaType == models.SubtitleRunMediaMovie || ref.MediaType == models.SubtitleRunMediaEpisode
	if p.asr == nil || !transcribable {
		return p.recordSkip(ctx, ref, run, decision, models.SubtitleStatusNoTextSource)
	}

	// The D6 stage stays inside the existing 12-value set: the media row
	// already reads `extracting`, ASR's first phase IS an ffmpeg audio
	// extraction, and a dedicated stage would bump the stamped D6 contract
	// this story deliberately leaves untouched. The service's job-keyed
	// `transcription_*` events fire during the same window — a DISTINCT
	// contract for the manual Route-C dialog (do NOT deduplicate,
	// project-context.md sub-1-3); automatic-pipeline observers stay on
	// `subtitle_progress`.
	p.emitProgress(ref, StageExtracting, "no embedded text source — transcribing audio (ASR fallback)")

	// Snapshot BEFORE the run (CR sub-3-1 M2): under Force a pre-existing
	// acceptable sidecar survives to this leg (the P5 pre-flight is bypassed),
	// and an `untranslated` outcome leaves it untouched — attributing it to
	// THIS run would record provenance for a file the run never produced,
	// while the media row simultaneously says `untranslated`.
	zhPath := ExpectedSidecarPath(item.FilePath)
	preStat, preErr := os.Stat(zhPath)

	err := p.asr.Transcribe(ctx, ref, item.FilePath, filepath.Dir(item.FilePath))
	switch {
	case err == nil:
		// fall through to the provenance record below
	case errors.Is(err, services.ErrTranscriptionDisabled):
		return p.recordSkip(ctx, ref, run, decision, models.SubtitleStatusNoTextSource)
	case errors.Is(err, ai.ErrBudgetExceeded):
		return p.pauseASRItem(ctx, ref, run, err)
	default:
		return p.failItem(ctx, ref, run, "asr fallback", err)
	}

	completedAt := p.now().UTC()
	run.Status = models.SubtitleRunCompleted
	run.CompletedAt = &completedAt
	outcome := &ProcessOutcome{Run: run, Kind: decision.Kind}
	// The sync seam returns only an error, so the run row records what is
	// verifiable on disk: a parseable zh-Hant sidecar at the exact path this
	// pipeline would have written — and only when this run actually wrote it
	// (new file, or overwritten vs the pre-run snapshot). Its absence is the
	// legitimate untranslated outcome (translation key unconfigured), where
	// the EN path lives on the media row and the run claims no sidecar.
	if cueCount, ok := sidecarCueCount(zhPath); ok && sidecarWrittenSince(zhPath, preStat, preErr) {
		run.OutputPath = zhPath
		run.CueCount = cueCount
		outcome.SubtitlePath = zhPath
	}
	if err := p.runs.Update(ctx, run); err != nil {
		return p.failItem(ctx, ref, run, "record asr provenance", err)
	}

	p.emitProgress(ref, StageComplete, "subtitle generated via ASR fallback")
	p.logger.Info("subtitle pipeline ASR fallback completed",
		"media_id", ref.ID, "media_type", ref.MediaType,
		"output_path", run.OutputPath, "cue_count", run.CueCount)
	return outcome, nil
}

// pauseASRItem closes the provenance row for a budget-ceiling hit WITHOUT
// touching the media row (see transcribeFallback's pause rationale) and
// propagates the sentinel so callers can classify the pause with errors.Is.
func (p *Pipeline) pauseASRItem(ctx context.Context, ref MediaRef, run *models.SubtitleRun, cause error) (*ProcessOutcome, error) {
	err := fmt.Errorf("subtitle pipeline: %s %s: asr fallback: %w", ref.MediaType, ref.ID, cause)

	// Same WithoutCancel rationale as failItem: the row must close even when
	// the ceiling hit rode in on a cancellation.
	cleanupCtx := context.WithoutCancel(ctx)
	completedAt := p.now().UTC()
	run.Status = models.SubtitleRunFailed
	run.ErrorMessage = truncateErrorMessage(err.Error())
	run.CompletedAt = &completedAt
	if uerr := p.runs.Update(cleanupCtx, run); uerr != nil {
		p.logger.Error("failed to record the budget-paused subtitle run",
			"media_id", ref.ID, "run_id", run.ID, "error", uerr)
	}

	p.emitProgress(ref, StageFailed, err.Error())
	p.logger.Info("subtitle pipeline ASR fallback paused on the budget ceiling",
		"media_id", ref.ID, "media_type", ref.MediaType)
	return nil, err
}

// sidecarWrittenSince reports whether the sidecar at path was (re)written
// after the pre-run snapshot: it did not exist before, or its mtime/size
// moved. The placer's atomic replace always advances mtime, so a stale
// pre-Force sidecar left untouched by an `untranslated` outcome is the only
// case that reads false.
func sidecarWrittenSince(path string, pre os.FileInfo, preErr error) bool {
	if preErr != nil {
		return true // did not exist before the run — the run made it
	}
	post, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !post.ModTime().Equal(pre.ModTime()) || post.Size() != pre.Size()
}

// sidecarCueCount reports how many cues a parseable sidecar at path carries.
// (0, false) for a missing, unreadable, unparseable or empty file — the same
// clauses as acceptableSidecar, reduced to the count the run row records.
func sidecarCueCount(path string) (int, bool) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is derived from the media file we were handed
	if err != nil {
		return 0, false
	}
	blocks, err := ParseSRT(string(raw))
	if err != nil || len(blocks) == 0 {
		return 0, false
	}
	return len(blocks), true
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

// glossaryKeyFor is the per-show glossary key (sub-5-5 AC #4): episodes and
// series accumulate under the SERIES id (ShowKey — any episode harvests, every
// episode benefits, in whatever order they are translated); movies carry no
// ShowKey and key on themselves (the spec's intra-film consistency, shared
// with re-translation and the 9R-13 NFO localizer).
func glossaryKeyFor(ref MediaRef, showKey string) string {
	if showKey != "" {
		return showKey
	}
	return ref.ID
}

// feedGlossary fills item.Context.Glossary from the per-show glossary (sub-5-5
// AC #5 去程). Entries are sorted by Source so the rendered prompt — and
// therefore the provider-side prompt cache — is deterministic for a given
// glossary state. Nil-safe and fail-soft: no port or a failed lookup feeds
// empty, which hashes to GlossaryVersion "" and keeps today's behavior.
func (p *Pipeline) feedGlossary(ctx context.Context, ref MediaRef, item *MediaItem) {
	if p.glossary == nil {
		return
	}
	key := glossaryKeyFor(ref, item.ShowKey)
	terms, err := p.glossary.Lookup(ctx, key)
	if err != nil {
		p.logger.Warn("glossary lookup failed — translating without glossary",
			"media_id", ref.ID, "media_type", ref.MediaType, "glossary_key", key, "error", err)
		return
	}
	if len(terms) == 0 {
		return
	}
	entries := make([]prompts.GlossaryEntry, 0, len(terms))
	for src, zh := range terms {
		entries = append(entries, prompts.GlossaryEntry{Source: src, Target: zh})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Source < entries[j].Source })
	item.Context.Glossary = entries
}

// harvestTerms writes the translate stage's trailer yield back to the glossary
// (sub-5-5 AC #4 回程), insert-if-absent only. Runs strictly AFTER the
// translate stage succeeded — a failed or cancelled run harvests nothing,
// because terms that never shipped with a subtitle would only pollute.
// Fail-soft (Rule 13 case 3): the translation already succeeded; a lost
// harvest is re-collected by some future run, never a failed item.
func (p *Pipeline) harvestTerms(ctx context.Context, ref MediaRef, scope *processScope, terms map[string]string) {
	if p.glossary == nil || len(terms) == 0 {
		return
	}
	showKey := ""
	if scope != nil {
		showKey = scope.showKey
	}
	key := glossaryKeyFor(ref, showKey)

	// CR sub-5-5 H2: run the s2twp safety net over each rendering BEFORE it
	// becomes a glossary row. The subtitle itself gets this pass, but a raw
	// harvested rendering does not — and a Simplified one would be fed back as
	// a MANDATORY fixed rendering, which the quality gate then rejects on every
	// future cue containing it: a self-reinforcing poisoning loop that only a
	// manual F6 correction could break. Fail-soft per value: on converter error
	// keep the raw rendering (F6 review still covers it).
	converted := make(map[string]string, len(terms))
	for src, zh := range terms {
		if out, err := p.converter.ConvertS2TWP([]byte(zh)); err == nil {
			converted[src] = string(out)
		} else {
			converted[src] = zh
		}
	}

	inserted, err := p.glossary.InsertNew(ctx, key, converted)
	if err != nil {
		p.logger.Warn("glossary harvest write failed — these terms will be re-collected by a future run",
			"media_id", ref.ID, "media_type", ref.MediaType, "glossary_key", key,
			"harvested", len(terms), "inserted", inserted, "error", err)
	}
	if scope != nil {
		scope.harvestedTerms = inserted
	}
}

// requireItemPorts fails fast and by NAME when sub-1-6's wiring is incomplete.
// A nil port discovered mid-run would panic a worker after the item had already
// been marked `extracting`.
//
// The ASR port (p.asr) is DELIBERATELY absent from this list: unlike the five
// required ports, nil is a legal, supported state (translate-only
// construction, ASR-less deployment — sub-3-1 AC #4) that degrades to the
// pre-M2 `no_text_source` record. Do not "fix" it into the missing-ports
// error.
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
