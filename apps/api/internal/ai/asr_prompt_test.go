package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestASRPrompt_ContextRoundTrip(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, ASRPromptFromContext(ctx))
	assert.Equal(t, ctx, WithASRPrompt(ctx, "   "), "an empty prompt leaves the ctx untouched")
	ctx = WithASRPrompt(ctx, "Jerlamarel, Baba Voss")
	assert.Equal(t, "Jerlamarel, Baba Voss", ASRPromptFromContext(ctx))
}

func TestBuildASRPrompt(t *testing.T) {
	assert.Empty(t, BuildASRPrompt(nil))
	assert.Empty(t, BuildASRPrompt([]string{"", "  "}))
	assert.Equal(t, "Jerlamarel, Baba Voss, Paris",
		BuildASRPrompt([]string{" Jerlamarel ", "Baba Voss", "baba voss", "Paris", "PARIS"}),
		"trimmed, de-duplicated case-insensitively, first spelling and order kept")

	many := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		many = append(many, strings.Repeat("N", 5)+string(rune('A'+i%26))+string(rune('a'+i/26)))
	}
	got := BuildASRPrompt(many)
	assert.LessOrEqual(t, len(strings.Split(got, ", ")), ASRPromptMaxNames)
	assert.LessOrEqual(t, len([]rune(got)), ASRPromptMaxRunes)
	assert.True(t, strings.HasPrefix(got, many[0]), "the FRONT of the list survives — callers put the important names first")

	long := []string{strings.Repeat("x", 695), "Kofun", "Haniwa"}
	assert.Equal(t, strings.Repeat("x", 695), BuildASRPrompt(long), "the rune cap stops before the next name would overflow")
}

func TestFilterPromptEcho(t *testing.T) {
	prompt := "Baba Voss, Paris, Maghra, Jerlamarel, Kofun, Haniwa"
	segs := []whisperSegment{
		{Start: 0, End: 2, Text: "Baba Voss, Paris, Maghra"},   // the prompt read back over silence
		{Start: 2, End: 4, Text: "Jerlamarel! Kofun, Haniwa."}, // another slice, different punctuation
		{Start: 4, End: 5, Text: "Paris?"},                     // one name is a line someone says — kept
		{Start: 5, End: 8, Text: "Speak to Paris."},            // dialogue that mentions a name — kept
		{Start: 8, End: 9, Text: "Maghra, no!"},                // kept: not a prompt slice
	}
	kept, dropped := filterPromptEcho(segs, prompt)
	assert.Len(t, dropped, 2)
	for _, d := range dropped {
		assert.Equal(t, dropReasonPromptEcho, d.Reason)
	}
	assert.Equal(t, []string{"Paris?", "Speak to Paris.", "Maghra, no!"}, []string{kept[0].Text, kept[1].Text, kept[2].Text})

	same, none := filterPromptEcho(segs, "")
	assert.Nil(t, none)
	assert.Equal(t, segs, same, "no prompt → untouched")
}
