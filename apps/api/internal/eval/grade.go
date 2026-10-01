package eval

// Grade thresholds, pre-registered in eval-1 AC #4 and NOT to be tuned after
// the fact: a model is watchable (A) when at most 5% of cues are unusable AND
// at least 60% need no edit at all. Meeting only one bar is B, neither is C.
const (
	MaxZeroRateForA = 0.05
	MinNaturalRateA = 0.60
	GradeA          = "A"
	GradeB          = "B"
	GradeC          = "C"
	GradeIncomplete = "incomplete"
)

// GradeFor maps the two rates to a letter.
func GradeFor(zeroRate, naturalRate float64) string {
	lowZero := zeroRate <= MaxZeroRateForA
	natural := naturalRate >= MinNaturalRateA
	switch {
	case lowZero && natural:
		return GradeA
	case lowZero || natural:
		return GradeB
	default:
		return GradeC
	}
}
