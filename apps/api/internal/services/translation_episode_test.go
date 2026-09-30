package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vido/api/internal/ai/prompts"
)

// sub-7-2a AC #3 on the ASR leg: the per-episode line is appended AFTER the
// per-show section, in the same order the extract leg's third system block
// takes, and a blank label leaves the prompt byte-identical.
func TestComposeSystemPrompt_EpisodeLineComesLast(t *testing.T) {
	md := prompts.MediaMetadata{Title: "Buffy the Vampire Slayer", Year: 1997}
	level := prompts.DefaultLocalizationLevel

	plain := composeSystemPrompt(md, level, "")
	withEpisode := composeSystemPrompt(md, level, "S05E16 · The Body")

	assert.Equal(t, prompts.ComposeInvariantSystemPrompt(level)+"\n\n"+prompts.BuildMetadataSection(md), plain,
		"no label → exactly the pre-sub-7-2a prompt")
	assert.True(t, strings.HasPrefix(withEpisode, plain), "the episode line only ever APPENDS")
	assert.True(t, strings.HasSuffix(withEpisode, prompts.BuildEpisodeSection("S05E16 · The Body")))
	assert.NotContains(t, prompts.BuildMetadataSection(md), "The Body")
}

func TestWithEpisodeLabel_IsNotPartOfTheRunVersion(t *testing.T) {
	cfg := newTranslateConfig([]TranslateOption{WithEpisodeLabel("S01E01 · Pilot")})
	assert.Equal(t, "S01E01 · Pilot", cfg.episode)
	assert.Equal(t, prompts.MediaMetadata{}, cfg.metadata, "the label lives beside the metadata, never inside it")
}
