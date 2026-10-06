package services

import (
	"context"

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
			if c.Character != "" {
				names = append(names, c.Character)
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

// asrPromptFor composes the conditioning text for one run. Fail-soft: any
// lookup that misses contributes nothing — a prompt-less run is exactly the
// pre-story behaviour, never a failed run.
func (s *TranscriptionService) asrPromptFor(ctx context.Context, mediaType, mediaID, glossaryKey string) string {
	var credits *models.Credits
	switch mediaType {
	case models.SubtitleRunMediaMovie:
		if s.stateReader != nil {
			if movie, err := s.stateReader.FindByID(ctx, mediaID); err == nil && movie != nil {
				credits = movie.Credits
			}
		}
	case models.SubtitleRunMediaEpisode, models.SubtitleRunMediaSeries:
		if s.seriesReader != nil && glossaryKey != "" && glossaryKey != mediaID {
			if series, err := s.seriesReader.FindByID(ctx, glossaryKey); err == nil && series != nil {
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
