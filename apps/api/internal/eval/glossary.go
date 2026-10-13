package eval

import (
	"fmt"
	"regexp"
	"sort"
)

// NameTerm is one fixed rendering the exam hands the translator, the way the
// app's per-show glossary does (prompts.BuildGlossarySection).
type NameTerm struct {
	Source string
	Target string
}

// NameGlossaries builds one glossary per cue source (tos / sita / trap — the
// exam's stand-in for "one show"), from every cue's `names`, taking the FIRST
// accepted rendering of each name as the fixed one. Terms are sorted by
// Source so the prompt and the version hash are stable. Two cues of the same
// source that disagree on a name's first rendering are a sample bug.
func NameGlossaries(cues []Cue) (map[string][]NameTerm, error) {
	seen := map[string]map[string]string{}
	for _, c := range cues {
		for name, variants := range c.Names {
			if len(variants) == 0 {
				continue
			}
			if seen[c.Source] == nil {
				seen[c.Source] = map[string]string{}
			}
			if prev, ok := seen[c.Source][name]; ok && prev != variants[0] {
				return nil, fmt.Errorf("eval: %s: name %q is %q in one cue and %q in %s", c.Source, name, prev, variants[0], c.ID)
			}
			seen[c.Source][name] = variants[0]
		}
	}
	out := make(map[string][]NameTerm, len(seen))
	for source, names := range seen {
		terms := make([]NameTerm, 0, len(names))
		for name, target := range names {
			terms = append(terms, NameTerm{Source: name, Target: target})
		}
		sort.Slice(terms, func(i, j int) bool { return terms[i].Source < terms[j].Source })
		out[source] = terms
	}
	return out, nil
}

// GlossaryFor is the glossary for a batch: the union of the glossaries of
// the sources its cues belong to (a batch can straddle two), sorted.
func GlossaryFor(glossaries map[string][]NameTerm, cues []Cue) []NameTerm {
	sources := map[string]bool{}
	for _, c := range cues {
		sources[c.Source] = true
	}
	return unionTerms(glossaries, sources)
}

// AllTerms is every source's glossary together — what the report's
// glossary_version hashes.
func AllTerms(glossaries map[string][]NameTerm) []NameTerm {
	sources := map[string]bool{}
	for s := range glossaries {
		sources[s] = true
	}
	return unionTerms(glossaries, sources)
}

func unionTerms(glossaries map[string][]NameTerm, sources map[string]bool) []NameTerm {
	type key struct{ s, t string }
	seen := map[key]bool{}
	var out []NameTerm
	for source := range sources {
		for _, t := range glossaries[source] {
			if k := (key{t.Source, t.Target}); !seen[k] {
				seen[k] = true
				out = append(out, t)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].Target < out[j].Target
	})
	return out
}

// TermsIn keeps the terms whose source name appears as a whole word in any of
// texts (case-insensitive; a "word" boundary is any non-letter, non-digit, so
// "José," and "THOM!" match while "Ram" does not match inside "Rama").
func TermsIn(terms []NameTerm, texts ...string) []NameTerm {
	var out []NameTerm
	for _, t := range terms {
		re := regexp.MustCompile(`(?i)(^|[^\p{L}\p{N}])` + regexp.QuoteMeta(t.Source) + `([^\p{L}\p{N}]|$)`)
		for _, text := range texts {
			if re.MatchString(text) {
				out = append(out, t)
				break
			}
		}
	}
	return out
}
