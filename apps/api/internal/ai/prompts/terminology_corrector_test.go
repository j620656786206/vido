package prompts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTerminologyCorrectorPrompt(t *testing.T) {
	content := "這個軟件很好用"
	prompt := BuildTerminologyCorrectorPrompt(content)

	assert.Contains(t, prompt, content, "prompt should contain the subtitle content")
	assert.Contains(t, prompt, "cross-strait terminology", "prompt should mention correction task")
}

func TestTerminologyCorrectorSystemPrompt_ContainsExamples(t *testing.T) {
	// System prompt should contain the key term mappings
	examples := []string{
		"軟件 → 軟體",
		"內存 → 記憶體",
		"數據 → 資料",
		"視頻 → 影片",
		"網絡 → 網路",
		"信息 → 資訊",
		"硬盤 → 硬碟",
		"默認 → 預設",
	}

	for _, ex := range examples {
		assert.Contains(t, TerminologyCorrectorSystemPrompt, ex, "system prompt should contain example: %s", ex)
	}
}

func TestTerminologyCorrectorSystemPrompt_PreservationRules(t *testing.T) {
	// Should instruct preservation of proper nouns
	assert.Contains(t, TerminologyCorrectorSystemPrompt, "人名")
	assert.Contains(t, TerminologyCorrectorSystemPrompt, "地名")
	assert.Contains(t, TerminologyCorrectorSystemPrompt, "作品名")

	// Should instruct preservation of formatting
	assert.Contains(t, TerminologyCorrectorSystemPrompt, "formatting")
	assert.Contains(t, TerminologyCorrectorSystemPrompt, "whitespace")
}

func TestTerminologyCorrectorSystemPrompt_NoRewriting(t *testing.T) {
	// Should explicitly disallow sentence rewriting
	prompt := strings.ToLower(TerminologyCorrectorSystemPrompt)
	assert.Contains(t, prompt, "not rewrite")
	assert.Contains(t, prompt, "only substitute")
}

func TestKnownTerminologyPairs_MinimumCount(t *testing.T) {
	// AC #1.3: at least 10 test pairs
	assert.GreaterOrEqual(t, len(KnownTerminologyPairs), 10, "must have at least 10 test pairs")
}

func TestKnownTerminologyPairs_HaveTechTerms(t *testing.T) {
	techCount := 0
	for _, pair := range KnownTerminologyPairs {
		if pair.Category == "tech" {
			techCount++
		}
	}
	assert.GreaterOrEqual(t, techCount, 5, "should have at least 5 tech term pairs")
}

func TestKnownTerminologyPairs_HaveProperNounPreservation(t *testing.T) {
	properNounCount := 0
	for _, pair := range KnownTerminologyPairs {
		if pair.Category == "proper-noun" {
			properNounCount++
			// For proper nouns, input should equal expected (no change)
			assert.Equal(t, pair.Input, pair.Expected,
				"proper noun pair should have identical input and expected: %s", pair.Input)
		}
	}
	assert.GreaterOrEqual(t, properNounCount, 2, "should have at least 2 proper noun preservation test pairs")
}

func TestKnownTerminologyPairs_InputDiffersFromExpected(t *testing.T) {
	// For non-proper-noun pairs, input should differ from expected
	for _, pair := range KnownTerminologyPairs {
		if pair.Category != "proper-noun" {
			assert.NotEqual(t, pair.Input, pair.Expected,
				"non-proper-noun pair should have different input and expected: %s", pair.Input)
		}
	}
}

func TestKnownTerminologyPairs_AllHaveCategories(t *testing.T) {
	for _, pair := range KnownTerminologyPairs {
		assert.NotEmpty(t, pair.Input, "input should not be empty")
		assert.NotEmpty(t, pair.Expected, "expected should not be empty")
		assert.NotEmpty(t, pair.Category, "category should not be empty")
	}
}

func TestTerminologyCorrectorSystemPrompt_ReturnTextOnly(t *testing.T) {
	// Should instruct to return only corrected text, no explanations
	assert.Contains(t, TerminologyCorrectorSystemPrompt, "ONLY the corrected text")
	assert.Contains(t, TerminologyCorrectorSystemPrompt, "no explanations")
}
