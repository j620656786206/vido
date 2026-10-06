package mine

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vido/api/internal/subtitle"
)

// Cue is one timed line, in milliseconds.
type Cue struct {
	StartMS int
	EndMS   int
	Text    string
}

// Segment is one aligned English/Chinese pair. Where cues split differently
// on the two sides (one English cue over two Chinese ones, or the reverse) the
// overlapping group is merged into one segment, so a name that straddles a cue
// boundary still lands in the same pair.
type Segment struct {
	En string
	Zh string
	// EnCues / ZhCues are how many source cues the segment merged.
	EnCues int
	ZhCues int
}

// minIoU is the time-overlap ratio two cues need to be considered the same
// line (AC #2). Official subtitles are usually cut within a few hundred ms of
// each other; 0.5 tolerates a ±300 ms drift on a 1 s line and still refuses
// to pair a line with its neighbour.
const minIoU = 0.5

// LoadCues parses an .srt / .ass / .ssa file into cues.
func LoadCues(path string) ([]Cue, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller-supplied subtitle path
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ass", ".ssa":
		return ParseASS(string(raw)), nil
	default:
		return ParseSRTCues(string(raw))
	}
}

// ParseSRTCues wraps subtitle.ParseSRT into millisecond cues.
func ParseSRTCues(content string) ([]Cue, error) {
	blocks, err := subtitle.ParseSRT(content)
	if err != nil {
		return nil, err
	}
	out := make([]Cue, 0, len(blocks))
	for _, b := range blocks {
		start, ok1 := subtitle.SRTTimestampMS(b.Start)
		end, ok2 := subtitle.SRTTimestampMS(b.End)
		if !ok1 || !ok2 || end < start {
			continue
		}
		out = append(out, Cue{StartMS: start, EndMS: end, Text: cleanText(b.Text)})
	}
	return out, nil
}

var (
	assDialogue = regexp.MustCompile(`(?m)^Dialogue:\s*[^,]*,([^,]+),([^,]+),(?:[^,]*,){6}(.*)$`)
	assOverride = regexp.MustCompile(`\{[^}]*\}`)
	htmlTag     = regexp.MustCompile(`<[^>]+>`)
)

// ParseASS pulls the Dialogue lines out of an .ass/.ssa file: start, end and
// the text with override tags removed. Styles, fonts and positions are
// ignored — only the words matter here.
func ParseASS(content string) []Cue {
	var out []Cue
	for _, m := range assDialogue.FindAllStringSubmatch(content, -1) {
		start, ok1 := assToMS(m[1])
		end, ok2 := assToMS(m[2])
		if !ok1 || !ok2 || end < start {
			continue
		}
		text := assOverride.ReplaceAllString(m[3], "")
		text = strings.ReplaceAll(text, `\N`, " ")
		text = strings.ReplaceAll(text, `\n`, " ")
		text = strings.ReplaceAll(text, `\h`, " ")
		out = append(out, Cue{StartMS: start, EndMS: end, Text: cleanText(text)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartMS < out[j].StartMS })
	return out
}

// assToMS parses "0:01:02.34" (centiseconds).
func assToMS(ts string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(ts), ":")
	if len(parts) != 3 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	secParts := strings.SplitN(parts[2], ".", 2)
	s, err3 := strconv.Atoi(secParts[0])
	cs := 0
	if len(secParts) == 2 {
		frac := secParts[1]
		if len(frac) > 3 {
			frac = frac[:3]
		}
		for len(frac) < 3 {
			frac += "0"
		}
		v, err := strconv.Atoi(frac)
		if err != nil {
			return 0, false
		}
		cs = v
	}
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, false
	}
	return ((h*60+m)*60+s)*1000 + cs, true
}

func cleanText(s string) string {
	s = htmlTag.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\u200e", "")
	s = strings.ReplaceAll(s, "\u200f", "")
	return strings.Join(strings.Fields(s), " ")
}

// Align pairs English and Chinese cues by time overlap into segments.
//
// Both lists are walked in time order. Two cues belong together when their
// intersection over union is at least minIoU; a cue that overlaps a cue the
// other side already grouped joins that group, which is how 1:N and N:1 splits
// merge. Cues with no counterpart are dropped — a name only one side has
// cannot be learned.
func Align(en, zh []Cue) []Segment {
	en = sortedCopy(en)
	zh = sortedCopy(zh)
	// A sidecar made for another release of the same episode is often a
	// constant few seconds off (Scorpion S01E02's fansub: −4 s). Estimate that
	// shift from cue-start coincidences and remove it before pairing; a drift
	// that is not constant (different cut, frame-rate) is beyond this miner
	// and simply pairs fewer lines.
	if shift := EstimateShift(en, zh); shift != 0 {
		for i := range zh {
			zh[i].StartMS += shift
			zh[i].EndMS += shift
		}
	}
	// group id per cue, -1 = unassigned
	enGroup := make([]int, len(en))
	zhGroup := make([]int, len(zh))
	for i := range enGroup {
		enGroup[i] = -1
	}
	for i := range zhGroup {
		zhGroup[i] = -1
	}
	nextGroup := 0
	j := 0
	for i := range en {
		// skip Chinese cues that end before this English cue starts
		for j < len(zh) && zh[j].EndMS < en[i].StartMS {
			j++
		}
		for k := j; k < len(zh) && zh[k].StartMS <= en[i].EndMS; k++ {
			if iou(en[i], zh[k]) < minIoU && !contained(en[i], zh[k]) {
				continue
			}
			switch {
			case enGroup[i] < 0 && zhGroup[k] < 0:
				enGroup[i], zhGroup[k] = nextGroup, nextGroup
				nextGroup++
			case enGroup[i] < 0:
				enGroup[i] = zhGroup[k]
			case zhGroup[k] < 0:
				zhGroup[k] = enGroup[i]
			case enGroup[i] != zhGroup[k]:
				// merge the two groups
				from, to := zhGroup[k], enGroup[i]
				for x := range enGroup {
					if enGroup[x] == from {
						enGroup[x] = to
					}
				}
				for x := range zhGroup {
					if zhGroup[x] == from {
						zhGroup[x] = to
					}
				}
			}
		}
	}
	type acc struct {
		en, zh         []string
		enCues, zhCues int
		first          int
	}
	groups := map[int]*acc{}
	for i, g := range enGroup {
		if g < 0 {
			continue
		}
		a := groups[g]
		if a == nil {
			a = &acc{first: en[i].StartMS}
			groups[g] = a
		}
		a.en = append(a.en, en[i].Text)
		a.enCues++
	}
	for k, g := range zhGroup {
		if g < 0 {
			continue
		}
		a := groups[g]
		if a == nil {
			continue
		}
		a.zh = append(a.zh, zh[k].Text)
		a.zhCues++
	}
	out := make([]Segment, 0, len(groups))
	for _, a := range groups {
		if len(a.en) == 0 || len(a.zh) == 0 {
			continue
		}
		out = append(out, Segment{En: strings.Join(a.en, " "), Zh: strings.Join(a.zh, " "), EnCues: a.enCues, ZhCues: a.zhCues})
	}
	// deterministic order: by first English start
	firsts := make(map[string]int, len(out))
	for _, a := range groups {
		if len(a.en) > 0 && len(a.zh) > 0 {
			firsts[strings.Join(a.en, " ")+"\x00"+strings.Join(a.zh, " ")] = a.first
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return firsts[out[i].En+"\x00"+out[i].Zh] < firsts[out[j].En+"\x00"+out[j].Zh]
	})
	return out
}

// EstimateShift returns the constant offset (ms) to ADD to the Chinese cues
// so their start times line up best with the English ones, searched over
// ±shiftSearchMS in shiftStepMS steps; 0 when the aligned-as-is count is
// already the best or the data is too thin to tell.
func EstimateShift(en, zh []Cue) int {
	if len(en) < 20 || len(zh) < 20 {
		return 0
	}
	enStarts := make(map[int]struct{}, len(en))
	for _, c := range en {
		enStarts[c.StartMS/shiftBucketMS] = struct{}{}
	}
	score := func(shift int) int {
		hits := 0
		for _, c := range zh {
			if _, ok := enStarts[(c.StartMS+shift)/shiftBucketMS]; ok {
				hits++
			}
		}
		return hits
	}
	base := score(0)
	best, bestHits := 0, base
	for shift := -shiftSearchMS; shift <= shiftSearchMS; shift += shiftStepMS {
		if shift == 0 {
			continue
		}
		if h := score(shift); h > bestHits || (h == bestHits && abs(shift) < abs(best)) {
			best, bestHits = shift, h
		}
	}
	// Only move when it clearly helps: a shift must beat "as is" by a margin,
	// or random coincidences would nudge a perfectly aligned pair.
	if bestHits < base+base/5+5 {
		return 0
	}
	return best
}

const (
	shiftSearchMS = 30000
	shiftStepMS   = 250
	shiftBucketMS = 500
)

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sortedCopy(c []Cue) []Cue {
	out := append([]Cue(nil), c...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartMS < out[j].StartMS })
	return out
}

func iou(a, b Cue) float64 {
	inter := min(a.EndMS, b.EndMS) - max(a.StartMS, b.StartMS)
	if inter <= 0 {
		return 0
	}
	union := max(a.EndMS, b.EndMS) - min(a.StartMS, b.StartMS)
	if union <= 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// contained reports whether the shorter cue sits almost entirely inside the
// longer one (≥ 80% of its own length) — the 1:N split shape, where IoU alone
// is low because the long cue dwarfs the short one.
func contained(a, b Cue) bool {
	inter := min(a.EndMS, b.EndMS) - max(a.StartMS, b.StartMS)
	if inter <= 0 {
		return false
	}
	shorter := min(a.EndMS-a.StartMS, b.EndMS-b.StartMS)
	if shorter <= 0 {
		return false
	}
	return float64(inter)/float64(shorter) >= 0.8
}
