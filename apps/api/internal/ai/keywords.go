package ai

import (
	"encoding/json"
	"fmt"
)

// KeywordVariants holds alternative search terms for a media title.
// It provides language variants and common aliases to improve search success rates.
type KeywordVariants struct {
	// Original is the original title as provided (required).
	Original string `json:"original"`
	// SimplifiedChinese is the Simplified Chinese version.
	SimplifiedChinese string `json:"simplified_chinese,omitempty"`
	// TraditionalChinese is the Traditional Chinese version.
	TraditionalChinese string `json:"traditional_chinese,omitempty"`
	// English is the English title or translation.
	English string `json:"english,omitempty"`
	// Romaji is the Japanese romaji romanization.
	Romaji string `json:"romaji,omitempty"`
	// Pinyin is the Chinese pinyin romanization.
	Pinyin string `json:"pinyin,omitempty"`
	// AlternativeSpellings are variant romanizations or spellings.
	AlternativeSpellings []string `json:"alternative_spellings,omitempty"`
	// CommonAliases are abbreviated names, fan translations, or regional titles.
	CommonAliases []string `json:"common_aliases,omitempty"`
}

// NewKeywordVariants creates a new KeywordVariants with the original title.
func NewKeywordVariants(original string) *KeywordVariants {
	return &KeywordVariants{
		Original: original,
	}
}

// KeywordVariantsFromJSON parses a JSON string into KeywordVariants.
// It handles markdown code blocks that AI responses sometimes include.
func KeywordVariantsFromJSON(jsonStr string) (*KeywordVariants, error) {
	// Clean markdown code blocks if present
	cleaned := CleanJSONResponse(jsonStr)

	var variants KeywordVariants
	if err := json.Unmarshal([]byte(cleaned), &variants); err != nil {
		return nil, fmt.Errorf("failed to parse keyword variants JSON: %w", err)
	}

	return &variants, nil
}

// GetPrioritizedList returns keywords in search priority order.
// Priority order:
// 1. English (most commonly supported by metadata providers)
// 2. Romaji (for Japanese content)
// 3. Simplified Chinese (for Chinese databases like Douban)
// 4. Traditional Chinese (if different from original)
// 5. Alternative spellings
// 6. Common aliases
//
// The original title is NOT included as it's assumed to have already been tried.
// Duplicates are removed.
func (k *KeywordVariants) GetPrioritizedList() []string {
	seen := make(map[string]bool)
	var keywords []string

	// Helper to add unique keywords
	add := func(keyword string) {
		if keyword != "" && keyword != k.Original && !seen[keyword] {
			seen[keyword] = true
			keywords = append(keywords, keyword)
		}
	}

	// Priority 1: English (best for TMDb)
	add(k.English)

	// Priority 2: Romaji (for anime/Japanese content)
	add(k.Romaji)

	// Priority 3: Simplified Chinese (for Douban)
	add(k.SimplifiedChinese)

	// Priority 4: Traditional Chinese (if different from original)
	add(k.TraditionalChinese)

	// Priority 5: Alternative spellings
	for _, spelling := range k.AlternativeSpellings {
		add(spelling)
	}

	// Priority 6: Common aliases
	for _, alias := range k.CommonAliases {
		add(alias)
	}

	return keywords
}

// IsEmpty returns true if there are no alternative keywords.
// A KeywordVariants with only the original title is considered empty.
func (k *KeywordVariants) IsEmpty() bool {
	return len(k.GetPrioritizedList()) == 0
}

// Count returns the number of unique alternative keywords.
func (k *KeywordVariants) Count() int {
	return len(k.GetPrioritizedList())
}

// ToJSON serializes the KeywordVariants to JSON.
func (k *KeywordVariants) ToJSON() (string, error) {
	data, err := json.Marshal(k)
	if err != nil {
		return "", fmt.Errorf("failed to marshal keyword variants: %w", err)
	}
	return string(data), nil
}
