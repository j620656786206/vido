package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vido/api/internal/models"
)

// disc-2026-10-asr-proper-names-inconsistent — what speech recognition is told
// to spell, and what it must never be told.
func TestASRPromptNames(t *testing.T) {
	credits := &models.Credits{Cast: []models.CastMember{
		{Name: "Jason Momoa", Character: "Baba Voss", Order: 0},
		{Name: "Alfre Woodard", Character: "Paris", Order: 1},
		{Name: "Hera Hilmar", Character: "Maghra", Order: 2},
		{Name: "Extra Person", Character: "", Order: 3},
	}}
	terms := []models.GlossaryTerm{
		{TermSrc: "Jerlamarel", TermZh: "謝拉馬威", Source: models.GlossarySourceOfficialSubtitle},
		{TermSrc: "Kofun", TermZh: "柯方", Source: models.GlossarySourceManual},
		{TermSrc: "Payan", TermZh: "帕揚", Source: models.GlossarySourceMetadata},
		// The LAST ASR run's own garble — must not be fed back.
		{TermSrc: "Chola Morel", TermZh: "丘拉·莫瑞爾", Source: models.GlossarySourceSubtitle},
		{TermSrc: "Durla Morel", TermZh: "杜拉·莫瑞爾", Source: models.GlossarySourceSubtitle},
	}

	got := asrPromptNames(credits, terms)

	assert.Equal(t, []string{
		"Baba Voss", "Paris", "Maghra", // characters first, billing order
		"Jason Momoa", "Alfre Woodard", "Hera Hilmar", "Extra Person", // then actors
		"Jerlamarel", "Kofun", "Payan", // then trusted glossary sources
	}, got)
	assert.NotContains(t, got, "Chola Morel")
	assert.NotContains(t, got, "Durla Morel")
}

func TestASRPromptNames_NothingKnownIsEmpty(t *testing.T) {
	assert.Empty(t, asrPromptNames(nil, nil))
	assert.Empty(t, asrPromptNames(&models.Credits{}, []models.GlossaryTerm{{TermSrc: "X", Source: models.GlossarySourceSubtitle}}))
}
