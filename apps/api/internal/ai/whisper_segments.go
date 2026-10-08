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
	//
	// Was 3 (disc-2026-10-asr-repeated-lines-dropped): on See S01E02 that ate
	// real dialogue — 「轉過來看我」shouted three times (4:21–4:37), the name
	// 「謝拉馬威」called three times (7:46–7:58) — and with the ♪♪ runs now
	// handled by music_only the biggest honest source of 3-runs is gone. A
	// decoder that is actually stuck re-emits the line far more than four
	// times; five identical segments in a row is where "someone is shouting"
	// stops being the likelier story. Tuned blind from one episode — the
	// 10-minute ASR clip verifies it.
	hallucinationRepeatRun = 5
	// hallucinationSpacedRepeatRun / hallucinationSpacedRepeatGapSeconds: the
	// OTHER loop shape (disc-2026-10-asr-repeat-run-bridging). Over score,
	// whisper emits one identical line per 30-second window — the 10-minute
	// See clip (sent whole, before chunk-at-silence) kept four of them because
	// four is under hallucinationRepeatRun. Chunked runs filter each ~120 s
	// chunk on its own, so a loop is caught only where 3+ windows land in one
	// chunk; a loop split across a cut can still leave a stray line.
	// Real repeated shouting is seconds apart ("Face me!" ×2, 「轉過來看我」×3
	// inside 16 s), so identical lines that keep coming 20 s+ apart are the
	// decoder, not the actor. Three, not two: a short answer ("Yes.") said
	// twice half a minute apart is ordinary dialogue, and letting one stray
	// fake line through costs less than eating a real one.
	hallucinationSpacedRepeatRun        = 3
	hallucinationSpacedRepeatGapSeconds = 20.0
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
	// dropReasonRepeatSpaced: the same line, again and again, 20 s+ apart —
	// see hallucinationSpacedRepeatRun.
	dropReasonRepeatSpaced = "repeat_spaced"
	dropReasonTail         = "tail"
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

// promptEchoMatcher returns the test for a segment being a slice of the prompt
// the request carried (case- and punctuation-insensitive, at least
// promptEchoMinRunes long, or a list of prompt names). A run without a prompt
// gets nil: nothing is an echo.
//
// It MARKS rather than removes (disc-2026-10-asr-repeat-run-bridging): the
// caller feeds it into filterHallucinationsWith alongside the other rules, so
// an echo sitting between two groups of the same shouted line keeps them apart
// instead of splicing them into one run long enough to look like a loop.
func promptEchoMatcher(prompt string) func(text string) bool {
	norm := normalizeForEcho(prompt)
	if norm == "" {
		return nil
	}
	names := promptNameSet(prompt)
	return func(raw string) bool {
		text := normalizeForEcho(raw)
		return (len([]rune(text)) >= promptEchoMinRunes && strings.Contains(norm, text)) || isNameList(raw, names)
	}
}

// promptNameSet is the prompt's comma-separated names, normalized.
func promptNameSet(prompt string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, n := range strings.Split(prompt, ",") {
		if k := normalizeForEcho(n); k != "" {
			set[k] = struct{}{}
		}
	}
	return set
}

// isNameList catches the echo the substring rule misses: whisper reading the
// list back out of order, with repeats or a dangling fragment ("The Bank,
// Lord Diego, Oloman, Shiloh, …, The Bank, Lord"). The mark of an echo is
// that EVERY comma-separated item is a prompt name or the cut-off start of
// one — the moment a free word appears ("Baba Voss, Maghra, come.") it is a
// line of dialogue and stays (CR 1). Two names at least; one alone is a line.
func isNameList(text string, names map[string]struct{}) bool {
	if len(names) == 0 {
		return false
	}
	items := strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '、' || r == ';' })
	if len(items) < 2 {
		return false
	}
	matched := 0
	for _, it := range items {
		k := normalizeForEcho(it)
		if k == "" {
			continue
		}
		if _, ok := names[k]; ok {
			matched++
			continue
		}
		if !isNamePrefix(k, names) {
			return false
		}
	}
	return matched >= 2
}

// isNamePrefix is true when k is the beginning of some prompt name (a
// fragment whisper cut mid-name: "Mag" of "Maghra", "Lord" of "Lord Diego")
// or one whole word of a multi-word name ("Kane" of "Queen Kane", "Jun" of
// "Tamacti Jun" — the fifth NAS run read the list back with exactly those).
func isNamePrefix(k string, names map[string]struct{}) bool {
	for n := range names {
		if len(k) < len(n) && strings.HasPrefix(n, k) {
			return true
		}
		if strings.Contains(n, " ") {
			for _, w := range strings.Fields(n) {
				if w == k {
					return true
				}
			}
		}
	}
	return false
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
	return filterHallucinationsWith(segs, true, "")
}

// filterHallucinationsWith is filterHallucinations with the R3 tail rule
// switchable: R3 is written for the END OF THE FILM (credits over score), and
// on a chunked run only the last upload is that
// (disc-2026-10-asr-chunk-at-silence, CR 1 — on a 120 s grid every cut sits
// in a pause, so every chunk's last lines are "quiet speech before a pause",
// exactly what the looser tail bar would eat).
//
// prompt is the name prompt the request carried ("" when none): segments that
// echo it are marked prompt_echo here, in place, like every other rule.
func filterHallucinationsWith(segs []whisperSegment, applyTail bool, prompt string) (kept []whisperSegment, dropped []droppedSegment) {
	if len(segs) == 0 {
		return nil, nil
	}

	reasons := make([]string, len(segs))
	isEcho := promptEchoMatcher(prompt)

	// R0 prompt echo / music-only, R1 silence, R2 per-segment repetition.
	for i, seg := range segs {
		switch {
		case isEcho != nil && isEcho(seg.Text):
			reasons[i] = dropReasonPromptEcho
		case IsMusicOnlyText(seg.Text):
			reasons[i] = dropReasonMusicOnly
		case seg.NoSpeechProb > hallucinationNoSpeechThreshold && seg.AvgLogprob < hallucinationLogprobThreshold:
			reasons[i] = dropReasonSilence
		case seg.CompressionRatio > hallucinationCompressionThreshold:
			reasons[i] = dropReasonRepetition
		}
	}

	// R2b repeat runs: a stretch of segments with identical text keeps the
	// FIRST one when it is a loop (a real repeated line is said once and
	// echoed; a decoder loop emits it forever). Only segments no other rule
	// has already claimed count (disc-2026-10-asr-repeat-run-bridging): four
	// real shouts plus one same-text window judged silence are four shouts,
	// not a five-long loop. A claimed segment with the SAME text stays inside
	// the stretch without adding to it; one with different text (a prompt
	// echo between two groups of shouts) ends it, like any other line.
	//
	// Two shapes are a loop:
	//   - hallucinationRepeatRun+ in a row, however close (a stuck decoder);
	//   - hallucinationSpacedRepeatRun+ in a row with every gap between
	//     starts at least hallucinationSpacedRepeatGapSeconds (whisper emitting
	//     one line per 30-second window over score — nobody shouts the same
	//     words at half-minute intervals).
	for i := 0; i < len(segs); {
		key := normalizedSegmentText(segs[i])
		j := i + 1
		for j < len(segs) && normalizedSegmentText(segs[j]) == key {
			j++
		}
		if key != "" {
			markRepeatRun(segs[i:j], reasons[i:j])
		}
		i = j
	}

	// R3 tail: walk back from the end over segments that clear the looser tail
	// bar. EVERY segment in the run must clear it on its own evidence (AC #3.5)
	// — CR M1: an earlier draft also let an already-dropped segment extend the
	// run, which let one compression-ratio drop pull two otherwise-innocent
	// quiet lines out with it. The looser bar is only defensible where every
	// member earns it.
	//
	// A prompt echo is transparent here: it used to be removed before this
	// filter ran (disc-2026-10-asr-repeat-run-bridging moved it in place), so
	// it neither ends the run nor counts toward hallucinationTailMinRun.
	tailStart, tailRun := len(segs), 0
	for applyTail && tailStart > 0 {
		prev := tailStart - 1
		if reasons[prev] == dropReasonPromptEcho {
			tailStart--
			continue
		}
		if segs[prev].NoSpeechProb <= hallucinationTailNoSpeechThreshold {
			break
		}
		tailStart--
		tailRun++
	}
	if applyTail && tailRun >= hallucinationTailMinRun {
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

// markRepeatRun applies R2b to one stretch of identical-text segments.
func markRepeatRun(run []whisperSegment, reasons []string) {
	var live []int
	for k := range run {
		if reasons[k] == "" {
			live = append(live, k)
		}
	}
	if len(live) < 2 {
		return
	}
	reason := ""
	switch {
	case len(live) >= hallucinationRepeatRun:
		reason = dropReasonRepeatRun
	case len(live) >= hallucinationSpacedRepeatRun && evenlySpacedApart(run, live):
		reason = dropReasonRepeatSpaced
	default:
		return
	}
	for _, k := range live[1:] {
		reasons[k] = reason
	}
}

// evenlySpacedApart is true when every consecutive pair in idx starts at least
// hallucinationSpacedRepeatGapSeconds after the one before it.
func evenlySpacedApart(run []whisperSegment, idx []int) bool {
	for n := 1; n < len(idx); n++ {
		if run[idx[n]].Start-run[idx[n-1]].Start < hallucinationSpacedRepeatGapSeconds {
			return false
		}
	}
	return true
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
