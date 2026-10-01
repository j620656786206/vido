package mine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/services"
)

// ─── AC #1: source classification (same rule as eval/scan-partial-zh.sh) ───

func TestIsOfficialZhSidecar(t *testing.T) {
	yes := []string{
		"Scorpion.S01E02.zh-TW.hi.srt", "Scorpion.S01E02.zh-tw.srt", "Show.cht.ass", "Show.TC.srt",
		"Show.zh-Hant.hi.srt", "Show.zh-Hant.sdh.srt", "Show.繁體.srt", "Show.zh-HK.srt",
	}
	no := []string{
		"Scorpion.S01E02.zh-Hant.srt",     // Vido's own delivery
		"Scorpion.S01E02.zh-Hant.srt.bak", // its backup
		"Scorpion.S01E02.zh-Hant.tmp.srt", // its in-flight write
		"Scorpion.S01E02.en.srt", "Scorpion.S01E02.srt", "Scorpion.S01E02.zh-CN.srt", "Scorpion.S01E02.chs.srt",
		"Scorpion.S01E02.zh-TW.sub", "Scorpion.S01E02.mkv",
	}
	for _, n := range yes {
		assert.True(t, IsOfficialZhSidecar(n), n)
	}
	for _, n := range no {
		assert.False(t, IsOfficialZhSidecar(n), n)
	}
	assert.True(t, IsEnglishSidecar("Show.en.srt"))
	assert.True(t, IsEnglishSidecar("Show.eng.ass"))
	assert.False(t, IsEnglishSidecar("Show.srt"), "an untagged sidecar is not assumed English")
	assert.False(t, IsEnglishSidecar("Show.en.srt.bak"))
}

func TestClassify_UsableNeedsBothSides(t *testing.T) {
	tracks := []services.SubtitleTrack{
		{Language: "eng", Format: "subrip", StreamIndex: 3},
		{Language: "chi", Format: "hdmv_pgs_subtitle", StreamIndex: 4}, // image track: not a source
		{Language: "zho", Format: "ass", StreamIndex: 5},
		{Language: "eng", Format: "srt", External: true}, // external entries come via sidecars, not tracks
	}
	s := Classify("/m/Show.S01E01.mkv", []string{"Show.S01E01.zh-TW.srt", "Show.S01E01.zh-Hant.srt"}, tracks)
	assert.Equal(t, []string{"Show.S01E01.zh-TW.srt"}, s.ZhSidecars)
	assert.Equal(t, []int{3}, s.EnStreams)
	assert.Equal(t, []int{5}, s.ZhStreams)
	assert.True(t, s.Usable())
	assert.True(t, s.HasZh())

	only := Classify("/m/x.mkv", nil, []services.SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 2}})
	assert.False(t, only.Usable(), "English alone is the translate route, nothing to learn from")
	assert.False(t, only.HasZh())
}

// ─── AC #2: alignment ───────────────────────────────────────────────────────

func cue(start, end int, text string) Cue { return Cue{StartMS: start, EndMS: end, Text: text} }

func TestAlign_OverlapMergeAndDrift(t *testing.T) {
	en := []Cue{
		cue(1000, 3000, "I'm Walter O'Brien."),
		cue(3500, 5000, "Toby, run the numbers."),
		cue(6000, 9000, "Paige went to the garage with Sylvester."),
		cue(20000, 21000, "Nothing matches this one."),
	}
	zh := []Cue{
		cue(1300, 3200, "我是華特·歐布萊恩"), // ±300 ms drift
		cue(3400, 5100, "托比，跑一下數字"),
		cue(6000, 7400, "佩姬去了車庫"), // one English cue split over two Chinese cues
		cue(7500, 9000, "跟希維斯特一起"),
		cue(30000, 31000, "另一邊獨有的"),
	}
	segs := Align(en, zh)
	require.Len(t, segs, 3, "unmatched cues on either side are dropped")
	assert.Equal(t, Segment{En: "I'm Walter O'Brien.", Zh: "我是華特·歐布萊恩", EnCues: 1, ZhCues: 1}, segs[0])
	assert.Equal(t, "托比，跑一下數字", segs[1].Zh)
	assert.Equal(t, Segment{En: "Paige went to the garage with Sylvester.", Zh: "佩姬去了車庫 跟希維斯特一起", EnCues: 1, ZhCues: 2}, segs[2])

	// N:1 the other way round.
	segs = Align([]Cue{cue(0, 1400, "Walter."), cue(1500, 3000, "Happy.")}, []Cue{cue(0, 3000, "華特 哈皮")})
	require.Len(t, segs, 1)
	assert.Equal(t, 2, segs[0].EnCues)
	assert.Equal(t, "Walter. Happy.", segs[0].En)

	// Neighbouring lines with a sliver of overlap are NOT paired.
	segs = Align([]Cue{cue(0, 2000, "A")}, []Cue{cue(1900, 4000, "乙")})
	assert.Empty(t, segs)
}

func TestParseASS_DialogueOnly(t *testing.T) {
	ass := "[Script Info]\nTitle: x\n[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n" +
		"Dialogue: 0,0:00:01.20,0:00:03.45,Default,,0,0,0,,{\\an8}我是華特\\N歐布萊恩\n" +
		"Comment: 0,0:00:05.00,0:00:06.00,Default,,0,0,0,,not a line\n" +
		"Dialogue: 0,0:00:04.00,0:00:05.00,Default,,0,0,0,,托比\n"
	cues := ParseASS(ass)
	require.Len(t, cues, 2)
	assert.Equal(t, Cue{StartMS: 1200, EndMS: 3450, Text: "我是華特 歐布萊恩"}, cues[0])
	assert.Equal(t, Cue{StartMS: 4000, EndMS: 5000, Text: "托比"}, cues[1])

	srt, err := ParseSRTCues("1\n00:00:01,000 --> 00:00:02,500\n<i>Hello</i> there\n\n2\n00:00:03,000 --> 00:00:04,000\n\u200eSecond\n")
	require.NoError(t, err)
	assert.Equal(t, []Cue{{1000, 2500, "Hello there"}, {3000, 4000, "Second"}}, srt)
}

// ─── AC #3: candidates and co-occurrence ────────────────────────────────────

func TestCandidates_CapitalisedRunsNotSentenceInitial(t *testing.T) {
	got := candidates("Walter said Toby Curtis was late. Paige and Happy Quinn waited. The FBI called Cabe Gallo.")
	assert.ElementsMatch(t, []string{"Toby Curtis", "Happy Quinn", "Cabe Gallo"}, got,
		"sentence-initial single words (Walter, Paige, The) are not names here; multi-word runs and mid-sentence capitals are; FBI is an acronym")
	assert.Equal(t, []string{"O'Brien"}, candidates("Walter O'Brien is a genius."),
		"a sentence-initial run loses its first token: the capital on Walter proves nothing here")
	assert.Equal(t, []string{"Toby"}, candidates("Ask Toby about it."), "Ask is only capitalised because it starts the sentence")
	assert.Empty(t, candidates("I think it's Monday. OK?"), "stopwords and acronyms never qualify")
	assert.Equal(t, []string{"Walter"}, candidates("Hey, Walter, come on."), "mid-sentence single capital is a candidate")
}

func TestMine_LearnsRenderingsByCooccurrence(t *testing.T) {
	var segs []Segment
	// Walter appears in 5 segments, always with 華特; 托比 only with Toby.
	for i := 0; i < 5; i++ {
		segs = append(segs, Segment{En: fmt.Sprintf("Hey, Walter, line %d.", i), Zh: fmt.Sprintf("嘿，華特，第%d句", i)})
	}
	segs = append(segs,
		Segment{En: "Ask Toby about it.", Zh: "去問托比"},
		Segment{En: "I told Toby already.", Zh: "我已經跟托比說了"},
		Segment{En: "Where's Toby?", Zh: "托比呢？"},
	)
	// Noise: a common word in every line must not be learned as a name.
	for i := 0; i < 6; i++ {
		segs = append(segs, Segment{En: "We need to go now.", Zh: "我們現在得走了"})
	}
	// Sentence-initial Walter in two more lines: adds nothing wrong.
	segs = append(segs, Segment{En: "Walter is here.", Zh: "華特來了"}, Segment{En: "Paige left.", Zh: "佩姬走了"})

	terms := Mine(segs, Options{})
	byName := map[string]Term{}
	for _, tm := range terms {
		byName[tm.Src] = tm
	}
	require.Contains(t, byName, "Walter")
	assert.Equal(t, "華特", byName["Walter"].Zh)
	assert.Equal(t, 6, byName["Walter"].Support, "the sentence-initial 「Walter is here.」 counts once Walter is a known candidate")
	assert.Equal(t, "cooccurrence", byName["Walter"].How)
	require.Contains(t, byName, "Toby")
	assert.Equal(t, "托比", byName["Toby"].Zh)
	assert.NotContains(t, byName, "Paige", "one segment is below the ≥3 bar")
	assert.NotContains(t, byName, "We", "never a candidate")
	assert.Equal(t, "Walter", terms[0].Src, "ordered by support")
}

func TestMine_KnownRenderingsAreVerifiedFirst(t *testing.T) {
	segs := []Segment{
		{En: "Call Cabe.", Zh: "打給凱布"},
		{En: "Cabe is outside.", Zh: "凱布在外面"},
		{En: "Where is Cabe?", Zh: "凱布在哪"},
		{En: "Sylvester, go.", Zh: "希維斯特，走"},
	}
	terms := Mine(segs, Options{Known: map[string]string{"cabe": "凱布", "Sylvester": "希維斯特", "Megan": "梅根"}})
	byName := map[string]Term{}
	for _, tm := range terms {
		byName[tm.Src] = tm
	}
	assert.Equal(t, Term{Src: "cabe", Zh: "凱布", Support: 3, Segments: 3, How: "known"}, byName["cabe"],
		"a known term is verified by plain word search, sentence-initial or not, case-insensitively")
	assert.Equal(t, "known", byName["Sylvester"].How, "a known term needs only one confirming line")
	assert.NotContains(t, byName, "Megan", "known but never seen in the subtitles → not emitted")
}

func TestMine_RejectsRenderingSharedWithOtherLines(t *testing.T) {
	// 我們 appears in every line; it must never be learned as Walter's rendering.
	var segs []Segment
	for i := 0; i < 4; i++ {
		segs = append(segs, Segment{En: "Hey, Walter, we go.", Zh: "我們走吧"})
	}
	for i := 0; i < 10; i++ {
		segs = append(segs, Segment{En: "We go.", Zh: "我們走吧"})
	}
	assert.Empty(t, Mine(segs, Options{}), "a rendering that is not specific to the term is refused")
}

func TestMine_RealWorldGuards(t *testing.T) {
	// "Tell" is capitalised after a dialogue dash in one line and lower-cased
	// elsewhere → a word, never a name. "Pekka's" is the name Pekka.
	segs := []Segment{
		{En: "-Tell him now.", Zh: "告訴他"},
		{En: "-Tell her too.", Zh: "也告訴她"},
		{En: "-Tell them all.", Zh: "告訴大家"},
		{En: "I will tell you later.", Zh: "我晚點告訴你"},
		{En: "That is Pekka's club.", Zh: "那是佩卡的賭場"},
		{En: "Pekka's men are here.", Zh: "佩卡的人來了"},
		{En: "Go and find Pekka's ledger.", Zh: "去找佩卡的帳本"},
	}
	terms := Mine(segs, Options{})
	byName := map[string]Term{}
	for _, tm := range terms {
		byName[tm.Src] = tm
	}
	assert.NotContains(t, byName, "Tell")
	assert.NotContains(t, byName, "Pekka's")
	require.Contains(t, byName, "Pekka")
	assert.Equal(t, "佩卡", byName["Pekka"].Zh, "the possessive 的 is not part of the name")
}

func TestHanSubstrings_UpToEightRunesWithoutParticles(t *testing.T) {
	subs := hanSubstrings("艾利娜史塔科夫來了嗎")
	_, has7 := subs["艾利娜史塔科夫"]
	_, has9 := subs["艾利娜史塔科夫來了"]
	_, endsWithLe := subs["科夫來了"]
	_, startsWithPronoun := hanSubstrings("我是華特")["我是"]
	_, tai := hanSubstrings("太陽召喚者")["太陽召喚者"]
	assert.True(t, has7)
	assert.False(t, has9)
	assert.False(t, endsWithLe, "a run ending in 了 is a phrase, not a name")
	assert.False(t, startsWithPronoun)
	assert.True(t, tai, "太 as in 太陽 must stay allowed")
}

func TestMine_SubTermsAndDashes(t *testing.T) {
	segs := []Segment{
		{En: "That was Pekka Rollins.", Zh: "那是佩卡羅林斯"},
		{En: "You work for Pekka Rollins now.", Zh: "你現在替佩卡羅林斯做事"},
		{En: "Tell Pekka Rollins no.", Zh: "跟佩卡羅林斯說不"},
		{En: "-Shut up. -Make me.", Zh: "閉嘴 你來啊"},
		{En: "-Shut it, Jesper. -Fine.", Zh: "閉嘴，傑斯柏 好"},
		{En: "-Shut the door.", Zh: "把門關上"},
	}
	byName := map[string]Term{}
	for _, tm := range Mine(segs, Options{}) {
		byName[tm.Src] = tm
	}
	assert.Contains(t, byName, "Pekka Rollins")
	assert.NotContains(t, byName, "Rollins", "the surname alone only ever appeared inside the full name")
	assert.NotContains(t, byName, "Shut", "a word after a dialogue dash is sentence-initial")
}
