package services

import (
	"context"
	"strings"
	"unicode"

	"github.com/vido/api/internal/wikipedia"
)

// WikiCastSource is the zh.wikipedia surface the seeder needs;
// *wikipedia.Client satisfies it.
type WikiCastSource interface {
	GetWikitext(ctx context.Context, titles ...string) (string, string, error)
}

var _ WikiCastSource = (*wikipedia.Client)(nil)

// SetWikiCastSource turns on the zh.wikipedia lookup for Chinese character
// names (disc-2026-10-glossary-seeds-actor-names-and-fan-surname-b).
func (s *GlossarySeeder) SetWikiCastSource(src WikiCastSource) {
	s.wiki = src
}

// glossaryWikiMaxPages caps how many page titles one title tries (the client
// allows one request a second).
const glossaryWikiMaxPages = 4

// fillCharactersFromWiki gives a Chinese title's characters their Chinese
// names when TMDb has only pinyin for them — 《百花殺》's zh-TW and zh-CN
// credits both say "Shen Xihe", so the glossary got no characters at all, and
// speech recognition had no 沈汐和 to listen for. zh.wikipedia's 演員陣容 table
// pairs each actor with the role in Chinese; rows are matched by the actor's
// Chinese name, which TMDb does have. A page is used only when at least two of
// the title's actors (one, if only one needs a name) are found in its table —
// so a page about the novel, or another show of the same name, is not.
func (s *GlossarySeeder) fillCharactersFromWiki(ctx context.Context, id int, facts titleFacts, pairs []CastPair, pairActors []string) []CastPair {
	if s.wiki == nil || !facts.chinese || len(facts.names) == 0 {
		return pairs
	}
	var gaps []int
	for i, p := range pairs {
		if p.Kind == "character" && i < len(pairActors) && pairActors[i] != "" && !hasHan(p.Zh) {
			gaps = append(gaps, i)
		}
	}
	if len(gaps) == 0 {
		return pairs
	}

	actors := make([]string, len(gaps))
	original := make([]string, len(gaps))
	for i, g := range gaps {
		actors[i] = pairActors[g]
		original[i] = pairs[g].Zh
	}
	actors = s.toTaiwan("", actors)
	need := 2
	if distinct(actors) < 2 {
		need = 1
	}

	names := facts.names
	if len(names) > glossaryWikiMaxPages {
		names = names[:glossaryWikiMaxPages]
	}
	for _, title := range names {
		text, page, err := s.wiki.GetWikitext(ctx, title)
		if err != nil {
			if !wikipedia.IsNotFound(err) {
				s.logger.Warn("glossary seed: zh.wikipedia unavailable; characters stay as TMDb has them", "tmdb_id", id, "error", err)
				return pairs
			}
			continue
		}
		byActor := s.wikiCharactersByActor(wikipedia.ParseCast(text))
		filled := 0
		for i, g := range gaps {
			if character, ok := byActor[nameKey(actors[i])]; ok {
				pairs[g].Zh = character
				filled++
			}
		}
		if filled >= need {
			s.logger.Info("glossary seed: character names filled from zh.wikipedia",
				"tmdb_id", id, "page", page, "filled", filled, "of", len(gaps))
			return pairs
		}
		for i, g := range gaps { // not this show's page: undo
			pairs[g].Zh = original[i]
		}
	}
	return pairs
}

// wikiCharactersByActor keys a cast table by the actor's Taiwan-script name;
// the first role wins, and "步疏林／步疏杳" keeps its first name.
func (s *GlossarySeeder) wikiCharactersByActor(entries []wikipedia.CastEntry) map[string]string {
	if len(entries) == 0 {
		return nil
	}
	actors := make([]string, len(entries))
	for i, e := range entries {
		actors[i] = e.Actor
	}
	actors = s.toTaiwan("", actors)
	out := make(map[string]string, len(entries))
	for i, e := range entries {
		key := nameKey(actors[i])
		if _, dup := out[key]; dup || key == "" {
			continue
		}
		character := strings.TrimSpace(strings.FieldsFunc(e.Character, func(r rune) bool { return r == '／' || r == '/' })[0])
		if character != "" {
			out[key] = character
		}
	}
	return out
}

// nameKey compares names without the spaces and middle dots transliterations
// disagree on.
func nameKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '·' || r == '・' || r == '•' {
			return -1
		}
		return r
	}, s)
}

func distinct(ss []string) int {
	set := map[string]bool{}
	for _, s := range ss {
		set[s] = true
	}
	return len(set)
}
