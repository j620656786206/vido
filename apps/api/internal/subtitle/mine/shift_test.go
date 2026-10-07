package mine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func cuesFromTimings(t [][2]int) []Cue {
	out := make([]Cue, len(t))
	for i, c := range t {
		out[i] = Cue{StartMS: c[0], EndMS: c[1], Text: "x"}
	}
	return out
}

// disc-2026-10-mine-shift-range-too-narrow — the real shape: See S01E02's
// zh-TW sidecar carries a 51.7 s recap the video lacks. The old ±30 s window
// could not reach the true shift and settled on a −13.75 s coincidence that
// paired every line 38 s off.
func TestEstimateShift_SeeS01E02RecapOffset(t *testing.T) {
	en := cuesFromTimings(seeS01E02EnTimings)
	zh := cuesFromTimings(seeS01E02ZhTimings)
	got := EstimateShift(en, zh)
	assert.InDelta(t, -51500, got, 500, "the Chinese cues need ~51.5 s taken off to meet the English ones")
}

func TestEstimateShift_SmallConstantOffsetStillFound(t *testing.T) {
	en := cuesFromTimings(seeS01E02EnTimings)
	zh := make([]Cue, len(en))
	for i, c := range en {
		zh[i] = Cue{StartMS: c.StartMS + 4000, EndMS: c.EndMS + 4000, Text: "x"} // the Scorpion fansub case
	}
	assert.Equal(t, -4000, EstimateShift(en, zh))
}

func TestEstimateShift_AlignedStaysPut(t *testing.T) {
	en := cuesFromTimings(seeS01E02EnTimings)
	assert.Equal(t, 0, EstimateShift(en, en))
}

// A wide window must not let noise win (CR M1 — these inputs clear the old
// "beats as-is" margin and are stopped only by the 35 % rule):
//   - the true offset sits just OUTSIDE the window, so the best in-window
//     shift is a coincidence at the window's edge;
//   - a frame-rate drift (25 vs 23.976) that no constant shift fits.
func TestEstimateShift_NoiseInsideTheWindowIsRejected(t *testing.T) {
	en := cuesFromTimings(seeS01E02EnTimings)
	t.Run("offset beyond the window", func(t *testing.T) {
		zh := make([]Cue, len(en))
		for i, c := range en {
			zh[i] = Cue{StartMS: c.StartMS + 181_000, EndMS: c.EndMS + 181_000, Text: "x"}
		}
		assert.Equal(t, 0, EstimateShift(en, zh))
	})
	t.Run("frame-rate drift", func(t *testing.T) {
		zh := make([]Cue, len(en))
		for i, c := range en {
			zh[i] = Cue{StartMS: c.StartMS*25/23976*1000 + 4000, EndMS: c.EndMS*25/23976*1000 + 4000, Text: "x"}
		}
		assert.Equal(t, 0, EstimateShift(en, zh))
	})
}

// CR L1: a tiny zh file (a forced-only sidecar) must not be shifted on a
// handful of coincidences — 35 % of 20 cues is seven hits, noise manages that.
func TestEstimateShift_TinyFileNeedsRealHits(t *testing.T) {
	en := cuesFromTimings(seeS01E02EnTimings)
	zh := make([]Cue, 0, 20)
	for i := 0; i < 20; i++ {
		zh = append(zh, Cue{StartMS: 913 + i*97_531, EndMS: 913 + i*97_531 + 1200, Text: "x"})
	}
	assert.Equal(t, 0, EstimateShift(en, zh))
	// …while a genuine 4 s offset on a 20-cue subset still hits all 20 and moves.
	sub := make([]Cue, 20)
	for i := range sub {
		c := en[i*20]
		sub[i] = Cue{StartMS: c.StartMS + 4000, EndMS: c.EndMS + 4000, Text: "x"}
	}
	assert.Equal(t, -4000, EstimateShift(cuesFromTimings(seeS01E02EnTimings), sub))
}

func TestEstimateShift_TooFewCuesIsZero(t *testing.T) {
	en := cuesFromTimings(seeS01E02EnTimings[:10])
	assert.Equal(t, 0, EstimateShift(en, en))
}
