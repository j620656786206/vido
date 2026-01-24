// Package prompts provides prompt templates for AI-powered operations.
package prompts

import (
	"errors"
	"fmt"
)

// KeywordGeneratorRequest represents the input for keyword generation.
type KeywordGeneratorRequest struct {
	// Title is the media title to generate alternative keywords for.
	Title string
	// Language is the detected language of the title (optional, will auto-detect if empty).
	Language string
}

// Validate validates the keyword generator request.
func (r *KeywordGeneratorRequest) Validate() error {
	if r.Title == "" {
		return errors.New("title is required")
	}
	return nil
}

// KeywordGeneratorResponse represents the expected JSON output from AI.
type KeywordGeneratorResponse struct {
	Original             string   `json:"original"`
	SimplifiedChinese    string   `json:"simplified_chinese,omitempty"`
	TraditionalChinese   string   `json:"traditional_chinese,omitempty"`
	English              string   `json:"english,omitempty"`
	Romaji               string   `json:"romaji,omitempty"`
	Pinyin               string   `json:"pinyin,omitempty"`
	AlternativeSpellings []string `json:"alternative_spellings,omitempty"`
	CommonAliases        []string `json:"common_aliases,omitempty"`
}

// KeywordPrompt is the prompt template for generating alternative search keywords.
// It's optimized for CJK media titles that may have multiple localizations.
const KeywordPrompt = `You are a media title translator and search keyword generator specialized in CJK (Chinese, Japanese, Korean) media.

Given a media title, generate alternative search keywords that could help find the same media on different databases (TMDb, Douban, Wikipedia).

Title: %s
Detected Language: %s

## Your Task

Generate alternative versions of this title that could be used as search keywords. Focus on:

1. **Language Variants**: If the title is in Chinese, provide both Simplified and Traditional Chinese versions.
2. **English Translation**: Provide the official English title if known, or a reasonable translation.
3. **Romanization**: For Japanese titles, provide romaji. For Chinese titles, provide pinyin if helpful.
4. **Alternative Spellings**: Include common misspellings or variant romanizations.
5. **Common Aliases**: Include abbreviated names, fan translations, or alternative titles used in different regions.

## Output JSON Schema

{
  "original": "the original title as provided",
  "simplified_chinese": "Simplified Chinese version (if applicable)",
  "traditional_chinese": "Traditional Chinese version (if applicable)",
  "english": "Official English title or translation",
  "romaji": "Japanese romaji romanization (if Japanese title)",
  "pinyin": "Chinese pinyin romanization (if Chinese title, optional)",
  "alternative_spellings": ["variant1", "variant2"],
  "common_aliases": ["alias1", "alias2"]
}

## Rules

1. Only include fields that are applicable and non-empty
2. For anime/manga, include both Japanese and localized titles
3. For Chinese content, always include both Simplified and Traditional Chinese
4. Include common fan translations or unofficial names if well-known
5. For Korean content, include romanization in alternative_spellings
6. Do NOT make up titles - only include verified/known alternatives
7. The "original" field must always match the input title exactly

## Examples

### Example 1: Traditional Chinese Anime Title
Input: "鬼滅之刃"
Output:
{
  "original": "鬼滅之刃",
  "simplified_chinese": "鬼灭之刃",
  "traditional_chinese": "鬼滅之刃",
  "english": "Demon Slayer",
  "romaji": "Kimetsu no Yaiba",
  "alternative_spellings": ["Demon Slayer: Kimetsu no Yaiba"],
  "common_aliases": ["鬼滅", "Demon Slayer"]
}

### Example 2: Japanese Title
Input: "進撃の巨人"
Output:
{
  "original": "進撃の巨人",
  "simplified_chinese": "进击的巨人",
  "traditional_chinese": "進擊的巨人",
  "english": "Attack on Titan",
  "romaji": "Shingeki no Kyojin",
  "alternative_spellings": ["AOT", "SnK"],
  "common_aliases": ["巨人"]
}

### Example 3: English Title
Input: "Parasite"
Output:
{
  "original": "Parasite",
  "simplified_chinese": "寄生虫",
  "traditional_chinese": "寄生上流",
  "english": "Parasite",
  "alternative_spellings": [],
  "common_aliases": ["기생충", "Gisaengchung"]
}

### Example 4: Korean Title
Input: "오징어 게임"
Output:
{
  "original": "오징어 게임",
  "simplified_chinese": "鱿鱼游戏",
  "traditional_chinese": "魷魚遊戲",
  "english": "Squid Game",
  "alternative_spellings": ["Ojingeo Geim"],
  "common_aliases": []
}

Respond ONLY with valid JSON. No markdown, no explanation, no code blocks.`

// BuildKeywordPrompt generates the complete prompt for keyword generation.
func BuildKeywordPrompt(title, language string) string {
	if language == "" {
		language = "auto-detect"
	}
	return fmt.Sprintf(KeywordPrompt, title, language)
}

// KeywordExamples contains test cases for validating the prompt.
// Used for testing and prompt refinement.
var KeywordExamples = []struct {
	Title    string
	Language string
	Expected KeywordGeneratorResponse
}{
	{
		Title:    "鬼滅之刃",
		Language: "Traditional Chinese",
		Expected: KeywordGeneratorResponse{
			Original:           "鬼滅之刃",
			SimplifiedChinese:  "鬼灭之刃",
			TraditionalChinese: "鬼滅之刃",
			English:            "Demon Slayer",
			Romaji:             "Kimetsu no Yaiba",
			AlternativeSpellings: []string{
				"Demon Slayer: Kimetsu no Yaiba",
			},
			CommonAliases: []string{
				"鬼滅",
				"Demon Slayer",
			},
		},
	},
	{
		Title:    "進撃の巨人",
		Language: "Japanese",
		Expected: KeywordGeneratorResponse{
			Original:           "進撃の巨人",
			SimplifiedChinese:  "进击的巨人",
			TraditionalChinese: "進擊的巨人",
			English:            "Attack on Titan",
			Romaji:             "Shingeki no Kyojin",
			AlternativeSpellings: []string{
				"AOT",
				"SnK",
			},
			CommonAliases: []string{
				"巨人",
			},
		},
	},
	{
		Title:    "Parasite",
		Language: "English",
		Expected: KeywordGeneratorResponse{
			Original:           "Parasite",
			SimplifiedChinese:  "寄生虫",
			TraditionalChinese: "寄生上流",
			English:            "Parasite",
			AlternativeSpellings: []string{},
			CommonAliases: []string{
				"기생충",
				"Gisaengchung",
			},
		},
	},
	{
		Title:    "我的英雄學院",
		Language: "Traditional Chinese",
		Expected: KeywordGeneratorResponse{
			Original:           "我的英雄學院",
			SimplifiedChinese:  "我的英雄学院",
			TraditionalChinese: "我的英雄學院",
			English:            "My Hero Academia",
			Romaji:             "Boku no Hero Academia",
			AlternativeSpellings: []string{
				"MHA",
				"BnHA",
			},
			CommonAliases: []string{
				"英雄學院",
			},
		},
	},
}
