package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/sse"
)

// SSE event type for translation progress (AC #6)
const (
	EventTranslationProgress sse.EventType = "translation_progress"
)

// Translation constants
const (
	// TranslationTimeout is the max time per batch request.
	TranslationTimeout = 60 * time.Second
	// TranslationMaxTokens is the max response tokens per batch (10 blocks × ~100 chars).
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

// Translate translates subtitle blocks from English to Traditional Chinese.
// progressFn (optional) receives percentage updates per batch.
// On partial failure, translated blocks are returned with untranslated blocks
// retaining their original English text (AC #5).
// Translate translates English subtitle blocks to Traditional Chinese with no
// glossary (back-compat entry point; existing callers unchanged).
func (s *TranslationService) Translate(ctx context.Context, blocks []TranslationBlock, progressFn func(float64)) ([]TranslationBlock, error) {
	return s.TranslateWithGlossary(ctx, blocks, nil, progressFn)
}

// TranslateWithGlossary is Translate plus a per-show glossary (Story 9R-7): the
// fixed renderings are injected into every batch prompt so proper nouns stay
// consistent across the whole subtitle and across runs. A nil glossary makes
// this byte-identical to the pre-9R-7 behavior.
//
// Signature deliberately unchanged (sub-5-5 AC #3): it delegates to
// TranslateWithGlossaryHarvest and drops the harvested terms — the
// Translate → TranslateWithGlossary precedent (9R-7).
func (s *TranslationService) TranslateWithGlossary(ctx context.Context, blocks []TranslationBlock, glossary []GlossaryPair, progressFn func(float64)) ([]TranslationBlock, error) {
	translated, _, err := s.TranslateWithGlossaryHarvest(ctx, blocks, glossary, progressFn)
	return translated, err
}

// TranslateWithGlossaryHarvest is TranslateWithGlossary plus the sub-5-5 AC #3
// harvest return: the proper-noun renderings the model reported in each batch's
// trailer, merged across batches with the FIRST occurrence winning (the model's
// later hindsight does not overturn an established rendering — the glossary's
// fixed-rendering spirit). nil terms = nothing harvested.
func (s *TranslationService) TranslateWithGlossaryHarvest(ctx context.Context, blocks []TranslationBlock, glossary []GlossaryPair, progressFn func(float64)) ([]TranslationBlock, map[string]string, error) {
	if len(blocks) == 0 {
		return nil, nil, nil
	}
	promptGlossary := toPromptGlossary(glossary)

	batchSize := prompts.SubtitleTranslatorBatchSize
	contextWindow := prompts.SubtitleTranslatorContextWindow

	// Copy blocks to result (will be modified in place)
	result := make([]TranslationBlock, len(blocks))
	copy(result, blocks)

	totalBlocks := len(blocks)
	processedBlocks := 0
	hasPartialFailure := false
	var harvested map[string]string

	for batchStart := 0; batchStart < totalBlocks; batchStart += batchSize {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("translation cancelled: %w", err)
		}

		batchEnd := batchStart + batchSize
		if batchEnd > totalBlocks {
			batchEnd = totalBlocks
		}

		batch := blocks[batchStart:batchEnd]

		// Build context from previous translated blocks (AC #2)
		var contextBlocks []prompts.SubtitleTranslatorBlock
		if batchStart > 0 {
			contextStart := batchStart - contextWindow
			if contextStart < 0 {
				contextStart = 0
			}
			for i := contextStart; i < batchStart; i++ {
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

		batchCtx, batchCancel := context.WithTimeout(ctx, TranslationTimeout)
		translated, err := s.provider.CompleteText(
			batchCtx,
			prompts.SubtitleTranslatorSystemPrompt,
			userPrompt,
			TranslationMaxTokens,
		)
		batchCancel()

		if err != nil {
			// 9R-16 AC 6c: the per-run budget sentinel must escape the
			// keep-English tolerance — remaining batches would all fail the
			// same way, and the caller needs the sentinel to pause the batch.
			if errors.Is(err, ai.ErrBudgetExceeded) {
				return nil, nil, fmt.Errorf("translation stopped at block %d: %w", batchStart, err)
			}

			// AC #5: on error, keep English text for failed blocks
			slog.Warn("Translation batch failed — keeping English text for blocks",
				"batch_start", batchStart,
				"batch_end", batchEnd,
				"error", err,
			)

			// Check if this is a context cancellation (propagate it)
			if ctx.Err() != nil {
				return nil, nil, fmt.Errorf("translation cancelled: %w", ctx.Err())
			}

			hasPartialFailure = true
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
			resultIdx := batchStart + i
			if text, ok := translations[b.Index]; ok {
				result[resultIdx].Text = text
			}
			// If translation not found for this block, original English text is kept
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

	if hasPartialFailure {
		slog.Warn("Translation completed with partial failures — some blocks retain English text",
			"total_blocks", totalBlocks,
		)
	}

	return result, harvested, nil
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

	batchCtx, cancel := context.WithTimeout(ctx, TranslationTimeout)
	defer cancel()
	translated, err := s.provider.CompleteText(batchCtx, prompts.SubtitleTranslatorSystemPrompt, userPrompt, TranslationMaxTokens)
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

	chunkCtx, cancel := context.WithTimeout(ctx, TranslationTimeout)
	defer cancel()

	// Claude implements CachingCompleter, so usage — including both cache
	// dimensions — flows back to the caller. Gemini deliberately does not:
	// it degrades to the plain text API with zero usage rather than blocking
	// the multi-provider moat (the established degradation shape).
	if caching, ok := s.provider.(ai.CachingCompleter); ok {
		res, err := caching.CompleteTextWithUsage(chunkCtx, ai.CompletionRequest{
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
	text, err := s.provider.CompleteText(chunkCtx, flattenSystemBlocks(sys), userPrompt, TranslationMaxTokens)
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
var responseLinePattern = regexp.MustCompile(`^\[(\d+)\]\s*(.+)$`)

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
				result[idx] = strings.TrimSpace(matches[2])
				lastIdx = idx
				hasLast = true
			}
		} else if hasLast && validIndices[lastIdx] {
			// Continuation line for multi-line subtitle block
			result[lastIdx] += "\n" + line
		}
	}

	return result, terms
}
