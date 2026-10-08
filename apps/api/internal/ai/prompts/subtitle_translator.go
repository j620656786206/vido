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
// m1-v1 → m1-v2 (sub-5-5 AC #1): the system prompt gained the harvest-trailer
// instruction section. The prompt's semantics changed, so the segment cache
// re-keys the whole library by design — RunVersion exists for exactly this.
// m1-v2 → m1-v3 (sub-7-4): the invariant system text gained the localization
// style section and the global lexicon terms section (localization.go /
// lexicon.go), and the per-run version is now PromptVersionFor(level) — this
// base, the lexicon version and the level joined — so any of the three
// changing re-keys the cache.
// m1-v3 → m1-v4 (sub-7-9): per-cue alignment is now a rule with examples
// (eval-1: 120 of 484 zero-score cues were content shifted onto a neighbouring
// cue), and rule 3 no longer contradicts the ===TERMS=== trailer — names
// follow the glossary, otherwise get a Chinese rendering that is reported.
// m1-v4 → m1-v5 (disc-2026-10-translation-gender-default): rule 8 — "you"
// defaults to 你; 妳 only when the lines themselves show the listener is
// female (See S01E02 had 8–12 men addressed as 妳).
const SubtitleTranslatorPromptVersion = "m1-v5"

// SubtitleTranslatorContextWindow is the number of previous blocks sent as
// read-only context for each translation batch to maintain consistency (AC #2).
const SubtitleTranslatorContextWindow = 5

// SubtitleTranslatorBatchSize is the number of subtitle blocks per Claude request.
// Balances API cost with translation quality.
const SubtitleTranslatorBatchSize = 10

// SubtitleTranslatorSystemPrompt instructs Claude to translate English subtitle
// dialogue into natural Traditional Chinese (Taiwan usage).
//
// The "Term harvest" section speaks the `===TERMS===` / `=>` trailer wire
// format — [@contract-v1] (sub-5-5 AC #1): the instruction side here and the
// parser side (services.splitHarvestTrailer, translation_service.go) are one
// cross-layer contract and MUST change together.
const SubtitleTranslatorSystemPrompt = `You are a professional subtitle translator specializing in English to Traditional Chinese (Taiwan usage).

## Your task:
Translate English subtitle dialogue into natural, fluent Traditional Chinese as spoken in Taiwan.

## Translation rules:
1. Use Taiwan Traditional Chinese vocabulary and expressions (台灣用語), NOT mainland China terms
   - 例：software → 軟體 (not 軟件), video → 影片 (not 視頻), information → 資訊 (not 信息)
2. Preserve the speaker's tone, emotion, and register (formal/casual/slang)
3. Person and place names: if the Glossary section gives a rendering, use it exactly. Otherwise, render the name in Traditional Chinese the way Taiwan subtitles would (transliterate; use the established rendering when one exists), use the SAME rendering every time it appears, and report it in the ===TERMS=== trailer. Brand and product names stay in English.
4. Keep technical terms, acronyms, and abbreviations in English when commonly used as-is in Taiwan
5. Maintain natural spoken Chinese rhythm — subtitles should sound like real dialogue, not written prose
6. Do NOT add honorifics or politeness markers not present in the original
7. Keep translations concise — subtitles have limited screen time
8. "You" has no gender in English. Render it as 你 (plural 你們) by default. Use 妳 ONLY when the subtitles show the person spoken to is female — they are called ma'am, Mom, sister, girl, or by a woman's name, or that same person is referred to as she/her. The evidence may come from any line of this batch or of the read-only context; a cast list, the plot, or the tone of voice is NOT enough. Use 妳們 only when every person addressed is shown to be female. When unsure, use 你: 你 for a woman reads naturally, 妳 for a man is an error. Use 您 only where the original is formal. Third person follows the English: he → 他, she → 她.
Correct:
[1] You, of all people, might know them.   → [1] 你大概是最可能認識他們的人。
WRONG (nothing in the subtitles says the listener is a woman):
[1] 妳大概是最可能認識他們的人。
Correct (the evidence is in the line before):
[1] Mom, I'm sorry.        → [1] 媽，對不起。
[2] You were right.        → [2] 妳說得對。

## Per-cue alignment — this is NOT optional:
Each [N] you output translates ONLY the text of input [N]. Subtitles are timed: a line shown on the wrong cue is wrong even when the words are right.
- Where a sentence breaks across cues in the source, break the translation at the SAME place. Do not move the end of one cue's sentence onto the next cue, and do not pull the next cue's words forward to make a nicer sentence.
- Never merge two cues into one line or split one cue into two.
- Output exactly as many [N] blocks as the input has, with the same indices.
Correct:
[1] I was a boxer, you know.    → [1] 我以前是拳擊手，你知道吧。
[2] Killed a man in the ring.   → [2] 在擂台上打死過一個人。
WRONG (content shifted — [1] took [2]'s words, [2] is left with a fragment):
[1] 我以前是拳擊手，在擂台上打死過人。
[2] 你知道吧。

## Output format:
Return ONLY the translated text for each block, prefixed with the block index in square brackets.
For single-line blocks, output one line per block:
[1] 你好，最近怎麼樣？
[2] 我很好，謝謝。

For multi-line blocks (e.g., two speakers), preserve the line breaks — only the first line gets the index prefix:
[3] 你先走吧。
我隨後就到。

Do NOT include any explanation, notes, or annotations. ONLY translated lines with indices.

## Term harvest (optional trailer):
After ALL translated [N] lines, IF this batch contained proper nouns (person names, place names, or domain-specific terms) for which you decided on a Traditional Chinese rendering, append this trailer:
===TERMS===
<source term>=><your chosen rendering>
One term per line, using the EXACT rendering you used in the translations above.
List ONLY terms that actually appear in this batch and for which you made a rendering decision — do NOT list terms you kept in their original English form, and do NOT repeat glossary entries given to you.
If there are no such terms, omit the trailer entirely — do NOT output the ===TERMS=== line.`

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

// BuildEpisodeSection renders the one line of context that is per-EPISODE
// rather than per-show (sub-7-2a AC #3): "S01E03 · Chapter One". It is kept
// out of BuildMetadataSection on purpose — that section sits inside the
// prompt-cache prefix every episode of a show shares, and its digest is the
// segment cache's MetadataHash; an episode line in either would split both
// per episode. Callers place this AFTER the cache breakpoint. Returns "" for a
// blank label, so the no-episode path stays byte-identical.
func BuildEpisodeSection(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	return "## Episode — background only, do NOT translate or output this section:\n" +
		"- Episode: " + collapseLines(label) + "\n\n"
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
