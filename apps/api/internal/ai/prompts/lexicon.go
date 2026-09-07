package prompts

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Built-in Taiwan lexicon (sub-7-4 AC #1/#2). Two things of different nature,
// deliberately NOT one table (party-mode 2026-09-03):
//
//   - replacements: post-processing, word-level, after OpenCC s2twp. OpenCC only
//     converts SCRIPT (简→繁); a model that writes 質量 or 視頻 in Traditional
//     characters sails straight through it. These are the mainland terms a
//     Taiwanese viewer trips over, mapped to what a Netflix subtitle would say.
//   - terms: brand / app / institution renderings injected into every prompt's
//     GLOBAL glossary section, ahead of the per-show glossary (which wins on a
//     clash). eval-1's zero-score example — Life360 rendered as 「360 號公路」 —
//     is exactly the class this closes.
//
// The file is embedded; its `version` rides PromptVersion and GlossaryVersion,
// so editing the lexicon re-keys the segment cache like any prompt change.

//go:embed lexicon/zh-tw.yaml
var zhTWLexiconYAML []byte

// Replacement is one word-level rewrite. Except lists longer phrases that
// CONTAIN From but must not be touched (電視頻道 ⊃ 視頻) — the "白名單" the
// story asks for, since CJK text has no word boundaries to lean on.
type Replacement struct {
	From   string   `yaml:"from"`
	To     string   `yaml:"to"`
	Except []string `yaml:"except,omitempty"`
}

// Lexicon is the parsed, validated zh-TW lexicon.
type Lexicon struct {
	Version      string          `yaml:"version"`
	Replacements []Replacement   `yaml:"replacements"`
	Terms        []GlossaryEntry `yaml:"terms"`
}

// GlossaryEntry's yaml tags live here so the prompts package does not have
// to know about YAML anywhere else; the struct itself is declared with the
// glossary builders.

var zhTWLexicon = mustLoadLexicon(zhTWLexiconYAML)

// ZhTWLexicon returns the embedded lexicon. It is parsed once at package
// init; a malformed file fails the binary at start-up (and the schema test
// long before that) rather than silently translating without it.
func ZhTWLexicon() *Lexicon { return zhTWLexicon }

// LexiconVersion is the lexicon's own version string — the part of
// PromptVersion / GlossaryVersion that changes when the YAML does.
func LexiconVersion() string { return zhTWLexicon.Version }

func mustLoadLexicon(raw []byte) *Lexicon {
	lex, err := ParseLexicon(raw)
	if err != nil {
		panic(fmt.Sprintf("prompts: embedded zh-TW lexicon is invalid: %v", err))
	}
	return lex
}

// ParseLexicon decodes and validates a lexicon document (AC #6 schema test
// surface): version required; no duplicate `from` / `source` (case-insensitive
// for Latin sources); no empty sides; every `except` must contain its `from`
// (an except that cannot match is a typo). Replacements are sorted longest
// `from` first so 打印機→印表機 is applied before 打印→列印.
func ParseLexicon(raw []byte) (*Lexicon, error) {
	var lex Lexicon
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if err := dec.Decode(&lex); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if strings.TrimSpace(lex.Version) == "" {
		return nil, fmt.Errorf("version is required")
	}

	seenFrom := map[string]struct{}{}
	for i := range lex.Replacements {
		r := &lex.Replacements[i]
		r.From, r.To = strings.TrimSpace(r.From), strings.TrimSpace(r.To)
		if r.From == "" || r.To == "" {
			return nil, fmt.Errorf("replacements[%d]: from and to are required", i)
		}
		if r.From == r.To {
			return nil, fmt.Errorf("replacements[%d] %q: from equals to", i, r.From)
		}
		if !containsHanRune(r.From) {
			return nil, fmt.Errorf("replacements[%d] %q: replacements are for Chinese text only", i, r.From)
		}
		if _, dup := seenFrom[r.From]; dup {
			return nil, fmt.Errorf("replacements[%d] %q: duplicate from", i, r.From)
		}
		seenFrom[r.From] = struct{}{}
		for j, ex := range r.Except {
			ex = strings.TrimSpace(ex)
			r.Except[j] = ex
			if ex == "" || !strings.Contains(ex, r.From) {
				return nil, fmt.Errorf("replacements[%d] %q: except[%d] %q must contain from", i, r.From, j, ex)
			}
		}
	}
	sort.SliceStable(lex.Replacements, func(a, b int) bool {
		return len([]rune(lex.Replacements[a].From)) > len([]rune(lex.Replacements[b].From))
	})

	seenSrc := map[string]struct{}{}
	for i := range lex.Terms {
		t := &lex.Terms[i]
		t.Source, t.Target = strings.TrimSpace(t.Source), strings.TrimSpace(t.Target)
		if t.Source == "" || t.Target == "" {
			return nil, fmt.Errorf("terms[%d]: source and target are required", i)
		}
		key := strings.ToLower(t.Source)
		if _, dup := seenSrc[key]; dup {
			return nil, fmt.Errorf("terms[%d] %q: duplicate source", i, t.Source)
		}
		seenSrc[key] = struct{}{}
	}
	return &lex, nil
}

// Apply rewrites mainland terms in s to Taiwan usage (AC #2 後處理). Word-level
// by construction of the table: an occurrence of `from` that sits inside one of
// its `except` phrases is left alone. Latin text is never touched — every
// `from` is Chinese (ParseLexicon enforces it). Idempotent: applying it to its
// own output changes nothing, so a cue that already reads Taiwanese is a
// no-op and the segment cache stays honest.
func (l *Lexicon) Apply(s string) string {
	if l == nil || s == "" || !containsHanRune(s) {
		return s
	}
	for _, r := range l.Replacements {
		if !strings.Contains(s, r.From) {
			continue
		}
		s = replaceOutsideExcepts(s, r)
	}
	return s
}

// replaceOutsideExcepts replaces every occurrence of r.From in s except the
// ones covered by an occurrence of one of r.Except. Byte ranges of the
// exceptions are computed on the ORIGINAL string, then a single left-to-right
// pass rebuilds it.
func replaceOutsideExcepts(s string, r Replacement) string {
	type span struct{ start, end int }
	var protected []span
	for _, ex := range r.Except {
		for from := 0; ; {
			i := strings.Index(s[from:], ex)
			if i < 0 {
				break
			}
			protected = append(protected, span{from + i, from + i + len(ex)})
			from += i + len(ex)
		}
	}
	covered := func(start, end int) bool {
		for _, p := range protected {
			if start >= p.start && end <= p.end {
				return true
			}
		}
		return false
	}

	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], r.From)
		if j < 0 {
			sb.WriteString(s[i:])
			break
		}
		start, end := i+j, i+j+len(r.From)
		sb.WriteString(s[i:start])
		if covered(start, end) {
			sb.WriteString(r.From)
		} else {
			sb.WriteString(r.To)
		}
		i = end
	}
	return sb.String()
}

// BuildLexiconTermsSection renders the GLOBAL glossary section (AC #2): the
// brand / app / institution renderings every translation gets, placed in the
// install-wide system prefix BEFORE the per-show glossary, which the model is
// told overrides it — so the prefix stays identical across shows (one shared
// prompt-cache breakpoint) and a show can still fix a rendering locally.
func BuildLexiconTermsSection(terms []GlossaryEntry) string {
	if len(terms) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Taiwan renderings for brands, apps and institutions (global glossary):\n")
	sb.WriteString("Render these EXACTLY as given — they are how a Taiwanese viewer knows them. A Chinese target means the Chinese name is the everyday one; an unchanged target means keep the original name as-is, never transliterate or explain it.\n")
	sb.WriteString("A per-show glossary below overrides any entry here.\n")
	for _, e := range terms {
		sb.WriteString(fmt.Sprintf("- %s → %s\n", e.Source, e.Target))
	}
	sb.WriteString("\n")
	return sb.String()
}

// IsMainlandContent is the PRD rule both legs share: content produced in
// mainland China keeps its own vocabulary — no OpenCC, no lexicon rewrite.
func IsMainlandContent(countries []string) bool {
	for _, c := range countries {
		if strings.EqualFold(strings.TrimSpace(c), "CN") {
			return true
		}
	}
	return false
}

func containsHanRune(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
