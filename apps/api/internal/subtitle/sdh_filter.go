package subtitle

import (
	"regexp"
	"strconv"
	"strings"
)

// maxSpeakerLabelLen caps how long an ALL-CAPS leading label may be before it
// stops looking like a speaker attribution and starts looking like dialogue.
const maxSpeakerLabelLen = 20

// speakerLabelName is the ALL-CAPS name grammar shared by both label forms. The
// first character must be A-Z (so `12:30 tomorrow` and `He said: hello` are
// never touched) and the name may not contain lowercase.
var speakerLabelName = `[A-Z][A-Z0-9 .'&#-]{0,` + strconv.Itoa(maxSpeakerLabelLen-1) + `}`

// speakerLabelPattern matches a leading ALL-CAPS speaker attribution, in either
// of the two forms SDH tracks use: `JOHN:` / `MAN ON RADIO:` or the bracketed
// `[JOHN]:`. The brackets must MATCH — a mismatched `[JOHN:` or `JOHN]:` is
// left alone (stripping a malformed line would be over-stripping, and this
// filter errs to under-strip).
var speakerLabelPattern = regexp.MustCompile(
	`^(?:\[` + speakerLabelName + `\]|` + speakerLabelName + `):[ \t]*`)

// musicMarks wrap a whole line of lyric/score annotation in SDH tracks.
var musicMarks = []rune{'♪', '#'}

// FilterSDH strips SDH (subtitles for the deaf and hard-of-hearing) annotations
// from parsed cues, returning the survivors and the number of cues DROPPED
// (cues, not lines — a cue that merely lost a speaker label still survives).
//
// The filter is CONSERVATIVE by design: only whole-line annotations and leading
// speaker labels are handled, so a parenthetical inside a dialogue line
// (`He said (quietly) yes`) is left alone. Over-stripping loses dialogue;
// under-stripping only costs translation tokens, and M1 errs to under-strip.
//
// P7: survivors keep their ORIGINAL Index and timestamps — no renumbering, so
// the resulting gaps are correct and FR17's later cue-equality assertion holds.
// The input slice is never mutated.
func FilterSDH(blocks []SubtitleBlock) (kept []SubtitleBlock, removed int) {
	for _, b := range blocks {
		lines := strings.Split(b.Text, "\n")
		survivors := make([]string, 0, len(lines))

		for _, line := range lines {
			if cleaned, ok := filterSDHLine(line); ok {
				survivors = append(survivors, cleaned)
			}
		}

		if len(survivors) == 0 {
			removed++
			continue
		}

		out := b // copy — Index/Start/End carried over untouched (P7)
		out.Text = strings.Join(survivors, "\n")
		kept = append(kept, out)
	}

	return kept, removed
}

// filterSDHLine applies the per-line rules, returning the surviving text and
// whether the line survives at all.
func filterSDHLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", false
	}
	if isWholeLineAnnotation(trimmed) {
		return "", false
	}

	stripped := stripSpeakerLabel(trimmed)
	if stripped == trimmed {
		return trimmed, true
	}

	// The label is gone; re-test the residue, because `JOHN: [laughs]` is an
	// annotation the first pass could not see behind the attribution. This is
	// the SAME rule applied to the same visible content, so it cannot
	// over-strip beyond what the rules already classify as annotation.
	if stripped == "" || isWholeLineAnnotation(stripped) {
		return "", false
	}
	return stripped, true
}

// isWholeLineAnnotation reports whether the ENTIRE line is a bracketed sound
// cue (`[door slams]`, `(sighs)`), a music-mark-wrapped line (`♪ … ♪`), or a
// line made of nothing but music marks (`♪`, `♪♪`, `#`).
func isWholeLineAnnotation(s string) bool {
	if isWrappedInBrackets(s, '[', ']') || isWrappedInBrackets(s, '(', ')') {
		return true
	}
	return isMusicLine(s) || isMusicOnly(s)
}

// isMusicOnly reports whether the line contains only music marks and spaces
// (`♪`, `♪♪`, `♪ ♪`, `#`). isMusicLine needs a mark at BOTH ends and at least
// two runes, so a bare `♪` — the commonest cue in an SDH track — slipped
// through it, was sent to the LLM, paid for, and then rejected by the quality
// gate as "echoed" (sub-6-4; eval-1 finding 8: ~40 such cues per model on
// Lioness). A line with any non-mark, non-space character is left alone — the
// AC #4 under-strip posture for `♪ lyrics` is unchanged.
func isMusicOnly(s string) bool {
	seenMark := false
	for _, r := range s {
		switch {
		case r == ' ' || r == '\t':
			continue
		case isMusicMark(r):
			seenMark = true
		default:
			return false
		}
	}
	return seenMark
}

func isMusicMark(r rune) bool {
	for _, mark := range musicMarks {
		if r == mark {
			return true
		}
	}
	return false
}

// isWrappedInBrackets reports whether s opens with `open`, closes with `close`,
// and the opening bracket's match is the FINAL character. The depth walk is
// what keeps `(quietly) he said (loudly)` — two separate parentheticals around
// real dialogue — from being mistaken for one wrapped annotation.
func isWrappedInBrackets(s string, open, close byte) bool {
	if len(s) < 2 || s[0] != open || s[len(s)-1] != close {
		return false
	}

	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 && i != len(s)-1 {
				return false
			}
		}
	}

	return depth == 0
}

// isMusicLine reports whether the line is wrapped in the SAME music mark at
// both ends (`♪ dramatic music ♪`). A line that merely STARTS with a mark is
// left alone — the conservative reading of AC #4.
func isMusicLine(s string) bool {
	runes := []rune(s)
	if len(runes) < 2 {
		return false
	}
	for _, mark := range musicMarks {
		if runes[0] == mark && runes[len(runes)-1] == mark {
			return true
		}
	}
	return false
}

// stripSpeakerLabel removes a leading ALL-CAPS speaker attribution, returning
// the line unchanged when there is none.
func stripSpeakerLabel(s string) string {
	loc := speakerLabelPattern.FindStringIndex(s)
	if loc == nil {
		return s
	}
	return strings.TrimSpace(s[loc[1]:])
}
