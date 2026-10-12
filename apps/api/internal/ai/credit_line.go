package ai

import "strings"

// Whisper learned from subtitle files that end with the subtitle group's
// signature, so over credits music it "hears" one: HP4 came back with
// 「Subs by www.zeoranger.co.uk」 at 2:29:54, twelve minutes after the last line
// of dialogue, and translation paid for it into the .zh-Hant.srt
// (disc-2026-09-whisper-credit-line-hallucination, NAS 2026-09-18). The line is
// confident and stands alone, so neither the score rules nor the tail rule
// (three lines in a row) can see it — only what it says gives it away.
//
// Both lists are matched on normalizeForEcho text (lower case, letters and
// digits, single spaces), so they are written in that form.

// creditLinePrefixes are signatures: the whole line starts with "who made the
// subtitles". Nobody opens a line of dialogue this way.
var creditLinePrefixes = []string{
	"subs by", // HP4, above
	"subtitles by",
	"subtitled by",
	"captions by",
	"captioning by", // 「Captioning by CaptionMax」, a broadcast caption credit
	"closed captioning by",
	"transcription by",
	"transcribed by",
	"字幕由",     // 「字幕由Amara.org社区提供」, zh runs
	"中文字幕志愿者", // 「中文字幕志愿者 李宗盛」, zh runs
	"中文字幕志願者",
}

// creditLineMarkers are names that only ever appear inside such a signature,
// matched anywhere in the line.
var creditLineMarkers = []string{
	"zeoranger", // HP4, above
	"amara org", // 「Subtitles by the Amara.org community」 and its zh form
	"明镜与点点",     // 「请不吝点赞 订阅 转发 打赏支持明镜与点点栏目」, zh runs
	"明鏡與點點",
}

// isCreditLineHallucination reports whether text is a subtitle-credit line
// whisper invented. Only the speech-recognition leg asks: a signature in an
// embedded track was put there by a person, and is a separate call.
//
// Deliberately NOT here: 「Thanks for watching.」 and other outro phrases. A
// person can say them, and an outro long enough to matter is the tail rule's.
func isCreditLineHallucination(text string) bool {
	norm := normalizeForEcho(text)
	if norm == "" {
		return false
	}
	for _, p := range creditLinePrefixes {
		if norm == p || strings.HasPrefix(norm, p+" ") || (!isASCII(p) && strings.HasPrefix(norm, p)) {
			return true
		}
	}
	for _, m := range creditLineMarkers {
		if strings.Contains(norm, m) {
			return true
		}
	}
	return false
}

// isASCII: a CJK prefix has no word boundary to wait for (「字幕由Amara」 runs on).
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
