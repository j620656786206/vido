package sidecarcheck

import (
	"encoding/binary"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Synthetic audio: speech where the "true" cues are, noise elsewhere ────

// speechEnergy is a window's frame energies: loud where a true cue is on,
// quieter (with noise, like a score under the dialogue) elsewhere.
func speechEnergy(startMS, frames int, truth []cue, seed uint64) []float64 {
	r := rand.New(rand.NewPCG(seed, 1))
	line := timeline(truth)
	out := make([]float64, frames)
	for k := range out {
		f := (startMS + k*frameMS + frameMS/2) / frameMS
		v := -3 + r.NormFloat64()*0.6
		if f < len(line) && line[f] {
			v = -1.5 + r.NormFloat64()*0.6
		}
		out[k] = v
	}
	return out
}

// dialogue is 20 minutes of lines 1.2–3.6 s long with gaps of 1–4 s.
func dialogue(seed uint64) []cue { return dialogueFor(seed, 20*60_000) }

func dialogueFor(seed uint64, lengthMS int) []cue {
	r := rand.New(rand.NewPCG(seed, 2))
	var cues []cue
	t := 5_000
	for t < lengthMS {
		d := 1_200 + r.IntN(2_400)
		cues = append(cues, cue{StartMS: t, EndMS: t + d})
		t += d + 1_000 + r.IntN(3_000)
	}
	return cues
}

func shifted(cues []cue, shiftMS int, scale float64) []cue {
	out := make([]cue, len(cues))
	for i, c := range cues {
		out[i] = cue{StartMS: int(float64(c.StartMS)*scale) + shiftMS, EndMS: int(float64(c.EndMS)*scale) + shiftMS}
	}
	return out
}

// run is the checker's pipeline over synthetic windows of a 20-minute film.
func run(truth, sub []cue) (Result, float64) {
	v := runFor(truth, sub, 20*60_000)
	return v.Result, v.OffsetSeconds
}

func runFor(truth, sub []cue, lengthMS int) Verdict {
	line := timeline(sub)
	var curves [][]float64
	var starts []int
	for i, w := range planWindows(lengthMS, windowCount, 90_000, minWindows) {
		e := speechEnergy(w.StartMS, w.LenMS/frameMS, truth, uint64(i+7))
		curves = append(curves, shiftScores(e, w.StartMS, line))
		starts = append(starts, w.StartMS)
	}
	return decide(curves, starts)
}

func TestDecide_Aligned(t *testing.T) {
	truth := dialogue(1)
	res, off := run(truth, truth)
	assert.Equal(t, Aligned, res)
	assert.InDelta(t, 0, off, 0.11)
}

func TestDecide_ShiftedWholeFile(t *testing.T) {
	truth := dialogue(2)
	res, off := run(truth, shifted(truth, 3_000, 1))
	assert.Equal(t, Offset, res)
	assert.InDelta(t, 3, off, 0.11, "positive = the subtitle is late")

	res, off = run(truth, shifted(truth, -2_000, 1))
	assert.Equal(t, Offset, res)
	assert.InDelta(t, -2, off, 0.11)
}

func TestDecide_FrameRateStretchIsMismatch(t *testing.T) {
	truth := dialogue(3)
	res, _ := run(truth, shifted(truth, 0, 25/23.976))
	assert.Equal(t, Mismatch, res, "the gap grows through the film — no single shift fits")
}

// 24 vs 23.976 fps: the gap grows by ~7 s over two hours — the sum still
// has one peak, the two halves of the film do not agree on it.
func TestDecide_SlowDriftIsMismatch(t *testing.T) {
	truth := dialogueFor(6, 120*60_000)
	v := runFor(truth, shifted(truth, 0, 24/23.976), 120*60_000)
	assert.Equal(t, Mismatch, v.Result, "%+v", v)

	v = runFor(truth, truth, 120*60_000)
	assert.Equal(t, Aligned, v.Result, "the same two-hour film, aligned, is not called a drift")
}

// The drift branch itself: no single window is strong enough to object, the
// sum still peaks, but the first half hears 0 s and the second +1.6 s.
func TestDecide_HalvesDisagreeIsDrift(t *testing.T) {
	bump := func(shiftMS int, seed uint64) []float64 {
		r := rand.New(rand.NewPCG(seed, 3))
		c := make([]float64, 2*maxShiftMS/frameMS+1)
		for i := range c {
			d := float64(i*frameMS-maxShiftMS-shiftMS) / 1000
			c[i] = r.NormFloat64() + 2.8*math.Exp(-d*d)
		}
		return c
	}
	var curves [][]float64
	var starts []int
	for i := range 12 {
		shift := 0
		if i >= 6 {
			shift = 1_600
		}
		curves = append(curves, bump(shift, uint64(i)))
		starts = append(starts, i)
	}
	v := decide(curves, starts)
	for _, w := range v.Windows {
		require.False(t, w.Found, "no window objects on its own: %+v", w)
	}
	require.GreaterOrEqual(t, v.Score, minGlobalPeak, "the sum alone would have passed")
	assert.Equal(t, Mismatch, v.Result)
	assert.Equal(t, "drift", v.Reason)
}

func TestDecide_AnotherFilmIsMismatch(t *testing.T) {
	res, _ := run(dialogue(4), dialogue(5))
	assert.Equal(t, Mismatch, res)
}

func TestDecide_WindowsHearingElsewhere(t *testing.T) {
	flat := make([]float64, 2*maxShiftMS/frameMS+1)
	at := func(shiftMS int) []float64 {
		c := append([]float64(nil), flat...)
		c[(shiftMS+maxShiftMS)/frameMS] = 10
		return c
	}
	v := decide([][]float64{at(0), at(0), at(8_000), at(8_000)}, []int{0, 1, 2, 3})
	assert.Equal(t, Mismatch, v.Result, "as many windows elsewhere as here")
	assert.Equal(t, "windows_disagree", v.Reason)

	v = decide([][]float64{at(0), at(0), at(0), at(8_000)}, []int{0, 1, 2, 3})
	assert.Equal(t, Aligned, v.Result, "one window elsewhere is outnumbered")

	v = decide([][]float64{at(0), at(0), at(0), at(200)}, []int{0, 1, 2, 3})
	assert.Equal(t, Aligned, v.Result, "within agreeMS")
	assert.Zero(t, v.OffsetSeconds)
	assert.Less(t, v.Score, maxProminence+0.01, "a spike over a flat curve stays a finite score")

	assert.Equal(t, Mismatch, decide(nil, nil).Result, "nothing decoded is not aligned")
}

func TestPlanWindows(t *testing.T) {
	ws := planWindows(120*60_000, 6, 90_000, 3)
	require.Len(t, ws, 6)
	assert.Equal(t, 6*60_000, ws[0].StartMS, "skips the opening (5% of two hours)")
	last := ws[5]
	assert.Equal(t, 120*60_000-12*60_000, last.StartMS+last.LenMS, "and the closing credits (10%)")
	for i := 1; i < len(ws); i++ {
		assert.Greater(t, ws[i].StartMS, ws[i-1].StartMS+ws[i-1].LenMS-1, "no overlap")
	}

	// Tears of Steel, 12:14: room for 6 windows of 90 s.
	assert.Len(t, planWindows(734_000, 6, 90_000, 3), 6)
	// Six minutes: too short for 3 windows, read whole.
	assert.Equal(t, []window{{StartMS: 0, LenMS: 360_000}}, planWindows(360_000, 6, 90_000, 3))
	assert.Nil(t, planWindows(0, 6, 90_000, 3))
}

func TestFrameEnergies(t *testing.T) {
	perFrame := sampleRate * frameMS / 1000
	pcm := make([]byte, 2*perFrame*2+10) // two frames + a partial one
	for s := range perFrame {
		binary.LittleEndian.PutUint16(pcm[(perFrame+s)*2:], uint16(int16(16384)))
	}
	e := frameEnergies(pcm)
	require.Len(t, e, 2, "the partial frame is dropped")
	assert.Less(t, e[0], -9.0, "silence")
	assert.InDelta(t, math.Log10(0.25), e[1], 0.001, "half scale")
}

func TestTimelineAndShiftScoresEdges(t *testing.T) {
	line := timeline([]cue{{StartMS: 100, EndMS: 300}})
	assert.Equal(t, []bool{false, true, true, false}, line)

	scores := shiftScores(make([]float64, 50), 0, line)
	for _, s := range scores {
		assert.Zero(t, s, "constant audio (silence) scores nothing")
	}
}
