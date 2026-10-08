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

// disc-2026-10-mine-cross-season-rendering-split: See's official zh-TW spells
// Jerlamarel 謝拉馬威 in season 1 and 傑拉馬瑞爾 in season 2. Their shared
// piece 拉馬 out-counts either full name; the miner must still learn the
// majority FULL rendering, never the fragment.
func TestMine_CrossSeasonSplitLearnsTheMajorityFullName(t *testing.T) {
	var segs []Segment
	s1 := []string{"謝拉馬威告訴我", "去找謝拉馬威", "謝拉馬威在等", "是謝拉馬威嗎", "謝拉馬威，過來", "我見過謝拉馬威", "謝拉馬威的孩子"}
	s2 := []string{"傑拉馬瑞爾告訴我", "去找傑拉馬瑞爾", "傑拉馬瑞爾在等", "是傑拉馬瑞爾嗎"}
	en := []string{"Jerlamarel told me.", "Go find Jerlamarel.", "Jerlamarel is waiting.", "Is that Jerlamarel?", "Jerlamarel, come here.", "I have met Jerlamarel.", "The children of Jerlamarel."}
	for i, zh := range s1 {
		segs = append(segs, Segment{En: en[i], Zh: zh, EnCues: 1, ZhCues: 1})
	}
	for i, zh := range s2 {
		segs = append(segs, Segment{En: en[i], Zh: zh, EnCues: 1, ZhCues: 1})
	}
	terms := Mine(segs, Options{})
	require.NotEmpty(t, terms)
	var got Term
	for _, tm := range terms {
		if tm.Src == "Jerlamarel" {
			got = tm
		}
	}
	assert.Equal(t, "謝拉馬威", got.Zh, "the season-1 spelling (7 of 11) wins, not the shared fragment 拉馬")
	assert.Equal(t, 7, got.Support)
}

// A phrase that merely contains the name must not be "grown" into — with or
// without a vocative line in the sample (CR 2).
func TestMine_GrowDoesNotSwallowAVerb(t *testing.T) {
	cases := map[string][]Segment{
		"with a vocative": {
			{En: "Ask Toby.", Zh: "去問托比"}, {En: "Ask Toby, now.", Zh: "去問托比吧"}, {En: "Toby, come here.", Zh: "托比，過來"},
			{En: "Where is Toby?", Zh: "托比在哪"}, {En: "Just ask Toby.", Zh: "去問托比就對了"}, {En: "I am Toby.", Zh: "我是托比"},
		},
		"never alone": {
			{En: "Ask Toby.", Zh: "去問托比"}, {En: "Ask Toby, now.", Zh: "去問托比"}, {En: "Go ask Toby.", Zh: "去問托比"}, {En: "Just ask Toby.", Zh: "去問托比"},
			{En: "Where is Toby?", Zh: "托比在哪"}, {En: "Where did Toby go?", Zh: "托比在哪裡"},
		},
		"three lines, verb after": {
			{En: "Toby said no.", Zh: "托比說不"}, {En: "Toby said yes.", Zh: "托比說好"}, {En: "Find Toby now.", Zh: "快找到托比"},
		},
	}
	for name, segs := range cases {
		t.Run(name, func(t *testing.T) {
			for run := 0; run < 10; run++ {
				var got string
				for _, tm := range Mine(segs, Options{}) {
					if tm.Src == "Toby" {
						got = tm.Zh
					}
				}
				require.Equal(t, "托比", got)
			}
		})
	}
}

// CR 3: a bare first name that is only ever seen glued to a title must not be
// grown into "name+title" (dropSubTerms would then delete it outright).
func TestMine_BareNameNextToTitledNameSurvives(t *testing.T) {
	segs := []Segment{
		{En: "Serve Princess Maghra.", Zh: "效忠瑪格拉公主"}, {En: "Serve Princess Maghra well.", Zh: "好好效忠瑪格拉公主"},
		{En: "Hail, Princess Maghra, hail.", Zh: "萬歲，瑪格拉公主，萬歲"},
		{En: "Where is Maghra?", Zh: "瑪格拉在哪"}, {En: "I love Maghra.", Zh: "我愛瑪格拉"},
	}
	by := map[string]string{}
	for _, tm := range Mine(segs, Options{}) {
		by[tm.Src] = tm.Zh
	}
	assert.Equal(t, "瑪格拉", by["Maghra"])
	assert.Equal(t, "瑪格拉公主", by["Princess Maghra"])
}

// Two spellings split evenly, same length: the pick must be deterministic,
// and the shared fragment must never be the answer.
func TestMine_EvenSplitIsDeterministic(t *testing.T) {
	segs := []Segment{
		{En: "Jerlamarel told me.", Zh: "謝拉馬威告訴我"}, {En: "Go find Jerlamarel.", Zh: "去找謝拉馬威"}, {En: "Jerlamarel is waiting.", Zh: "謝拉馬威在等"},
		{En: "Is that Jerlamarel?", Zh: "是謝拉馬威嗎"}, {En: "I met Jerlamarel.", Zh: "我見過謝拉馬威"},
		{En: "Jerlamarel left.", Zh: "傑拉馬爾走了"}, {En: "Call Jerlamarel.", Zh: "叫傑拉馬爾來"}, {En: "Jerlamarel knows.", Zh: "傑拉馬爾知道"},
		{En: "Trust Jerlamarel.", Zh: "相信傑拉馬爾"}, {En: "Jerlamarel again.", Zh: "又是傑拉馬爾"},
	}
	first := ""
	for run := 0; run < 20; run++ {
		got := ""
		for _, tm := range Mine(segs, Options{}) {
			if tm.Src == "Jerlamarel" {
				got = tm.Zh
			}
		}
		assert.NotEqual(t, "拉馬", got)
		if first == "" {
			first = got
		}
		assert.Equal(t, first, got, "same input, same answer")
	}
}

// A 2+2 split too thin to complete: the fragment is dropped, not learned.
func TestMine_FragmentThatCannotGrowIsNotLearned(t *testing.T) {
	segs := []Segment{
		{En: "Go find Jerlamarel now.", Zh: "現在去找謝拉馬威"}, {En: "Go find Jerlamarel now.", Zh: "現在去找謝拉馬威"},
		{En: "Only Jerlamarel decides.", Zh: "只有傑拉馬瑞爾能決定"}, {En: "Where did Jerlamarel go?", Zh: "傑拉馬瑞爾去哪了"},
	}
	for _, tm := range Mine(segs, Options{}) {
		assert.NotEqual(t, "Jerlamarel", tm.Src, "got %q", tm.Zh)
	}
}

// disc-2026-10-mine-en-source-selection: See S01E07's English tracks are
// [2 forced, 3 full, 4 SDH]; the forced one must never be the English side,
// and the full one comes before the SDH one.
func TestClassify_EnglishStreamsSkipForcedAndPreferFull(t *testing.T) {
	tracks := []services.SubtitleTrack{
		{Language: "eng", Format: "subrip", StreamIndex: 2, Forced: true},
		{Language: "eng", Format: "subrip", StreamIndex: 4, HearingImpaired: true},
		{Language: "eng", Format: "subrip", StreamIndex: 3},
		{Language: "chi", Format: "subrip", StreamIndex: 7},
	}
	src := Classify("/m/x.mkv", nil, tracks)
	assert.Equal(t, []int{3, 4}, src.EnStreams)
	assert.Equal(t, []int{7}, src.ZhStreams)
	onlyForced := Classify("/m/x.mkv", nil, []services.SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 2, Forced: true}})
	assert.Empty(t, onlyForced.EnStreams, "a forced-only file has no English dialogue track")
	// Title-only releases (no disposition flags) are read the same way.
	byTitle := Classify("/m/x.mkv", nil, []services.SubtitleTrack{
		{Language: "eng", Format: "subrip", StreamIndex: 2, Title: "English (Forced)"},
		{Language: "eng", Format: "subrip", StreamIndex: 4, Title: "English [SDH]"},
		{Language: "eng", Format: "subrip", StreamIndex: 3, Title: "English"},
	})
	assert.Equal(t, []int{3, 4}, byTitle.EnStreams)
}

// disc-2026-10-mine-accented-names-truncated AC #1/#2: names with accented
// letters are whole candidates — The Rings of Power's official subtitles
// taught 凱薩督姆 to "Khazad-d", 法拉松 to "Pharaz" and 盧恩 to "Rh".
func TestCandidates_AccentedNamesAreWhole(t *testing.T) {
	assert.Equal(t, []string{"Khazad-dûm"}, candidates("They fled to Khazad-dûm."))
	assert.Equal(t, []string{"Pharazôn"}, candidates("We must tell Pharazôn."))
	assert.Equal(t, []string{"Rhûn", "Míriel"}, candidates("The road to Rhûn is long, Míriel."))
	assert.Equal(t, []string{"Númenor"}, candidates("Welcome to Númenor."))
	// CR H1/M4: other scripts never join or become a candidate.
	assert.Equal(t, []string{"Gandalf", "Rivendell"}, candidates("I saw Gandalf的朋友 and 他去了 Rivendell today"))
	assert.Empty(t, candidates("He said Привет Борис."))
}

func TestMentionsWord_UnicodeBoundaries(t *testing.T) {
	cases := []struct {
		text, term string
		want       bool
	}{
		{"The road to Rhûn.", "Rh", false},  // the old byte-wise check said yes
		{"The road to Rhûn.", "Rhûn", true}, // whole word
		{"Rhûn's armies", "Rhûn", true},     // apostrophe ends the word
		{"KHAZAD-DÛM", "Khazad-dûm", true},  // case-insensitive, accent too
		{"Pharazôn", "Pharaz", false},       // a fragment is not the word
		{"Ask Walter.", "Walt", false},
		{"Ask Walter.", "Walter", true},
		{"Gandalf的朋友", "Gandalf", true}, // Han after a name still ends the word (CR M3)
		{"見到Gandalf了", "Gandalf", true},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, MentionsWord(tc.text, tc.term), "%q in %q", tc.term, tc.text)
	}
}

func TestMine_LearnsAccentedNamesWhole(t *testing.T) {
	// Varied lines, as real dialogue is: the name is the only thing they share.
	segs := []Segment{
		{En: "We march to Khazad-dûm.", Zh: "我們前往凱薩督姆"},
		{En: "The doors of Khazad-dûm are shut.", Zh: "凱薩督姆的大門關了"},
		{En: "Is Khazad-dûm far?", Zh: "凱薩督姆很遠嗎？"},
		{En: "Nobody leaves Khazad-dûm.", Zh: "沒有人離開凱薩督姆"},
		{En: "Tell Pharazôn.", Zh: "告訴法拉松"},
		{En: "Where is Pharazôn now?", Zh: "法拉松現在在哪？"},
		{En: "I serve Pharazôn.", Zh: "我效忠法拉松"},
		{En: "Even Pharazôn was afraid.", Zh: "連法拉松都怕了"},
		{En: "The east of Rhûn burns.", Zh: "盧恩以東在燃燒"},
		{En: "They came from Rhûn.", Zh: "他們從盧恩來"},
		{En: "Is that Rhûn?", Zh: "那是盧恩嗎？"},
		{En: "Far beyond Rhûn, nothing.", Zh: "盧恩之外，什麼也沒有"},
	}
	for i := 0; i < 6; i++ {
		segs = append(segs, Segment{En: "We need to go now.", Zh: "我們現在得走了"})
	}
	byName := map[string]string{}
	for _, tm := range Mine(segs, Options{}) {
		byName[tm.Src] = tm.Zh
	}
	assert.Equal(t, "凱薩督姆", byName["Khazad-dûm"])
	assert.Equal(t, "法拉松", byName["Pharazôn"])
	assert.Equal(t, "盧恩", byName["Rhûn"])
	for _, frag := range []string{"Khazad-d", "Pharaz", "Rh"} {
		assert.NotContains(t, byName, frag)
	}
}
