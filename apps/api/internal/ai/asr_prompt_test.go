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

	long := []string{strings.Repeat("x", 195), "Kofun", "Haniwa"}
	assert.Equal(t, strings.Repeat("x", 195), BuildASRPrompt(long), "the rune cap stops before the next name would overflow")
	assert.Equal(t, "Kofun, Haniwa", BuildASRPrompt([]string{"Kofun", strings.Repeat("y", 300), "Haniwa"}),
		"CR 4: one over-long phrase is skipped, the short names behind it still go")
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

// disc-2026-10-asr-name-prompt-too-long: the list read back out of order /
// with a dangling fragment is still an echo.
func TestFilterPromptEcho_NameListOutOfOrder(t *testing.T) {
	prompt := "Baba Voss, Maghra, Paris, Jerlamarel, Kofun, Haniwa, Tamacti Jun, Lord Harlan, Oloman, Shiloh, The Bank, Lord Diego"
	segs := []whisperSegment{
		{Start: 0, End: 2, Text: "The Bank, Lord Diego, Oloman, Shiloh, Lord Harlan, The Bank, Lord"}, // reordered, repeated, cut off
		{Start: 2, End: 3, Text: "Maghra, Kofun"},                                                     // two names → echo
		{Start: 3, End: 4, Text: "Kofun, Haniwa, Tamacti"},                                            // cut mid-name → echo
		{Start: 4, End: 5, Text: "Kofun, come here, Haniwa, now, please"},                             // free words — dialogue
		{Start: 5, End: 6, Text: "Baba Voss, Maghra, come."},                                          // CR 1: two names + a verb — dialogue
		{Start: 6, End: 7, Text: "Kofun, Haniwa, run!"},                                               // dialogue
		{Start: 7, End: 8, Text: "Jerlamarel's children, Kofun, Haniwa."},                             // dialogue
		{Start: 8, End: 9, Text: "Maghra."},                                                           // one name — a line
	}
	kept, dropped := filterPromptEcho(segs, prompt)
	assert.Len(t, dropped, 3)
	var texts []string
	for _, k := range kept {
		texts = append(texts, k.Text)
	}
	assert.Equal(t, []string{"Kofun, come here, Haniwa, now, please", "Baba Voss, Maghra, come.", "Kofun, Haniwa, run!", "Jerlamarel's children, Kofun, Haniwa.", "Maghra."}, texts)
}
