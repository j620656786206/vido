package subtitle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oneCue builds a single-cue chunk carrying the given English source text.
func oneCue(index int, source string) []SubtitleBlock {
	return []SubtitleBlock{{Index: index, Start: "00:00:01,000", End: "00:00:02,000", Text: source}}
}

// TestCheckChunk_FailureClasses is the AC #1 class table: one case per rule,
// plus the echo guards that must NOT false-positive.
func TestCheckChunk_FailureClasses(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		translated map[int]string
		wantReason string // "" = the cue must pass the gate
	}{
		// ─── missing ───
		{
			name:       "missing: index absent from response",
			source:     "This software is great.",
			translated: map[int]string{},
			wantReason: GateReasonMissing,
		},
		{
			name:       "missing: model answered a different index only",
			source:     "This software is great.",
			translated: map[int]string{99: "這個軟體很好用"},
			wantReason: GateReasonMissing,
		},

		// ─── empty ───
		{
			name:       "empty: empty string",
			source:     "This software is great.",
			translated: map[int]string{7: ""},
			wantReason: GateReasonEmpty,
		},
		{
			name:       "empty: whitespace only",
			source:     "This software is great.",
			translated: map[int]string{7: "   \n\t "},
			wantReason: GateReasonEmpty,
		},

		// ─── echoed ───
		{
			name:       "echoed: verbatim English returned",
			source:     "This software is great.",
			translated: map[int]string{7: "This software is great."},
			wantReason: GateReasonEchoed,
		},
		{
			name:       "echoed: differs only by case and whitespace",
			source:     "This  software\nis great.",
			translated: map[int]string{7: "this software is great."},
			wantReason: GateReasonEchoed,
		},
		{
			name:       "echoed: differs only by dropped punctuation (sub-1-5a CR M3)",
			source:     "This software is great.",
			translated: map[int]string{7: "This software is great"},
			wantReason: GateReasonEchoed,
		},
		{
			name:       "echoed: punctuation swapped without a space",
			source:     "Hi,Bob. Come on!",
			translated: map[int]string{7: "Hi Bob — come on"},
			wantReason: GateReasonEchoed,
		},

		// ─── echo false-positive guards (each has < 3 consecutive Latin letters) ───
		{
			name:       "guard: interjection survives translation unchanged",
			source:     "OK!",
			translated: map[int]string{7: "OK!"},
		},
		{
			name:       "guard: pure numerals",
			source:     "1999",
			translated: map[int]string{7: "1999"},
		},
		{
			name:       "guard: initialised name kept as-is",
			source:     "J.T.",
			translated: map[int]string{7: "J.T."},
		},
		{
			name:       "guard: short-segment name kept as-is",
			source:     "Mr. Ed",
			translated: map[int]string{7: "Mr. Ed"},
		},
		{
			name:       "guard: Latin run present but text differs from source",
			source:     "Call the FBI.",
			translated: map[int]string{7: "打電話給 FBI。"},
		},

		// ─── simplified_leak ───
		{
			name:       "simplified_leak: whole cue in Simplified",
			source:     "This software is great.",
			translated: map[int]string{7: "这个软件很好用"},
			wantReason: GateReasonSimplifiedLeak,
		},
		{
			name:       "simplified_leak: a single simplified-only character is enough",
			source:     "Say something.",
			translated: map[int]string{7: "說点什麼"},
			wantReason: GateReasonSimplifiedLeak,
		},

		// ─── pass ───
		{
			name:       "pass: clean Traditional translation",
			source:     "This software is great.",
			translated: map[int]string{7: "這個軟體很好用"},
		},
		{
			name:       "pass: multi-line translation",
			source:     "You go first.\nI'll catch up.",
			translated: map[int]string{7: "你先走吧。\n我隨後就到。"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict := CheckChunk(oneCue(7, tt.source), tt.translated)

			if tt.wantReason == "" {
				assert.True(t, verdict.Passed(), "cue should pass, got reason %q", verdict.Reasons[7])
				assert.Empty(t, verdict.FailedIndexes)
				return
			}

			require.False(t, verdict.Passed())
			assert.Equal(t, []int{7}, verdict.FailedIndexes)
			assert.Equal(t, tt.wantReason, verdict.Reasons[7])
			assert.True(t, verdict.Failed(7))
		})
	}
}

// TestCheckChunk_MultipleFailuresReturnExactIndexSet covers a chunk where three
// different classes fail at once — the retry must resend precisely those cues.
func TestCheckChunk_MultipleFailuresReturnExactIndexSet(t *testing.T) {
	source := cues(
		"Good morning.",           // 1 — clean
		"This software is great.", // 2 — leaks Simplified
		"See you later.",          // 3 — echoed
		"Wait for me.",            // 4 — clean
		"Where are you going?",    // 5 — missing
	)

	verdict := CheckChunk(source, map[int]string{
		1: "早安",
		2: "这个软件很好用",
		3: "See you later.",
		4: "等等我",
	})

	assert.Equal(t, []int{2, 3, 5}, verdict.FailedIndexes, "failed indexes stay in source order")
	assert.Equal(t, map[int]string{
		2: GateReasonSimplifiedLeak,
		3: GateReasonEchoed,
		5: GateReasonMissing,
	}, verdict.Reasons)
	assert.False(t, verdict.Failed(1))
	assert.False(t, verdict.Failed(4))
}

// TestCheckChunk_HonoursNonContiguousIndexes guards the P1/P7 contract: SDH
// filtering leaves gaps in the cue numbering, so the gate must key on Index,
// never on position.
func TestCheckChunk_HonoursNonContiguousIndexes(t *testing.T) {
	source := []SubtitleBlock{
		{Index: 4, Text: "Good morning."},
		{Index: 9, Text: "See you later."},
		{Index: 17, Text: "Wait for me."},
	}

	verdict := CheckChunk(source, map[int]string{
		4:  "早安",
		9:  "",
		17: "等等我",
	})

	assert.Equal(t, []int{9}, verdict.FailedIndexes)
	assert.Equal(t, GateReasonEmpty, verdict.Reasons[9])
}

// TestCheckChunk_UnexpectedIndexesAreIgnored — a model inventing cue indexes
// must not corrupt the verdict for the cues that were actually requested.
func TestCheckChunk_UnexpectedIndexesAreIgnored(t *testing.T) {
	verdict := CheckChunk(oneCue(1, "Good morning."), map[int]string{
		1:  "早安",
		2:  "這個不存在",
		42: "這個也不存在",
	})

	assert.True(t, verdict.Passed())
	assert.Empty(t, verdict.FailedIndexes)
}

// TestCheckChunk_EmptyChunkPasses — a degenerate chunk has nothing to reject.
func TestCheckChunk_EmptyChunkPasses(t *testing.T) {
	verdict := CheckChunk(nil, nil)

	assert.True(t, verdict.Passed())
	assert.Empty(t, verdict.Reasons)
}
