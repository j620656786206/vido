package services

import (
	"context"
	"strings"

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
func asrPromptNames(credits *models.Credits, terms []models.GlossaryTerm) []string {
	var names []string
	if credits != nil {
		for _, c := range credits.Cast {
			if n := promptCharacterName(c.Character); n != "" {
				names = append(names, n)
			}
		}
		for _, c := range credits.Cast {
			if c.Name != "" {
				names = append(names, c.Name)
			}
		}
	}
	for _, t := range terms {
		if t.Source == models.GlossarySourceSubtitle {
			continue
		}
		if t.TermSrc != "" {
			names = append(names, t.TermSrc)
		}
	}
	return names
}

// promptCharacterName cleans a TMDb character credit for the prompt: the
// parenthetical tail goes ("Baba Voss (voice)" → "Baba Voss") and the
// non-names TMDb uses for people playing themselves ("Self", "Himself",
// "Herself", "Narrator") are skipped — they are not words anyone says.
func promptCharacterName(character string) string {
	n := strings.TrimSpace(character)
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
