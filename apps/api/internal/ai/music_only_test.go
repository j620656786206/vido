package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMusicOnlyText(t *testing.T) {
	for _, s := range []string{"♪", "♪♪", "♪ ♪", "♫", "♬", "#", " ♪♪ ", "\u200b♪\ufeff", "♪\n♪", "-♪♪", "- ♪♪", "—♪", "♪♪-"} {
		assert.True(t, IsMusicOnlyText(s), "%q", s)
	}
	for _, s := range []string{"", "   ", "-", "- -", "♪ lyrics ♪", "Hello", "♪ 1", "# comment", "-Hello"} {
		assert.False(t, IsMusicOnlyText(s), "%q", s)
	}
}

// disc-2026-10-asr-music-only-cues — See S01E02's speech-recognition run shipped
// 22 cues that said only ♪♪; they were paid for in translation and floated on
// screen over score. The ASR leg now drops them exactly as FilterSDH does for
// an embedded track.
func TestFilterHallucinations_R0MusicOnly(t *testing.T) {
	segs := []whisperSegment{
		speech(1, 7, "♪♪"),
		speech(8, 9, "Who's there?"),
		speech(10, 12, "♪ ♪"),
		speech(13, 14, "♪ Happy birthday to you ♪"), // lyrics are words — kept
		speech(15, 16, "Shh."),
	}
	kept, dropped := filterHallucinations(segs)

	require.Len(t, dropped, 2)
	for _, d := range dropped {
		assert.Equal(t, dropReasonMusicOnly, d.Reason)
	}
	require.Len(t, kept, 3)
	assert.Equal(t, "Who's there?", kept[0].Text)
	assert.Equal(t, "♪ Happy birthday to you ♪", kept[1].Text)
	assert.Equal(t, "Shh.", kept[2].Text)
}

func TestFilterHallucinations_MusicOnlyRunIsNotCountedAsARepeatRun(t *testing.T) {
	// Five ♪♪ in a row used to be dropped as repeat_run after the first; now
	// every one is music_only — the log tells the truth about why.
	segs := []whisperSegment{speech(1, 2, "♪♪"), speech(3, 4, "♪♪"), speech(5, 6, "♪♪"), speech(7, 8, "♪♪"), speech(9, 10, "Hi")}
	kept, dropped := filterHallucinations(segs)
	require.Len(t, kept, 1)
	assert.Equal(t, 4, dropReasonCounts(dropped)[dropReasonMusicOnly])
	assert.Zero(t, dropReasonCounts(dropped)[dropReasonRepeatRun])
}

// disc-2026-10-asr-symbol-only-cues
func TestIsSymbolOnlyText(t *testing.T) {
	for _, s := range []string{"$%&!", "-$%&!", "- $%&!", "...", "?!", "#$%&!", "\u200b$%&!"} {
		assert.True(t, IsSymbolOnlyText(s), "%q", s)
	}
	for _, s := range []string{"", "   ", "Ugh!", "3!", "啊！", "-Hello", "♪ lyrics ♪", "$5"} {
		assert.False(t, IsSymbolOnlyText(s), "%q", s)
	}
}

// See S01E02's 10-minute clip (2026-10-11): whisper wrote shouting during a
// fight as 「$%&!」 three times, and translation copied it into the
// subtitle as 「$%&！」. A line with no letter and no digit is not a subtitle.
func TestFilterHallucinations_R0SymbolOnly(t *testing.T) {
	segs := []whisperSegment{
		speech(263.886, 264.166, "-$%&!"),
		speech(264.546, 264.686, "$%&!"),
		speech(265, 267, "Face me!"),
		speech(268, 269, "♪♪"),
		speech(270, 271, "3!"),
		speech(272, 273, "Ugh!"),
		speech(284.566, 285.966, "$%&!"),
	}
	kept, dropped := filterHallucinations(segs)

	counts := dropReasonCounts(dropped)
	assert.Equal(t, 3, counts[dropReasonSymbolOnly])
	assert.Equal(t, 1, counts[dropReasonMusicOnly], "♪♪ stays music_only — the log tells the two apart")
	require.Len(t, kept, 3)
	assert.Equal(t, []string{"Face me!", "3!", "Ugh!"}, []string{kept[0].Text, kept[1].Text, kept[2].Text})
}
