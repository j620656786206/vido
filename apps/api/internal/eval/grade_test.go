package eval

import "testing"

// Boundaries are inclusive exactly where eval-1 AC #4 put them.
func TestGradeFor_Boundaries(t *testing.T) {
	cases := []struct {
		zero, natural float64
		want          string
	}{
		{0.05, 0.60, GradeA},
		{0.013, 0.896, GradeA}, // eval-1 Sonnet
		{0.036, 0.718, GradeA}, // eval-1 Haiku (merged corpus)
		{0.051, 0.60, GradeB},
		{0.05, 0.599, GradeB},
		{0.20, 0.90, GradeB},
		{0.00, 0.10, GradeB},
		{0.06, 0.59, GradeC},
		{1, 0, GradeC},
	}
	for _, tc := range cases {
		if got := GradeFor(tc.zero, tc.natural); got != tc.want {
			t.Errorf("GradeFor(%v, %v) = %s, want %s", tc.zero, tc.natural, got, tc.want)
		}
	}
}
