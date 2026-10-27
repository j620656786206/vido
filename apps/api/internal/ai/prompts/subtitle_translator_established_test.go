package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// disc-2026-10-movie-name-drift-in-one-run
func TestEstablishedNames(t *testing.T) {
	assert.Nil(t, EstablishedNames(nil, nil))
	got := EstablishedNames(
		map[string]string{"Thom": "湯姆", "Celia": "西莉亞", " Barley ": " 巴利 ", "Empty": " "},
		[]GlossaryEntry{{Source: "thom", Target: "湯姆"}},
	)
	assert.Equal(t, []GlossaryEntry{{Source: "Barley", Target: "巴利"}, {Source: "Celia", Target: "西莉亞"}}, got,
		"sorted, trimmed, blanks dropped, and nothing the glossary already fixes (case-insensitive)")
}

func TestBuildPrompt_WithoutEstablishedNamesIsUnchanged(t *testing.T) {
	blocks := []SubtitleTranslatorBlock{{Index: 1, Text: "Look Celia."}}
	assert.Equal(t, BuildSubtitleTranslatorPrompt(blocks, nil), BuildSubtitleTranslatorPromptWithGlossary(blocks, nil, EstablishedNames(nil, nil)))
}
