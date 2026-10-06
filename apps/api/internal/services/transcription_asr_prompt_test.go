package services

import (
	"context"
	"errors"
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
		assert.Equal(t, "Chuck Noland, Tom Hanks", svc.asrPromptFor(context.Background(), models.SubtitleRunMediaMovie, "mv-1", "mv-1"))
	})
	t.Run("episode: the parent series' credits", func(t *testing.T) {
		svc := NewTranscriptionService(nil, nil, nil, nil)
		reader := &metadataSeriesReader{series: show}
		svc.SetSeriesMetadataReader(reader)
		assert.Equal(t, "Baba Voss, Jason Momoa", svc.asrPromptFor(context.Background(), models.SubtitleRunMediaEpisode, "ep-1", "series-1"))
		assert.Equal(t, "series-1", reader.lastID)
	})
	t.Run("series run: its own credits (glossaryKey == mediaID is NOT a miss here)", func(t *testing.T) {
		svc := NewTranscriptionService(nil, nil, nil, nil)
		reader := &metadataSeriesReader{series: show}
		svc.SetSeriesMetadataReader(reader)
		assert.Equal(t, "Baba Voss, Jason Momoa", svc.asrPromptFor(context.Background(), models.SubtitleRunMediaSeries, "series-1", "series-1"))
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
