package handlers

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/subtitle/providers"
)

// episodeSubtitleStore is what the manual search/download paths need about an
// episode (disc-2026-11-online-subs-for-episodes); repos.Episodes satisfies it.
type episodeSubtitleStore interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
	UpdateEpisodeSubtitleStatus(ctx context.Context, episodeID string, status models.SubtitleStatus, path, language string) error
}

// ManualSubtitleLookup gives the manual subtitle paths read access to the
// library: titles to build a search query from, and episodes to search for
// and record a download against.
type ManualSubtitleLookup struct {
	Movies   countryMovieFinder
	Series   countrySeriesFinder
	Episodes episodeSubtitleStore
}

// SetMediaLookup wires the library lookup. Unset, a search must carry its own
// query text and episodes cannot be searched.
func (h *SubtitleHandler) SetMediaLookup(l ManualSubtitleLookup) {
	h.lookup = l
}

// searchTitle picks the name an online subtitle site indexes a title under:
// the original title, since Assrt and OpenSubtitles list the original or the
// mainland name, not the Taiwan one (「天蠍行動」 finds nothing,「Scorpion」
// does). Japanese kana and Korean hangul originals fall back to the Chinese
// title, which is what Assrt's Chinese index carries for those.
func searchTitle(original models.NullString, title string) string {
	o := strings.TrimSpace(original.String)
	if !original.Valid || o == "" {
		return title
	}
	for _, r := range o {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			return title
		}
	}
	return o
}

func yearOf(date string) int {
	if len(date) < 4 {
		return 0
	}
	var y int
	if _, err := fmt.Sscanf(date[:4], "%d", &y); err != nil {
		return 0
	}
	return y
}

// queryFor builds the search query for a library item when the request names
// no query text: movie → original title + year; series → original title;
// episode → the show's original title + season + episode.
func (h *SubtitleHandler) queryFor(ctx context.Context, mediaType, mediaID string) (providers.SubtitleQuery, error) {
	switch mediaType {
	case "movie":
		if h.lookup.Movies == nil {
			break
		}
		movie, err := h.lookup.Movies.FindByID(ctx, mediaID)
		if err != nil || movie == nil {
			return providers.SubtitleQuery{}, fmt.Errorf("movie %s not found: %w", mediaID, err)
		}
		return providers.SubtitleQuery{
			Title: searchTitle(movie.OriginalTitle, movie.Title),
			Year:  yearOf(movie.ReleaseDate),
		}, nil
	case "series":
		if h.lookup.Series == nil {
			break
		}
		show, err := h.lookup.Series.FindByID(ctx, mediaID)
		if err != nil || show == nil {
			return providers.SubtitleQuery{}, fmt.Errorf("series %s not found: %w", mediaID, err)
		}
		return providers.SubtitleQuery{Title: searchTitle(show.OriginalTitle, show.Title)}, nil
	case "episode":
		if h.lookup.Episodes == nil || h.lookup.Series == nil {
			break
		}
		ep, err := h.lookup.Episodes.FindByID(ctx, mediaID)
		if err != nil || ep == nil {
			return providers.SubtitleQuery{}, fmt.Errorf("episode %s not found: %w", mediaID, err)
		}
		show, err := h.lookup.Series.FindByID(ctx, ep.SeriesID)
		if err != nil || show == nil {
			return providers.SubtitleQuery{}, fmt.Errorf("series %s of episode %s not found: %w", ep.SeriesID, mediaID, err)
		}
		return providers.SubtitleQuery{
			Title:   searchTitle(show.OriginalTitle, show.Title),
			Season:  ep.SeasonNumber,
			Episode: ep.EpisodeNumber,
		}, nil
	}
	return providers.SubtitleQuery{}, fmt.Errorf("no library lookup for %s", mediaType)
}

// seriesOfEpisode returns the episode's show id, "" when unknown. The
// own-wording rule reads production countries off the show.
func (h *SubtitleHandler) seriesOfEpisode(ctx context.Context, episodeID string) string {
	if h.lookup.Episodes == nil {
		return ""
	}
	ep, err := h.lookup.Episodes.FindByID(ctx, episodeID)
	if err != nil || ep == nil {
		return ""
	}
	return ep.SeriesID
}
