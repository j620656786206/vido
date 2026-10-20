package sidecarcheck

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func repeat(line string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = line
	}
	return out
}

func TestEnglishVerdict(t *testing.T) {
	ok, why := englishVerdict(repeat("You're a jerk, Thom.", 30))
	assert.True(t, ok, why)

	ok, why = englishVerdict(repeat("Hello", 19))
	assert.False(t, ok)
	assert.Equal(t, "too_few_cues", why)

	ok, why = englishVerdict(repeat("你有你的機器人學，我只是想在太空裡耍帥而已。", 30))
	assert.False(t, ok)
	assert.Equal(t, "not_latin", why)

	ok, why = englishVerdict(repeat("Je ne sais pas pourquoi tu es là.", 30))
	assert.False(t, ok, "French is Latin script but not English")
	assert.Equal(t, "not_english", why)

	// Latin-script neighbours that share short words with English.
	ok, why = englishVerdict(repeat("¿Qué quieres que te diga? Me voy a casa.", 30))
	assert.False(t, ok, "Spanish")
	assert.Equal(t, "not_english", why)
	ok, _ = englishVerdict(repeat("Ik weet het niet, we gaan naar huis in de auto.", 30))
	assert.False(t, ok, "Dutch")
	ok, _ = englishVerdict(repeat("Ich weiß nicht, was du willst.", 30))
	assert.False(t, ok, "German")

	// Curly apostrophes count: don’t, it’s.
	ok, _ = englishVerdict(repeat("Don’t move. It’s over.", 30))
	assert.True(t, ok)

	// A real file mixes in music cues and names; a third with function words is enough.
	mixed := append(repeat("♪ ♪", 20), repeat("Where is the ship?", 10)...)
	ok, _ = englishVerdict(mixed)
	assert.True(t, ok, strings.Join(mixed[:2], "|"))
}
