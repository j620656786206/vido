package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// speech is a healthy segment: the decoder was confident and heard speech.
func speech(start, end float64, text string) whisperSegment {
	return whisperSegment{
		Start: start, End: end, Text: text,
		NoSpeechProb: 0.01, AvgLogprob: -0.25, CompressionRatio: 1.4,
	}
}

// ─── segmentsToSRT ──────────────────────────────────────────────────────────

func TestSegmentsToSRT_RendersOneCuePerSegment(t *testing.T) {
	srt := segmentsToSRT([]whisperSegment{
		speech(1.0, 3.5, " Hello world "),
		speech(5.0, 8.25, "Goodbye"),
	})

	assert.Equal(t,
		"1\n00:00:01,000 --> 00:00:03,500\nHello world\n\n"+
			"2\n00:00:05,000 --> 00:00:08,250\nGoodbye\n\n",
		srt)
}

// The renderer's output feeds MergeSRTChunks, which only re-numbers and shifts
// lines it can RECOGNISE. If either predicate stops matching, multi-chunk files
// silently merge into garbage.
func TestSegmentsToSRT_OutputIsRecognisedByTheMergePath(t *testing.T) {
	srt := segmentsToSRT([]whisperSegment{speech(1, 2, "One"), speech(3, 4, "Two")})
	lines := splitLines(srt)

	require.True(t, isSequenceNumber(lines[0]), "line 0 must parse as a sequence number")
	require.True(t, isTimestampLine(lines[1]), "line 1 must parse as a timestamp line")

	// And a real round-trip through the merge helper keeps it well-formed.
	merged := MergeSRTChunks([]string{srt, srt}, []int{0, 600_000})
	assert.Contains(t, merged, "3\n00:10:01,000 --> 00:10:02,000\nOne")
	assert.Contains(t, merged, "4\n00:10:03,000 --> 00:10:04,000\nTwo")
}

func TestSegmentsToSRT_SkipsBlankTextWithoutLeavingGapsInNumbering(t *testing.T) {
	srt := segmentsToSRT([]whisperSegment{
		speech(1, 2, "One"),
		speech(2, 3, "   "),
		speech(3, 4, "Two"),
	})
	assert.Equal(t,
		"1\n00:00:01,000 --> 00:00:02,000\nOne\n\n"+
			"2\n00:00:03,000 --> 00:00:04,000\nTwo\n\n",
		srt)
}

func TestSegmentsToSRT_Empty(t *testing.T) {
	assert.Equal(t, "", segmentsToSRT(nil))
}

func TestSecondsToMillis_RoundsToNearest(t *testing.T) {
	assert.Equal(t, 0, secondsToMillis(0))
	assert.Equal(t, 0, secondsToMillis(-1))
	assert.Equal(t, 1000, secondsToMillis(1.0))
	assert.Equal(t, 2000, secondsToMillis(1.9995), "must round up, not truncate to 1999")
	assert.Equal(t, 1234, secondsToMillis(1.2344))
}

// ─── parseVerboseTranscription ──────────────────────────────────────────────

func TestParseVerboseTranscription_Success(t *testing.T) {
	vt, err := parseVerboseTranscription(`{"language":"english","duration":8.4,"text":"Hi",
		"segments":[{"id":0,"seek":0,"start":0.0,"end":3.3,"text":"Hi","tokens":[1,2],
		"temperature":0.0,"avg_logprob":-0.28,"compression_ratio":1.23,"no_speech_prob":0.008}]}`)
	require.NoError(t, err)
	require.Len(t, vt.Segments, 1)
	assert.Equal(t, "Hi", vt.Segments[0].Text)
	assert.InDelta(t, 3.3, vt.Segments[0].End, 0.001)
	assert.InDelta(t, 0.008, vt.Segments[0].NoSpeechProb, 0.0001)
	assert.InDelta(t, -0.28, vt.Segments[0].AvgLogprob, 0.0001)
	assert.InDelta(t, 1.23, vt.Segments[0].CompressionRatio, 0.0001)
}

func TestParseVerboseTranscription_NotJSON(t *testing.T) {
	_, err := parseVerboseTranscription("1\n00:00:01,000 --> 00:00:02,000\nHi\n")
	require.Error(t, err)
}

// A transcript with NO segment array means the engine served the wrong JSON
// shape: the words exist but nothing says when they were said. Falling back to
// `srt` recovers a cueable transcript.
func TestParseVerboseTranscription_TextWithoutSegmentsIsTheWrongShape(t *testing.T) {
	_, err := parseVerboseTranscription(`{"text":"Hi there","segments":[]}`)
	require.ErrorIs(t, err, errWrongJSONShape)
}

// CR H1: an empty segment array with an EMPTY transcript is a silent chunk —
// the single most on-topic input this story has. It must parse cleanly so the
// caller can report "no speech" instead of condemning the engine.
func TestParseVerboseTranscription_SilenceParsesCleanly(t *testing.T) {
	vt, err := parseVerboseTranscription(`{"language":"english","duration":600.0,"text":"","segments":[]}`)
	require.NoError(t, err)
	assert.Empty(t, vt.Segments)
	assert.InDelta(t, 600.0, vt.Duration, 0.001)
}

// CR M3: blank-text segments are dropped at parse time so SegmentsIn /
// SegmentsKept can never disagree with the number of cues actually rendered.
func TestParseVerboseTranscription_DropsBlankTextSegments(t *testing.T) {
	vt, err := parseVerboseTranscription(`{"segments":[
	  {"id":0,"start":1.0,"end":2.0,"text":"Real","avg_logprob":-0.2,"compression_ratio":1.2,"no_speech_prob":0.01},
	  {"id":1,"start":2.0,"end":3.0,"text":"   ","avg_logprob":-0.2,"compression_ratio":1.2,"no_speech_prob":0.01}
	]}`)
	require.NoError(t, err)
	require.Len(t, vt.Segments, 1)
	assert.Equal(t, "Real", vt.Segments[0].Text)
}

// ─── filterHallucinations ───────────────────────────────────────────────────

func TestFilterHallucinations_KeepsCleanDialogueUntouched(t *testing.T) {
	segs := []whisperSegment{speech(1, 2, "One"), speech(3, 4, "Two"), speech(5, 6, "Three")}
	kept, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped)
	assert.Equal(t, segs, kept)
}

func TestFilterHallucinations_Empty(t *testing.T) {
	kept, dropped := filterHallucinations(nil)
	assert.Empty(t, kept)
	assert.Empty(t, dropped)
}

// R1: whisper's own "this window is silence" condition — BOTH signals must
// agree, which is what keeps the false-positive rate low.
func TestFilterHallucinations_R1Silence(t *testing.T) {
	cases := []struct {
		name        string
		noSpeech    float64
		avgLogprob  float64
		wantDropped bool
	}{
		{"both bars breached", 0.9, -1.6, true},
		{"loud but improbable text only", 0.1, -1.6, false},
		{"silent but confident text only", 0.9, -0.2, false},
		{"exactly at both thresholds", 0.6, -1.0, false},
		{"clean speech", 0.01, -0.25, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			segs := []whisperSegment{
				speech(1, 2, "Real line"),
				{Start: 3, End: 4, Text: "Suspect", NoSpeechProb: tc.noSpeech, AvgLogprob: tc.avgLogprob, CompressionRatio: 1.2},
				speech(5, 6, "Another real line"),
			}
			kept, dropped := filterHallucinations(segs)
			if tc.wantDropped {
				require.Len(t, dropped, 1)
				assert.Equal(t, dropReasonSilence, dropped[0].Reason)
				assert.Equal(t, "Suspect", dropped[0].Segment.Text)
				assert.Len(t, kept, 2)
			} else {
				assert.Empty(t, dropped)
				assert.Len(t, kept, 3)
			}
		})
	}
}

// R2a: a single segment whose text compresses far too well is the decoder
// stuttering inside one window.
func TestFilterHallucinations_R2Repetition(t *testing.T) {
	segs := []whisperSegment{
		speech(1, 2, "Real line"),
		{Start: 3, End: 4, Text: "ha ha ha ha ha ha ha ha", NoSpeechProb: 0.05, AvgLogprob: -0.4, CompressionRatio: 3.1},
		speech(5, 6, "Another real line"),
	}
	kept, dropped := filterHallucinations(segs)
	require.Len(t, dropped, 1)
	assert.Equal(t, dropReasonRepetition, dropped[0].Reason)
	assert.Len(t, kept, 2)
}

func TestFilterHallucinations_R2RepetitionAtThresholdIsKept(t *testing.T) {
	segs := []whisperSegment{
		speech(1, 2, "Real"),
		{Start: 3, End: 4, Text: "borderline", NoSpeechProb: 0.05, AvgLogprob: -0.4, CompressionRatio: 2.4},
		speech(5, 6, "Real"),
	}
	_, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped, "2.4 is the threshold, not a violation of it")
}

// R2b: a decoder loop repeats the SAME line forever. The first utterance is
// probably real, so it survives.
func TestFilterHallucinations_R2RepeatRunKeepsTheFirst(t *testing.T) {
	segs := []whisperSegment{
		speech(1, 2, "Thank you."),
		speech(2, 3, "thank you"),
		speech(3, 4, " Thank you. "),
		speech(4, 5, "Thank you."),
		speech(5, 6, "Thank you!"),
		speech(6, 7, "Thank you."),
		speech(8, 9, "Real closing line"),
	}
	kept, dropped := filterHallucinations(segs)
	require.Len(t, dropped, 5)
	for _, d := range dropped {
		assert.Equal(t, dropReasonRepeatRun, d.Reason)
	}
	require.Len(t, kept, 2)
	assert.Equal(t, "Thank you.", kept[0].Text, "the FIRST utterance survives")
	assert.Equal(t, "Real closing line", kept[1].Text)
}

// disc-2026-10-asr-repeated-lines-dropped: See S01E02 — 「Face me!」 shouted
// three times, 「Jerlamarel!」 called three times — is dialogue. A run under
// hallucinationRepeatRun (now 5) is kept whole.
func TestFilterHallucinations_R2GenuineRepeatedShoutsSurvive(t *testing.T) {
	segs := []whisperSegment{
		speech(261, 263, "Face me!"),
		speech(263, 265, "Face me!"),
		speech(265, 268, "Face me!"),
		speech(466, 468, "Jerlamarel!"),
		speech(468, 470, "Jerlamarel!"),
		speech(470, 472, "Jerlamarel!"),
		speech(472, 474, "Jerlamarel!"),
		speech(480, 482, "Come back."),
	}
	kept, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped, "three or four genuine shouts are under the loop bar")
	assert.Len(t, kept, 8)
}

// Exactly hallucinationRepeatRun identical lines is where the loop rule
// starts: the first stays, the rest go.
func TestFilterHallucinations_R2ExactlyFiveIsALoop(t *testing.T) {
	segs := []whisperSegment{
		speech(1, 2, "Thank you."),
		speech(2, 3, "Thank you."),
		speech(3, 4, "Thank you."),
		speech(4, 5, "Thank you."),
		speech(5, 6, "Thank you."),
	}
	kept, dropped := filterHallucinations(segs)
	require.Len(t, dropped, 4)
	require.Len(t, kept, 1)
	for _, d := range dropped {
		assert.Equal(t, dropReasonRepeatRun, d.Reason)
	}
}

// Two identical lines are dialogue, not a loop.
func TestFilterHallucinations_R2TwoIdenticalLinesSurvive(t *testing.T) {
	segs := []whisperSegment{
		speech(1, 2, "No."),
		speech(2, 3, "No."),
		speech(4, 5, "Listen to me"),
	}
	_, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped, "a run of 2 is under hallucinationRepeatRun")
}

// disc-2026-10-asr-repeat-run-bridging: a same-text window another rule has
// already claimed does not count toward the run. Four real shouts plus one
// same-text window judged silence are four shouts — all four stay.
func TestFilterHallucinations_R2ClaimedSegmentsDoNotLengthenTheRun(t *testing.T) {
	silent := whisperSegment{Start: 263, End: 264, Text: "Face me!", NoSpeechProb: 0.9, AvgLogprob: -1.6, CompressionRatio: 1.2}
	segs := []whisperSegment{
		speech(261, 263, "Face me!"),
		silent,
		speech(264, 266, "Face me!"),
		speech(266, 268, "Face me!"),
		speech(268, 270, "Face me!"),
	}
	kept, dropped := filterHallucinations(segs)
	require.Len(t, dropped, 1)
	assert.Equal(t, dropReasonSilence, dropped[0].Reason)
	assert.Len(t, kept, 4, "the four real shouts are under the loop bar")
}

// An echo of the name prompt between two groups of the same shout is marked in
// place, so it keeps the groups apart instead of splicing 3+3 into a six-long
// "loop" (the old filterPromptEcho removed it first).
func TestFilterHallucinations_PromptEchoDoesNotBridgeTwoShoutGroups(t *testing.T) {
	prompt := "Baba Voss, Maghra, Jerlamarel, Kofun, Haniwa"
	segs := []whisperSegment{
		speech(466, 468, "Jerlamarel!"),
		speech(468, 470, "Jerlamarel!"),
		speech(470, 472, "Jerlamarel!"),
		speech(472, 474, "Baba Voss, Maghra, Kofun"),
		speech(474, 476, "Jerlamarel!"),
		speech(476, 478, "Jerlamarel!"),
		speech(478, 480, "Jerlamarel!"),
	}
	kept, dropped := filterHallucinationsWith(segs, true, prompt)
	require.Len(t, dropped, 1)
	assert.Equal(t, dropReasonPromptEcho, dropped[0].Reason)
	assert.Len(t, kept, 6, "both groups of three shouts survive")
}

// The other loop shape: over score whisper emits one identical line per
// 30-second window (10-minute See clip, run1-prompt — the run of four that
// slipped under hallucinationRepeatRun). The first stays, the rest go.
func TestFilterHallucinations_R2SpacedRepeatIsALoop(t *testing.T) {
	segs := []whisperSegment{
		speech(0, 2, "Welcome to the guesthouse."),
		speech(30, 32, "Welcome to the guesthouse."),
		speech(60, 62, "Welcome to the guesthouse."),
		speech(90, 92, "Welcome to the guesthouse!"),
		speech(125, 127, "Stop the van."),
	}
	kept, dropped := filterHallucinations(segs)
	require.Len(t, dropped, 3)
	for _, d := range dropped {
		assert.Equal(t, dropReasonRepeatSpaced, d.Reason)
	}
	require.Len(t, kept, 2)
	assert.Equal(t, "Welcome to the guesthouse.", kept[0].Text, "the FIRST utterance survives")
	assert.InDelta(t, 0.0, kept[0].Start, 0.001)
}

// Exactly three, exactly 20 s apart, is where the spaced rule starts.
func TestFilterHallucinations_R2SpacedRepeatAtTheBar(t *testing.T) {
	segs := []whisperSegment{
		speech(10, 11, "Hello?"),
		speech(30, 31, "Hello?"),
		speech(50, 51, "Hello?"),
	}
	kept, dropped := filterHallucinations(segs)
	assert.Len(t, dropped, 2)
	assert.Len(t, kept, 1)
}

// Two of the same short answer half a minute apart is ordinary dialogue.
func TestFilterHallucinations_R2TwoSpacedIdenticalLinesSurvive(t *testing.T) {
	segs := []whisperSegment{
		speech(10, 11, "Yes."),
		speech(35, 36, "Yes."),
	}
	_, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped, "two is under hallucinationSpacedRepeatRun")
}

// One gap under the bar means someone is actually repeating themselves — the
// spaced rule does not apply (the five-in-a-row rule still could).
func TestFilterHallucinations_R2SpacedRuleNeedsEveryGapWide(t *testing.T) {
	segs := []whisperSegment{
		speech(10, 11, "Hello?"),
		speech(40, 41, "Hello?"),
		speech(45, 46, "Hello?"),
		speech(75, 76, "Hello?"),
	}
	_, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped)
}

// A claimed window in the middle neither counts nor shortens the gap: the gap
// is measured between the live segments either side of it.
func TestFilterHallucinations_R2SpacedGapSkipsClaimedSegments(t *testing.T) {
	segs := []whisperSegment{
		speech(0, 2, "Welcome to the guesthouse."),
		{Start: 5, End: 6, Text: "Welcome to the guesthouse.", NoSpeechProb: 0.9, AvgLogprob: -1.6, CompressionRatio: 1.2},
		speech(30, 32, "Welcome to the guesthouse."),
		speech(60, 62, "Welcome to the guesthouse."),
	}
	kept, dropped := filterHallucinations(segs)
	assert.Equal(t, map[string]int{dropReasonSilence: 1, dropReasonRepeatSpaced: 2}, dropReasonCounts(dropped))
	assert.Len(t, kept, 1)
}

// Just under the gap bar is someone repeating themselves: all three stay.
func TestFilterHallucinations_R2SpacedRepeatJustUnderTheBarSurvives(t *testing.T) {
	segs := []whisperSegment{
		speech(10, 11, "Hello?"),
		speech(29.9, 30.9, "Hello?"),
		speech(49.8, 50.8, "Hello?"),
	}
	_, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped)
}

// Five or more is the stuck-decoder rule whatever the spacing — the log says
// repeat_run, not repeat_spaced.
func TestFilterHallucinations_R2FiveSpacedReportsRepeatRun(t *testing.T) {
	var segs []whisperSegment
	for i := 0; i < 5; i++ {
		segs = append(segs, speech(float64(i*30), float64(i*30+2), "Welcome to the guesthouse."))
	}
	_, dropped := filterHallucinations(segs)
	assert.Equal(t, map[string]int{dropReasonRepeatRun: 4}, dropReasonCounts(dropped))
}

// A claimed FIRST window: the first LIVE one is the one kept.
func TestFilterHallucinations_R2ClaimedFirstSegmentKeepsTheFirstLiveOne(t *testing.T) {
	segs := []whisperSegment{
		{Start: 0, End: 1, Text: "Hello?", NoSpeechProb: 0.9, AvgLogprob: -1.6, CompressionRatio: 1.2},
		speech(30, 31, "Hello?"),
		speech(60, 61, "Hello?"),
		speech(90, 91, "Hello?"),
	}
	kept, dropped := filterHallucinations(segs)
	assert.Equal(t, map[string]int{dropReasonSilence: 1, dropReasonRepeatSpaced: 2}, dropReasonCounts(dropped))
	require.Len(t, kept, 1)
	assert.InDelta(t, 30.0, kept[0].Start, 0.001)
}

// The spaced rule runs on mid-file chunks too (only R3 is end-of-file only).
func TestFilterHallucinationsWith_SpacedRepeatAppliesMidFile(t *testing.T) {
	segs := []whisperSegment{
		speech(0, 2, "Welcome to the guesthouse."),
		speech(30, 32, "Welcome to the guesthouse."),
		speech(60, 62, "Welcome to the guesthouse."),
		speech(90, 92, "Welcome to the guesthouse."),
	}
	_, dropped := filterHallucinationsWith(segs, false, "")
	assert.Equal(t, map[string]int{dropReasonRepeatSpaced: 3}, dropReasonCounts(dropped))
}

// R3 + prompt echo: an echo is transparent to the tail walk, as it was when
// echoes were removed before the filter — it neither ends the run nor pads it.
func TestFilterHallucinations_R3TailIgnoresPromptEcho(t *testing.T) {
	prompt := "Baba Voss, Maghra, Jerlamarel, Kofun, Haniwa"
	quiet := func(start float64, text string) whisperSegment {
		return whisperSegment{Start: start, End: start + 1, Text: text, NoSpeechProb: 0.5, AvgLogprob: -0.3, CompressionRatio: 1.3}
	}
	echo := speech(20, 21, "Baba Voss, Maghra, Kofun")

	t.Run("echo last does not end the run", func(t *testing.T) {
		segs := []whisperSegment{speech(1, 2, "Real line"), quiet(10, "Outro one"), quiet(12, "Outro two"), quiet(14, "Outro three"), echo}
		_, dropped := filterHallucinationsWith(segs, true, prompt)
		assert.Equal(t, map[string]int{dropReasonTail: 3, dropReasonPromptEcho: 1}, dropReasonCounts(dropped))
	})
	t.Run("echo does not pad a short run to the bar", func(t *testing.T) {
		quietEcho := echo
		quietEcho.NoSpeechProb = 0.5
		segs := []whisperSegment{speech(1, 2, "Real line"), quiet(10, "Soft one"), quiet(12, "Soft two"), quietEcho}
		_, dropped := filterHallucinationsWith(segs, true, prompt)
		assert.Equal(t, map[string]int{dropReasonPromptEcho: 1}, dropReasonCounts(dropped))
	})
}

// R3: the POC failure. Silent credits, an invented outro, nothing after it.
func TestFilterHallucinations_R3TailDropsThePOCOutro(t *testing.T) {
	segs := []whisperSegment{
		speech(10, 12, "So that's the whole story."),
		speech(13, 15, "Goodbye."),
		// Credits roll: soft-scoring, invented.
		{Start: 100, End: 103, Text: "Thanks for watching!", NoSpeechProb: 0.55, AvgLogprob: -0.9, CompressionRatio: 1.5},
		{Start: 103, End: 106, Text: "Please like and subscribe.", NoSpeechProb: 0.72, AvgLogprob: -1.2, CompressionRatio: 1.6},
		{Start: 106, End: 109, Text: "See you next time.", NoSpeechProb: 0.61, AvgLogprob: -0.95, CompressionRatio: 1.4},
	}
	kept, dropped := filterHallucinations(segs)

	require.Len(t, kept, 2)
	assert.Equal(t, "So that's the whole story.", kept[0].Text)
	assert.Equal(t, "Goodbye.", kept[1].Text)
	// Real dialogue survives byte-identical, timestamps included.
	assert.Equal(t, segs[0], kept[0])
	assert.Equal(t, segs[1], kept[1])
	require.Len(t, dropped, 3)
	assert.Contains(t, []string{dropReasonTail, dropReasonSilence}, dropped[0].Reason)
}

// The looser tail bar must NOT reach into the middle of the file.
func TestFilterHallucinations_R3TailDoesNotEatMidFileSoftSegments(t *testing.T) {
	soft := func(start, end float64, text string) whisperSegment {
		return whisperSegment{Start: start, End: end, Text: text, NoSpeechProb: 0.5, AvgLogprob: -0.5, CompressionRatio: 1.3}
	}
	segs := []whisperSegment{
		speech(1, 2, "Opening"),
		soft(3, 4, "Muffled one"),
		soft(4, 5, "Muffled two"),
		soft(5, 6, "Muffled three"),
		speech(7, 8, "Clear closing line"),
	}
	kept, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped, "the soft run does not reach the end, so the tail rule must not fire")
	assert.Len(t, kept, 5)
}

// A short soft tail is left alone — one quiet last line is normal.
func TestFilterHallucinations_R3TailBelowMinRunIsKept(t *testing.T) {
	segs := []whisperSegment{
		speech(1, 2, "Opening"),
		speech(3, 4, "Middle"),
		{Start: 5, End: 6, Text: "Quiet last line", NoSpeechProb: 0.5, AvgLogprob: -0.5, CompressionRatio: 1.3},
		{Start: 6, End: 7, Text: "Another quiet one", NoSpeechProb: 0.45, AvgLogprob: -0.5, CompressionRatio: 1.3},
	}
	kept, dropped := filterHallucinations(segs)
	assert.Empty(t, dropped, "a 2-segment tail is under hallucinationTailMinRun")
	assert.Len(t, kept, 4)
}

// A chunk that is nothing but hallucinated credits SHOULD collapse to nothing.
// This is legal at the chunk level and is exactly the scenario 9R-5 exists for.
func TestFilterHallucinations_WholeChunkOfCreditsCollapses(t *testing.T) {
	segs := []whisperSegment{
		{Start: 1, End: 3, Text: "Thanks for watching!", NoSpeechProb: 0.8, AvgLogprob: -1.3, CompressionRatio: 1.5},
		{Start: 3, End: 5, Text: "Like and subscribe.", NoSpeechProb: 0.85, AvgLogprob: -1.4, CompressionRatio: 1.5},
		{Start: 5, End: 7, Text: "See you.", NoSpeechProb: 0.9, AvgLogprob: -1.5, CompressionRatio: 1.5},
	}
	kept, dropped := filterHallucinations(segs)
	assert.Empty(t, kept)
	assert.Len(t, dropped, 3)
}

func TestDropReasonCounts(t *testing.T) {
	assert.Nil(t, dropReasonCounts(nil))
	counts := dropReasonCounts([]droppedSegment{
		{Reason: dropReasonSilence}, {Reason: dropReasonSilence}, {Reason: dropReasonTail},
	})
	assert.Equal(t, map[string]int{dropReasonSilence: 2, dropReasonTail: 1}, counts)
}

// ─── end-to-end: verbose_json body → filtered SRT ───────────────────────────

func TestVerboseJSONToFilteredSRT_DropsOutroKeepsDialogueVerbatim(t *testing.T) {
	body := `{"language":"english","duration":109.0,"segments":[
	  {"id":0,"start":10.0,"end":12.0,"text":" So that's the whole story.","avg_logprob":-0.25,"compression_ratio":1.4,"no_speech_prob":0.01},
	  {"id":1,"start":13.0,"end":15.5,"text":" Goodbye.","avg_logprob":-0.3,"compression_ratio":1.3,"no_speech_prob":0.02},
	  {"id":2,"start":100.0,"end":103.0,"text":" Thanks for watching!","avg_logprob":-0.9,"compression_ratio":1.5,"no_speech_prob":0.55},
	  {"id":3,"start":103.0,"end":106.0,"text":" Please like and subscribe.","avg_logprob":-1.2,"compression_ratio":1.6,"no_speech_prob":0.72},
	  {"id":4,"start":106.0,"end":109.0,"text":" See you next time.","avg_logprob":-0.95,"compression_ratio":1.4,"no_speech_prob":0.61}
	]}`

	vt, err := parseVerboseTranscription(body)
	require.NoError(t, err)

	unfiltered := segmentsToSRT(vt.Segments)
	kept, dropped := filterHallucinations(vt.Segments)
	filtered := segmentsToSRT(kept)

	assert.Len(t, dropped, 3)
	assert.Equal(t,
		"1\n00:00:10,000 --> 00:00:12,000\nSo that's the whole story.\n\n"+
			"2\n00:00:13,000 --> 00:00:15,500\nGoodbye.\n\n",
		filtered)
	assert.Contains(t, unfiltered, "like and subscribe", "the unfiltered rendering keeps everything")
	assert.Equal(t, 5, strings.Count(unfiltered, " --> "))
}

// CR M1: a mid-file drop must NOT extend the tail run. Before the fix, the
// R2-dropped segment at index 2 bridged the gap and pulled the two quiet
// closing lines out with it even though a 2-segment tail is under the minimum.
func TestFilterHallucinations_R3TailRunIsNotBridgedByAnAlreadyDroppedSegment(t *testing.T) {
	segs := []whisperSegment{
		speech(1, 2, "Opening"),
		speech(2, 3, "Middle"),
		// Dropped by R2 (compression), but LOUD — no tail evidence of its own.
		{Start: 3, End: 4, Text: "la la la la la la", NoSpeechProb: 0.02, AvgLogprob: -0.4, CompressionRatio: 3.0},
		{Start: 5, End: 6, Text: "Quiet close one", NoSpeechProb: 0.5, AvgLogprob: -0.5, CompressionRatio: 1.3},
		{Start: 6, End: 7, Text: "Quiet close two", NoSpeechProb: 0.45, AvgLogprob: -0.5, CompressionRatio: 1.3},
	}
	kept, dropped := filterHallucinations(segs)

	require.Len(t, dropped, 1)
	assert.Equal(t, dropReasonRepetition, dropped[0].Reason)
	require.Len(t, kept, 4)
	assert.Equal(t, "Quiet close one", kept[2].Text, "a 2-segment tail is under the minimum and must survive")
	assert.Equal(t, "Quiet close two", kept[3].Text)
}

// disc-2026-10-asr-chunk-at-silence CR 1: the end-of-film tail rule must not
// run on a mid-file upload — on a 120 s grid every cut sits in a pause, so
// every chunk ends with quiet lines that the looser tail bar would eat.
func TestFilterHallucinationsWith_TailRuleOnlyAtTheFileEnd(t *testing.T) {
	quiet := func(start, end float64, text string) whisperSegment {
		s := speech(start, end, text)
		s.NoSpeechProb = 0.5 // above the tail bar, below the normal silence bar
		return s
	}
	segs := []whisperSegment{speech(1, 2, "Loud line"), quiet(3, 4, "soft one"), quiet(4, 5, "soft two"), quiet(5, 6, "soft three")}

	kept, dropped := filterHallucinationsWith(segs, true, "")
	assert.Len(t, kept, 1, "at the end of the film the quiet run is credits over score")
	assert.Len(t, dropped, 3)

	kept, dropped = filterHallucinationsWith(segs, false, "")
	assert.Len(t, kept, 4, "mid-file the same run is dialogue before a pause")
	assert.Empty(t, dropped)
}
