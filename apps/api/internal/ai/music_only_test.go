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
