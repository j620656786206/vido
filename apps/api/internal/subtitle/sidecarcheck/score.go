// Package sidecarcheck tells, for free and on the NAS itself, whether an
// English subtitle file beside a video matches that video's timing
// (disc-2026-10-generate-keeps-user-en-srt-b1).
//
// A subtitle the user downloaded may be for another cut (a longer intro: the
// whole file late by a few seconds), another frame rate (23.976 vs 25: the
// gap grows through the film) or another film altogether. Translating it
// would cost money and produce Chinese that is just as wrong, so -b2 only
// routes a sidecar to translation after this check says it lines up.
//
// The first plan reused ffmpeg silencedetect (ai/whisper.go detectSilences);
// on Tears of Steel it found 3 silences in 12 minutes — the score never stops
// — which cannot judge 76 cues. So this listens for speech instead: the audio
// is band-passed to the voice range, its energy taken every 100 ms, and the
// subtitle's on/off timeline is slid ±15 s across it to find where the two
// agree best. Only a few windows of the film are decoded, so a 4K remux costs
// seconds, not the minutes a full audio extraction takes.
package sidecarcheck

import (
	"encoding/binary"
	"math"
)

// Result is the check's verdict.
type Result string

const (
	// Aligned: the subtitle matches the audio as it is.
	Aligned Result = "aligned"
	// Offset: the whole subtitle is early or late by the same amount.
	Offset Result = "offset"
	// Mismatch: no single shift fits — another frame rate, another cut, or
	// another film.
	Mismatch Result = "mismatch"
	// Unusable: not an English subtitle that can be translated (unreadable,
	// fewer than minCues cues, or not English).
	Unusable Result = "unusable"
)

const (
	// frameMS is the time resolution of both the audio and the subtitle.
	frameMS = 100
	// maxShiftMS is how far the subtitle is slid either way.
	maxShiftMS = 15_000
	// sampleRate is what ffmpeg decodes the window to (16-bit mono).
	sampleRate = 16_000
)

// Thresholds, measured with the exam in
// _bmad-output/audit/sidecar-alignment-exam-2026-10.md.
var (
	// minWindowPeak is how far (in standard deviations of the 301 shift
	// scores) a window's best shift must stand out to count as "found".
	minWindowPeak = 5.0
	// minGlobalPeak is the same for the curve summed over all windows. The
	// exam's weakest real match scored 6.4, its strongest wrong one 3.35.
	minGlobalPeak = 4.9
	// agreeMS is how close a found window's best shift must be to the
	// overall best shift.
	agreeMS = 1_000
	// alignedMS is the largest overall shift still reported as aligned.
	alignedMS = 500
	// edgeMS: a window's best shift this close to ±maxShiftMS is not "found".
	edgeMS = 1_000
	// minHalfWindows: the drift check needs this many windows in each half.
	minHalfWindows = 3
	// minHalfPeak: each half's own sum must stand out this much before its
	// shift is believed.
	minHalfPeak = 4.9
	// maxProminence caps a peak's score so it stays a finite number.
	maxProminence = 99.0
)

// window is one stretch of the film to decode, in ms.
type window struct{ StartMS, LenMS int }

// WindowResult is what one window found.
type WindowResult struct {
	StartSeconds  float64 `json:"start_seconds"`
	OffsetSeconds float64 `json:"offset_seconds"`
	// Peak is how far the best shift stands out, in standard deviations.
	Peak float64 `json:"peak"`
	// Found is Peak ≥ minWindowPeak.
	Found bool `json:"found"`
}

// planWindows picks n windows of lenMS spread over the film, skipping the
// opening and the closing credits. A film too short for minWindows of them
// is read whole, as one window. PURE.
func planWindows(durationMS, n, lenMS, minWindows int) []window {
	if durationMS <= 0 {
		return nil
	}
	head := max(60_000, durationMS*5/100)
	tail := max(120_000, durationMS*10/100)
	usable := durationMS - head - tail
	count := min(n, usable/lenMS)
	if usable <= 0 || count < minWindows {
		return []window{{StartMS: 0, LenMS: durationMS}}
	}
	// Even spacing: the gaps between windows are equal, the first starts at
	// head and the last ends at durationMS - tail.
	gap := 0
	if count > 1 {
		gap = (usable - count*lenMS) / (count - 1)
	}
	out := make([]window, count)
	for i := range out {
		out[i] = window{StartMS: head + i*(lenMS+gap), LenMS: lenMS}
	}
	return out
}

// frameEnergies turns 16-bit little-endian mono PCM into one log energy per
// frameMS. A trailing partial frame is dropped. PURE.
func frameEnergies(pcm []byte) []float64 {
	perFrame := sampleRate * frameMS / 1000
	n := len(pcm) / 2 / perFrame
	out := make([]float64, n)
	for f := range n {
		var sum float64
		for s := range perFrame {
			off := (f*perFrame + s) * 2
			v := float64(int16(binary.LittleEndian.Uint16(pcm[off:]))) / 32768
			sum += v * v
		}
		out[f] = math.Log10(sum/float64(perFrame) + 1e-10)
	}
	return out
}

// cue is one subtitle line in ms.
type cue struct{ StartMS, EndMS int }

// maxTimelineMS bounds the timeline: a corrupt cue end cannot allocate more
// than ten hours of frames.
const maxTimelineMS = 10 * 3_600_000

// timeline marks, per frame from 0 to the last cue's end, whether a cue is on
// screen. PURE.
func timeline(cues []cue) []bool {
	end := 0
	for _, c := range cues {
		end = max(end, min(c.EndMS, maxTimelineMS))
	}
	out := make([]bool, end/frameMS+1)
	for _, c := range cues {
		for f := max(c.StartMS, 0) / frameMS; f*frameMS < c.EndMS && f < len(out); f++ {
			out[f] = true
		}
	}
	return out
}

// shiftScores correlates a window's energy with the subtitle timeline slid by
// every shift in ±maxShiftMS (step frameMS). Index i is shift
// (i*frameMS - maxShiftMS); a positive shift means the subtitle is LATE:
// speech heard at t is on screen at t + shift. A shift where the subtitle
// never changes inside the window scores 0. PURE.
func shiftScores(energy []float64, startMS int, line []bool) []float64 {
	steps := 2*maxShiftMS/frameMS + 1
	out := make([]float64, steps)
	z := zscore(energy)
	if z == nil {
		return out
	}
	mask := make([]float64, len(z))
	for i := range steps {
		shift := i*frameMS - maxShiftMS
		var on float64
		for k := range z {
			at := startMS + k*frameMS + frameMS/2 + shift
			f := at / frameMS
			mask[k] = 0
			if at >= 0 && f < len(line) && line[f] {
				mask[k] = 1
				on++
			}
		}
		p := on / float64(len(z))
		if p == 0 || p == 1 {
			continue
		}
		// Pearson correlation of z (mean 0, sd 1) with a 0/1 mask.
		var dot float64
		for k := range z {
			dot += z[k] * (mask[k] - p)
		}
		out[i] = dot / float64(len(z)) / math.Sqrt(p*(1-p))
	}
	return out
}

// zscore standardizes xs; nil when they are constant (silence). PURE.
func zscore(xs []float64) []float64 {
	if len(xs) < 2 {
		return nil
	}
	var mean float64
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	var sd float64
	for _, x := range xs {
		sd += (x - mean) * (x - mean)
	}
	sd = math.Sqrt(sd / float64(len(xs)))
	if sd == 0 {
		return nil
	}
	out := make([]float64, len(xs))
	for i, x := range xs {
		out[i] = (x - mean) / sd
	}
	return out
}

// peakGuardMS is the neighbourhood around the best shift left out when
// measuring how much it stands out: a correlation peak is a hill a couple of
// seconds wide (a line lasts ~2 s), not a spike, and counting its own slopes
// as "the rest of the curve" would hide even a perfect match.
const peakGuardMS = 2_000

// peak returns the best shift (ms) of a score curve and its prominence: how
// many standard deviations it stands above the curve away from the peak. PURE.
func peak(scores []float64) (shiftMS int, prominence float64) {
	best := 0
	for i, s := range scores {
		if s > scores[best] {
			best = i
		}
	}
	shiftMS = best*frameMS - maxShiftMS
	guard := peakGuardMS / frameMS
	var rest []float64
	for i, s := range scores {
		if i < best-guard || i > best+guard {
			rest = append(rest, s)
		}
	}
	if len(rest) < 2 {
		return shiftMS, 0
	}
	var mean, sd float64
	for _, s := range rest {
		mean += s
	}
	mean /= float64(len(rest))
	for _, s := range rest {
		sd += (s - mean) * (s - mean)
	}
	sd = math.Sqrt(sd / float64(len(rest)))
	if sd == 0 {
		if scores[best] > mean {
			// A spike over a flat curve: as clear as it gets, but kept
			// finite — the verdict is stored as JSON, which has no Inf.
			return shiftMS, maxProminence
		}
		return shiftMS, 0
	}
	return shiftMS, math.Min((scores[best]-mean)/sd, maxProminence)
}

// decide turns per-window score curves into the verdict (Result, offset,
// score, windows and the reason for a Mismatch). PURE.
func decide(curves [][]float64, starts []int) Verdict {
	v := Verdict{Windows: make([]WindowResult, len(curves))}
	for i, c := range curves {
		shift, strength := peak(c)
		v.Windows[i] = WindowResult{
			StartSeconds:  float64(starts[i]) / 1000,
			OffsetSeconds: float64(shift) / 1000,
			Peak:          round2(strength),
			// A best shift at the edge of the search is where the curve was
			// cut off, not a hill: it cannot overrule the sum.
			Found: strength >= minWindowPeak && abs(shift) < maxShiftMS-edgeMS,
		}
	}
	if len(curves) == 0 {
		v.Result, v.Reason = Mismatch, "no_peak"
		return v
	}
	shift, strength := peak(zsum(curves))
	v.OffsetSeconds, v.Score = float64(shift)/1000, round2(strength)
	if strength < minGlobalPeak {
		v.Result, v.Reason = Mismatch, "no_peak"
		return v
	}

	// The summed curve decides; a single window is often too quiet to find
	// anything on its own (Tears of Steel: 2 of 6 windows found the shift,
	// the sum found it at 9 sd). Windows that clearly hear the subtitle
	// somewhere else overrule it only when they are not outnumbered by
	// windows that agree: one episode had a stretch the English track does
	// not subtitle the way it is spoken, and that single window found +12.8 s
	// against two that found the true -2 s.
	agree, disagree := 0, 0
	for _, w := range v.Windows {
		if !w.Found {
			continue
		}
		if math.Abs(w.OffsetSeconds*1000-float64(shift)) > float64(agreeMS) {
			disagree++
		} else {
			agree++
		}
	}
	if disagree > 0 && disagree >= agree {
		v.Result, v.Reason = Mismatch, "windows_disagree"
		return v
	}

	// A slow drift (24 vs 23.976 fps: ~7 s over two hours) still leaves the
	// sum one peak, and few single windows are strong enough to object. The
	// first and second half of the film, summed apart, are: if both find a
	// clear shift and they differ, the gap moves through the film.
	if n := len(curves); n >= 2*minHalfWindows {
		s1, p1 := peak(zsum(curves[:n/2]))
		s2, p2 := peak(zsum(curves[n/2:]))
		if p1 >= minHalfPeak && p2 >= minHalfPeak && abs(s1-s2) > agreeMS {
			v.Result, v.Reason = Mismatch, "drift"
			return v
		}
	}

	if abs(shift) <= alignedMS {
		v.Result = Aligned
	} else {
		v.Result = Offset
	}
	return v
}

// zsum adds the curves after standardizing each, so one loud window (an
// action scene) does not outweigh eleven quiet ones. On the exam this widened
// the gap between the weakest real match and the strongest wrong one from
// +1.9 to +3.1 sd. PURE.
func zsum(curves [][]float64) []float64 {
	var sum []float64
	for _, c := range curves {
		if sum == nil {
			sum = make([]float64, len(c))
		}
		for k, z := range zscore(c) {
			sum[k] += z
		}
	}
	return sum
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }
