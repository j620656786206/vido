package subtitle

import (
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Quality-gate failure classes. These are the per-cue verdict reasons the
// pipeline logs and, for a stubborn cue, carries into the pilot data.
const (
	GateReasonMissing        = "missing"
	GateReasonEmpty          = "empty"
	GateReasonEchoed         = "echoed"
	GateReasonSimplifiedLeak = "simplified_leak"
)

// GateVerdict is one chunk's inspection result.
type GateVerdict struct {
	FailedIndexes []int          // cue Index values needing retry, in source order
	Reasons       map[int]string // per-cue failure class
}

// Passed reports whether every cue in the inspected chunk is deliverable.
func (v GateVerdict) Passed() bool { return len(v.FailedIndexes) == 0 }

// Failed reports whether a specific cue Index needs a retry.
func (v GateVerdict) Failed(index int) bool {
	_, bad := v.Reasons[index]
	return bad
}

// latinRunPattern matches a run of at least three consecutive Latin letters.
// It is the echo check's false-positive guard: an interjection ("OK!"), a
// number ("1999") or a short initialism legitimately survives translation
// unchanged, so equality alone must not condemn a cue.
var latinRunPattern = regexp.MustCompile(`[A-Za-z]{3,}`)

// CheckChunk inspects RAW (pre-OpenCC) translations for one chunk.
//
// It runs BEFORE the OpenCC pass on purpose (architecture P4): converting first
// would silently repair a Simplified leak, the retry could never fire, and this
// whole gate would be dead code. The order is proven by
// TestTranslateTrack_ConverterRunsOnlyAfterGatePasses.
//
// Exactly one reason is recorded per failing cue — the first class that
// matches, checked in escalating order of how much the model got wrong.
func CheckChunk(source []SubtitleBlock, got map[int]string) GateVerdict {
	return checkChunk(source, got, slog.Default())
}

// checkChunk is CheckChunk with an injected logger, so the pipeline's
// component-tagged logger carries the unexpected-index warnings into the pilot
// logs. The exported wrapper keeps the AC #1 contract signature intact.
func checkChunk(source []SubtitleBlock, got map[int]string, logger *slog.Logger) GateVerdict {
	verdict := GateVerdict{Reasons: make(map[int]string)}

	expected := make(map[int]struct{}, len(source))
	for _, b := range source {
		expected[b.Index] = struct{}{}

		text, ok := got[b.Index]
		switch {
		case !ok:
			// Covers the cue-count mismatch case too: a short response simply
			// leaves the tail cues unanswered.
			verdict.fail(b.Index, GateReasonMissing)
		case strings.TrimSpace(text) == "":
			verdict.fail(b.Index, GateReasonEmpty)
		case isEchoed(b.Text, text):
			verdict.fail(b.Index, GateReasonEchoed)
		case Detect([]byte(text)).SimplifiedCount > 0:
			// STRICT by design: any simplified-only character fails the cue.
			// The detector's ratio thresholds classify a document's variant;
			// they are not a leak detector, and one 这 in a delivered cue is
			// exactly the defect FR16 exists to catch.
			verdict.fail(b.Index, GateReasonSimplifiedLeak)
		}
	}

	logUnexpectedIndexes(expected, got, logger)
	return verdict
}

// fail records one cue's failure class, keeping FailedIndexes ordered.
func (v *GateVerdict) fail(index int, reason string) {
	if _, dup := v.Reasons[index]; dup {
		return
	}
	v.Reasons[index] = reason
	v.FailedIndexes = append(v.FailedIndexes, index)
}

// isEchoed reports whether the model handed back the source instead of a
// translation. Equality is normalized (case, surrounding and interior
// whitespace) and must be paired with a run of Latin letters — see
// latinRunPattern for why.
//
// Known, accepted trade-off: a cue that is ONLY a long proper noun ("John
// Smith") is normalized-equal and does carry a Latin run, so it is flagged and
// retried. Cost is bounded at two extra chunk calls, after which the stubborn
// policy delivers the very text that was already correct — the gate can waste a
// call here, never corrupt output.
func isEchoed(source, translated string) bool {
	if !latinRunPattern.MatchString(translated) {
		return false
	}
	return normalizeForEcho(source) == normalizeForEcho(translated)
}

// normalizeForEcho collapses case, whitespace and punctuation so
// "This  Software\nis great." and "this software is great" compare equal — a
// model that echoes the source while dropping the final period is still an
// echo, and without this an untranslated English cue would ship uncounted.
// Punctuation becomes a space (not deleted) so "Hi,Bob" still splits into the
// same fields as "Hi Bob". The Latin-run guard in isEchoed keeps "OK!" /
// "J.T." / "Mr. Ed" from false-positiving.
func normalizeForEcho(s string) string {
	depunct := strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return ' '
		}
		return r
	}, s)
	return strings.ToLower(strings.Join(strings.Fields(depunct), " "))
}

// logUnexpectedIndexes reports indexes the model invented. They are ignored
// rather than fatal — the requested cues are all accounted for by the loop
// above — but a model answering for cues nobody asked about is worth seeing in
// the pilot logs (Rule 13: surfaced, not silently dropped).
func logUnexpectedIndexes(expected map[int]struct{}, got map[int]string, logger *slog.Logger) {
	var extra []int
	for idx := range got {
		if _, want := expected[idx]; !want {
			extra = append(extra, idx)
		}
	}
	if len(extra) == 0 {
		return
	}
	sort.Ints(extra)
	logger.Warn("subtitle quality gate ignored unexpected cue indexes in LLM response",
		"unexpected_indexes", extra,
		"chunk_size", len(expected),
	)
}
