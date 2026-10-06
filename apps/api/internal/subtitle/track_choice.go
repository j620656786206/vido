package subtitle

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/vido/api/internal/services"
)

// trackKind is what a subtitle track IS, as far as the file can tell us. The
// order matters: chooseMain prefers the lower value when two tracks qualify.
type trackKind int

const (
	trackRegular trackKind = iota // the full dialogue track
	trackSDH                      // subtitles for the deaf and hard-of-hearing
	trackForced                   // forced narrative: on-screen text, foreign-language lines
)

func (k trackKind) String() string {
	switch k {
	case trackSDH:
		return "sdh"
	case trackForced:
		return "forced"
	default:
		return "regular"
	}
}

// Whole-word title markers. `\b` keeps "Accent" from reading as CC and is an
// ASCII boundary, so it still fires next to CJK ("繁中SDH"). Titles are matched
// with `_` turned into a space first, because `_` is a word character and
// "English_SDH" would otherwise have no boundary.
var (
	forcedTitle    = regexp.MustCompile(`(?i)\bforced\b`)
	notForcedTitle = regexp.MustCompile(`(?i)\b(non|not)[ -]?forced\b`)
	sdhTitle       = regexp.MustCompile(`(?i)\b(sdh|cc|hi|hearing[ -]impaired)\b`)
)

// classifyTrack reads the muxer's disposition flags first and the track title
// second — not every release sets the flags, but most name the track. A track
// that claims to be both forced and SDH is forced: it is the short one.
func classifyTrack(t services.SubtitleTrack) trackKind {
	title := strings.ReplaceAll(t.Title, "_", " ")
	titleForced := forcedTitle.MatchString(title) && !notForcedTitle.MatchString(title)
	if t.Forced || titleForced || strings.Contains(title, "強制") || strings.Contains(title, "强制") {
		return trackForced
	}
	if t.HearingImpaired || sdhTitle.MatchString(title) || strings.Contains(title, "聽障") || strings.Contains(title, "听障") {
		return trackSDH
	}
	return trackRegular
}

// coversHalf reports whether a track's last cue ends at or past half of the
// file. An unknown duration (0) or unreadable timestamps never disqualify a
// track — the rule only removes tracks it can prove are partial.
func coversHalf(blocks []SubtitleBlock, durationSeconds float64) bool {
	if durationSeconds <= 0 {
		return true
	}
	lastEnd, parsedAny := 0, false
	for _, b := range blocks {
		if end, ok := SRTTimestampMS(b.End); ok {
			lastEnd = max(lastEnd, end)
			parsedAny = true
		}
	}
	if !parsedAny {
		return true
	}
	return float64(lastEnd)*2 >= durationSeconds*1000
}

// mergeForcedCues folds every OTHER forced English candidate into the main
// track's cues. A forced cue that overlaps an already-kept cue (main, or one
// merged from another forced track) in time and says the same words (ignoring
// case, spaces and punctuation) is a duplicate and is dropped; an overlapping
// cue with different words is kept — players stack overlapping cues.
//
// A "forced" track with at least half the main track's cues is not a forced
// narrative at all: many releases set forced=1 on the FULL track so players
// always show it. Merging it would print every line twice and double the
// translation bill, so it is skipped and reported in skippedTracks.
//
// Main cues keep their original Index (P7). Merged cues are numbered upward
// from the main track's HIGHEST Index, not its length: SDH filtering leaves
// gaps, and cue identity downstream is the Index (pipeline.go's `final` map).
// The result is ordered by start time, a main cue first on a tie.
func mergeForcedCues(main []SubtitleBlock, mainStream int, parsed []parsedCandidate) (out []SubtitleBlock, merged, dropped int, skippedTracks []int) {
	var extra []SubtitleBlock
	for _, p := range parsed {
		if p.kind != trackForced || p.track.StreamIndex == mainStream || p.variant != LangUndetermined {
			continue
		}
		if len(p.track.Blocks)*2 >= len(main) {
			skippedTracks = append(skippedTracks, p.track.StreamIndex)
			continue
		}
		for _, f := range p.track.Blocks {
			if duplicatesKeptCue(f, main) || duplicatesKeptCue(f, extra) {
				dropped++
				continue
			}
			extra = append(extra, f)
		}
	}
	if len(extra) == 0 {
		return main, 0, dropped, skippedTracks
	}

	next := 0
	for _, b := range main {
		next = max(next, b.Index)
	}
	out = make([]SubtitleBlock, 0, len(main)+len(extra))
	out = append(out, main...)
	for _, f := range extra {
		next++
		f.Index = next
		out = append(out, f)
	}

	mainCount := len(main)
	order := make([]int, len(out))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		sa, _ := SRTTimestampMS(out[order[a]].Start)
		sb, _ := SRTTimestampMS(out[order[b]].Start)
		if sa != sb {
			return sa < sb
		}
		return order[a] < mainCount && order[b] >= mainCount
	})
	sorted := make([]SubtitleBlock, len(out))
	for i, idx := range order {
		sorted[i] = out[idx]
	}
	return sorted, len(extra), dropped, skippedTracks
}

// duplicatesKeptCue reports whether a forced cue repeats a kept cue it
// overlaps in time.
func duplicatesKeptCue(f SubtitleBlock, kept []SubtitleBlock) bool {
	fs, ok1 := SRTTimestampMS(f.Start)
	fe, ok2 := SRTTimestampMS(f.End)
	if !ok1 || !ok2 {
		return false
	}
	words := normalizedWords(f.Text)
	for _, m := range kept {
		ms, ok1 := SRTTimestampMS(m.Start)
		me, ok2 := SRTTimestampMS(m.End)
		if !ok1 || !ok2 || fs >= me || ms >= fe {
			continue
		}
		if normalizedWords(m.Text) == words {
			return true
		}
	}
	return false
}

// normalizedWords keeps only letters and digits, lower-cased, so "We are
// leaving." and "we are LEAVING!" compare equal.
func normalizedWords(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
