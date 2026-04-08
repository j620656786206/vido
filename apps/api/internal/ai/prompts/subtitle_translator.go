// Package prompts provides prompt templates for AI-powered text processing.
package prompts

import (
	"fmt"
	"strings"
)

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
Return ONLY the translated text for each block, one per line, prefixed with the block index in square brackets.
Example:
[1] 你好，最近怎麼樣？
[2] 我很好，謝謝。

Do NOT include any explanation, notes, or annotations. ONLY translated lines with indices.`

// SubtitleTranslatorBlock represents a subtitle block for translation.
type SubtitleTranslatorBlock struct {
	Index int
	Text  string
}

// BuildSubtitleTranslatorPrompt generates the user prompt for a batch of subtitle blocks.
// contextBlocks are previous blocks sent as read-only context (not re-translated).
// blocks are the blocks to be translated.
func BuildSubtitleTranslatorPrompt(blocks []SubtitleTranslatorBlock, contextBlocks []SubtitleTranslatorBlock) string {
	var sb strings.Builder

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
