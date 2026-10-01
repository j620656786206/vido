package eval

import (
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
