package prompts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── AC #1 / #6: the embedded file's schema ───────────────────────────────

func TestZhTWLexicon_EmbeddedFileMeetsTheStoryFloor(t *testing.T) {
	lex := ZhTWLexicon()
	require.NotNil(t, lex)
	assert.Equal(t, "zh-tw-lex-1", lex.Version)
	assert.GreaterOrEqual(t, len(lex.Replacements), 60, "AC #1: ≥ 60 簡→台 replacements")
	assert.GreaterOrEqual(t, len(lex.Terms), 40, "AC #1: ≥ 40 brand / app / institution terms")
	assert.Equal(t, lex.Version, LexiconVersion())

	// Longest `from` first, so 打印機 → 印表機 wins over 打印 → 列印.
	for i := 1; i < len(lex.Replacements); i++ {
		assert.GreaterOrEqual(t,
			len([]rune(lex.Replacements[i-1].From)), len([]rune(lex.Replacements[i].From)),
			"replacements must be sorted longest-first")
	}
	// eval-1's zero-score example is in the terms table.
	found := false
	for _, e := range lex.Terms {
		if e.Source == "Life360" {
			found = true
		}
	}
	assert.True(t, found, "Life360 (eval-1 零分例) must be a global term")
}

func TestParseLexicon_SchemaErrors(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{"missing version", "replacements: []\nterms: []\n", "version is required"},
		{"duplicate from", "version: v\nreplacements:\n  - {from: 視頻, to: 影片}\n  - {from: 視頻, to: 影像}\n", "duplicate from"},
		{"empty to", "version: v\nreplacements:\n  - {from: 視頻, to: ''}\n", "from and to are required"},
		{"from equals to", "version: v\nreplacements:\n  - {from: 影片, to: 影片}\n", "from equals to"},
		{"latin from", "version: v\nreplacements:\n  - {from: video, to: 影片}\n", "Chinese text only"},
		{"except without from", "version: v\nreplacements:\n  - {from: 視頻, to: 影片, except: [電影]}\n", "must contain from"},
		{"duplicate term (case-insensitive)", "version: v\nterms:\n  - {source: Venmo, target: Venmo}\n  - {source: venmo, target: Venmo}\n", "duplicate source"},
		{"empty term target", "version: v\nterms:\n  - {source: Venmo, target: ''}\n", "source and target are required"},
		{"unknown key", "version: v\nreplacementz: []\n", "field replacementz not found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseLexicon([]byte(c.yaml))
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.want)
		})
	}
}

func TestParseLexicon_ValidDocument(t *testing.T) {
	lex, err := ParseLexicon([]byte("version: v9\nreplacements:\n  - {from: 打印, to: 列印}\n  - {from: 打印機, to: 印表機}\nterms:\n  - {source: ' Venmo ', target: ' Venmo '}\n"))
	require.NoError(t, err)
	assert.Equal(t, "v9", lex.Version)
	assert.Equal(t, "打印機", lex.Replacements[0].From, "sorted longest-first")
	assert.Equal(t, GlossaryEntry{Source: "Venmo", Target: "Venmo"}, lex.Terms[0], "trimmed")
}

// ─── AC #2 / #6: word-level replacement with the counter-example table ────

func TestLexicon_Apply_Table(t *testing.T) {
	lex := ZhTWLexicon()
	cases := []struct{ in, want string }{
		// the story's own trap
		{"我在看電視頻道", "我在看電視頻道"},
		{"我在看視頻", "我在看影片"},
		{"這個視頻在電視頻道上播", "這個影片在電視頻道上播"},
		// OpenCC's own s2twp output for 视频 is 視訊; only the video-call sense keeps it
		{"我們來視訊通話吧", "我們來視訊通話吧"},
		{"傳個視訊給我", "傳個影片給我"},
		// longest first
		{"打印機壞了，不能打印", "印表機壞了，不能列印"},
		{"智能手機的人工智能", "智慧型手機的人工智慧"},
		// exceptions
		{"這是質量守恆定律", "這是質量守恆定律"},
		{"這件衣服質量很好", "這件衣服品質很好"},
		{"信息素的作用", "信息素的作用"},
		{"我收到一條信息", "我收到一條資訊"},
		{"大數據時代的數據", "大數據時代的資料"},
		{"朝鮮半島的朝鮮", "朝鮮半島的北韓"},
		// several occurrences, mixed
		{"視頻、視頻、還是視頻", "影片、影片、還是影片"},
		// Latin untouched
		{"Life360 is an app", "Life360 is an app"},
		{"", ""},
		// already Taiwanese: no-op
		{"我用軟體看影片", "我用軟體看影片"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.want, lex.Apply(c.in))
		})
	}
}

func TestLexicon_Apply_IsIdempotentOverTheWholeTable(t *testing.T) {
	lex := ZhTWLexicon()
	var sb strings.Builder
	for _, r := range lex.Replacements {
		sb.WriteString("他說" + r.From + "。")
	}
	once := lex.Apply(sb.String())
	assert.Equal(t, once, lex.Apply(once), "applying the table to its own output must change nothing")
	for _, r := range lex.Replacements {
		assert.NotContains(t, once, "他說"+r.From+"。", "every from in a neutral context is replaced")
	}
}

func TestLexicon_Apply_NilIsANoop(t *testing.T) {
	var lex *Lexicon
	assert.Equal(t, "視頻", lex.Apply("視頻"))
}

func TestIsMainlandContent(t *testing.T) {
	assert.True(t, IsMainlandContent([]string{"US", "cn"}))
	assert.False(t, IsMainlandContent([]string{"TW", "HK"}))
	assert.False(t, IsMainlandContent(nil))
}

// ─── AC #3 / #6: the three style sections and the version composition ────

func TestBuildLocalizationSection_ThreeLevels(t *testing.T) {
	literal := BuildLocalizationSection(LocalizationLiteral)
	standard := BuildLocalizationSection(LocalizationStandard)
	ott := BuildLocalizationSection(LocalizationOTT)

	assert.True(t, strings.HasPrefix(literal, "## Localization style (literal):"))
	assert.True(t, strings.HasPrefix(standard, "## Localization style (standard):"))
	assert.True(t, strings.HasPrefix(ott, "## Localization style (ott):"))

	assert.Contains(t, literal, "超市")
	assert.NotContains(t, literal, "全聯")
	assert.Contains(t, standard, "never 全聯")
	assert.Contains(t, ott, "去全聯")
	assert.Contains(t, ott, "ONLY when the source is a generic noun AND the scene is everyday life", "the AC #3 guardrail")
	assert.Contains(t, ott, "never replaced with a different one", "named brands are never swapped")

	// An unknown level renders as the default, never as a fourth style.
	assert.Equal(t, standard, BuildLocalizationSection("bogus"))
	assert.Equal(t, standard, BuildLocalizationSection(""))
}

func TestParseLocalizationLevel(t *testing.T) {
	for in, want := range map[string]LocalizationLevel{"": LocalizationStandard, " OTT ": LocalizationOTT, "literal": LocalizationLiteral, "Standard": LocalizationStandard} {
		got, err := ParseLocalizationLevel(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	_, err := ParseLocalizationLevel("netflix")
	require.Error(t, err)
	assert.Equal(t, LocalizationStandard, LocalizationLevel("netflix").Normalized())
}

func TestPromptVersionFor_CarriesBaseLexiconAndLevel(t *testing.T) {
	assert.Equal(t, "m1-v3+zh-tw-lex-1+standard", PromptVersionFor(LocalizationStandard))
	assert.Equal(t, "m1-v3+zh-tw-lex-1+ott", PromptVersionFor(LocalizationOTT))
	assert.Equal(t, "m1-v3+zh-tw-lex-1+literal", PromptVersionFor(LocalizationLiteral))
	assert.Equal(t, PromptVersionFor(LocalizationStandard), PromptVersionFor(""), "zero value is the default level")
}

func TestComposeInvariantSystemPrompt_Order(t *testing.T) {
	sys := ComposeInvariantSystemPrompt(LocalizationOTT)
	assert.True(t, strings.HasPrefix(sys, SubtitleTranslatorSystemPrompt))
	i := strings.Index(sys, "## Localization style (ott)")
	j := strings.Index(sys, "## Taiwan renderings for brands")
	assert.Greater(t, i, 0)
	assert.Greater(t, j, i, "style before the global lexicon terms")
	assert.Contains(t, sys, "- Life360 → Life360")
	assert.Contains(t, sys, "- Costco → 好市多")
	assert.Contains(t, sys, "A per-show glossary below overrides any entry here.")
	assert.NotContains(t, sys, "## Media context", "nothing per-show in the invariant prefix")
}
