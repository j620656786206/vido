package eval

import (
	"strings"

	"github.com/vido/api/internal/subtitle"
)

// Rule-layer failure reasons. The first four are the pipeline's own
// quality-gate classes (subtitle/quality_gate.go) so a report reads in the
// same vocabulary as the production logs; the rest are checks the gate does
// not have yet (sub-7-9 will add misaligned to the gate).
const (
	ReasonNameMismatch  = "name_mismatch"
	ReasonTimeShift     = "time_shift"
	ReasonForbiddenTerm = "forbidden_term"
)

// RuleCheck inspects the RAW model output for one cue (before OpenCC, the same
// place the pipeline's gate looks) and returns every failure class that
// applies, in a stable order. prev/next are the neighbouring cues (nil at the
// edges) — time_shift is only detectable against them.
//
// A cue with any failure scores 0 regardless of what the judge would say
// (AC #4): a line that is on the wrong cue or leaks simplified characters is
// unwatchable however fluent it is.
func RuleCheck(prev, cue, next *Cue, zh string) []string {
	var reasons []string

	// The gate expects a chunk; one cue is a chunk of one.
	verdict := subtitle.CheckChunk(
		[]subtitle.SubtitleBlock{{Index: 1, Text: cue.Text}},
		map[int]string{1: zh},
	)
	if r, bad := verdict.Reasons[1]; bad {
		reasons = append(reasons, r)
	}
	if verdict.Failed(1) && strings.TrimSpace(zh) == "" {
		// empty / missing: nothing else can be judged.
		return reasons
	}

	for _, variants := range cue.Names {
		if !containsAny(zh, variants) {
			reasons = append(reasons, ReasonNameMismatch)
			break
		}
	}

	if containsAny(zh, cue.Forbid) {
		reasons = append(reasons, ReasonForbiddenTerm)
	}

	own := anchorsOf(cue)
	if len(own) > 0 && !containsAny(zh, own) {
		neighbours := append(anchorsOf(prev), anchorsOf(next)...)
		if containsAny(zh, neighbours) {
			reasons = append(reasons, ReasonTimeShift)
		}
	}
	return reasons
}

// anchorsOf returns the tokens that identify a cue's own content: explicit
// Anchors when authored, otherwise the first rendering of each name.
func anchorsOf(c *Cue) []string {
	if c == nil {
		return nil
	}
	if len(c.Anchors) > 0 {
		return c.Anchors
	}
	var out []string
	for _, variants := range c.Names {
		if len(variants) > 0 {
			out = append(out, variants[0])
		}
	}
	return out
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if n != "" && strings.Contains(s, n) {
			return true
		}
	}
	return false
}
