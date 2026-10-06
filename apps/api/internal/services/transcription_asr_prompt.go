package services

import (
	"context"
	"strings"
	"unicode"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// asrPromptNames is the ordered list of names speech recognition should
// spell correctly (disc-2026-10-asr-proper-names-inconsistent, the 聽 half):
//
//  1. character names from TMDb credits (what people SAY — Jerlamarel, Baba
//     Voss), in billing order;
//  2. actor names (said in a making-of, sung in a theme — rarer, so after);
//  3. the show's trusted glossary sources — TMDb seed, official-subtitle
//     mining, manual, community — but NEVER the `subtitle` harvest: those
//     rows are what the LAST speech-recognition run heard (Chola Morel,
//     Durla Morel, Jorna Morel…), and feeding them back would teach the
//     decoder its own mistakes (disc-2026-10-asr-harvest-pollutes-glossary).
//
// Pure, so the whole policy is one table test; de-duplication and the
// whisper window cap happen in ai.BuildASRPrompt.
//
// disc-2026-10-asr-name-prompt-too-long: CHARACTERS only, in billing order,
// then trusted glossary terms — actor names are not words anyone says in the
// episode, and the zh-TW spellings TMDb returns for them ("傑森·摩莫亞") were
// read back verbatim over the score. Anything not written in Latin letters
// is skipped: the prompt conditions an English transcript.
//
// The stored credits are TMDb's zh-TW answer (glossary_seeder FetchCredits),
// so a character may arrive translated ("帕里斯"); the metadata glossary holds
// that same pair the other way round (TermZh 帕里斯 → TermSrc Paris), which
// is how the English name is recovered without losing billing order (CR 2).
// Glossary rows whose TermZh is an ACTOR's stored name are actor pairs and are
// skipped for the same reason actors are not sent at all.
func asrPromptNames(credits *models.Credits, terms []models.GlossaryTerm) []string {
	zhToSrc := make(map[string]string, len(terms))
	for _, t := range terms {
		if t.Source != models.GlossarySourceSubtitle && t.TermZh != "" && t.TermSrc != "" {
			zhToSrc[t.TermZh] = t.TermSrc
		}
	}
	actorZh := map[string]struct{}{}
	var names []string
	if credits != nil {
		for _, c := range credits.Cast {
			if c.Name != "" {
				actorZh[c.Name] = struct{}{}
			}
			n := promptCharacterName(c.Character)
			if n == "" {
				continue
			}
			if !isLatinName(n) {
				n = zhToSrc[n]
			}
			if n != "" && isLatinName(n) {
				names = append(names, n)
			}
		}
	}
	for _, t := range terms {
		if t.Source == models.GlossarySourceSubtitle {
			continue
		}
		if _, actor := actorZh[t.TermZh]; actor {
			continue
		}
		if t.TermSrc != "" && isLatinName(t.TermSrc) {
			names = append(names, t.TermSrc)
		}
	}
	return names
}

// isLatinName is true when the name has at least one letter and every letter
// is in the Latin script (Zoë, O'Neil, Jean-Luc, Ægir pass; 帕里斯, ジョン,
// Иван do not — the prompt conditions an English transcript).
func isLatinName(s string) bool {
	hasLetter := false
	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		if !unicode.Is(unicode.Latin, r) {
			return false
		}
		hasLetter = true
	}
	return hasLetter
}

// promptCharacterName cleans a TMDb character credit for the prompt: the
// parenthetical tail goes ("Baba Voss (voice)" → "Baba Voss") and the
// non-names TMDb uses for people playing themselves ("Self", "Himself",
// "Herself", "Narrator") are skipped — they are not words anyone says.
func promptCharacterName(character string) string {
	n := strings.TrimSpace(character)
	// A multi-role credit is stored joined with " / " (glossary_seeder
	// tvCreditsToModel); the first role is the one billed (CR 3).
	if i := strings.Index(n, " / "); i > 0 {
		n = strings.TrimSpace(n[:i])
	}
	if i := strings.Index(n, "("); i > 0 {
		n = strings.TrimSpace(n[:i])
	}
	switch strings.ToLower(n) {
	case "", "self", "himself", "herself", "themselves", "narrator":
		return ""
	}
	return n
}

// asrPromptFor composes the conditioning text for one run. Fail-soft: any
// lookup that misses contributes nothing — a prompt-less run is exactly the
// pre-story behaviour, never a failed run — but never silently (the
// mediaMetadataFor CR M1 posture): a miss is logged with what it cost.
func (s *TranscriptionService) asrPromptFor(ctx context.Context, mediaType, mediaID, glossaryKey string) string {
	var credits *models.Credits
	switch mediaType {
	case models.SubtitleRunMediaMovie:
		if s.stateReader != nil {
			movie, err := s.stateReader.FindByID(ctx, mediaID)
			if err != nil || movie == nil {
				s.logger.Warn("asr prompt: movie lookup failed — prompting without credits",
					"media_id", mediaID, "error", err)
			} else {
				credits = movie.Credits
			}
		}
	case models.SubtitleRunMediaEpisode, models.SubtitleRunMediaSeries:
		// glossaryKey is the SHOW: the parent series for an episode, the row
		// itself for a series run. The only unusable case is an episode whose
		// parent could not be resolved (glossaryKeyFor fell back to the
		// episode's own id) — querying the series table with it would miss.
		parentUnresolved := mediaType == models.SubtitleRunMediaEpisode && glossaryKey == mediaID
		if s.seriesReader != nil && glossaryKey != "" && !parentUnresolved {
			series, err := s.seriesReader.FindByID(ctx, glossaryKey)
			if err != nil || series == nil {
				s.logger.Warn("asr prompt: series lookup failed — prompting without credits",
					"media_id", mediaID, "media_type", mediaType, "series_id", glossaryKey, "error", err)
			} else {
				credits = series.Credits
			}
		}
	}

	var terms []models.GlossaryTerm
	if s.glossaryRepo != nil {
		scope := s.glossaryScopeFor(ctx, glossaryKey)
		if rows, err := s.glossaryRepo.ListByScope(ctx, scope); err == nil {
			terms = rows
		} else {
			s.logger.Warn("asr prompt: glossary lookup failed — prompting from credits only",
				"media_id", mediaID, "scope", scope, "error", err)
		}
	}

	return ai.BuildASRPrompt(asrPromptNames(credits, terms))
}
