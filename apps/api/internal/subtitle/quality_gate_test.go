package subtitle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai/prompts"
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

// ─── sub-7-9 AC #2: misaligned ──────────────────────────────────────────────

func TestCheckChunkAnchored_Misaligned(t *testing.T) {
	src := []SubtitleBlock{
		{Index: 1, Text: "I was a boxer, you know."},
		{Index: 2, Text: "Killed a man in the ring."},
		{Index: 3, Text: "The train leaves at 14:30."},
		{Index: 4, Text: "The bus doesn't come till 2002."},
	}
	glossary := []prompts.GlossaryEntry{{Source: "boxer", Target: "拳擊手"}, {Source: "ring", Target: "擂台"}}
	anchors := AnchorsFor(src, glossary, nil)
	assert.Equal(t, map[int][]string{1: {"拳擊手"}, 2: {"擂台"}, 3: {"14", "30"}, 4: {"2002"}}, anchors)

	tests := []struct {
		name string
		got  map[int]string
		want map[int]string // index → reason; absent = passes
	}{
		{
			name: "aligned: every anchor on its own cue",
			got:  map[int]string{1: "我以前是拳擊手，你知道吧。", 2: "在擂台上打死過一個人。", 3: "火車 14:30 開。", 4: "公車要到 2002 才來。"},
		},
		{
			name: "shifted: cue 2's content moved onto cue 1, cue 2 left with a fragment",
			got:  map[int]string{1: "我以前是拳擊手，在擂台上打死過人。", 2: "你知道吧。", 3: "火車 14:30 開。", 4: "公車要到 2002 才來。"},
			want: map[int]string{2: GateReasonMisaligned},
		},
		{
			name: "shifted forward: cue 3's time landed on cue 4",
			got:  map[int]string{1: "我以前是拳擊手，你知道吧。", 2: "在擂台上打死過一個人。", 3: "火車開了。", 4: "14:30，公車要到 2002 才來。"},
			want: map[int]string{3: GateReasonMisaligned},
		},
		{
			name: "dropped but NOT next door: a translator may paraphrase a number away",
			got:  map[int]string{1: "我以前是拳擊手，你知道吧。", 2: "在擂台上打死過一個人。", 3: "火車下午兩點半開。", 4: "公車要到 2002 才來。"},
		},
		{
			name: "anchor shared by neighbours' sources is not a shift",
			got:  map[int]string{1: "我以前是拳擊手，你知道吧。", 2: "擂台上那個拳擊手死了。", 3: "火車 14:30 開。", 4: "公車要到 2002 才來。"},
		},
		{
			name: "no anchors on the cue → never triggers, whatever the neighbour says",
			got:  map[int]string{1: "我以前是拳擊手，你知道吧。", 2: "在擂台上打死過一個人，2002 年。", 3: "火車 14:30 開。", 4: "公車要到 2002 才來。"},
		},
		{
			name: "earlier classes win: an echoed cue is echoed, not misaligned",
			got:  map[int]string{1: "I was a boxer, you know.", 2: "在擂台上打死過一個拳擊手。", 3: "火車 14:30 開。", 4: "公車要到 2002 才來。"},
			want: map[int]string{1: GateReasonEchoed},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := CheckChunkAnchored(src, tc.got, anchors, nil)
			got := map[int]string{}
			for i, r := range v.Reasons {
				got[i] = r
			}
			if tc.want == nil {
				tc.want = map[int]string{}
			}
			assert.Equal(t, tc.want, got)
			assert.Len(t, v.FailedIndexes, len(tc.want))
		})
	}
}

func TestCheckChunkAnchored_RetriedCueSeesAcceptedNeighbours(t *testing.T) {
	// On a quality retry only cue 2 is pending; its drifted anchor sits in
	// cue 1's ALREADY-ACCEPTED translation, which comes in via neighbours.
	src := []SubtitleBlock{{Index: 2, Text: "Killed a man in the ring."}}
	anchors := map[int][]string{1: {"拳擊手"}, 2: {"擂台"}}
	accepted := map[int]string{1: "我以前是拳擊手，在擂台上打死過人。"}
	v := CheckChunkAnchored(src, map[int]string{2: "你知道吧。"}, anchors, accepted)
	assert.Equal(t, GateReasonMisaligned, v.Reasons[2])

	// Same text, no neighbour knowledge: the gate has no evidence and passes it.
	v = CheckChunkAnchored(src, map[int]string{2: "你知道吧。"}, anchors, nil)
	assert.True(t, v.Passed())

	// Plain CheckChunk never sees anchors: byte-for-byte the pre-7-9 verdict.
	assert.True(t, CheckChunk(src, map[int]string{2: "你知道吧。"}).Passed())
}

func TestAnchorsFor_RulesOfThumb(t *testing.T) {
	src := []SubtitleBlock{
		{Index: 1, Text: "Rick wants to hire me! 4 wives, 14 years, room 1999."},
		{Index: 2, Text: "Nothing to anchor here."},
		{Index: 3, Text: "ricky and Rick's hat"},
	}
	got := AnchorsFor(src, []prompts.GlossaryEntry{{Source: "Rick", Target: "瑞克"}}, map[string]string{"hat": "帽子", "wives": ""})
	assert.Equal(t, []string{"14", "1999", "瑞克"}, got[1], "multi-digit numbers and the glossary rendering; a lone 4 is not an anchor")
	assert.Nil(t, got[2], "a cue with nothing to anchor has no entry")
	assert.Equal(t, []string{"帽子", "瑞克"}, got[3], "whole-word match: ricky does not count, Rick's does; an empty rendering is ignored")
}
