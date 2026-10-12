package ai

import (
	"strings"
	"unicode"
)

// MusicMarks are the glyphs subtitle tracks and speech recognizers use to say
// "music here, no words": ♪ ♫ ♬ and the SDH-track `#`.
const MusicMarks = "♪♫♬#"

// IsMusicOnlyText reports whether text contains nothing but music marks and
// whitespace (`♪`, `♪♪`, `♪ ♪`, `#`). "Whitespace" is unicode.IsSpace plus the
// zero-width characters subtitle tracks smuggle in (ZWSP / ZWNJ / BOM).
//
// It lives in this leaf package because TWO legs need the same answer and
// cannot see each other (Rule 19): subtitle.FilterSDH drops such cues from an
// embedded track before translation (sub-6-4), and the Whisper hallucination
// filter drops the ♪♪ segments speech recognition emits over score
// (disc-2026-10-asr-music-only-cues — See S01E02 shipped 22 of them). A line
// with any other character is NOT music-only, so `♪ lyrics ♪` is untouched.
func IsMusicOnlyText(text string) bool {
	seenMark := false
	for _, r := range text {
		switch {
		case isBlankRune(r):
			continue
		case r == '-' || r == '–' || r == '—':
			// SDH speaker dash ("-♪♪"): See S01E02's 10-minute clip came back
			// with 38 such cues and 10 slipped past the rule
			// (disc-2026-10-asr-music-only-dash).
			continue
		case strings.ContainsRune(MusicMarks, r):
			seenMark = true
		default:
			return false
		}
	}
	return seenMark
}

// IsSymbolOnlyText reports whether text has something to show but no letter
// (CJK included) and no digit: 「$%&!」, 「-$%&!」, 「...」. Whisper writes
// shouting or bleeped swearing that way, and translation copied it into the
// subtitle as 「$%&！」 (See S01E02's 10-minute clip, three cues —
// disc-2026-10-asr-symbol-only-cues). Music-only text matches too; callers
// that tell the two apart check IsMusicOnlyText first.
//
// Only the speech-recognition leg uses it: an embedded track's 「#$%&!」 was
// typed by a person, and dropping it there is a separate call.
func IsSymbolOnlyText(text string) bool {
	seen := false
	for _, r := range text {
		switch {
		case isBlankRune(r):
			continue
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			return false
		default:
			seen = true
		}
	}
	return seen
}

// isBlankRune is whitespace plus the zero-width characters subtitle tracks
// smuggle in (ZWSP / ZWNJ / BOM).
func isBlankRune(r rune) bool {
	return unicode.IsSpace(r) || r == '\u200b' || r == '\u200c' || r == '\ufeff'
}
