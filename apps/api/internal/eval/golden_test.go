package eval

import (
	"fmt"
	"strings"
	"testing"
)

// TestGolden_SchemaAndComposition pins AC #1: 200 cues, 76/84/40, unique ids,
// two references each, every trap category represented at least three times,
// and no song lyric slipped in from the Sita source.
func TestGolden_SchemaAndComposition(t *testing.T) {
	cues, err := Golden()
	if err != nil {
		t.Fatalf("Golden(): %v", err)
	}
	if len(cues) != 200 {
		t.Fatalf("want 200 cues, got %d", len(cues))
	}
	cats := map[string]int{}
	for _, c := range cues {
		if c.Attribution == "" {
			t.Errorf("%s: missing attribution", c.ID)
		}
		if !strings.HasPrefix(c.ID, c.Source+"-") {
			t.Errorf("%s: id does not carry its source %q", c.ID, c.Source)
		}
		for _, cat := range c.Traps {
			cats[cat]++
		}
		if c.Source == "sita" && (strings.Contains(c.Text, "♪") || strings.Contains(c.Attribution, "lyric")) {
			t.Errorf("%s: looks like a lyric", c.ID)
		}
		for _, r := range c.Refs {
			if strings.ContainsAny(r, "这没软") {
				t.Errorf("%s: reference carries simplified characters: %q", c.ID, r)
			}
		}
	}
	for _, cat := range []string{"slang", "pun", "name_consistency", "time_shift", "simplified_bait", "tw_lexicon", "number_unit", "register", "idiom"} {
		if cats[cat] < 3 {
			t.Errorf("trap category %s: want ≥3 cues, got %d", cat, cats[cat])
		}
	}
}

func TestValidate_RejectsBrokenSamples(t *testing.T) {
	good := func() []Cue {
		var out []Cue
		for src, n := range ExpectedCounts {
			for i := 0; i < n; i++ {
				c := Cue{ID: src + "-" + string(rune('a'+i%26)) + string(rune('a'+i/26)), Source: src, Text: "hi", Refs: []string{"嗨", "你好"}}
				if src == "trap" {
					c.Traps = []string{"slang"}
				}
				out = append(out, c)
			}
		}
		return out
	}
	if err := Validate(good()); err != nil {
		t.Fatalf("baseline should validate: %v", err)
	}
	cases := map[string]func([]Cue) []Cue{
		"duplicate id":   func(c []Cue) []Cue { c[1].ID = c[0].ID; return c },
		"one ref":        func(c []Cue) []Cue { c[0].Refs = c[0].Refs[:1]; return c },
		"empty text":     func(c []Cue) []Cue { c[0].Text = "  "; return c },
		"unknown source": func(c []Cue) []Cue { c[0].Source = "youtube"; return c },
		"short by one":   func(c []Cue) []Cue { return c[1:] },
		"trap without cat": func(c []Cue) []Cue {
			for i := range c {
				if c[i].Source == "trap" {
					c[i].Traps = nil
					break
				}
			}
			return c
		},
	}
	for name, mutate := range cases {
		if err := Validate(mutate(good())); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestParseJSONL_ReportsLine(t *testing.T) {
	_, err := ParseJSONL([]byte("{\"id\":\"a\"}\n\nnot json\n"))
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("want line-3 error, got %v", err)
	}
}

func TestNameGlossaries(t *testing.T) {
	cues, err := Golden()
	if err != nil {
		t.Fatal(err)
	}
	g, err := NameGlossaries(cues)
	if err != nil {
		t.Fatal(err)
	}
	if len(g) != 3 {
		t.Fatalf("one glossary per source, got %d", len(g))
	}
	want := []NameTerm{{"Barley", "貝里"}, {"Celia", "希莉亞"}, {"Thom", "湯姆"}}
	if fmt.Sprint(g["tos"]) != fmt.Sprint(want) {
		t.Fatalf("tos glossary: %v", g["tos"])
	}
	if got := GlossaryFor(g, cues[:2]); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("a tos batch carries the tos glossary only: %v", got)
	}
	mixed := []Cue{cues[0], cues[len(cues)-1]}
	if got := GlossaryFor(g, mixed); len(got) != len(g["tos"])+len(g[cues[len(cues)-1].Source]) {
		t.Fatalf("a batch straddling two sources gets both glossaries: %v", got)
	}
	if n := len(AllTerms(g)); n != len(g["tos"])+len(g["sita"])+len(g["trap"]) {
		t.Fatalf("AllTerms: %d", n)
	}
}

func TestNameGlossaries_FirstRenderingWinsAndConflictsFail(t *testing.T) {
	ok := []Cue{{ID: "a", Source: "s", Names: map[string][]string{"Thom": {"湯姆", "湯瑪斯"}}}}
	g, err := NameGlossaries(ok)
	if err != nil || g["s"][0].Target != "湯姆" {
		t.Fatalf("first rendering is the fixed one: %v %v", g, err)
	}
	bad := append(ok, Cue{ID: "b", Source: "s", Names: map[string][]string{"Thom": {"托姆"}}})
	if _, err := NameGlossaries(bad); err == nil {
		t.Fatal("two cues of one source disagreeing on a name is a sample bug")
	}
}

func TestTermsIn_WholeWordsOnly(t *testing.T) {
	terms := []NameTerm{{"Ram", "羅摩"}, {"Rama", "羅摩"}, {"Sita", "西妲"}}
	got := TermsIn(terms, "Rama is here.", "Nothing")
	if len(got) != 1 || got[0].Source != "Rama" {
		t.Fatalf("Ram must not match inside Rama: %v", got)
	}
	accented := []NameTerm{{"José", "荷西"}, {"Thom", "湯姆"}}
	if got := TermsIn(accented, "Hello José, hi", "THOM!"); len(got) != 2 {
		t.Fatalf("non-ASCII names and shouted names still match: %v", got)
	}
	if got := TermsIn(accented, "Thomas came", "Josélito"); len(got) != 0 {
		t.Fatalf("a name inside a longer word must not match: %v", got)
	}
}
