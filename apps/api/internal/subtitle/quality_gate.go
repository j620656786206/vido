package subtitle

import (
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/vido/api/internal/ai/prompts"
)

// Quality-gate failure classes. These are the per-cue verdict reasons the
// pipeline logs and, for a stubborn cue, carries into the pilot data.
const (
	GateReasonMissing        = "missing"
	GateReasonEmpty          = "empty"
	GateReasonEchoed         = "echoed"
	GateReasonSimplifiedLeak = "simplified_leak"
	// GateReasonMisaligned (sub-7-9): the model moved this cue's content onto
	// a neighbouring cue — an anchor token the source carries is missing from
	// this translation and shows up in the previous or next cue's translation,
	// whose own source never had it. eval-1 counted 120 of 484 zero-score cues
	// as exactly this. ADDITIVE on the reason vocabulary (0 bump).
	GateReasonMisaligned = "misaligned"
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

// CheckChunkAnchored is CheckChunk plus the sub-7-9 alignment check.
//
// anchors maps a cue Index to the tokens its translation must carry (see
// AnchorsFor); neighbours supplies already-accepted translations for cues
// OUTSIDE got (the previous attempt's survivors, the previous chunk's tail) so
// a cue retried alone can still be compared against the line it drifted
// into. Both may be nil, which reduces this to CheckChunk.
func CheckChunkAnchored(source []SubtitleBlock, got map[int]string, anchors map[int][]string, neighbours map[int]string) GateVerdict {
	return checkChunkAnchored(source, got, anchors, neighbours, slog.Default())
}

// checkChunk is CheckChunk with an injected logger, so the pipeline's
// component-tagged logger carries the unexpected-index warnings into the pilot
// logs. The exported wrapper keeps the AC #1 contract signature intact.
func checkChunk(source []SubtitleBlock, got map[int]string, logger *slog.Logger) GateVerdict {
	return checkChunkAnchored(source, got, nil, nil, logger)
}

func checkChunkAnchored(source []SubtitleBlock, got map[int]string, anchors map[int][]string, neighbours map[int]string, logger *slog.Logger) GateVerdict {
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
		case Detect([]byte(text)).SimplifiedCount > 0 || containsSimplifiedWord(text):
			// STRICT by design: any simplified-only character fails the cue.
			// The detector's ratio thresholds classify a document's variant;
			// they are not a leak detector, and one 这 in a delivered cue is
			// exactly the defect FR16 exists to catch. Characters that are
			// ALSO legitimate Traditional (里、几、准…) left the detector's set
			// (disc-2026-10-simplified-leak-false-positive-li); their common
			// SIMPLIFIED words are caught here as words instead.
			verdict.fail(b.Index, GateReasonSimplifiedLeak)
		case isMisaligned(b.Index, text, anchors, got, neighbours):
			verdict.fail(b.Index, GateReasonMisaligned)
		}
	}

	logUnexpectedIndexes(expected, got, logger)
	return verdict
}

// isMisaligned reports whether one of this cue's anchors is missing from its
// own translation yet present in an ADJACENT cue's translation that had no
// business carrying it. Adjacency is by cue Index (±1), not by position: SDH
// filtering leaves gaps, and a cue two indexes away is not where a sentence
// break drifts to.
//
// It is deliberately conservative: a translation that merely drops a number
// (「四個老婆」 for "4 wives") never trips it — the anchor also has to turn up
// next door, unexplained. That is the shape of a shifted line, and nothing
// else the gate has seen produces it.
func isMisaligned(index int, text string, anchors map[int][]string, got map[int]string, neighbours map[int]string) bool {
	own := anchors[index]
	if len(own) == 0 {
		return false
	}
	textOf := func(i int) (string, bool) {
		if t, ok := got[i]; ok {
			return t, true
		}
		t, ok := neighbours[i]
		return t, ok
	}
	for _, a := range own {
		if strings.Contains(text, a) {
			continue
		}
		for _, n := range []int{index - 1, index + 1} {
			nt, ok := textOf(n)
			if !ok || !strings.Contains(nt, a) {
				continue
			}
			if containsAnchor(anchors[n], a) {
				// The neighbour legitimately carries it too ("Line 7" / "7 of
				// them"): not evidence of a shift.
				continue
			}
			return true
		}
	}
	return false
}

func containsAnchor(list []string, a string) bool {
	for _, x := range list {
		if x == a {
			return true
		}
	}
	return false
}

// anchorDigits matches the numeric tokens a translation is expected to keep as
// digits (Taiwan subtitles write 14 and 2002, not 十四 and 二〇〇二). Lone
// digits are too common to anchor on: "4 wives" → 「四個老婆」 is good
// translation, and a false misaligned verdict costs a paid retry.
var anchorDigits = regexp.MustCompile(`\d{2,}`)

// AnchorsFor derives, per cue, the tokens its translation must carry:
//   - multi-digit numbers from the source text;
//   - the fixed rendering of every glossary term the source mentions;
//   - the rendering of every harvested term (the model's own ===TERMS===) the
//     source mentions.
//
// Proper nouns are NOT anchored by their English spelling: the translator prompt
// (since m1-v4) renders names in Chinese, so the English token is expected to vanish.
// Glossary and harvest give the rendering to look for instead.
func AnchorsFor(source []SubtitleBlock, glossary []prompts.GlossaryEntry, harvested map[string]string) map[int][]string {
	out := make(map[int][]string, len(source))
	for _, b := range source {
		var list []string
		seen := map[string]struct{}{}
		add := func(a string) {
			a = strings.TrimSpace(a)
			if a == "" {
				return
			}
			if _, dup := seen[a]; dup {
				return
			}
			seen[a] = struct{}{}
			list = append(list, a)
		}
		for _, d := range anchorDigits.FindAllString(b.Text, -1) {
			add(d)
		}
		for _, g := range glossary {
			if mentions(b.Text, g.Source) {
				add(g.Target)
			}
		}
		for src, rendering := range harvested {
			if mentions(b.Text, src) {
				add(rendering)
			}
		}
		if len(list) > 0 {
			sort.Strings(list)
			out[b.Index] = list
		}
	}
	return out
}

// mentions is a whole-word, case-insensitive match of term inside text.
func mentions(text, term string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return false
	}
	lt, lterm := strings.ToLower(text), strings.ToLower(term)
	for start := 0; ; {
		i := strings.Index(lt[start:], lterm)
		if i < 0 {
			return false
		}
		i += start
		end := i + len(lterm)
		before := i == 0 || !isWordByte(lt[i-1])
		after := end == len(lt) || !isWordByte(lt[end])
		if before && after {
			return true
		}
		start = i + 1
	}
}

func isWordByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
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

// simplifiedWords are the everyday Simplified words built from characters the
// detector no longer counts as Simplified-only because each is ALSO a
// legitimate Traditional character on its own (里 公里／帕里斯, 几 茶几, 余 余光中,
// 丰 丰采, 准 不准, 么 老么). As WORDS these are unambiguously Simplified — their
// Traditional spellings are 哪裡／家裡／心裡／那裡／幾乎／幾個／多餘／其餘／豐富／
// 什麼／怎麼／這麼／那麼／準備／標準 — so a cue containing one is still a leak.
var simplifiedWords = []string{
	"哪里", "家里", "心里", "那里", "这里", "里面", "里边",
	"几乎", "几个", "几天", "几次", "几年", "几点",
	"多余", "其余", "剩余",
	"丰富", "丰收",
	"什么", "怎么", "这么", "那么", "为什么", "多么",
	"准备", "标准", "准确", "准时",
	"佣人", "雇佣",
}

// containsSimplifiedWord reports whether text carries one of simplifiedWords.
func containsSimplifiedWord(text string) bool {
	for _, w := range simplifiedWords {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}
