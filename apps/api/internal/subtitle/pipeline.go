package subtitle

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
)

const (
	// maxQualityRetries caps SEMANTIC resends per cue (FR16). Transport-level
	// retries are the ai client's job (retryTransient, D8) — do not add another
	// transport loop here.
	maxQualityRetries = 2

	// stubbornCeilingDenominator expresses the stubborn-cue ceiling as 1/N of
	// the track's cues: 20 → 5%.
	stubbornCeilingDenominator = 20
)

// ChunkTranslator is the narrow port the pipeline needs from the services-side
// TranslationService: ONE chunk in, per-index translations plus token usage out.
//
// Chunking, the quality gate and every semantic retry live on THIS side of the
// boundary (architecture P3): retry granularity is the cue, transport
// granularity is the chunk. If the service batched internally the pipeline
// could not resend only the failed cues without re-sending the clean ones —
// which is the quality gate's whole point.
type ChunkTranslator interface {
	TranslateChunk(ctx context.Context, sys []ai.SystemBlock, contextBlocks, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error)
}

// VariantConverter is the narrow port over *Converter. Injected once and reused
// for the whole track (Rule 14 — never constructed per call).
type VariantConverter interface {
	ConvertS2TWP(content []byte) ([]byte, error)
}

// TranslateContext carries the FR26 show metadata injected into the translation
// prompt as context. Every field is optional; a zero value yields a prompt that
// is byte-identical to the no-metadata path.
//
// [@contract-v1] — consumed by sub-1-5b (delivery + provenance + cache policy)
// and sub-1-6. Changing the signature, the stubborn-cue policy, or the usage
// semantics = Rule 20 bump + downstream stale-mark.
type TranslateContext struct {
	Title, OriginalTitle string
	Year                 int
	Genres               []string
	Overview             string
	Cast                 []string // capped at 10 by the builder
	Countries            []string
	Glossary             []prompts.GlossaryEntry // M1: always empty — field exists NOW (D4 versioning)
}

// TranslateResult is the translate stage's output.
//
// [@contract-v1] — see TranslateContext.
type TranslateResult struct {
	Blocks       []SubtitleBlock    // translated + gated + OpenCC'd; Index/Start/End byte-equal source
	Usage        ai.CompletionUsage // aggregated across all chunks + retries
	StubbornCues int                // cues delivered with English fallback (see policy)
}

// Pipeline owns the generation-side stages that must not live in services:
// the quality gate needs detector.Detect, and Rule 19 forbids services →
// subtitle. The thin LLM call stays on the services side behind ChunkTranslator.
type Pipeline struct {
	translator ChunkTranslator
	converter  VariantConverter
	logger     *slog.Logger
}

// NewPipeline wires the pipeline to its two ports. Both are mandatory: this is
// a wiring-time invariant, so a nil port panics at construction rather than
// degrading at runtime — a nil converter in particular would silently void
// FR15's deterministic Traditional-script guarantee on every delivered cue.
func NewPipeline(translator ChunkTranslator, converter VariantConverter, logger *slog.Logger) *Pipeline {
	if translator == nil {
		panic("subtitle.NewPipeline: ChunkTranslator must not be nil")
	}
	if converter == nil {
		panic("subtitle.NewPipeline: VariantConverter must not be nil — FR15's OpenCC guarantee would silently disappear")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Pipeline{
		translator: translator,
		converter:  converter,
		logger:     logger.With("component", "subtitle_pipeline"),
	}
}

// TranslateTrack translates one routed English track to Traditional Chinese.
//
// [@contract-v1] — see TranslateContext.
func (p *Pipeline) TranslateTrack(ctx context.Context, track *ExtractedTrack, tctx TranslateContext) (*TranslateResult, error) {
	if track == nil || len(track.Blocks) == 0 {
		return nil, fmt.Errorf("%w: translate stage received no cues", ErrSubtitleTranslateFailed)
	}

	source := track.Blocks
	sys := buildSystemBlocks(tctx)

	// Snapshot cue identity BEFORE any work so the FR17 check compares against
	// the track as it was routed — not against whatever state it is in when the
	// stage finishes. SubtitleBlock is all value fields, so a shallow copy is a
	// full one.
	routed := append([]SubtitleBlock(nil), source...)

	// final holds the accepted translation per cue Index. Cue identity is the
	// Index carried from the source track, never the position: SDH filtering
	// leaves gaps in the numbering (P1/P7, sub-1-4 AC #1).
	final := make(map[int]string, len(source))
	var usage ai.CompletionUsage

	stubborn := 0

	for start := 0; start < len(source); start += prompts.SubtitleTranslatorBatchSize {
		end := min(start+prompts.SubtitleTranslatorBatchSize, len(source))
		chunk := source[start:end]

		// A whole track is tens of sequential chunks; check cancellation at the
		// boundary so a shutdown stops promptly. Both sentinels are chained so
		// the orchestrator can classify the run AND still tell a shutdown apart
		// from a genuine translate failure with errors.Is(context.Canceled) —
		// the sub-1-4 CR M1/M2 precedent.
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: cancelled at cue %d: %w", ErrSubtitleTranslateFailed, chunk[0].Index, err)
		}

		// Read-only context is fixed for the whole chunk: it looks BACKWARDS at
		// already-translated cues, so a retry inside this chunk cannot change it.
		contextBlocks := p.contextWindow(source, start, final)

		// pending starts as the whole chunk and narrows to just the cues the
		// gate rejected. Quality retries are SEMANTIC only, capped per cue —
		// transport retries already live inside the ai client (D8); a second
		// loop here would multiply them.
		pending := chunk
		var verdict GateVerdict

		for attempt := 0; ; attempt++ {
			got, chunkUsage, err := p.translator.TranslateChunk(ctx, sys, contextBlocks, promptBlocksOf(pending))
			usage = addUsage(usage, chunkUsage)
			if err != nil {
				return nil, fmt.Errorf("%w: cues %d-%d: %w",
					ErrSubtitleTranslateFailed, pending[0].Index, pending[len(pending)-1].Index, err)
			}

			verdict = checkChunk(pending, got, p.logger)
			for _, b := range pending {
				if !verdict.Failed(b.Index) {
					final[b.Index] = got[b.Index]
				}
			}

			if verdict.Passed() || attempt == maxQualityRetries {
				break
			}
			pending = subsetByIndex(pending, verdict.FailedIndexes)
		}

		for _, index := range verdict.FailedIndexes {
			stubborn++
			p.logger.Warn("subtitle cue kept its English original after quality retries",
				"cue_index", index,
				"reason", verdict.Reasons[index],
				"retries", maxQualityRetries,
				"stream_index", track.StreamIndex,
			)
		}
	}

	// Stubborn-cue policy (NFR-R1 fail-soft, bounded): one flaky cue must not
	// kill a 1000-cue episode, but a broken track must not ship half-English.
	// stubborn*20 > total is the integer form of stubborn/total > 5%.
	if stubborn*stubbornCeilingDenominator > len(source) {
		return nil, fmt.Errorf("%w: %d of %d cues still failed the quality gate after %d retries — over the %d%% ceiling",
			ErrSubtitleTranslateFailed, stubborn, len(source), maxQualityRetries, 100/stubbornCeilingDenominator)
	}

	blocks := p.convertAndStitch(source, final)
	if err := checkTimestampInvariant(routed, blocks); err != nil {
		return nil, err
	}

	return &TranslateResult{Blocks: blocks, Usage: usage, StubbornCues: stubborn}, nil
}

// subsetByIndex narrows a chunk to the cues the gate rejected, preserving
// source order so the resent prompt keeps reading top-to-bottom.
func subsetByIndex(blocks []SubtitleBlock, indexes []int) []SubtitleBlock {
	wanted := make(map[int]struct{}, len(indexes))
	for _, index := range indexes {
		wanted[index] = struct{}{}
	}

	out := make([]SubtitleBlock, 0, len(indexes))
	for _, b := range blocks {
		if _, ok := wanted[b.Index]; ok {
			out = append(out, b)
		}
	}
	return out
}

// contextWindow returns the previous SubtitleTranslatorContextWindow cues as
// read-only context, carrying their TRANSLATED text so the model stays
// terminologically consistent across chunk boundaries. A cue with no accepted
// translation yet contributes its source text.
func (p *Pipeline) contextWindow(source []SubtitleBlock, start int, final map[int]string) []prompts.SubtitleTranslatorBlock {
	if start == 0 {
		return nil
	}
	from := max(start-prompts.SubtitleTranslatorContextWindow, 0)

	out := make([]prompts.SubtitleTranslatorBlock, 0, start-from)
	for _, b := range source[from:start] {
		out = append(out, prompts.SubtitleTranslatorBlock{Index: b.Index, Text: textOf(b, final)})
	}
	return out
}

// convertAndStitch runs the OpenCC final pass over every cue and writes the
// result back onto a COPY of the source cue, so Index/Start/End are preserved
// by construction (FR11/FR17). It runs only once every chunk has cleared the
// quality gate — converting earlier would repair a Simplified leak the gate
// exists to catch (P4).
func (p *Pipeline) convertAndStitch(source []SubtitleBlock, final map[int]string) []SubtitleBlock {
	out := make([]SubtitleBlock, len(source))
	copy(out, source)

	failures := 0
	var firstErr error

	for i := range out {
		text := textOf(source[i], final)
		converted, err := p.converter.ConvertS2TWP([]byte(text))
		if err != nil {
			// Deliberate non-fatal discard (Rule 13 case 3): the gate has
			// already guaranteed this cue carries no simplified-only
			// character, so s2twp here is phrase-level polish rather than
			// the correctness barrier. Delivering the gated text beats
			// failing the item (NFR-R1) — the same call the Converter's own
			// contract makes ("unconverted subtitle is better than no
			// subtitle"). Surfaced in one aggregate warning below.
			failures++
			if firstErr == nil {
				firstErr = err
			}
		} else {
			text = string(converted)
		}
		out[i].Text = text
	}

	if failures > 0 {
		p.logger.Warn("OpenCC final pass failed for some cues — delivering the gated text unconverted",
			"failed_cues", failures,
			"total_cues", len(out),
			"first_error", firstErr,
		)
	}
	return out
}

// textOf returns the accepted translation for a cue, falling back to its
// original source text.
func textOf(b SubtitleBlock, final map[int]string) string {
	if text, ok := final[b.Index]; ok {
		return text
	}
	return b.Text
}

// promptBlocksOf projects source cues onto the prompt's numbered-text shape.
// Start/End never cross this boundary — timestamps stay in Go (P2/FR11).
func promptBlocksOf(blocks []SubtitleBlock) []prompts.SubtitleTranslatorBlock {
	out := make([]prompts.SubtitleTranslatorBlock, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, prompts.SubtitleTranslatorBlock{Index: b.Index, Text: b.Text})
	}
	return out
}

// buildSystemBlocks composes the system prompt in stable-first order, the shape
// sub-1-5b's prompt caching depends on: [0] the invariant translator prompt
// (most stable), [1] the per-show metadata + glossary sections. Block [1] is
// omitted entirely when there is nothing per-show to say, so the no-metadata
// path is byte-identical to Route C's.
//
// Every block is CacheTTLNone in this story ON PURPOSE — cache policy (the TTL
// flip, the versioned key, the <4096-token disable-and-record rule) is sub-1-5b's,
// and turning caching on here would ship the mechanism without its policy.
func buildSystemBlocks(tctx TranslateContext) []ai.SystemBlock {
	blocks := []ai.SystemBlock{
		{Text: prompts.SubtitleTranslatorSystemPrompt, CacheTTL: ai.CacheTTLNone},
	}

	perShow := prompts.BuildMetadataSection(metadataOf(tctx)) + prompts.BuildGlossarySection(tctx.Glossary)
	if perShow != "" {
		blocks = append(blocks, ai.SystemBlock{Text: perShow, CacheTTL: ai.CacheTTLNone})
	}
	return blocks
}

// metadataOf maps the stage's context onto the prompts package's mirror type.
// prompts cannot import subtitle (that would cycle), so the mapping lives here.
func metadataOf(tctx TranslateContext) prompts.MediaMetadata {
	return prompts.MediaMetadata{
		Title:         tctx.Title,
		OriginalTitle: tctx.OriginalTitle,
		Year:          tctx.Year,
		Genres:        tctx.Genres,
		Overview:      tctx.Overview,
		Cast:          tctx.Cast,
		Countries:     tctx.Countries,
	}
}

// addUsage accumulates token usage across chunks and retries.
func addUsage(acc, u ai.CompletionUsage) ai.CompletionUsage {
	acc.InputTokens += u.InputTokens
	acc.OutputTokens += u.OutputTokens
	acc.CacheCreationInputTokens += u.CacheCreationInputTokens
	acc.CacheReadInputTokens += u.CacheReadInputTokens
	return acc
}

// checkTimestampInvariant is the FR17 structural guard: the translated track
// must carry exactly the routed cue count, and every cue must keep its routed
// Index/Start/End byte-for-byte. It passes by construction today — that is the
// point. It exists so a future refactor that starts reordering, dropping or
// re-numbering cues fails loudly instead of shipping desynced subtitles.
//
// source is the snapshot taken at stage entry, which also makes the check
// meaningful when the track is mutated underneath the stage.
func checkTimestampInvariant(source, translated []SubtitleBlock) error {
	if len(source) != len(translated) {
		return fmt.Errorf("%w: cue count %d != source %d",
			ErrSubtitleTimestampMismatch, len(translated), len(source))
	}
	for i := range source {
		s, t := source[i], translated[i]
		if s.Index != t.Index || s.Start != t.Start || s.End != t.End {
			return fmt.Errorf("%w: cue %d: got [%d] %s --> %s, want [%d] %s --> %s",
				ErrSubtitleTimestampMismatch, i, t.Index, t.Start, t.End, s.Index, s.Start, s.End)
		}
	}
	return nil
}
