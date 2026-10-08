package models

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ChineseSubtitle is the ONE answer to "does this title have Chinese
// subtitles?" — story disc-2026-10-subtitle-filter-disagrees-with-badges.
//
// AC #1 [@contract-v1]: serialized as json:"chinese_subtitle" on Movie and
// Series, always present, one of the five values below. It is computed on read
// (scanMovie / scanSeries) and, through the vido_chinese_subtitle SQL function
// the repository registers from ChineseSubtitleVerdict, it is ALSO what the
// library's chinese_subtitle filter matches on — so the badge and the filter
// cannot disagree. Never compute it anywhere else (AC #3).
type ChineseSubtitle string

const (
	// ChineseSubtitleZhHant — there is Traditional Chinese.
	ChineseSubtitleZhHant ChineseSubtitle = "zh_hant"
	// ChineseSubtitleZhHans — there is Simplified Chinese and no Traditional.
	ChineseSubtitleZhHans ChineseSubtitle = "zh_hans"
	// ChineseSubtitleZh — there is Chinese, script untold (an embedded `chi`).
	ChineseSubtitleZh ChineseSubtitle = "zh"
	// ChineseSubtitleNone — known to have no Chinese (English only, nothing at
	// all, Cantonese only, or searched online and not found).
	ChineseSubtitleNone ChineseSubtitle = "none"
	// ChineseSubtitleUnknown — not enough evidence either way.
	ChineseSubtitleUnknown ChineseSubtitle = "unknown"
)

// AllChineseSubtitleVerdicts lists every value, in display order.
func AllChineseSubtitleVerdicts() []ChineseSubtitle {
	return []ChineseSubtitle{
		ChineseSubtitleZhHant, ChineseSubtitleZhHans, ChineseSubtitleZh,
		ChineseSubtitleNone, ChineseSubtitleUnknown,
	}
}

// IsValid reports whether c is one of the five verdicts.
func (c ChineseSubtitle) IsValid() bool {
	for _, v := range AllChineseSubtitleVerdicts() {
		if c == v {
			return true
		}
	}
	return false
}

// Library filter groups — AC #2 [@contract-v1]: the values of the
// `chinese_subtitle` query param. The three groups partition the verdicts.
const (
	ChineseSubtitleFilterHas     = "has"
	ChineseSubtitleFilterMissing = "missing"
	ChineseSubtitleFilterUnknown = "unknown"
)

// AllChineseSubtitleFilters lists the accepted `chinese_subtitle` values.
func AllChineseSubtitleFilters() []string {
	return []string{ChineseSubtitleFilterHas, ChineseSubtitleFilterMissing, ChineseSubtitleFilterUnknown}
}

// ChineseSubtitleFilterVerdicts maps a filter group to the verdicts it covers;
// nil for an unknown group (values are case-sensitive, like subtitle_status).
func ChineseSubtitleFilterVerdicts(group string) []ChineseSubtitle {
	switch group {
	case ChineseSubtitleFilterHas:
		return []ChineseSubtitle{ChineseSubtitleZhHant, ChineseSubtitleZhHans, ChineseSubtitleZh}
	case ChineseSubtitleFilterMissing:
		return []ChineseSubtitle{ChineseSubtitleNone}
	case ChineseSubtitleFilterUnknown:
		return []ChineseSubtitle{ChineseSubtitleUnknown}
	}
	return nil
}

// TitleScript is what a subtitle track's TITLE tag says about its Chinese
// variant. ffprobe usually reports only `chi`; the title ("繁體中文",
// "Chinese (Simplified)", "粵語") is where the script lives.
type TitleScript int

const (
	TitleScriptNone TitleScript = iota
	TitleScriptCantonese
	TitleScriptTraditional
	TitleScriptSimplified
)

// ChineseTitleScript classifies a track title. Cantonese wins over the script
// words (a "粵語繁體" track is Cantonese), then Traditional, then Simplified.
// This is the single keyword list shared by subtitle.embeddedLanguage (the
// manage-subtitles dialog) and ChineseSubtitleVerdict.
func ChineseTitleScript(title string) TitleScript {
	t := strings.ToLower(title)
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(t, w) {
				return true
			}
		}
		return false
	}
	// Release-scene abbreviations are matched as whole words only: "chs"
	// inside a longer word, or "gb" in "gbp", must not decide a script
	// (disc-2026-10-episode-list-subtitle-badge-c AC #5).
	words := map[string]bool{}
	for _, w := range titleWordSplit.Split(t, -1) {
		if w != "" {
			words[w] = true
		}
	}
	hasWord := func(ws ...string) bool {
		for _, w := range ws {
			if words[w] {
				return true
			}
		}
		return false
	}
	switch {
	case has("粵", "粤", "cantonese", "yue"):
		return TitleScriptCantonese
	case has("繁", "traditional", "hant", "zh-tw", "taiwan", "hong kong", "zh-hk"), hasWord("cht", "big5"):
		return TitleScriptTraditional
	case has("简", "簡", "simplified", "hans", "zh-cn"), hasWord("chs", "gb", "gbk", "gb2312", "gb18030"):
		return TitleScriptSimplified
	}
	return TitleScriptNone
}

// titleWordSplit splits a lowercased title into ASCII words.
var titleWordSplit = regexp.MustCompile(`[^a-z0-9]+`)

// trackClass is a single language tag's (plus title's) verdict.
type trackClass int

const (
	classUndetermined trackClass = iota // und / untagged / not a language code
	classNotChinese                     // a known language that is not Chinese (eng, jpn, yue…)
	classZh                             // Chinese, script untold
	classZhHans
	classZhHant
)

var (
	hantTags = map[string]bool{"zh-hant": true, "zh-tw": true, "zh-hk": true, "zh-mo": true, "cht": true, "zh-hant-tw": true, "zh-hant-hk": true}
	hansTags = map[string]bool{"zh-hans": true, "zh-cn": true, "zh-sg": true, "chs": true, "zh-hans-cn": true}
	zhTags   = map[string]bool{"chi": true, "zho": true, "zh": true}
	// Sidecar filename flags that are not languages (Movie.eng.forced.srt).
	nonLanguageSegments = map[string]bool{"forced": true, "sdh": true, "cc": true, "default": true}
	// A language-code shape: 2–3 letters, optional subtags. A full word such as
	// "Chinese" or "forced" does NOT qualify, so it cannot claim "known non-Chinese".
	languageCodeShape = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{1,8})*$`)
	// One NFO token (applyNFOTechInfo joins tags with commas: "chi,eng").
	nfoTokenShape = regexp.MustCompile(`^[A-Za-z0-9_.\-]*$`)
)

// classifyTag classifies one language tag. Lowercase, `_` as `-`, split on
// `.`; ANY segment that is Chinese makes the tag Chinese (zh-TW.forced,
// chi.sdh). Cantonese (`yue`, `zh-yue`) is a known NON-Chinese language —
// Alexyu 2026-10-06 ruling, matching subtitle.isChineseTag.
func classifyTag(tag string) trackClass {
	norm := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(tag)), "_", "-")
	var langSegs []string
	best := classUndetermined
	for _, seg := range strings.Split(norm, ".") {
		seg = strings.TrimSpace(seg)
		if seg == "" || nonLanguageSegments[seg] {
			continue
		}
		langSegs = append(langSegs, seg)
		if c := classifyChineseSegment(seg); c > best {
			best = c
		}
	}
	if best >= classZh {
		return best
	}
	if len(langSegs) == 0 {
		return classUndetermined
	}
	first := langSegs[0]
	switch {
	case first == "und" || first == "mul" || first == "mis":
		return classUndetermined
	case first == "yue" || first == "zh-yue":
		return classNotChinese
	case languageCodeShape.MatchString(first):
		return classNotChinese
	}
	return classUndetermined
}

// classifyChineseSegment returns the Chinese class of one segment, or
// classUndetermined when it is not Chinese.
func classifyChineseSegment(seg string) trackClass {
	switch {
	case hantTags[seg]:
		return classZhHant
	case hansTags[seg]:
		return classZhHans
	case zhTags[seg]:
		return classZh
	case strings.HasPrefix(seg, "zh-"):
		sub := strings.Split(seg, "-")[1:]
		for _, s := range sub {
			switch s {
			case "yue":
				return classUndetermined // zh-yue is Cantonese — handled as non-Chinese by the caller
			case "hant", "tw", "hk", "mo":
				return classZhHant
			case "hans", "cn", "sg":
				return classZhHans
			}
		}
		return classZh
	}
	return classUndetermined
}

// classifyTrack folds the title and the detected script into the tag's class:
// any Chinese tag whose title says Cantonese is Cantonese (same precedence as
// embeddedLanguage); otherwise the script read from the text wins; otherwise
// an untold-script Chinese tag takes its script from the title.
func classifyTrack(tag, title, detected string) trackClass {
	c := classifyTag(tag)
	if c < classZh {
		return c
	}
	if ChineseTitleScript(title) == TitleScriptCantonese {
		return classNotChinese
	}
	// What the text says outranks the title and the tag: Vido read a sample
	// of the track (disc-2026-10-episode-list-subtitle-badge-c AC #1).
	switch detected {
	case "zh-Hant":
		return classZhHant
	case "zh-Hans":
		return classZhHans
	}
	if title == "" {
		return c
	}
	switch ChineseTitleScript(title) {
	case TitleScriptTraditional:
		if c == classZh {
			return classZhHant
		}
	case TitleScriptSimplified:
		if c == classZh {
			return classZhHans
		}
	}
	return c
}

type verdictTrack struct {
	Language string `json:"language"`
	Title    string `json:"title"`
	// DetectedLanguage is the script Vido read from a sample of the track's
	// text. Only "zh-Hant" / "zh-Hans" decide anything; "zh" / "und" / "" =
	// the sample did not tell, or was never taken.
	DetectedLanguage string `json:"detected_language"`
}

// parseVerdictTracks reads subtitle_tracks: a JSON array (ffprobe + sidecars)
// or the NFO comma string ("chi,eng"). ok=false means "no usable track
// information" — NULL, blank, or garbage — which the verdict treats as NULL.
func parseVerdictTracks(raw string) (tracks []verdictTrack, ok bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, false
	}
	if strings.HasPrefix(s, "[") {
		if err := json.Unmarshal([]byte(s), &tracks); err != nil {
			return nil, false
		}
		if tracks == nil {
			tracks = []verdictTrack{}
		}
		return tracks, true
	}
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if !nfoTokenShape.MatchString(tok) {
			return nil, false // neither JSON nor a comma list of tags
		}
		tracks = append(tracks, verdictTrack{Language: tok})
	}
	return tracks, true
}

// ChineseSubtitleVerdict is the ONLY implementation of the "has Chinese
// subtitles" rule (AC #3). Inputs are the raw subtitle_status,
// subtitle_language and subtitle_tracks columns; "" stands for NULL. It never
// fails: anything it cannot read is "unknown" (it runs inside library queries).
//
// Rule (Alexyu 2026-10-06):
//  1. Chinese evidence → has: subtitle_status=found with a Chinese
//     subtitle_language, or any Chinese track (embedded or sidecar, text or
//     image). Traditional beats Simplified beats untold.
//  2. Otherwise "none" only when certain: the tracks are readable and every one
//     is a known non-Chinese language (incl. `[]`); or there is no track info
//     and subtitle_status is not_found / untranslated.
//  3. Everything else is "unknown" — notably an `und` track, which in a
//     Taiwanese library is quite possibly Chinese.
func ChineseSubtitleVerdict(status, language, tracks string) ChineseSubtitle {
	best := classUndetermined
	if SubtitleStatus(status) == SubtitleStatusFound {
		if c := classifyTag(language); c >= classZh {
			best = c
		}
	}

	parsed, haveTracks := parseVerdictTracks(tracks)
	allKnownNonChinese := true
	for _, t := range parsed {
		c := classifyTrack(t.Language, t.Title, t.DetectedLanguage)
		if c > best && c >= classZh {
			best = c
		}
		if c != classNotChinese {
			allKnownNonChinese = false
		}
	}

	switch best {
	case classZhHant:
		return ChineseSubtitleZhHant
	case classZhHans:
		return ChineseSubtitleZhHans
	case classZh:
		return ChineseSubtitleZh
	}

	if haveTracks {
		if allKnownNonChinese {
			return ChineseSubtitleNone
		}
		return ChineseSubtitleUnknown
	}
	switch SubtitleStatus(status) {
	case SubtitleStatusNotFound, SubtitleStatusUntranslated:
		return ChineseSubtitleNone
	}
	return ChineseSubtitleUnknown
}

// ChineseSubtitleOfTrack is the verdict for ONE track (or one language tag
// with an empty title): zh_hant / zh_hans / zh when it is Chinese, "" when it
// is not or cannot be told. Same classification ChineseSubtitleVerdict folds
// over every track — the season list uses it to say WHICH sources are Chinese
// (disc-2026-10-episode-list-subtitle-badge-a AC #1).
func ChineseSubtitleOfTrack(language, title, detected string) ChineseSubtitle {
	switch classifyTrack(language, title, detected) {
	case classZhHant:
		return ChineseSubtitleZhHant
	case classZhHans:
		return ChineseSubtitleZhHans
	case classZh:
		return ChineseSubtitleZh
	}
	return ""
}

// chineseSubtitleNeed ranks how much a verdict needs the user's attention:
// missing Chinese first, then "we do not know", then Simplified only, then
// Chinese of untold script, then Traditional. An unrecognised value ranks as
// unknown.
func chineseSubtitleNeed(c ChineseSubtitle) int {
	switch c {
	case ChineseSubtitleNone:
		return 5
	case ChineseSubtitleZhHans:
		return 3
	case ChineseSubtitleZh:
		return 2
	case ChineseSubtitleZhHant:
		return 1
	}
	return 4 // unknown, or anything unreadable
}

// WorstChineseSubtitle folds per-episode verdicts into the series verdict
// (disc-2026-10-subtitle-filter-series-phase-2, SM ruling 1): the one that
// most needs handling wins — any episode missing Chinese makes the series
// "missing"; one unknown episode keeps the series from claiming "has".
// ok=false when there were no verdicts at all (no episode with a file).
func WorstChineseSubtitle(verdicts []ChineseSubtitle) (ChineseSubtitle, bool) {
	if len(verdicts) == 0 {
		return "", false
	}
	worst := verdicts[0]
	if !worst.IsValid() {
		worst = ChineseSubtitleUnknown
	}
	for _, v := range verdicts[1:] {
		if !v.IsValid() {
			v = ChineseSubtitleUnknown
		}
		if chineseSubtitleNeed(v) > chineseSubtitleNeed(worst) {
			worst = v
		}
	}
	return worst, true
}
