package prompts

import (
	"crypto/sha256"
	"fmt"
	"strconv"
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

// --- sub-1-5a: FR26 metadata section + P11 prompt version ---

func sampleMetadata() MediaMetadata {
	return MediaMetadata{
		Title:         "怪奇物語",
		OriginalTitle: "Stranger Things",
		Year:          2016,
		Genres:        []string{"Drama", "Fantasy", "Mystery"},
		Overview:      "A group of kids\nuncover a secret lab.",
		Cast:          []string{"Winona Ryder", "David Harbour", "Millie Bobby Brown"},
		Countries:     []string{"US"},
	}
}

func TestBuildMetadataSection_ZeroValueYieldsNoSection(t *testing.T) {
	assert.Equal(t, "", BuildMetadataSection(MediaMetadata{}),
		"a zero-value context must render nothing (byte-identical no-metadata path)")

	// Whitespace-only / zero fields are the same as absent.
	assert.Equal(t, "", BuildMetadataSection(MediaMetadata{
		Title:     "   ",
		Overview:  "\n\t",
		Genres:    []string{"", "  "},
		Cast:      []string{" "},
		Countries: []string{""},
	}))
}

func TestBuildMetadataSection_RendersEveryField(t *testing.T) {
	section := BuildMetadataSection(sampleMetadata())

	assert.Contains(t, section, "Media context")
	assert.Contains(t, section, "do NOT translate")
	assert.Contains(t, section, "- Title: 怪奇物語\n")
	assert.Contains(t, section, "- Original title: Stranger Things\n")
	assert.Contains(t, section, "- Year: 2016\n")
	assert.Contains(t, section, "- Genres: Drama, Fantasy, Mystery\n")
	assert.Contains(t, section, "- Production countries: US\n")
	assert.Contains(t, section, "- Cast: Winona Ryder, David Harbour, Millie Bobby Brown\n")
	assert.Contains(t, section, "- Overview: A group of kids uncover a secret lab.\n",
		"a multi-line overview is collapsed so it cannot read as dialogue")
	assert.True(t, strings.HasSuffix(section, "\n\n"), "section ends with a blank separator line")
}

func TestBuildMetadataSection_PartialMetadataOmitsAbsentRows(t *testing.T) {
	section := BuildMetadataSection(MediaMetadata{Title: "Dune", Year: 0})

	assert.Contains(t, section, "- Title: Dune\n")
	assert.NotContains(t, section, "Year:", "a zero year is absent, not 0")
	assert.NotContains(t, section, "Genres:")
	assert.NotContains(t, section, "Cast:")
	assert.NotContains(t, section, "Overview:")
}

func TestBuildMetadataSection_CapsCastAtTen(t *testing.T) {
	cast := make([]string, 0, 15)
	for i := 1; i <= 15; i++ {
		cast = append(cast, "Actor "+strconv.Itoa(i))
	}

	section := BuildMetadataSection(MediaMetadata{Cast: cast})

	assert.Contains(t, section, "Actor 10")
	assert.NotContains(t, section, "Actor 11")
	assert.Equal(t, MetadataCastLimit, strings.Count(section, "Actor "))
}

// TestSubtitleTranslatorPromptVersion_PinsPromptText is the P11 guard: it
// fingerprints every prompt surface in subtitle_translator.go. Editing any of
// them changes the digest, which fails this test and forces the author to bump
// SubtitleTranslatorPromptVersion in the same edit — the alternative is a
// re-run silently serving the previous translation from cache.
func TestSubtitleTranslatorPromptVersion_PinsPromptText(t *testing.T) {
	// The pinned metadata deliberately carries MORE cast entries than
	// MetadataCastLimit, so the cap itself is inside the fingerprint: raising
	// the limit changes real prompts for any show with a large cast, and must
	// fail this pin like any other prompt-surface edit (sub-1-5a CR M1).
	pinned := sampleMetadata()
	pinned.Cast = nil
	for i := 1; i <= MetadataCastLimit+2; i++ {
		pinned.Cast = append(pinned.Cast, "Pin Actor "+strconv.Itoa(i))
	}

	var sb strings.Builder
	sb.WriteString(SubtitleTranslatorSystemPrompt)
	sb.WriteString(BuildGlossarySection([]GlossaryEntry{{Source: "Vecna", Target: "維克那"}}))
	sb.WriteString(BuildMetadataSection(pinned))
	sb.WriteString(BuildSubtitleTranslatorPrompt(
		[]SubtitleTranslatorBlock{{Index: 2, Text: "Hello"}},
		[]SubtitleTranslatorBlock{{Index: 1, Text: "Hi"}},
	))
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(sb.String())))

	assert.Equal(t, "m1-v2", SubtitleTranslatorPromptVersion)
	assert.Equal(t, "dd8e754fa4bf2abaea2cf148df69ce0c52c21b8b451adb309a098c682b70c7b8", digest,
		"prompt text changed — bump SubtitleTranslatorPromptVersion and update this digest in the SAME edit (P11)")
}
