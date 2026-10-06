package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode"
)

// ─── Whisper verbose_json wire types (Story 9R-5) ───────────────────────────
//
// `response_format=srt` returns rendered text and nothing else, so there is no
// way to tell a real line from a hallucinated one. `verbose_json` returns the
// same segments the SRT is rendered FROM, each carrying the decoder's own
// confidence signals — which is what makes post-filtering possible at all.

// whisperSegment is one decoded segment of a verbose_json transcription.
// Only the fields the filter and the SRT renderer need are modelled; unknown
// fields (seek, tokens, temperature) are ignored by encoding/json.
type whisperSegment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"` // seconds
	End   float64 `json:"end"`   // seconds
	Text  string  `json:"text"`
	// NoSpeechProb is the decoder's own probability that this window contains
	// no speech at all — the primary hallucination signal.
	NoSpeechProb float64 `json:"no_speech_prob"`
	// AvgLogprob is the mean token log-probability. Very negative = the model
	// was guessing.
	AvgLogprob float64 `json:"avg_logprob"`
	// CompressionRatio is len(text)/len(gzip(text)). A high ratio means the
	// text repeats itself — the signature of a decoder stuck in a loop.
	CompressionRatio float64 `json:"compression_ratio"`
}

// whisperWord is one word with its own timing — present only when the request
// asked for `timestamp_granularities[]=word` and the engine honours it.
type whisperWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// verboseTranscription is the verbose_json response envelope.
type verboseTranscription struct {
	Language string           `json:"language"`
	Duration float64          `json:"duration"`
	Text     string           `json:"text"`
	Segments []whisperSegment `json:"segments"`
	// Words is the flat word list (not nested per segment) OpenAI returns
	// alongside Segments when word granularity was requested; nil otherwise.
	Words []whisperWord `json:"words,omitempty"`
}

// errWrongJSONShape reports a 200 that parsed as JSON but is not verbose_json:
// a transcript came back with NO segment array to cue it from (the engine
// served plain `json`, or ignored the requested format).
//
// CR H1 — this is deliberately NOT raised for an empty segment array with an
// empty transcript. That combination is a genuinely SILENT chunk, which is the
// single most on-topic input this story has: ten minutes of credits. Treating
// it as an engine capability failure latched the fallback and disabled
// hallucination filtering for every later chunk and every later media item.
var errWrongJSONShape = fmt.Errorf("whisper: response parsed as JSON but carried no segment array")

// parseVerboseTranscription decodes a verbose_json body and drops segments that
// carry no renderable text, so SegmentsIn/SegmentsKept and the rendered cue
// count can never disagree (CR M3).
func parseVerboseTranscription(body string) (*verboseTranscription, error) {
	var vt verboseTranscription
	if err := json.Unmarshal([]byte(body), &vt); err != nil {
		return nil, fmt.Errorf("whisper: decode verbose_json: %w", err)
	}
	if len(vt.Segments) == 0 && strings.TrimSpace(vt.Text) != "" {
		// Text but no segments = the wrong JSON shape. Falling back to `srt`
		// recovers a cueable transcript; returning "silence" would throw away
		// dialogue the engine actually heard.
		return nil, errWrongJSONShape
	}

	renderable := vt.Segments[:0]
	for _, seg := range vt.Segments {
		if strings.TrimSpace(seg.Text) != "" {
			renderable = append(renderable, seg)
		}
	}
	vt.Segments = renderable
	return &vt, nil
}

// tightenSegmentsWithWords moves each segment's cue boundaries in to the first
// and last word spoken inside it (disc-2026-10-asr-coarse-timestamps).
//
// whisper's SEGMENT times are coarse — whole seconds, back to back, the next
// cue starting the instant the previous one ends — so on See S01E02 138 of 637
// cues appeared while nobody was speaking, some six seconds early. Its WORD
// times are not. The cue count and order are untouched (ONE SEGMENT = ONE CUE
// still holds for segmentsToSRT); only Start/End move, and only INWARD: a
// boundary is never pushed outside the segment the engine gave, and a segment
// with no word inside it, or whose words would leave it shorter than
// minTightenedCueSeconds, keeps its original timing. Returns how many segments
// changed, for the log line.
func tightenSegmentsWithWords(segs []whisperSegment, words []whisperWord) int {
	if len(words) == 0 {
		return 0
	}
	tightened := 0
	wi := 0
	for i := range segs {
		seg := &segs[i]
		// Words are time-ordered; skip the ones that ended before this segment.
		for wi < len(words) && words[wi].End <= seg.Start {
			wi++
		}
		// CR M1: the previous line's last word usually runs a little PAST the
		// whole-second boundary ("name." 10.4–15.9 against a segment ending at
		// 15). Letting that straggler be this segment's first word would pin
		// the cue to the coarse seg.Start — the exact symptom this exists to
		// fix. So the start comes from the first word that BEGINS inside the
		// segment; a straggler only counts when no word begins inside.
		first, last, firstInside := -1, -1, -1
		for j := wi; j < len(words) && words[j].Start < seg.End; j++ {
			if strings.TrimSpace(words[j].Word) == "" {
				continue
			}
			if first < 0 {
				first = j
			}
			if firstInside < 0 && words[j].Start >= seg.Start {
				firstInside = j
			}
			last = j
		}
		if first < 0 {
			continue
		}
		if firstInside >= 0 {
			first = firstInside
		}
		start := math.Max(seg.Start, words[first].Start)
		end := math.Min(seg.End, words[last].End)
		if end-start < minTightenedCueSeconds {
			continue
		}
		if start != seg.Start || end != seg.End {
			seg.Start, seg.End = start, end
			tightened++
		}
	}
	return tightened
}

// minTightenedCueSeconds is the shortest cue word timing may produce; below
// it the segment keeps the engine's own boundaries (a one-word "No!" still
// needs to stay on screen long enough to read).
const minTightenedCueSeconds = 0.6

// segmentsToSRT renders segments as SRT.
//
// ONE SEGMENT = ONE CUE, start/end/text carried through untouched. This is the
// same rule whisper's own write_srt applies, and it is deliberate: OpenAI's
// `response_format=srt` is rendered from THIS EXACT segment array, so keeping
// the mapping 1:1 is what stops 9R-5 from silently re-cueing every subtitle the
// ASR leg has ever produced. Do NOT re-wrap, merge, or split here.
func segmentsToSRT(segs []whisperSegment) string {
	var sb strings.Builder
	seq := 1
	for _, seg := range segs {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		fmt.Fprintf(&sb, "%d\n%s --> %s\n%s\n\n",
			seq,
			formatSRTTimestamp(secondsToMillis(seg.Start)),
			formatSRTTimestamp(secondsToMillis(seg.End)),
			text,
		)
		seq++
	}
	return sb.String()
}

// secondsToMillis converts whisper's float seconds to the integer milliseconds
// formatSRTTimestamp expects, rounding to nearest rather than truncating so a
// 1.9995s boundary does not render as 1.999.
func secondsToMillis(s float64) int {
	if s <= 0 {
		return 0
	}
	return int(s*1000 + 0.5)
}

// ─── Hallucination filter ───────────────────────────────────────────────────

// Thresholds. These are whisper's OWN reference-implementation defaults, not
// numbers invented here — the upstream decoder uses the same values to decide a
// window is unusable, so a segment failing them is one whisper itself would
// have treated as suspect.
const (
	// hallucinationNoSpeechThreshold mirrors whisper's --no_speech_threshold.
	hallucinationNoSpeechThreshold = 0.6
	// hallucinationLogprobThreshold mirrors whisper's --logprob_threshold.
	hallucinationLogprobThreshold = -1.0
	// hallucinationCompressionThreshold mirrors whisper's
	// --compression_ratio_threshold. Above it the text repeats itself.
	hallucinationCompressionThreshold = 2.4
	// hallucinationTailNoSpeechThreshold is a DELIBERATELY LOOSER bar that
	// applies ONLY to a run of segments reaching the very end of the audio.
	// The POC failure (an invented "like & subscribe" outro over silent
	// credits) sits exactly there, and the tail is the least likely place for
	// real dialogue — the same bar applied mid-file would eat real lines.
	hallucinationTailNoSpeechThreshold = 0.4
	// hallucinationTailMinRun is how many trailing segments must agree before
	// the tail rule fires, so one soft-scoring last line is never dropped alone.
	hallucinationTailMinRun = 3
	// hallucinationRepeatRun is how many consecutive identical segments count
	// as a decoder loop rather than genuine repetition (a chant, a countdown).
	hallucinationRepeatRun = 3
	// hallucinationDropRatioWarn is the share of dropped segments above which
	// the caller should shout: the POC's real-vs-generated gap was ~5%
	// (1029 official cues vs 1082 generated), so a fifth of the file
	// disappearing means the thresholds are wrong, not the audio.
	hallucinationDropRatioWarn = 0.2
)

// Drop reasons (stable strings — they appear in logs and tests).
const (
	dropReasonSilence    = "silence"
	dropReasonRepetition = "repetition"
	dropReasonRepeatRun  = "repeat_run"
	dropReasonTail       = "tail"
	// dropReasonMusicOnly: the segment's text is only music marks (♪♪) —
	// whisper's way of saying "score, no words". Not a hallucination in the
	// strict sense, but not a subtitle either: it would be paid for in
	// translation and shown as a floating ♪♪ (See S01E02: 22 of them,
	// disc-2026-10-asr-music-only-cues). The embedded-track leg already drops
	// these in subtitle.FilterSDH (sub-6-4); this is the same rule for the
	// speech-recognition leg.
	dropReasonMusicOnly = "music_only"
	// dropReasonPromptEcho: the segment is a verbatim slice of the name
	// prompt we sent (disc-2026-10-asr-proper-names-inconsistent, CR M2) —
	// whisper's known habit over silence or score, when given a prompt, is
	// to "hear" the prompt. Confident, non-repeating, mid-file: none of the
	// other rules catch it.
	dropReasonPromptEcho = "prompt_echo"
)

// promptEchoMinRunes is how long a prompt slice must be before a segment
// matching it counts as an echo. One name alone ("Paris.") is a line someone
// could genuinely say; two or more names in a row are not dialogue.
const promptEchoMinRunes = 12

// filterPromptEcho removes segments whose text is a substring of the prompt
// the request carried (case- and punctuation-insensitive), at least
// promptEchoMinRunes long. PURE like filterHallucinations; a run without a
// prompt returns its input untouched.
func filterPromptEcho(segs []whisperSegment, prompt string) (kept []whisperSegment, dropped []droppedSegment) {
	norm := normalizeForEcho(prompt)
	if norm == "" {
		return segs, nil
	}
	for _, seg := range segs {
		text := normalizeForEcho(seg.Text)
		if len([]rune(text)) >= promptEchoMinRunes && strings.Contains(norm, text) {
			dropped = append(dropped, droppedSegment{Segment: seg, Reason: dropReasonPromptEcho})
			continue
		}
		kept = append(kept, seg)
	}
	return kept, dropped
}

// normalizeForEcho lower-cases and keeps only letters, digits and single
// spaces, so "Baba Voss, Paris" and "baba voss paris." compare equal.
func normalizeForEcho(s string) string {
	var sb strings.Builder
	lastSpace := true
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(r)
			lastSpace = false
		default:
			if !lastSpace {
				sb.WriteRune(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// droppedSegment is one filtered-out segment plus why it went.
type droppedSegment struct {
	Segment whisperSegment
	Reason  string
}

// filterHallucinations removes segments whisper most likely invented.
//
// PURE: no I/O, no logging, no clock — the caller owns observability. That is
// what makes every rule directly table-testable.
//
// Returning an EMPTY kept slice is legal and expected: a ten-minute chunk that
// is entirely silent credits SHOULD collapse to nothing. Guarding against an
// empty result belongs at the whole-FILE level, where "everything vanished"
// really is a bug (see TranscriptionDetail / transcribeAudio).
func filterHallucinations(segs []whisperSegment) (kept []whisperSegment, dropped []droppedSegment) {
	if len(segs) == 0 {
		return nil, nil
	}

	reasons := make([]string, len(segs))

	// R0 music-only, R1 silence, R2 per-segment repetition.
	for i, seg := range segs {
		switch {
		case IsMusicOnlyText(seg.Text):
			reasons[i] = dropReasonMusicOnly
		case seg.NoSpeechProb > hallucinationNoSpeechThreshold && seg.AvgLogprob < hallucinationLogprobThreshold:
			reasons[i] = dropReasonSilence
		case seg.CompressionRatio > hallucinationCompressionThreshold:
			reasons[i] = dropReasonRepetition
		}
	}

	// R2b repeat runs: N+ consecutive segments with identical text keep the
	// FIRST one (a real repeated line is said once and echoed; a decoder loop
	// emits it forever).
	for i := 0; i < len(segs); {
		j := i + 1
		for j < len(segs) && normalizedSegmentText(segs[j]) == normalizedSegmentText(segs[i]) {
			j++
		}
		if runLen := j - i; runLen >= hallucinationRepeatRun && normalizedSegmentText(segs[i]) != "" {
			for k := i + 1; k < j; k++ {
				if reasons[k] == "" {
					reasons[k] = dropReasonRepeatRun
				}
			}
		}
		i = j
	}

	// R3 tail: walk back from the end over segments that clear the looser tail
	// bar. EVERY segment in the run must clear it on its own evidence (AC #3.5)
	// — CR M1: an earlier draft also let an already-dropped segment extend the
	// run, which let one compression-ratio drop pull two otherwise-innocent
	// quiet lines out with it. The looser bar is only defensible where every
	// member earns it.
	tailStart := len(segs)
	for tailStart > 0 && segs[tailStart-1].NoSpeechProb > hallucinationTailNoSpeechThreshold {
		tailStart--
	}
	if len(segs)-tailStart >= hallucinationTailMinRun {
		for i := tailStart; i < len(segs); i++ {
			if reasons[i] == "" {
				reasons[i] = dropReasonTail
			}
		}
	}

	for i, seg := range segs {
		if reasons[i] == "" {
			kept = append(kept, seg)
			continue
		}
		dropped = append(dropped, droppedSegment{Segment: seg, Reason: reasons[i]})
	}
	return kept, dropped
}

// normalizedSegmentText is the comparison key for repeat-run detection.
//
// Case and trailing punctuation are stripped: a stuck decoder re-emits the same
// line with drifting terminal punctuation ("Thank you." / "thank you" /
// "Thank you!"), and comparing raw strings lets that drift hide the loop. The
// run-length bar (hallucinationRepeatRun) is what keeps this from touching real
// dialogue — two matching lines are a conversation, five are a malfunction.
func normalizedSegmentText(seg whisperSegment) string {
	lowered := strings.ToLower(strings.TrimSpace(seg.Text))
	return strings.TrimRight(lowered, " \t.,!?;:…。、！？，；：")
}

// dropReasonCounts summarizes a dropped slice for a single log line.
func dropReasonCounts(dropped []droppedSegment) map[string]int {
	if len(dropped) == 0 {
		return nil
	}
	counts := make(map[string]int, 4)
	for _, d := range dropped {
		counts[d.Reason]++
	}
	return counts
}

// ─── Transcription detail seam (Story 9R-5) ─────────────────────────────────

// Response-format field values. Named so the fallback path cannot drift from
// the request path by a typo.
const (
	transcribeFormatSRT         = "srt"
	transcribeFormatVerboseJSON = "verbose_json"
)

// TranscriptionDetail is one transcription plus what the hallucination filter
// removed from it.
//
// Unfiltered exists for ONE reason: the chunking loop owns the whole file and
// this package does not. A single chunk collapsing to nothing is legitimate
// (ten minutes of silent credits), but a whole FILE collapsing to nothing is a
// filter bug, and recovering from it needs the unfiltered rendering the caller
// never otherwise sees.
//
// Filtered is false when the engine could not serve verbose_json: there were no
// segments to judge, so SRT and Unfiltered are the same untouched response.
type TranscriptionDetail struct {
	SRT          string
	Unfiltered   string
	SegmentsIn   int
	SegmentsKept int
	Filtered     bool
}

// DropRatio is the share of segments the filter removed (0 when nothing was
// filtered or nothing came back).
func (d TranscriptionDetail) DropRatio() float64 {
	if !d.Filtered || d.SegmentsIn == 0 {
		return 0
	}
	return float64(d.SegmentsIn-d.SegmentsKept) / float64(d.SegmentsIn)
}

// DetailedTranscriber is the OPTIONAL companion to ASRProvider for engines that
// post-filter their own output (Story 9R-5).
//
// Deliberately separate from ASRProvider: 9R-9's contract is that any
// OpenAI-compatible engine can be dropped in behind that interface, and adding
// a method would break every alternative implementation. Consumers type-assert
// and degrade — the ai.CachingCompleter pattern.
type DetailedTranscriber interface {
	TranscribeDetailed(ctx context.Context, audioPath, lang string) (TranscriptionDetail, error)
}

// Compile-time proof the Whisper client serves the detailed seam.
var _ DetailedTranscriber = (*WhisperClient)(nil)
