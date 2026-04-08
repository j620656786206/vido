package prompts

import (
	"strings"
	"testing"
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
