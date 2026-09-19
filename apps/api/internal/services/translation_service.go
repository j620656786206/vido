package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/segkey"
	"github.com/vido/api/internal/sse"
)

// SSE event type for translation progress (AC #6)
const (
	EventTranslationProgress sse.EventType = "translation_progress"
)

// Translation constants
const (
	// TranslationMaxTokens is the max response tokens per batch (10 blocks × ~100 chars).
	//
	// It is ALSO the input to the request deadline: the ai layer derives each
	// attempt's timeout from the model family plus this figure
	// (ai.RequestTimeoutFor, sub-6-2 AC #1/#2). This service deliberately
	// wraps NO deadline of its own around a call — the 60 s TranslationTimeout
	// it used to stack on top of the client's 15 s could never fire first, and
	// two deadlines on one request is the D8 bug class where the shorter one
	// wins silently. One bound, owned by the layer that knows the model.
	TranslationMaxTokens = 4096
)

// TranslationBlock mirrors subtitle.SubtitleBlock. services ↛ subtitle — see project-context.md Rule 19.
type TranslationBlock struct {
	Index int
	Start string
	End   string
	Text  string
}

// GlossaryPair is one proper-noun mapping (source term → fixed zh rendering)
// carried into a translation so it renders consistently (Story 9R-7 keystone).
type GlossaryPair struct {
	Source string
	Target string
}

// TranslateOption configures optional translation behavior (Story 9R-8).
//
// VARIADIC on purpose, mirroring TranscriptionOption in this same package: a
// positional parameter would rewrite every existing TranslateWithGlossary /
// TranslateWithGlossaryHarvest call site and their tests for a feature only one
// caller uses, while a fourth TranslateWithGlossaryXxx method would extend an
// already three-deep naming chain. Zero options = the pre-9R-8 behavior.
type TranslateOption func(*translateConfig)

type translateConfig struct {
	metadata prompts.MediaMetadata
	level    prompts.LocalizationLevel
	// store + version are the resume seam
	// (disc-2026-09-generation-resume-a-translation-cache). nil store = the
	// pre-story behaviour, byte for byte.
	store   SegmentStore
	version models.RunVersion
}

// WithLocalizationLevel selects the sub-7-4 style section for this run. The
// zero value is the default level.
func WithLocalizationLevel(level prompts.LocalizationLevel) TranslateOption {
	return func(cfg *translateConfig) { cfg.level = level }
}

// WithMediaMetadata attaches the FR26 show context (title, year, genres,
// overview, countries) to a translation, so the model renders character names,
// register and setting consistently across the whole subtitle. The zero value
// renders nothing — see composeSystemPrompt.
func WithMediaMetadata(md prompts.MediaMetadata) TranslateOption {
	return func(cfg *translateConfig) { cfg.metadata = md }
}

// WithSegmentCache makes this translation resumable: every cue already
// translated under this exact RunVersion is served from the store instead of
// the model, and every batch the model answers is written back at once
// (disc-2026-09-generation-resume-a-translation-cache AC #4).
//
// The version is what makes a hit SAFE to serve — change the metadata, the
// glossary, the prompt or the model and the key changes, so the cue is
// re-translated rather than answered with a rendering that no longer matches
// what was asked. A nil store leaves every existing caller untouched.
func WithSegmentCache(store SegmentStore, version models.RunVersion) TranslateOption {
	return func(cfg *translateConfig) {
		cfg.store = store
		cfg.version = version
	}
}

func newTranslateConfig(opts []TranslateOption) *translateConfig {
	cfg := &translateConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// composeSystemPrompt puts the FR26 media-context section behind the invariant
// translator prompt — the same stable-first order the extract leg's
// buildSystemBlocks uses (subtitle/pipeline.go: block[0] invariant, block[1]
// per-show), so a future prompt-caching breakpoint lands on the same prefix.
//
// P11 — DELIBERATELY NO PromptVersion BUMP (Story 9R-8). The bump rule exists so
// a changed prompt cannot silently serve a cached translation of the OLD prompt.
// Neither half of that risk applies here:
//   - this composes the already-shipped BuildMetadataSection at the call site
//     and edits NO text in prompts/subtitle_translator.go, so the P11 pin digest
//     is unchanged;
//   - a bump WOULD re-key BOTH legs' whole cached library (RunVersion embeds
//     the prompt version) to re-translate for zero gain. Since
//     disc-2026-09-generation-resume-a this leg has the same per-cue segment
//     cache the extract leg has, keyed by the same function
//     (`internal/segkey`), so the cost of a needless bump is now doubled.
//
// TestSubtitleTranslatorPromptVersion_NotBumpedBy9R8 pins the decision.
//
// KNOWN DIVERGENCE (CR L1): on the extract leg the metadata AND the glossary
// share one system block; here the metadata rides the system prompt while the
// glossary stays where 9R-7 put it — inside the per-batch USER prompt. Both
// reach the model, so this is a structural difference rather than a behavioural
// one, but the two legs should converge when the ASR path is folded into the
// gated TranslateTrack (sprint-status `backlog-asr-leg-unify-gated-pipeline`).
func composeSystemPrompt(md prompts.MediaMetadata, level prompts.LocalizationLevel) string {
	// sub-7-4: the invariant prefix (translator prompt + localization style +
	// global lexicon terms) is the same text the extract leg puts in
	// block[0]; the per-show media context follows it.
	invariant := prompts.ComposeInvariantSystemPrompt(level)
	section := prompts.BuildMetadataSection(md)
	if section == "" {
		return invariant
	}
	return invariant + "\n\n" + section
}

// TranslationField is one arbitrary keyed piece of text to translate. Key is a
// stable identifier the caller uses to match the result back (a subtitle block
// index as a string, or a metadata field name like "plot"/"title"). This is the
// generic unit the same engine translates for BOTH subtitles (9R-7) and .nfo
// metadata localization (9R-13).
type TranslationField struct {
	Key  string
	Text string
}

// TranslationRequest is the generalized translation input (Story 9R-7): a set
// of fields to translate plus a glossary of fixed proper-noun renderings. The
// subtitle path builds one internally per batch; 9R-13 metadata localization
// reuses TranslateRequest directly.
type TranslationRequest struct {
	Fields   []TranslationField
	Glossary []GlossaryPair
}

func toPromptGlossary(pairs []GlossaryPair) []prompts.GlossaryEntry {
	if len(pairs) == 0 {
		return nil
	}
	out := make([]prompts.GlossaryEntry, 0, len(pairs))
	for _, p := range pairs {
		if strings.TrimSpace(p.Source) == "" || strings.TrimSpace(p.Target) == "" {
			continue
		}
		out = append(out, prompts.GlossaryEntry{Source: p.Source, Target: p.Target})
	}
	return out
}

// modelNamer is the OPTIONAL seam a provider implements to say which model it
// would actually dispatch to (`*ClaudeProviderHolder` does). The
// ai.DetailedTranscriber precedent: a provider without it is not an error, it
// simply falls through to the deployment default.
type modelNamer interface {
	EffectiveModel() string
}

// TranslationService uses Claude API to translate English subtitles to Traditional Chinese.
type TranslationService struct {
	provider ai.TextCompleter
	sseHub   *sse.Hub
}

// NewTranslationService creates a new translation service.
// Returns nil if provider is nil (graceful degradation per AC #4).
func NewTranslationService(provider ai.TextCompleter, sseHub *sse.Hub) *TranslationService {
	if provider == nil {
		slog.Info("Translation service not configured - no AI provider")
		return nil
	}

	slog.Info("Translation service initialized")
	return &TranslationService{
		provider: provider,
		sseHub:   sseHub,
	}
}

// IsConfigured returns true if Claude API key is available for translation.
func (s *TranslationService) IsConfigured() bool {
	if s == nil || s.provider == nil {
		return false
	}
	// A provider may be a LAZY holder whose key is resolved per call (sub-2-1a
	// AC #2). Since sub-2-1a these services are constructed UNCONDITIONALLY, so
	// "the provider object exists" no longer implies "a key is configured" —
	// without this probe a keyless install would report configured, attempt the
	// work, and fail with ErrAINotConfigured where it used to skip cleanly.
	// A plain provider (Gemini, a direct Claude client) has no probe and stays
	// configured, exactly as before.
	if probe, ok := s.provider.(interface {
		IsConfigured(ctx context.Context) bool
	}); ok {
		return probe.IsConfigured(context.Background())
	}
	return true
}

// EffectiveModelID names the model this run's translations will be attributed
// to — the ModelID half of models.RunVersion, and therefore part of every
// segment-cache key (disc-2026-09-generation-resume-a AC #3).
//
// It is never empty, and that is the point: if the default-model runs keyed on
// "" while the same model picked explicitly keyed on its id, the two
// populations would never share a cached cue and every hit rate would silently
// halve. Three sources, most specific first:
//  1. the model the user picked for this run (ai.WithModelID on the ctx);
//  2. the model the provider would actually dispatch to (a holder honouring
//     CLAUDE_MODEL knows this; a plain client does not implement the seam);
//  3. the deployment default — the same fallback the extract leg's
//     currentModelID uses.
func (s *TranslationService) EffectiveModelID(ctx context.Context) string {
	if id := ai.ModelIDFromContext(ctx); id != "" {
		return id
	}
	if s != nil && s.provider != nil {
		if namer, ok := s.provider.(modelNamer); ok {
			if id := namer.EffectiveModel(); id != "" {
				return id
			}
		}
	}
	return ai.DefaultClaudeModel
}

// Translate translates subtitle blocks from English to Traditional Chinese.
// progressFn (optional) receives percentage updates per batch.
// On partial failure, translated blocks are returned with untranslated blocks
// retaining their original English text (AC #5).
// Translate translates English subtitle blocks to Traditional Chinese with no
// glossary (back-compat entry point; existing callers unchanged).
func (s *TranslationService) Translate(ctx context.Context, blocks []TranslationBlock, progressFn func(float64)) ([]TranslationBlock, error) {
	// Outcome deliberately dropped HERE ONLY (bugfix-j): the sole production
	// caller is the route-c POC cmd, which never persists a verdict. Verdict
	// paths MUST consume TranslateWithGlossaryHarvest's TranslationOutcome.
	translated, _, err := s.TranslateWithGlossary(ctx, blocks, nil, progressFn)
	return translated, err
}

// TranslationOutcome reports translation-fidelity facts the caller must not
// silently drop (bugfix-j): a batch that errors keeps English text wholesale
// (AC #5 of 9-2b) and a response missing a block keeps English silently — both
// used to vanish into a slog.Warn while the verdict recorded found/zh-Hant.
// EnglishKeptBlocks counts every cue that still carries the English source.
type TranslationOutcome struct {
	EnglishKeptBlocks int
	TotalBlocks       int
}

// Partial reports whether any cue kept its English text — the DISCLOSURE
// signal (bugfix-j): it drives the SSE partial flag and the english_kept
// log field unconditionally. Whether the verdict demotes is a separate,
// thresholded question — see DemotesVerdict.
func (o TranslationOutcome) Partial() bool { return o.EnglishKeptBlocks > 0 }

// bugfix-j H3 ruling (Alexyu 2026-08-20, option B「門檻制」): disclosure is
// unconditional, but DEMOTION to `untranslated` fires only when the English
// residue is material:
//   - ≥ a whole batch's worth of cues (a batch failure leaves a contiguous
//     English RUN — unwatchable regardless of file size), or
//   - ≥5% of the ITEM'S OWN cue count. Per the ruling, the percentage
//     denominator is this movie's/episode's total cue count (TotalBlocks of
//     this run), the numerator its untranslated English cues.
//
// Below both bars the verdict stays `found` (zh file placed, scattered EN
// cues disclosed via Partial) so a 1-cue miss doesn't force a full re-run.
const (
	demoteAbsoluteKeptBlocks = prompts.SubtitleTranslatorBatchSize
	demoteKeptPercent        = 5
)

// DemotesVerdict reports whether the English residue is material enough to
// persist `untranslated` instead of `found` (bugfix-j H3 ruling, option B).
func (o TranslationOutcome) DemotesVerdict() bool {
	if o.EnglishKeptBlocks == 0 {
		return false
	}
	if o.EnglishKeptBlocks >= demoteAbsoluteKeptBlocks {
		return true
	}
	// Integer cross-multiplication: kept/total ≥ 5% without float drift.
	return o.EnglishKeptBlocks*100 >= o.TotalBlocks*demoteKeptPercent
}

// TranslateWithGlossary is Translate plus a per-show glossary (Story 9R-7): the
// fixed renderings are injected into every batch prompt so proper nouns stay
// consistent across the whole subtitle and across runs. A nil glossary makes
// this byte-identical to the pre-9R-7 behavior.
//
// It delegates to TranslateWithGlossaryHarvest and drops the harvested terms —
// the Translate → TranslateWithGlossary precedent (9R-7). The partial-failure
// outcome is NOT dropped (bugfix-j AC #1 red line): it passes through so no
// wrapper caller can record a lying verdict.
func (s *TranslationService) TranslateWithGlossary(ctx context.Context, blocks []TranslationBlock, glossary []GlossaryPair, progressFn func(float64), opts ...TranslateOption) ([]TranslationBlock, TranslationOutcome, error) {
	translated, _, outcome, err := s.TranslateWithGlossaryHarvest(ctx, blocks, glossary, progressFn, opts...)
	return translated, outcome, err
}

// splitCachedBlocks is the ONE batched read that decides what this run has to
// pay for (disc-2026-09-generation-resume-a AC #4).
//
// Hits are written straight into `result`, so they serve double duty: they are
// the finished translation AND the context the first fresh batch reads. Misses
// come back as SOURCE POSITIONS in source order, which is what lets the batch
// loop keep `result`, the progress count and the cache keys aligned without a
// second lookup table.
//
// Every failure here is a miss, never an error: a cache that cannot be read
// costs tokens, and failing a paid run over it would be strictly worse than
// paying again (Rule 13 case 3).
func (s *TranslationService) splitCachedBlocks(ctx context.Context, blocks, result []TranslationBlock, cfg *translateConfig) ([]string, []int) {
	pending := make([]int, 0, len(blocks))
	if cfg.store == nil {
		for i := range blocks {
			pending = append(pending, i)
		}
		return nil, pending
	}

	keys := make([]string, len(blocks))
	for i, b := range blocks {
		keys[i] = segkey.SegmentKey(b.Text, cfg.version)
	}

	values, err := cfg.store.GetMany(ctx, keys)
	if err != nil {
		slog.Warn("segment cache read failed — translating every cue",
			"cue_count", len(blocks), "error", err)
		for i := range blocks {
			pending = append(pending, i)
		}
		return keys, pending
	}

	hits := 0
	for i := range blocks {
		if text, ok := values[keys[i]]; ok {
			result[i].Text = text
			hits++
			continue
		}
		pending = append(pending, i)
	}
	slog.Info("segment cache", "hits", hits, "total", len(blocks), "to_translate", len(pending))
	return keys, pending
}

// TranslateWithGlossaryHarvest is TranslateWithGlossary plus the sub-5-5 AC #3
// harvest return: the proper-noun renderings the model reported in each batch's
// trailer, merged across batches with the FIRST occurrence winning (the model's
// later hindsight does not overturn an established rendering — the glossary's
// fixed-rendering spirit). nil terms = nothing harvested.
func (s *TranslationService) TranslateWithGlossaryHarvest(ctx context.Context, blocks []TranslationBlock, glossary []GlossaryPair, progressFn func(float64), opts ...TranslateOption) ([]TranslationBlock, map[string]string, TranslationOutcome, error) {
	if len(blocks) == 0 {
		return nil, nil, TranslationOutcome{}, nil
	}
	promptGlossary := toPromptGlossary(glossary)
	// 9R-8: composed ONCE — the media context is per-run, not per-batch.
	cfg := newTranslateConfig(opts)
	systemPrompt := composeSystemPrompt(cfg.metadata, cfg.level)

	batchSize := prompts.SubtitleTranslatorBatchSize
	contextWindow := prompts.SubtitleTranslatorContextWindow

	// Copy blocks to result (will be modified in place)
	result := make([]TranslationBlock, len(blocks))
	copy(result, blocks)

	totalBlocks := len(blocks)
	processedBlocks := 0
	// bugfix-j: count every cue that keeps English (failed batches + per-cue
	// misses) instead of a dropped bool — the caller persists the verdict.
	englishKept := 0
	var harvested map[string]string

	// disc-2026-09-generation-resume-a: one batched read decides which cues an
	// earlier run already paid for. `pending` is the source positions that
	// still need the model, in source order; with no store it is simply every
	// position, which keeps the pre-story path byte-identical.
	keys, pending := s.splitCachedBlocks(ctx, blocks, result, cfg)
	processedBlocks = totalBlocks - len(pending)
	if processedBlocks > 0 && progressFn != nil {
		// The first frame already reflects what the earlier run finished —
		// a resumed run that reported 0% would look like it lost the work.
		progressFn(float64(processedBlocks) / float64(totalBlocks) * 100)
	}
	writeFailures := 0

	for batchStart := 0; batchStart < len(pending); batchStart += batchSize {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return nil, nil, TranslationOutcome{}, fmt.Errorf("translation cancelled: %w", err)
		}

		batchEnd := batchStart + batchSize
		if batchEnd > len(pending) {
			batchEnd = len(pending)
		}

		// The batch's SOURCE positions: with a cache these skip the hits, so a
		// batch is always ten cues the model has not been paid for yet.
		batchAt := pending[batchStart:batchEnd]
		batch := make([]TranslationBlock, len(batchAt))
		for i, at := range batchAt {
			batch[i] = blocks[at]
		}

		// Build context from previous translated blocks (AC #2).
		// Taken from `result` at the SOURCE positions before this batch, so a
		// resumed run reads the cached translations rather than the English
		// they replaced — otherwise the first fresh batch after a resume would
		// see English context and drift in register and proper nouns.
		var contextBlocks []prompts.SubtitleTranslatorBlock
		if first := batchAt[0]; first > 0 {
			contextStart := first - contextWindow
			if contextStart < 0 {
				contextStart = 0
			}
			for i := contextStart; i < first; i++ {
				contextBlocks = append(contextBlocks, prompts.SubtitleTranslatorBlock{
					Index: result[i].Index,
					Text:  result[i].Text, // Use translated text as context
				})
			}
		}

		// Build prompt blocks
		var promptBlocks []prompts.SubtitleTranslatorBlock
		for _, b := range batch {
			promptBlocks = append(promptBlocks, prompts.SubtitleTranslatorBlock{
				Index: b.Index,
				Text:  b.Text,
			})
		}

		// Call Claude API
		userPrompt := prompts.BuildSubtitleTranslatorPromptWithGlossary(promptBlocks, contextBlocks, promptGlossary)

		translated, err := s.provider.CompleteText(
			ctx,
			systemPrompt,
			userPrompt,
			TranslationMaxTokens,
		)

		if err != nil {
			// 9R-16 AC 6c: the per-run budget sentinel must escape the
			// keep-English tolerance — remaining batches would all fail the
			// same way, and the caller needs the sentinel to pause the batch.
			if errors.Is(err, ai.ErrBudgetExceeded) {
				// The batches already written to the cache above ARE the
				// progress: the next run reads them back and starts here.
				return nil, nil, TranslationOutcome{}, fmt.Errorf("translation stopped at block %d: %w", batchAt[0], err)
			}

			// AC #5: on error, keep English text for failed blocks
			slog.Warn("Translation batch failed — keeping English text for blocks",
				"batch_start", batchStart,
				"batch_end", batchEnd,
				"error", err,
			)

			// Check if this is a context cancellation (propagate it)
			if ctx.Err() != nil {
				return nil, nil, TranslationOutcome{}, fmt.Errorf("translation cancelled: %w", ctx.Err())
			}

			englishKept += len(batch)
			processedBlocks += len(batch)
			if progressFn != nil {
				progressFn(float64(processedBlocks) / float64(totalBlocks) * 100)
			}
			continue
		}

		// Parse response and apply translations
		indices := make([]int, len(batch))
		for i, b := range batch {
			indices[i] = b.Index
		}

		translations, batchTerms := parseTranslationResponseWithTerms(translated, indices)
		harvested = mergeHarvestedTerms(harvested, batchTerms)

		for i, b := range batch {
			resultIdx := batchAt[i]
			if text, ok := translations[b.Index]; ok {
				result[resultIdx].Text = text
				// Written the moment it arrives, not at the end of the track:
				// the run may never reach the end, and what is not written
				// here is what the next run pays for again. The model's RAW
				// output is stored — OpenCC and the Taiwan lexicon run over
				// the whole assembled SRT afterwards, so a hit goes through
				// exactly the same post-processing as a fresh translation.
				if cfg.store != nil {
					if err := cfg.store.Set(ctx, keys[resultIdx], text, segkey.TTL); err != nil {
						writeFailures++
						if writeFailures == 1 {
							slog.Warn("segment cache write failed — the next run pays for these cues again",
								"cue_index", b.Index, "error", err)
						}
					}
				}
			} else {
				// bugfix-j: a response missing this block keeps English — that
				// is a partial outcome, not a silent nothing. It is also NOT
				// cached: storing the English would make every later run serve
				// it back as though it had been translated.
				englishKept++
			}
		}

		processedBlocks += len(batch)
		if progressFn != nil {
			progressFn(float64(processedBlocks) / float64(totalBlocks) * 100)
		}

		slog.Info("Translation batch completed",
			"batch", fmt.Sprintf("%d-%d", batch[0].Index, batch[len(batch)-1].Index),
			"translated", len(translations),
			"total", len(batch),
		)
	}

	if writeFailures > 0 {
		slog.Warn("segment cache writes failed", "failed_cues", writeFailures, "translated_cues", len(pending))
	}

	outcome := TranslationOutcome{EnglishKeptBlocks: englishKept, TotalBlocks: totalBlocks}
	if outcome.Partial() {
		slog.Warn("Translation completed with partial failures — some blocks retain English text",
			"total_blocks", totalBlocks,
			"english_kept_blocks", englishKept,
		)
	}

	return result, harvested, outcome, nil
}

// mergeHarvestedTerms folds one batch's trailer terms into the accumulated map,
// FIRST occurrence winning across batches. Returns acc unchanged (possibly nil)
// when the batch harvested nothing.
func mergeHarvestedTerms(acc, batch map[string]string) map[string]string {
	if len(batch) == 0 {
		return acc
	}
	if acc == nil {
		acc = make(map[string]string, len(batch))
	}
	for src, zh := range batch {
		if _, seen := acc[src]; !seen {
			acc[src] = zh
		}
	}
	return acc
}

// TranslateRequest translates a set of arbitrary keyed fields in a single
// glossary-aware batch (Story 9R-7). It is the generic entry point the .nfo
// metadata localizer (9R-13) uses — no batching/context-window (a handful of
// named fields, not thousands of subtitle blocks). Returns fields with Text
// replaced by the translation; a field with no returned translation keeps its
// original Text (fail-soft, mirrors the subtitle path).
//
// Fields are addressed by 1-based ordinal in the prompt/response; Key is
// preserved verbatim so callers match results back by name.
func (s *TranslationService) TranslateRequest(ctx context.Context, req TranslationRequest) ([]TranslationField, error) {
	if len(req.Fields) == 0 {
		return nil, nil
	}

	promptBlocks := make([]prompts.SubtitleTranslatorBlock, len(req.Fields))
	indices := make([]int, len(req.Fields))
	for i, f := range req.Fields {
		promptBlocks[i] = prompts.SubtitleTranslatorBlock{Index: i + 1, Text: f.Text}
		indices[i] = i + 1
	}

	userPrompt := prompts.BuildSubtitleTranslatorPromptWithGlossary(promptBlocks, nil, toPromptGlossary(req.Glossary))

	translated, err := s.provider.CompleteText(ctx, prompts.SubtitleTranslatorSystemPrompt, userPrompt, TranslationMaxTokens)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("translation cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("translate fields: %w", err)
	}

	translations := parseTranslationResponse(translated, indices)

	out := make([]TranslationField, len(req.Fields))
	for i, f := range req.Fields {
		out[i] = f // keep Key; default to original Text (fail-soft)
		if text, ok := translations[i+1]; ok {
			out[i].Text = text
		}
	}
	return out, nil
}

// TranslateChunk sends ONE chunk (numbered cues plus its read-only context
// window) and returns the parsed per-index map plus token usage. No internal
// batching, no retry — the subtitle-side pipeline owns both, because retry
// granularity is the cue while transport granularity is the chunk (P3). If this
// method batched, the pipeline could not resend only the cues its quality gate
// rejected without re-sending the clean ones.
//
// The prompt carries numbered text only: Start/End never cross this boundary
// (P2/FR11). Prompt building and response parsing reuse the Route C machinery
// verbatim — no second prompt or parser path exists.
//
// The second return is the chunk's harvested term map (sub-5-5 AC #3) — the
// proper-noun renderings the model reported in its optional trailer; nil when
// the response carried none. Widening the ChunkTranslator port (unstamped) in
// the same change; fakes sync alongside.
func (s *TranslationService) TranslateChunk(ctx context.Context, sys []ai.SystemBlock, contextBlocks, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, map[string]string, ai.CompletionUsage, error) {
	if len(blocks) == 0 {
		return nil, nil, ai.CompletionUsage{}, nil
	}

	indices := make([]int, len(blocks))
	for i, b := range blocks {
		indices[i] = b.Index
	}
	userPrompt := prompts.BuildSubtitleTranslatorPrompt(blocks, contextBlocks)

	// Claude implements CachingCompleter, so usage — including both cache
	// dimensions — flows back to the caller. Gemini deliberately does not:
	// it degrades to the plain text API with zero usage rather than blocking
	// the multi-provider moat (the established degradation shape).
	if caching, ok := s.provider.(ai.CachingCompleter); ok {
		res, err := caching.CompleteTextWithUsage(ctx, ai.CompletionRequest{
			System:     sys,
			UserPrompt: userPrompt,
			MaxTokens:  TranslationMaxTokens,
		})
		if err != nil {
			return nil, nil, ai.CompletionUsage{}, fmt.Errorf("translate chunk [%d-%d]: %w", indices[0], indices[len(indices)-1], err)
		}
		translations, terms := parseTranslationResponseWithTerms(res.Text, indices)
		return translations, terms, res.Usage, nil
	}

	slog.Info("AI provider does not implement ai.CachingCompleter — translating without prompt caching or token usage",
		"chunk_size", len(blocks),
	)
	text, err := s.provider.CompleteText(ctx, flattenSystemBlocks(sys), userPrompt, TranslationMaxTokens)
	if err != nil {
		return nil, nil, ai.CompletionUsage{}, fmt.Errorf("translate chunk [%d-%d]: %w", indices[0], indices[len(indices)-1], err)
	}
	translations, terms := parseTranslationResponseWithTerms(text, indices)
	return translations, terms, ai.CompletionUsage{}, nil
}

// flattenSystemBlocks collapses ordered system blocks into the single string
// the non-caching TextCompleter API accepts. Block order is preserved, so the
// flattened prompt reads exactly like the block sequence it stands in for.
func flattenSystemBlocks(sys []ai.SystemBlock) string {
	parts := make([]string, 0, len(sys))
	for _, b := range sys {
		if strings.TrimSpace(b.Text) != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// Harvest-trailer wire format — [@contract-v1] (sub-5-5 AC #1): the parser side
// of the cross-layer contract whose instruction side lives in
// prompts.SubtitleTranslatorSystemPrompt ("Term harvest" section). The sentinel
// line and the `=>` separator MUST change together with that prompt text.
const (
	harvestTrailerSentinel = "===TERMS==="
	harvestTermSeparator   = "=>"
)

// splitHarvestTrailer cuts the optional harvest trailer off a translation
// response BEFORE cue parsing (sub-5-5 AC #2, red line 1): the cue parser's
// continuation rule appends un-prefixed lines to the most recent cue, so an
// unstripped trailer would be stitched into the LAST subtitle cue.
//
// Everything from the first sentinel line onward is trailer — even a [N] line
// after a mid-response sentinel is deliberately NOT parsed as a cue. Malformed
// term lines (no separator, empty source, empty rendering) are skipped per
// line with a Debug log: harvest is an opportunistic yield, never a
// correctness obligation (fail-soft). A duplicate source keeps its first
// occurrence, matching the glossary's fixed-rendering spirit.
func splitHarvestTrailer(response string) (string, map[string]string) {
	lines := strings.Split(response, "\n")
	sentinelAt := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == harvestTrailerSentinel {
			sentinelAt = i
			break
		}
	}
	if sentinelAt < 0 {
		return response, nil
	}

	var terms map[string]string
	for _, raw := range lines[sentinelAt+1:] {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		src, zh, found := strings.Cut(line, harvestTermSeparator)
		src, zh = strings.TrimSpace(src), strings.TrimSpace(zh)
		if !found || src == "" || zh == "" {
			slog.Debug("harvest trailer line ignored — malformed term entry", "line", line)
			continue
		}
		if src == zh {
			// The prompt forbids listing terms kept in their original form, but
			// a disobedient model still emits `Vecna=>Vecna` — an identity
			// mapping carries no rendering decision and would only be junk the
			// F6 review has to clean up (CR sub-5-5 M2).
			slog.Debug("harvest trailer line ignored — identity mapping", "line", line)
			continue
		}
		if terms == nil {
			terms = make(map[string]string)
		}
		if _, dup := terms[src]; !dup {
			terms[src] = zh
		}
	}
	return strings.Join(lines[:sentinelAt], "\n"), terms
}

// responseLinePattern matches "[N] text" format from Claude's response.
// The text group is `(.*)` (may be empty): a bare "[N]" line — the model
// emitting an index with no translation — must be recognized as that cue's
// (empty) slot, NOT fall through to the continuation branch where the literal
// "[N]" gets appended into the PREVIOUS cue's subtitle text (bugfix-j CR M3:
// probe showed cue1 ending as "你好\n[2]" in the placed file).
var responseLinePattern = regexp.MustCompile(`^\[(\d+)\]\s*(.*)$`)

// parseTranslationResponse extracts translated text from Claude's response.
// Response format: "[1] 翻譯文字\n[2] 翻譯文字"
// Handles multi-line blocks: continuation lines (no [N] prefix) are appended
// to the most recent indexed block. The optional harvest trailer is stripped
// (and discarded) first — callers that want the terms use
// parseTranslationResponseWithTerms.
func parseTranslationResponse(response string, indices []int) map[int]string {
	result, _ := parseTranslationResponseWithTerms(response, indices)
	return result
}

// parseTranslationResponseWithTerms strips the harvest trailer BEFORE cue
// parsing (sub-5-5 AC #2/#3) and returns both the per-index translations and
// the harvested term map (nil when the response carried no trailer).
func parseTranslationResponseWithTerms(response string, indices []int) (map[int]string, map[string]string) {
	response, terms := splitHarvestTrailer(response)
	result := make(map[int]string)

	// Build index lookup for validation
	validIndices := make(map[int]bool)
	for _, idx := range indices {
		validIndices[idx] = true
	}

	lines := strings.Split(strings.TrimSpace(response), "\n")
	var lastIdx int
	hasLast := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := responseLinePattern.FindStringSubmatch(line)
		if matches != nil {
			idx, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			if validIndices[idx] {
				// bugfix-j CR M3: an empty translation is a MISSING one — do
				// not store "" (it would blank the cue and count as success).
				// Still advance lastIdx so any continuation lines attach to
				// THIS cue instead of polluting the previous one.
				if text := strings.TrimSpace(matches[2]); text != "" {
					result[idx] = text
				}
				lastIdx = idx
				hasLast = true
			}
		} else if hasLast && validIndices[lastIdx] {
			// Continuation line for multi-line subtitle block. If the indexed
			// line itself was empty, the continuation IS the cue's first text.
			if existing, ok := result[lastIdx]; ok {
				result[lastIdx] = existing + "\n" + line
			} else {
				result[lastIdx] = line
			}
		}
	}

	return result, terms
}
