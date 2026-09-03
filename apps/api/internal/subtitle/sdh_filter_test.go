package subtitle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Per-rule table (sub-1-4 AC #4; sub-6-4 rows marked) ───────────────────

func TestFilterSDH_Rules(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantOut string // "" means the whole cue is dropped
	}{
		// Rule 1 — line entirely bracketed / parenthesised
		{"square-bracket annotation", "[door slams]", ""},
		{"parenthesised annotation", "(sighs)", ""},
		{"uppercase bracket annotation", "[DOOR SLAMS]", ""},
		{"annotation with inner brackets", "[door slams [twice]]", ""},
		{"annotation padded with spaces", "   [door slams]   ", ""},

		// Rule 2 — music marks
		{"music note wrapped", "♪ dramatic music ♪", ""},
		{"hash wrapped", "# dramatic music #", ""},

		// Rule 2b (sub-6-4, eval-1 finding 8) — a line that is NOTHING but music
		// marks is an annotation too. Lioness's SDH track had ~40 bare ♪ cues per
		// model that were paid for and then rejected by the gate as "echoed".
		// These four were RED before isMusicOnly existed:
		{"lone music note", "♪", ""},
		{"lone hash", "#", ""},
		{"note padded with spaces", "   ♪   ", ""},
		{"music-only line dropped, dialogue line kept", "♪\nHello there", "Hello there"},
		// sub-6-4 CR M4/H2 — other marks and non-ASCII whitespace (RED before fix):
		{"lone beamed note", "♫", ""},
		{"lone beamed sixteenth", "♬", ""},
		{"notes separated by nbsp", "♪\u00a0#", ""},
		{"note with zero-width space", "♪\u200b", ""},
		{"note with ideographic space", "♪\u3000♪", ""},
		{"note after a speaker label", "JOHN: ♪", ""},
		{"note after a bracketed speaker label", "[JOHN]: ♪", ""},
		// Regression guards (already green via isMusicLine, kept so a future
		// isMusicOnly change cannot widen into these):
		{"double music note", "♪♪", ""},
		{"two notes with a space", "♪ ♪", ""},
		{"mixed marks only", "♪ # ♪", ""},
		{"note after lyrics keeps its line", "lyrics ♪", "lyrics ♪"},
		{"note inside dialogue keeps its line", "a ♪ b", "a ♪ b"},
		{"hash inside dialogue keeps its line", "I'm his #1 fan", "I'm his #1 fan"},

		// Rule 3 — leading ALL-CAPS speaker label is stripped, dialogue survives
		{"speaker label", "JOHN: Hello", "Hello"},
		{"speaker label in brackets", "[JOHN]: Hello", "Hello"},
		{"multi-word speaker label", "MAN ON RADIO: Evacuate now", "Evacuate now"},
		{"label exactly at the 20-char cap", "ABCDEFGHIJKLMNOPQRST: Hi", "Hi"},
		{"label with residual annotation is also cleared", "JOHN: [laughs]", ""},

		// Conservative — under-strip beats over-strip
		{"mid-line parenthetical is untouched", "He said (quietly) yes", "He said (quietly) yes"},
		{"leading and trailing parentheticals are not a whole-line annotation", "(quietly) he said (loudly)", "(quietly) he said (loudly)"},
		{"lowercase colon is not a speaker label", "He said: hello", "He said: hello"},
		{"mixed-case name is not a speaker label", "John: Hello", "John: Hello"},
		{"mismatched opening bracket is not a speaker label", "[JOHN: Hello", "[JOHN: Hello"},
		{"mismatched closing bracket is not a speaker label", "JOHN]: Hello", "JOHN]: Hello"},
		{"label longer than 20 chars is not stripped", "ABCDEFGHIJKLMNOPQRSTU: Hi", "ABCDEFGHIJKLMNOPQRSTU: Hi"},
		{"timestamps are not speaker labels", "12:30 tomorrow", "12:30 tomorrow"},
		{"unwrapped music note keeps its line", "♪ dramatic music", "♪ dramatic music"},
		{"plain dialogue untouched", "Hello there", "Hello there"},

		// Multi-line cues — per-line decisions, survivors rejoin
		{"annotation line dropped, dialogue kept", "[door slams]\nHello there", "Hello there"},
		{"dialogue kept, annotation line dropped", "Hello there\n(sighs)", "Hello there"},
		{"both lines annotations → cue dropped", "[door slams]\n(sighs)", ""},
		{"label stripped on one line only", "JOHN: Hello\nHow are you", "Hello\nHow are you"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := []SubtitleBlock{{Index: 7, Start: "00:00:01,000", End: "00:00:02,000", Text: tt.in}}

			kept, removed := FilterSDH(in)

			if tt.wantOut == "" {
				assert.Empty(t, kept)
				assert.Equal(t, 1, removed)
				return
			}
			require.Len(t, kept, 1)
			assert.Equal(t, tt.wantOut, kept[0].Text)
			assert.Equal(t, 0, removed)
		})
	}
}

// ─── P7 — survivors keep their original numbering and timings ──────────────

func TestFilterSDH_PreservesIndexAndTimestamps(t *testing.T) {
	in := []SubtitleBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "[door slams]"},
		{Index: 2, Start: "00:00:03,500", End: "00:00:05,120", Text: "JOHN: Hello there"},
		{Index: 3, Start: "00:00:06,000", End: "00:00:07,000", Text: "♪ dramatic music ♪"},
		{Index: 4, Start: "00:00:08,250", End: "00:00:09,999", Text: "General Kenobi"},
	}

	kept, removed := FilterSDH(in)

	require.Len(t, kept, 2)
	assert.Equal(t, 2, removed)

	// Original Index values survive — NO renumbering. Gaps (1 and 3 gone) are
	// correct; FR17's later cue-equality assertion depends on this.
	assert.Equal(t, 2, kept[0].Index)
	assert.Equal(t, "00:00:03,500", kept[0].Start)
	assert.Equal(t, "00:00:05,120", kept[0].End)
	assert.Equal(t, "Hello there", kept[0].Text)

	assert.Equal(t, 4, kept[1].Index)
	assert.Equal(t, "00:00:08,250", kept[1].Start)
	assert.Equal(t, "00:00:09,999", kept[1].End)

	// A cue with no SDH content is byte-identical to its input.
	assert.Equal(t, in[3], kept[1])
}

func TestFilterSDH_DoesNotMutateInput(t *testing.T) {
	in := []SubtitleBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "JOHN: Hello"},
		{Index: 2, Start: "00:00:03,000", End: "00:00:04,000", Text: "[door slams]"},
	}

	_, _ = FilterSDH(in)

	assert.Equal(t, "JOHN: Hello", in[0].Text)
	assert.Equal(t, "[door slams]", in[1].Text)
	assert.Len(t, in, 2)
}

// ─── Degenerate inputs ─────────────────────────────────────────────────────

func TestFilterSDH_EmptyInput(t *testing.T) {
	kept, removed := FilterSDH(nil)
	assert.Empty(t, kept)
	assert.Equal(t, 0, removed)
}

func TestFilterSDH_AllSDH(t *testing.T) {
	// A pure-SDH track: every cue is annotation. The router treats an empty
	// survivor set as "no usable text source" (AC #5).
	in := []SubtitleBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "[door slams]"},
		{Index: 2, Start: "00:00:03,000", End: "00:00:04,000", Text: "♪ theme ♪"},
		{Index: 3, Start: "00:00:05,000", End: "00:00:06,000", Text: "(sighs)"},
		{Index: 4, Start: "00:00:07,000", End: "00:00:08,000", Text: "♪"}, // sub-6-4
	}

	kept, removed := FilterSDH(in)

	assert.Empty(t, kept)
	assert.Equal(t, 4, removed)
}

func TestFilterSDH_BlankCueIsDropped(t *testing.T) {
	in := []SubtitleBlock{{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "   "}}

	kept, removed := FilterSDH(in)

	assert.Empty(t, kept)
	assert.Equal(t, 1, removed)
}

func TestFilterSDH_RemovedCountEqualsDroppedCues(t *testing.T) {
	// `removed` counts CUES dropped, not lines stripped — a cue that merely lost
	// its speaker label survives and is not counted.
	in := []SubtitleBlock{
		{Index: 1, Text: "JOHN: Hello"},
		{Index: 2, Text: "MARY: Hi"},
		{Index: 3, Text: "[door slams]"},
	}

	kept, removed := FilterSDH(in)

	assert.Len(t, kept, 2)
	assert.Equal(t, 1, removed)
	assert.Equal(t, len(in)-len(kept), removed)
}
