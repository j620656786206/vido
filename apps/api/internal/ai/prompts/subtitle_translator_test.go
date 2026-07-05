package prompts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSubtitleTranslatorPrompt(t *testing.T) {
	blocks := []SubtitleTranslatorBlock{
		{Index: 1, Text: "Hello, how are you?"},
		{Index: 2, Text: "I'm doing fine, thanks."},
	}

	context := []SubtitleTranslatorBlock{
		{Index: 0, Text: "Previously on the show..."},
	}

	prompt := BuildSubtitleTranslatorPrompt(blocks, context)

	// Must contain the block text
	if !strings.Contains(prompt, "Hello, how are you?") {
		t.Error("prompt should contain block text")
	}
	if !strings.Contains(prompt, "I'm doing fine, thanks.") {
		t.Error("prompt should contain second block text")
	}

	// Must contain context blocks
	if !strings.Contains(prompt, "Previously on the show...") {
		t.Error("prompt should contain context blocks")
	}

	// Must contain block indices
	if !strings.Contains(prompt, "[1]") {
		t.Error("prompt should contain block index [1]")
	}
	if !strings.Contains(prompt, "[2]") {
		t.Error("prompt should contain block index [2]")
	}
}

func TestBuildSubtitleTranslatorPrompt_NoContext(t *testing.T) {
	blocks := []SubtitleTranslatorBlock{
		{Index: 1, Text: "Hello world."},
	}

	prompt := BuildSubtitleTranslatorPrompt(blocks, nil)

	if !strings.Contains(prompt, "Hello world.") {
		t.Error("prompt should contain block text")
	}
	// Should not have context section header when no context
	if strings.Contains(prompt, "Previous context") {
		t.Error("prompt should not contain context section when no context provided")
	}
}

func TestBuildSubtitleTranslatorPrompt_EmptyBlocks(t *testing.T) {
	prompt := BuildSubtitleTranslatorPrompt(nil, nil)
	if prompt == "" {
		t.Error("prompt should not be empty even with no blocks")
	}
}

func TestSubtitleTranslatorSystemPrompt(t *testing.T) {
	if SubtitleTranslatorSystemPrompt == "" {
		t.Fatal("system prompt must not be empty")
	}

	// Must mention Traditional Chinese / Taiwan
	if !strings.Contains(SubtitleTranslatorSystemPrompt, "Traditional Chinese") {
		t.Error("system prompt must mention Traditional Chinese")
	}
	if !strings.Contains(SubtitleTranslatorSystemPrompt, "Taiwan") {
		t.Error("system prompt must mention Taiwan")
	}

	// Must instruct to preserve proper nouns
	lower := strings.ToLower(SubtitleTranslatorSystemPrompt)
	if !strings.Contains(lower, "proper noun") {
		t.Error("system prompt must instruct proper noun preservation")
	}

	// Must instruct to preserve speaker tone
	if !strings.Contains(lower, "tone") {
		t.Error("system prompt must instruct tone preservation")
	}
}

func TestSubtitleTranslatorContextWindow(t *testing.T) {
	// Verify the context window constant is 5 (per story spec)
	if SubtitleTranslatorContextWindow != 5 {
		t.Errorf("context window should be 5, got %d", SubtitleTranslatorContextWindow)
	}
}

func TestSubtitleTranslatorBatchSize(t *testing.T) {
	// Verify batch size is 10 (per story spec)
	if SubtitleTranslatorBatchSize != 10 {
		t.Errorf("batch size should be 10, got %d", SubtitleTranslatorBatchSize)
	}
}

func TestBuildSubtitleTranslatorPrompt_MultiLineBlock(t *testing.T) {
	blocks := []SubtitleTranslatorBlock{
		{Index: 1, Text: "Line one\nLine two"},
		{Index: 2, Text: "Single line"},
	}

	prompt := BuildSubtitleTranslatorPrompt(blocks, nil)

	// Multi-line text should be preserved in prompt
	if !strings.Contains(prompt, "Line one\nLine two") {
		t.Error("prompt should preserve multi-line block text")
	}
	if !strings.Contains(prompt, "[1]") && !strings.Contains(prompt, "[2]") {
		t.Error("prompt should contain block indices")
	}
}

func TestSubtitleTranslatorSystemPrompt_MultiLineInstructions(t *testing.T) {
	// System prompt must instruct Claude on multi-line block handling
	if !strings.Contains(SubtitleTranslatorSystemPrompt, "multi-line") {
		t.Error("system prompt must mention multi-line block handling")
	}
}

// --- 9R-7: glossary section ---

func TestBuildGlossarySection(t *testing.T) {
	assert.Equal(t, "", BuildGlossarySection(nil), "empty glossary yields no section (no-regression)")

	section := BuildGlossarySection([]GlossaryEntry{
		{Source: "Demogorgon", Target: "魔王獸"},
		{Source: "Vecna", Target: "維克那"},
	})
	assert.Contains(t, section, "Glossary")
	assert.Contains(t, section, "Demogorgon → 魔王獸")
	assert.Contains(t, section, "Vecna → 維克那")
}

func TestBuildSubtitleTranslatorPrompt_NoGlossaryUnchanged(t *testing.T) {
	blocks := []SubtitleTranslatorBlock{{Index: 1, Text: "Hello"}}
	base := BuildSubtitleTranslatorPrompt(blocks, nil)
	withNil := BuildSubtitleTranslatorPromptWithGlossary(blocks, nil, nil)
	assert.Equal(t, base, withNil, "nil glossary must produce byte-identical prompt")
	assert.NotContains(t, base, "Glossary")
}

func TestBuildSubtitleTranslatorPromptWithGlossary(t *testing.T) {
	blocks := []SubtitleTranslatorBlock{{Index: 1, Text: "The Demogorgon"}}
	p := BuildSubtitleTranslatorPromptWithGlossary(blocks, nil, []GlossaryEntry{{Source: "Demogorgon", Target: "魔王獸"}})
	assert.Contains(t, p, "Demogorgon → 魔王獸")
	// Glossary must come before the translate section.
	assert.Less(t, strings.Index(p, "Glossary"), strings.Index(p, "Translate the following"))
}
