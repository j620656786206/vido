package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-09-whisper-credit-line-hallucination — whisper's known habit of
// "hearing" a subtitle group's signature over credits music.
func TestIsCreditLineHallucination(t *testing.T) {
	for _, s := range []string{
		"Subs by www.zeoranger.co.uk", // HP4 2:29:54, NAS 2026-09-18
		"Subtitles by the Amara.org community",
		"subs by",
		"- Subs by someone",
		"Captioning by CaptionMax",
		"字幕由Amara.org社区提供",
		"请不吝点赞 订阅 转发 打赏支持明镜与点点栏目",
		"中文字幕志愿者 李宗盛",
	} {
		assert.True(t, isCreditLineHallucination(s), "%q", s)
	}
	for _, s := range []string{
		"",
		"Thanks for watching.",
		"Who subs by the river?",
		"He subtitles books for a living.",
		"Subsidies by the state are gone.",
		"字幕呢？",
	} {
		assert.False(t, isCreditLineHallucination(s), "%q", s)
	}
}

// The HP4 line came back confident and alone — no score rule and no tail rule
// (it needs three lines in a row) could see it. Run with the tail rule off, as
// on a mid-file chunk, so only the new rule can be the one that drops it.
func TestFilterHallucinations_R0CreditLine(t *testing.T) {
	segs := []whisperSegment{
		speech(8580, 8582, "It's not over."),
		speech(8583, 8585, "He subtitles books for a living."),
		speech(8994, 8997, "Subs by www.zeoranger.co.uk"),
	}
	segs[2].NoSpeechProb, segs[2].AvgLogprob = 0.02, -0.1 // a confident line

	kept, dropped := filterHallucinationsWith(segs, false, "")

	require.Len(t, dropped, 1)
	assert.Equal(t, dropReasonCreditLine, dropped[0].Reason)
	assert.Equal(t, "Subs by www.zeoranger.co.uk", dropped[0].Segment.Text)
	require.Len(t, kept, 2)
	assert.Equal(t, "It's not over.", kept[0].Text)
	assert.Equal(t, "He subtitles books for a living.", kept[1].Text)
}
