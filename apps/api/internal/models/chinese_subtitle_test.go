package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Story disc-2026-10-subtitle-filter-disagrees-with-badges AC #4: the ONE rule
// that both the library badge (via scanMovie/scanSeries) and the library filter
// (via the vido_chinese_subtitle SQL function) use. Every row below is a case
// the AC names; extra rows pin the edges the rule text calls out.
func TestChineseSubtitleVerdict(t *testing.T) {
	cases := []struct {
		name     string
		status   string
		language string
		tracks   string // "" = NULL
		want     ChineseSubtitle
	}{
		// --- AC #4 named cases ---
		{"embedded chi + eng → zh", "not_searched", "", `[{"language":"chi","format":"subrip","external":false,"stream_index":2},{"language":"eng","format":"subrip","external":false,"stream_index":3}]`, ChineseSubtitleZh},
		{"only eng → none", "not_searched", "", `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`, ChineseSubtitleNone},
		{"empty array → none", "not_searched", "", `[]`, ChineseSubtitleNone},
		{"NULL + not_searched → unknown", "not_searched", "", "", ChineseSubtitleUnknown},
		{"NULL + not_found → none", "not_found", "", "", ChineseSubtitleNone},
		{"NULL + untranslated + en → none", "untranslated", "en", "", ChineseSubtitleNone},
		{"found + zh-Hant → zh_hant", "found", "zh-Hant", "", ChineseSubtitleZhHant},
		{"found + zh → zh", "found", "zh", "", ChineseSubtitleZh},
		{"NFO comma string chi,eng → zh", "not_searched", "", "chi,eng", ChineseSubtitleZh},
		{"sidecar zh-TW → zh_hant", "not_searched", "", `[{"language":"zh-TW","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleZhHant},
		{"sidecar chi.forced → zh", "not_searched", "", `[{"language":"chi.forced","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleZh},
		{"only und sidecar → unknown", "not_searched", "", `[{"language":"und","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleUnknown},
		{"eng + und → unknown", "not_searched", "", `[{"language":"eng","format":"subrip","external":false,"stream_index":2},{"language":"und","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleUnknown},
		{"chi + title 繁體中文 → zh_hant", "not_searched", "", `[{"language":"chi","format":"subrip","external":false,"title":"繁體中文","stream_index":2}]`, ChineseSubtitleZhHant},
		{"chi + title 简体 → zh_hans", "not_searched", "", `[{"language":"chi","format":"subrip","external":false,"title":"简体","stream_index":2}]`, ChineseSubtitleZhHans},
		{"only chi titled 粵語 → none (Cantonese is not Chinese)", "not_searched", "", `[{"language":"chi","format":"subrip","external":false,"title":"粵語","stream_index":2}]`, ChineseSubtitleNone},
		{"only yue → none", "not_searched", "", `[{"language":"yue","format":"subrip","external":false,"stream_index":2}]`, ChineseSubtitleNone},
		{"yue + untitled chi → zh", "not_searched", "", `[{"language":"yue","format":"subrip","external":false,"stream_index":2},{"language":"chi","format":"subrip","external":false,"stream_index":3}]`, ChineseSubtitleZh},
		{"image-format chi + no_text_source → zh", "no_text_source", "", `[{"language":"chi","format":"hdmv_pgs_subtitle","external":false,"stream_index":4}]`, ChineseSubtitleZh},
		{"found + en + NULL → unknown", "found", "en", "", ChineseSubtitleUnknown},
		{"garbage (neither JSON nor a comma list) → unknown", "not_searched", "", `{oops this is not json`, ChineseSubtitleUnknown},

		// --- edges the rule text calls out ---
		{"Traditional evidence beats Simplified", "not_searched", "", `[{"language":"zh-CN","format":"srt","external":true,"stream_index":0},{"language":"cht","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleZhHant},
		{"Simplified beats untold", "not_searched", "", `[{"language":"chi","format":"subrip","external":false,"stream_index":2},{"language":"chs","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleZhHans},
		{"found zh-Hant + embedded chi → zh_hant", "found", "zh-Hant", `[{"language":"chi","format":"subrip","external":false,"stream_index":2}]`, ChineseSubtitleZhHant},
		{"found zh-Hans → zh_hans", "found", "zh-Hans", "", ChineseSubtitleZhHans},
		{"Chinese language without found status is not evidence", "searching", "zh-Hant", "", ChineseSubtitleUnknown},
		{"untranslated + NULL + no language → none", "untranslated", "", "", ChineseSubtitleNone},
		{"not_found + und track → unknown (an untagged file may be Chinese)", "not_found", "", `[{"language":"und","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleUnknown},
		{"not_found + eng track → none", "not_found", "", `[{"language":"eng","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleNone},
		{"underscore tag zh_TW → zh_hant", "not_searched", "", `[{"language":"zh_TW","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleZhHant},
		{"region subtag zh-Hant-HK → zh_hant", "not_searched", "", `[{"language":"zh-Hant-HK","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleZhHant},
		{"zh-SG → zh_hans", "not_searched", "", `[{"language":"zh-SG","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleZhHans},
		{"eng.forced sidecar is still English → none", "not_searched", "", `[{"language":"eng.forced","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleNone},
		{"bare flag sidecar (Movie.forced.srt) → unknown", "not_searched", "", `[{"language":"forced","format":"srt","external":true,"stream_index":0}]`, ChineseSubtitleUnknown},
		{"a full word like 'Chinese' is not a code → unknown, never none", "not_searched", "", "Chinese", ChineseSubtitleUnknown},
		{"zh-HK titled Cantonese → none", "not_searched", "", `[{"language":"zh-HK","format":"subrip","external":false,"title":"Cantonese","stream_index":2}]`, ChineseSubtitleNone},
		{"Cantonese title + English → none", "not_searched", "", `[{"language":"chi","format":"subrip","external":false,"title":"粤语","stream_index":2},{"language":"eng","format":"subrip","external":false,"stream_index":3}]`, ChineseSubtitleNone},
		{"found + yue → not evidence", "found", "yue", "", ChineseSubtitleUnknown},
		{"NFO string with an empty token → unknown", "not_searched", "", "eng,", ChineseSubtitleUnknown},
		{"NFO string eng only → none", "not_searched", "", "eng", ChineseSubtitleNone},
		{"JSON that is not an array → unknown", "not_found", "", `{"language":"chi"}`, ChineseSubtitleNone},
		{"truncated JSON array + not_searched → unknown", "not_searched", "", `[{"language":"chi"`, ChineseSubtitleUnknown},
		{"whitespace-only tracks behave like NULL", "not_found", "", "   ", ChineseSubtitleNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ChineseSubtitleVerdict(tc.status, tc.language, tc.tracks))
		})
	}
}

// Broken input must degrade to a verdict, never panic — the function runs
// inside every library SQL query (AC #4 last clause).
func TestChineseSubtitleVerdict_NeverPanics(t *testing.T) {
	inputs := []string{"", "[", "]", "[null]", "[1,2]", `[{"language":null}]`, "\x00\xff", ",,,", "[{}]", "null"}
	for _, in := range inputs {
		assert.NotPanics(t, func() {
			v := ChineseSubtitleVerdict("not_searched", "", in)
			assert.True(t, v.IsValid(), "%q gave %q", in, v)
		}, "%q", in)
	}
}

func TestChineseSubtitleFilterGroups(t *testing.T) {
	assert.Equal(t, []ChineseSubtitle{ChineseSubtitleZhHant, ChineseSubtitleZhHans, ChineseSubtitleZh},
		ChineseSubtitleFilterVerdicts(ChineseSubtitleFilterHas))
	assert.Equal(t, []ChineseSubtitle{ChineseSubtitleNone}, ChineseSubtitleFilterVerdicts(ChineseSubtitleFilterMissing))
	assert.Equal(t, []ChineseSubtitle{ChineseSubtitleUnknown}, ChineseSubtitleFilterVerdicts(ChineseSubtitleFilterUnknown))
	assert.Nil(t, ChineseSubtitleFilterVerdicts("HAS"))

	// The three groups partition the verdict set: every verdict in exactly one.
	seen := map[ChineseSubtitle]int{}
	for _, g := range AllChineseSubtitleFilters() {
		for _, v := range ChineseSubtitleFilterVerdicts(g) {
			seen[v]++
		}
	}
	for _, v := range AllChineseSubtitleVerdicts() {
		assert.Equal(t, 1, seen[v], "verdict %q must sit in exactly one filter group", v)
	}
	assert.Len(t, seen, len(AllChineseSubtitleVerdicts()))
}

// The keyword rule the manage-subtitles dialog (subtitle.embeddedLanguage) and
// the verdict share. Order matters: Cantonese wins, then Traditional, then
// Simplified.
func TestChineseTitleScript(t *testing.T) {
	cases := []struct {
		title string
		want  TitleScript
	}{
		{"繁體中文", TitleScriptTraditional},
		{"Chinese (Traditional)", TitleScriptTraditional},
		{"zh-TW", TitleScriptTraditional},
		{"Chinese (Hong Kong)", TitleScriptTraditional},
		{"简体", TitleScriptSimplified},
		{"簡體中文", TitleScriptSimplified},
		{"Chinese (Simplified)", TitleScriptSimplified},
		{"粵語", TitleScriptCantonese},
		{"粤语", TitleScriptCantonese},
		{"Cantonese", TitleScriptCantonese},
		{"粵語繁體", TitleScriptCantonese},
		{"", TitleScriptNone},
		{"English [SDH]", TitleScriptNone},
		// Release-scene abbreviations, whole words only (badge-c AC #5).
		{"CHT", TitleScriptTraditional},
		{"Chinese Big5", TitleScriptTraditional},
		{"CHS", TitleScriptSimplified},
		{"chi.GB", TitleScriptSimplified},
		{"GB2312", TitleScriptSimplified},
		{"Chinese (GBP prices)", TitleScriptNone},
		{"Chinese", TitleScriptNone},
		{"Richtsubs", TitleScriptNone},
		{"CHT/CHS", TitleScriptTraditional}, // a dual title: Traditional is checked first
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ChineseTitleScript(tc.title), "%q", tc.title)
	}
}

// AC #1 [@contract-v1]: the field is always on the wire (no omitempty) on both
// Movie and Series.
func TestChineseSubtitle_AlwaysSerialized(t *testing.T) {
	mb, err := json.Marshal(Movie{ChineseSubtitle: ChineseSubtitleNone})
	assert.NoError(t, err)
	assert.Contains(t, string(mb), `"chinese_subtitle":"none"`)

	sb, err := json.Marshal(Series{ChineseSubtitle: ChineseSubtitleUnknown})
	assert.NoError(t, err)
	assert.Contains(t, string(sb), `"chinese_subtitle":"unknown"`)
}

// disc-2026-10-episode-list-subtitle-badge-c AC #1: what Vido read from the
// track's text outranks its title and its tag; a sample that did not tell
// ("zh" / "und") changes nothing; Cantonese stays not-Chinese.
func TestChineseSubtitleVerdict_DetectedLanguage(t *testing.T) {
	cases := []struct {
		name, tracks string
		want         ChineseSubtitle
	}{
		{"untold chi, text Traditional", `[{"language":"chi","detected_language":"zh-Hant"}]`, ChineseSubtitleZhHant},
		{"untold chi, text Simplified", `[{"language":"chi","detected_language":"zh-Hans"}]`, ChineseSubtitleZhHans},
		{"sample mixed", `[{"language":"chi","detected_language":"zh"}]`, ChineseSubtitleZh},
		{"sample had no Chinese", `[{"language":"chi","detected_language":"und"}]`, ChineseSubtitleZh},
		{"text beats a wrong title", `[{"language":"chi","title":"繁體中文","detected_language":"zh-Hans"}]`, ChineseSubtitleZhHans},
		{"text beats a wrong tag", `[{"language":"zh-TW","detected_language":"zh-Hans"}]`, ChineseSubtitleZhHans},
		{"Cantonese title still wins", `[{"language":"chi","title":"粵語","detected_language":"zh-Hant"}]`, ChineseSubtitleNone},
		{"English track ignores it", `[{"language":"eng","detected_language":"zh-Hant"}]`, ChineseSubtitleNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ChineseSubtitleVerdict("not_searched", "", tc.tracks))
		})
	}
}
