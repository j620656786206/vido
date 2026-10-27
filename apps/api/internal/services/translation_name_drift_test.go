package services

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
)

// ─── disc-2026-10-movie-name-drift-in-one-run ───────────────────────────────

func celiaBlocks(n int) []TranslationBlock {
	out := make([]TranslationBlock, n)
	for i := range out {
		out[i] = TranslationBlock{Index: i + 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "Look Celia."}
	}
	return out
}

func TestTranslateWithGlossaryHarvest_LaterBatchesGetTheNamesEarlierBatchesSettledOn(t *testing.T) {
	mock := &mockTranslationCompleter{response: "[1] 聽著西莉亞\n[2] 聽著西莉亞\n[3] 聽著西莉亞\n===TERMS===\nCelia=>西莉亞"}
	svc := NewTranslationService(mock, nil)

	_, _, _, err := svc.TranslateWithGlossaryHarvest(context.Background(), celiaBlocks(3), nil, nil, WithBatchSize(1))
	require.NoError(t, err)
	require.Len(t, mock.calls, 3)

	assert.NotContains(t, mock.calls[0].UserPrompt, "Celia → 西莉亞", "nothing settled before the first batch")
	assert.Contains(t, mock.calls[1].UserPrompt, "- Celia → 西莉亞", "batch 2 keeps batch 1's rendering")
	assert.Contains(t, mock.calls[2].UserPrompt, "- Celia → 西莉亞")
}

func TestTranslateWithGlossaryHarvest_TheRunsGlossaryStillWins(t *testing.T) {
	mock := &mockTranslationCompleter{response: "[1] 聽著西莉亞\n[2] 聽著西莉亞\n===TERMS===\nCelia=>西莉亞"}
	svc := NewTranslationService(mock, nil)

	_, _, _, err := svc.TranslateWithGlossaryHarvest(context.Background(), celiaBlocks(2),
		[]GlossaryPair{{Source: "Celia", Target: "希莉亞"}}, nil, WithBatchSize(1))
	require.NoError(t, err)
	require.Len(t, mock.calls, 2)
	assert.Contains(t, mock.calls[1].UserPrompt, "- Celia → 希莉亞")
	assert.NotContains(t, mock.calls[1].UserPrompt, "- Celia → 西莉亞", "the harvested rendering does not overrule the glossary")
	assert.Equal(t, 1, strings.Count(mock.calls[1].UserPrompt, "- Celia →"))
}

func TestTranslateChunk_EstablishedNamesRideTheUserMessage(t *testing.T) {
	mock := &cachingTranslationMock{result: ai.CompletionResult{Text: "[1] 聽著西莉亞"}}
	svc := NewTranslationService(mock, nil)

	_, _, _, err := svc.TranslateChunk(context.Background(), chunkSystemBlocks(), nil,
		[]prompts.SubtitleTranslatorBlock{{Index: 1, Text: "Look Celia."}},
		[]prompts.GlossaryEntry{{Source: "Celia", Target: "西莉亞"}})
	require.NoError(t, err)
	require.Len(t, mock.reqs, 1)
	assert.Contains(t, mock.reqs[0].UserPrompt, "- Celia → 西莉亞")
	assert.Equal(t, chunkSystemBlocks(), mock.reqs[0].System, "the cached system blocks are untouched")
}
