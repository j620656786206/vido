package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vido/api/internal/ai"
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
		"Baba Voss", "Paris", "Maghra", // characters first, billing order — actors are never spoken, so never sent
		"Jerlamarel", "Kofun", "Payan", // then trusted glossary sources
	}, got)
	assert.NotContains(t, got, "Chola Morel")
	assert.NotContains(t, got, "Durla Morel")
}

func TestASRPromptNames_NothingKnownIsEmpty(t *testing.T) {
	assert.Empty(t, asrPromptNames(nil, nil))
	assert.Empty(t, asrPromptNames(&models.Credits{}, []models.GlossaryTerm{{TermSrc: "X", Source: models.GlossarySourceSubtitle}}))
}

func TestPromptCharacterName(t *testing.T) {
	assert.Equal(t, "Baba Voss", promptCharacterName("Baba Voss (voice)"))
	assert.Equal(t, "Jerlamarel", promptCharacterName("  Jerlamarel "))
	for _, skip := range []string{"Self", "Himself", "Herself", "Narrator (voice)", ""} {
		assert.Empty(t, promptCharacterName(skip), "%q is not a name anyone says", skip)
	}
}

// asrPromptFor's lookup glue (CR M1: the series case used to be unreachable).
func TestASRPromptFor_ReadsCreditsPerMediaType(t *testing.T) {
	show := &models.Series{Credits: &models.Credits{Cast: []models.CastMember{{Name: "Jason Momoa", Character: "Baba Voss"}}}}
	movie := &models.Movie{Credits: &models.Credits{Cast: []models.CastMember{{Name: "Tom Hanks", Character: "Chuck Noland"}}}}

	t.Run("movie: its own credits", func(t *testing.T) {
		svc := NewTranscriptionService(nil, nil, nil, nil)
		svc.SetSubtitleStateReader(&fakeStateReader{movie: movie})
		assert.Equal(t, "Chuck Noland", svc.asrPromptFor(context.Background(), models.SubtitleRunMediaMovie, "mv-1", "mv-1"))
	})
	t.Run("episode: the parent series' credits", func(t *testing.T) {
		svc := NewTranscriptionService(nil, nil, nil, nil)
		reader := &metadataSeriesReader{series: show}
		svc.SetSeriesMetadataReader(reader)
		assert.Equal(t, "Baba Voss", svc.asrPromptFor(context.Background(), models.SubtitleRunMediaEpisode, "ep-1", "series-1"))
		assert.Equal(t, "series-1", reader.lastID)
	})
	t.Run("series run: its own credits (glossaryKey == mediaID is NOT a miss here)", func(t *testing.T) {
		svc := NewTranscriptionService(nil, nil, nil, nil)
		reader := &metadataSeriesReader{series: show}
		svc.SetSeriesMetadataReader(reader)
		assert.Equal(t, "Baba Voss", svc.asrPromptFor(context.Background(), models.SubtitleRunMediaSeries, "series-1", "series-1"))
	})
	t.Run("episode whose parent is unresolved: no series query, no prompt", func(t *testing.T) {
		svc := NewTranscriptionService(nil, nil, nil, nil)
		reader := &metadataSeriesReader{series: show}
		svc.SetSeriesMetadataReader(reader)
		assert.Empty(t, svc.asrPromptFor(context.Background(), models.SubtitleRunMediaEpisode, "ep-1", "ep-1"))
		assert.Zero(t, reader.callCount)
	})
	t.Run("lookup failure: no prompt, no panic", func(t *testing.T) {
		svc := NewTranscriptionService(nil, nil, nil, nil)
		svc.SetSeriesMetadataReader(&metadataSeriesReader{err: errors.New("db down")})
		assert.Empty(t, svc.asrPromptFor(context.Background(), models.SubtitleRunMediaEpisode, "ep-1", "series-1"))
	})
}

// disc-2026-10-asr-name-prompt-too-long: TMDb hands back zh-TW spellings for
// a zh-TW library; those are not words in an English soundtrack.
func TestASRPromptNames_SkipsNonLatinNames(t *testing.T) {
	credits := &models.Credits{Cast: []models.CastMember{
		{Name: "傑森·摩莫亞", Character: "Baba Voss"},
		{Name: "阿爾法·伍達德", Character: "帕里斯"},
	}}
	terms := []models.GlossaryTerm{{TermSrc: "謝拉馬威", Source: models.GlossarySourceManual}, {TermSrc: "Jerlamarel", Source: models.GlossarySourceManual}}
	assert.Equal(t, []string{"Baba Voss", "Jerlamarel"}, asrPromptNames(credits, terms))
	assert.False(t, isLatinName("·"), "punctuation alone is not a name")
}

// CR 2: TMDb's zh-TW credits may translate a main character; the metadata
// glossary carries the reverse pair, so billing order survives and the actor
// pairs in the same glossary are not mistaken for words anyone says.
func TestASRPromptNames_RecoversEnglishFromZhCredits(t *testing.T) {
	credits := &models.Credits{Cast: []models.CastMember{
		{Name: "傑森·摩莫亞", Character: "Baba Voss"},
		{Name: "阿爾法·伍達德", Character: "帕里斯"},
		{Name: "赫拉·希爾瑪", Character: "瑪格拉 / Young Maghra"},
		{Name: "某人", Character: "The Bank"},
	}}
	terms := []models.GlossaryTerm{
		{TermSrc: "Paris", TermZh: "帕里斯", Source: models.GlossarySourceMetadata},
		{TermSrc: "Maghra", TermZh: "瑪格拉", Source: models.GlossarySourceMetadata},
		{TermSrc: "Jason Momoa", TermZh: "傑森·摩莫亞", Source: models.GlossarySourceMetadata}, // actor pair — skipped
		{TermSrc: "Alfre Woodard", TermZh: "阿爾法·伍達德", Source: models.GlossarySourceMetadata},
		{TermSrc: "Jerlamarel", TermZh: "謝拉馬威", Source: models.GlossarySourceOfficialSubtitle},
	}
	assert.Equal(t, []string{"Baba Voss", "Paris", "Maghra", "The Bank", "Paris", "Maghra", "Jerlamarel"}, asrPromptNames(credits, terms),
		"billing order first (duplicates are folded by BuildASRPrompt), then the trusted glossary without actor pairs")
	assert.Equal(t, "Baba Voss, Paris, Maghra, The Bank, Jerlamarel", ai.BuildASRPrompt(asrPromptNames(credits, terms)))
	for _, ok := range []string{"Zoë", "O'Neil", "Jean-Luc", "Tamacti Jun", "Ægir"} {
		assert.True(t, isLatinName(ok), ok)
	}
	for _, no := range []string{"Иван", "ジョン", "帕里斯", "·", "123"} {
		assert.False(t, isLatinName(no), no)
	}
}
