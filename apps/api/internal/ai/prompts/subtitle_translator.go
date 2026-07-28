// Package prompts provides prompt templates for AI-powered text processing.
package prompts

import (
	"fmt"
	"strconv"
	"strings"
)

// SubtitleTranslatorPromptVersion identifies the prompt revision for cache-key
// and provenance purposes (P11). ANY text change to the prompts or section
// builders in this file REQUIRES bumping this constant IN THE SAME EDIT —
// otherwise a re-run silently returns the previous translation and the pilot's
// A/B comparison is invalid (the architecture's named silent-failure trap).
//
// TestSubtitleTranslatorPromptVersion_PinsPromptText enforces this: it hashes
// every prompt surface in this file, so an edit that forgets the bump fails.
const SubtitleTranslatorPromptVersion = "m1-v1"

// SubtitleTranslatorContextWindow is the number of previous blocks sent as
// read-only context for each translation batch to maintain consistency (AC #2).
const SubtitleTranslatorContextWindow = 5

// SubtitleTranslatorBatchSize is the number of subtitle blocks per Claude request.
// Balances API cost with translation quality.
const SubtitleTranslatorBatchSize = 10

// SubtitleTranslatorSystemPrompt instructs Claude to translate English subtitle
// dialogue into natural Traditional Chinese (Taiwan usage).
const SubtitleTranslatorSystemPrompt = `You are a professional subtitle translator specializing in English to Traditional Chinese (Taiwan usage).

## Your task:
Translate English subtitle dialogue into natural, fluent Traditional Chinese as spoken in Taiwan.

## Translation rules:
1. Use Taiwan Traditional Chinese vocabulary and expressions (台灣用語), NOT mainland China terms
   - 例：software → 軟體 (not 軟件), video → 影片 (not 視頻), information → 資訊 (not 信息)
2. Preserve the speaker's tone, emotion, and register (formal/casual/slang)
3. Keep proper nouns (person names, place names, brand names) in their original English form
4. Keep technical terms, acronyms, and abbreviations in English when commonly used as-is in Taiwan
5. Maintain natural spoken Chinese rhythm — subtitles should sound like real dialogue, not written prose
6. Do NOT add honorifics or politeness markers not present in the original
7. Keep translations concise — subtitles have limited screen time

## Output format:
Return ONLY the translated text for each block, prefixed with the block index in square brackets.
For single-line blocks, output one line per block:
[1] 你好，最近怎麼樣？
[2] 我很好，謝謝。

For multi-line blocks (e.g., two speakers), preserve the line breaks — only the first line gets the index prefix:
[3] 你先走吧。
我隨後就到。

Do NOT include any explanation, notes, or annotations. ONLY translated lines with indices.`

// SubtitleTranslatorBlock represents a subtitle block for translation.
type SubtitleTranslatorBlock struct {
	Index int
	Text  string
}

// GlossaryEntry is one proper-noun mapping injected into a translation prompt
// so a term renders consistently across runs (Story 9R-7 keystone). Source is
// the original-language term; Target is the fixed zh rendering.
type GlossaryEntry struct {
	Source string
	Target string
}

// BuildGlossarySection renders the do-not-retranslate / use-this-rendering
// instruction block for a glossary. Returns "" when the glossary is empty so
// existing prompts are byte-identical (no-regression for the non-glossary path).
func BuildGlossarySection(glossary []GlossaryEntry) string {
	if len(glossary) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Glossary — MANDATORY fixed renderings:\n")
	sb.WriteString("Whenever any of these source terms appears, render it EXACTLY as given.\n")
	sb.WriteString("Do NOT re-translate, transliterate, or vary these — use the fixed target verbatim:\n")
	for _, e := range glossary {
		sb.WriteString(fmt.Sprintf("- %s → %s\n", e.Source, e.Target))
	}
	sb.WriteString("\n")
	return sb.String()
}

// MetadataCastLimit caps how many cast names are rendered into the media
// context section — enough to anchor character names, short enough not to crowd
// the cached prefix.
const MetadataCastLimit = 10

// MediaMetadata is the FR26 show context injected into a translation prompt so
// the model renders character names, genre register and setting consistently.
//
// It mirrors the metadata half of subtitle.TranslateContext: prompts cannot
// import subtitle (that would cycle), so the pipeline maps across at the call
// site. Every field is optional.
type MediaMetadata struct {
	Title         string
	OriginalTitle string
	Year          int
	Genres        []string
	Overview      string
	Cast          []string
	Countries     []string
}

// BuildMetadataSection renders the FR26 media-context block. Returns "" when
// the metadata carries nothing renderable, so the no-metadata path produces a
// byte-identical prompt (the BuildGlossarySection precedent).
func BuildMetadataSection(md MediaMetadata) string {
	var rows []string
	addRow := func(label, value string) {
		if v := strings.TrimSpace(value); v != "" {
			rows = append(rows, fmt.Sprintf("- %s: %s\n", label, v))
		}
	}

	addRow("Title", md.Title)
	addRow("Original title", md.OriginalTitle)
	if md.Year > 0 {
		addRow("Year", strconv.Itoa(md.Year))
	}
	addRow("Genres", joinNonEmpty(md.Genres, 0))
	addRow("Production countries", joinNonEmpty(md.Countries, 0))
	addRow("Cast", joinNonEmpty(md.Cast, MetadataCastLimit))
	addRow("Overview", collapseLines(md.Overview))

	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Media context — background only, do NOT translate or output this section:\n")
	sb.WriteString("Use it to keep character names, register and setting consistent across the whole subtitle.\n")
	for _, row := range rows {
		sb.WriteString(row)
	}
	sb.WriteString("\n")
	return sb.String()
}

// joinNonEmpty renders a comma-separated list, dropping blank entries and
// keeping at most limit of them (limit <= 0 means no cap).
func joinNonEmpty(values []string, limit int) string {
	kept := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			kept = append(kept, v)
		}
		if limit > 0 && len(kept) == limit {
			break
		}
	}
	return strings.Join(kept, ", ")
}

// collapseLines folds a multi-line overview onto one line so the section stays
// a flat key/value list the model cannot mistake for dialogue.
func collapseLines(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// BuildSubtitleTranslatorPrompt generates the user prompt for a batch of subtitle blocks.
// contextBlocks are previous blocks sent as read-only context (not re-translated).
// blocks are the blocks to be translated.
func BuildSubtitleTranslatorPrompt(blocks []SubtitleTranslatorBlock, contextBlocks []SubtitleTranslatorBlock) string {
	return BuildSubtitleTranslatorPromptWithGlossary(blocks, contextBlocks, nil)
}

// BuildSubtitleTranslatorPromptWithGlossary is BuildSubtitleTranslatorPrompt plus
// an optional glossary section prepended (Story 9R-7). A nil/empty glossary
// yields the exact same prompt as BuildSubtitleTranslatorPrompt.
func BuildSubtitleTranslatorPromptWithGlossary(blocks []SubtitleTranslatorBlock, contextBlocks []SubtitleTranslatorBlock, glossary []GlossaryEntry) string {
	var sb strings.Builder

	// Glossary first so the fixed renderings are established before the model
	// reads any dialogue.
	sb.WriteString(BuildGlossarySection(glossary))

	// Add context section if there are previous blocks
	if len(contextBlocks) > 0 {
		sb.WriteString("## Previous context (do NOT translate, for reference only):\n")
		for _, b := range contextBlocks {
			sb.WriteString(fmt.Sprintf("[%d] %s\n", b.Index, b.Text))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Translate the following blocks:\n")
	for _, b := range blocks {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", b.Index, b.Text))
	}

	return sb.String()
}
