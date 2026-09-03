package subtitle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
)

// ErrMediaNotFound is returned by Load when the referenced row does not exist.
//
// It is a SENTINEL rather than a formatted string because sub-1-6's endpoint
// has to tell "no such media id" (404) apart from "the database is locked"
// (500) — collapsing the two would send an operator hunting for a missing row
// that is actually there.
var ErrMediaNotFound = errors.New("media row not found")

// ─── Repository ports ──────────────────────────────────────────────────────
//
// Narrow by media type rather than one fat interface, so a test fakes three
// methods instead of the ~30 on the real repository interfaces. The concrete
// repositories satisfy these directly.

// MovieMediaRepo is the movie half of the MediaStore adapter.
type MovieMediaRepo interface {
	FindByID(ctx context.Context, id string) (*models.Movie, error)
	UpdateSubtitleGenerationStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string) error
}

// SeriesMediaRepo is the series half.
type SeriesMediaRepo interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
	UpdateSubtitleGenerationStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string) error
}

// EpisodeMediaRepo is the episode half. Episodes already had a search-column-safe
// writer (UpdateEpisodeSubtitleStatus) before this story.
type EpisodeMediaRepo interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
	UpdateEpisodeSubtitleStatus(ctx context.Context, episodeID string, status models.SubtitleStatus, path, language string) error
}

// repoMediaStore resolves a MediaRef to the file path + FR26 metadata the
// pipeline needs, and writes terminal status back (sub-1-5b's MediaStore port).
//
// WRITE CONTRACT: every write goes through a generation-specific repository
// method that touches subtitle_status / subtitle_path / subtitle_language ONLY.
// The search columns (`subtitle_search_score`, `subtitle_last_searched`) stay
// untouched — extract-and-translate is not a search, and stamping them would
// make a generated sidecar indistinguishable from a provider hit in the library
// UI (sub-1-5b AC #6.3). This is what `backlog-subtitle-status-writer-search-columns`
// was filed for, and sub-1-6 owns the fix because it owns this adapter.
type repoMediaStore struct {
	movies   MovieMediaRepo
	series   SeriesMediaRepo
	episodes EpisodeMediaRepo
}

// NewMediaStore builds the MediaStore adapter over the three media repositories.
func NewMediaStore(movies MovieMediaRepo, series SeriesMediaRepo, episodes EpisodeMediaRepo) MediaStore {
	return &repoMediaStore{movies: movies, series: series, episodes: episodes}
}

func (s *repoMediaStore) Load(ctx context.Context, ref MediaRef) (*MediaItem, error) {
	switch ref.MediaType {
	case models.SubtitleRunMediaMovie:
		return s.loadMovie(ctx, ref.ID)
	case models.SubtitleRunMediaSeries:
		return s.loadSeries(ctx, ref.ID)
	case models.SubtitleRunMediaEpisode:
		return s.loadEpisode(ctx, ref.ID)
	default:
		return nil, fmt.Errorf("unknown media type %q for %s", ref.MediaType, ref.ID)
	}
}

func (s *repoMediaStore) loadMovie(ctx context.Context, id string) (*MediaItem, error) {
	if s.movies == nil {
		return nil, fmt.Errorf("media store: no movie repository wired")
	}
	movie, err := s.movies.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load movie %s: %w", id, err)
	}
	if movie == nil {
		return nil, fmt.Errorf("movie %s: %w", id, ErrMediaNotFound)
	}

	return &MediaItem{
		FilePath: movie.FilePath.String,
		TMDbID:   nullInt64Ptr(movie.TMDbID),
		// 9R-10b AC #3: captured so the FreeOnly brake can put the row back
		// exactly where it found it instead of flattening it to not_searched.
		SubtitleStatus:   movie.SubtitleStatus,
		SubtitlePath:     movie.SubtitlePath.String,
		SubtitleLanguage: movie.SubtitleLanguage.String,
		// Movies carry no ShowKey: nothing shares their prompt prefix, so the
		// D10 gate bypasses them entirely (sub-1-5b AC #5.1).
		ShowKey: "",
		Context: withoutUntrustedIdentity(TranslateContext{
			Title:         movie.Title,
			OriginalTitle: movie.OriginalTitle.String,
			Year:          yearOf(movie.ReleaseDate),
			Genres:        movie.Genres,
			Overview:      movie.Overview.String,
			Countries:     countryCodes(movie.ProductionCountries),
		}, movie.TMDbID.Valid, id, models.SubtitleRunMediaMovie),
	}, nil
}

// withoutUntrustedIdentity blanks the identity fields (Title, OriginalTitle,
// Year, Overview) when they cannot be trusted as show context (sub-6-7;
// eval-1 finding 12): the row never matched TMDb — so every one of those
// fields came from filename parsing — or the title itself is filename-shaped
// even though a match exists. Genres and Countries are left as they are:
// they are empty on an unmatched row anyway, and on a matched one they are
// TMDb's, not the filename's. Blank fields make BuildMetadataSection render
// nothing and MetadataHash equal the no-identity hash, so the cache key does
// not carry the release name either.
func withoutUntrustedIdentity(ctx TranslateContext, tmdbMatched bool, mediaID, mediaType string) TranslateContext {
	reason := ""
	switch {
	case !tmdbMatched:
		reason = "unmatched"
	case prompts.LooksLikeFilename(ctx.Title):
		reason = "filename-shaped"
	default:
		return ctx
	}
	slog.Info("metadata title skipped — not sent to the translation prompt",
		"reason", reason, "media_id", mediaID, "media_type", mediaType)
	ctx.Title = ""
	ctx.OriginalTitle = ""
	ctx.Year = 0
	ctx.Overview = ""
	return ctx
}

func (s *repoMediaStore) loadSeries(ctx context.Context, id string) (*MediaItem, error) {
	series, err := s.loadSeriesRow(ctx, id)
	if err != nil {
		return nil, err
	}
	return &MediaItem{
		FilePath:         series.FilePath.String,
		TMDbID:           nullInt64Ptr(series.TMDbID),
		SubtitleStatus:   series.SubtitleStatus,
		SubtitlePath:     series.SubtitlePath.String,
		SubtitleLanguage: series.SubtitleLanguage.String,
		// A series row IS its own show, so it keys the gate on itself.
		ShowKey: series.ID,
		Context: seriesContext(series),
	}, nil
}

// loadEpisode resolves the episode's own file path but the PARENT SERIES'
// metadata: the FR26 prompt context is show-level (title, cast, genres), and
// keying it per episode would split the segment cache's MetadataHash for every
// episode of the same show — silently halving the cache hit rate.
func (s *repoMediaStore) loadEpisode(ctx context.Context, id string) (*MediaItem, error) {
	if s.episodes == nil {
		return nil, fmt.Errorf("media store: no episode repository wired")
	}
	episode, err := s.episodes.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load episode %s: %w", id, err)
	}
	if episode == nil {
		return nil, fmt.Errorf("episode %s: %w", id, ErrMediaNotFound)
	}

	item := &MediaItem{
		FilePath:         episode.FilePath.String,
		TMDbID:           nullInt64Ptr(episode.TMDbID),
		SubtitleStatus:   episode.SubtitleStatus,
		SubtitlePath:     episode.SubtitlePath.String,
		SubtitleLanguage: episode.SubtitleLanguage.String,
		// The D10 gate keys on the SERIES id, never the episode id — an
		// episode-keyed gate would serialize nothing while defeating the exact
		// season-batch case D10 exists for (sub-1-5b AC #5.1).
		ShowKey: episode.SeriesID,
	}

	// A missing parent is not fatal: the metadata is optional context for the
	// prompt, and refusing to translate an episode because its series row is
	// unmatched would be worse than translating it with less context.
	if series, err := s.loadSeriesRow(ctx, episode.SeriesID); err == nil {
		item.Context = seriesContext(series)
	}
	return item, nil
}

func (s *repoMediaStore) loadSeriesRow(ctx context.Context, id string) (*models.Series, error) {
	if s.series == nil {
		return nil, fmt.Errorf("media store: no series repository wired")
	}
	series, err := s.series.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load series %s: %w", id, err)
	}
	if series == nil {
		return nil, fmt.Errorf("series %s: %w", id, ErrMediaNotFound)
	}
	return series, nil
}

func (s *repoMediaStore) SetSubtitleStatus(ctx context.Context, ref MediaRef, status models.SubtitleStatus, subtitlePath, language string) error {
	switch ref.MediaType {
	case models.SubtitleRunMediaMovie:
		if s.movies == nil {
			return fmt.Errorf("media store: no movie repository wired")
		}
		return s.movies.UpdateSubtitleGenerationStatus(ctx, ref.ID, status, subtitlePath, language)
	case models.SubtitleRunMediaSeries:
		if s.series == nil {
			return fmt.Errorf("media store: no series repository wired")
		}
		return s.series.UpdateSubtitleGenerationStatus(ctx, ref.ID, status, subtitlePath, language)
	case models.SubtitleRunMediaEpisode:
		if s.episodes == nil {
			return fmt.Errorf("media store: no episode repository wired")
		}
		return s.episodes.UpdateEpisodeSubtitleStatus(ctx, ref.ID, status, subtitlePath, language)
	default:
		return fmt.Errorf("unknown media type %q for %s", ref.MediaType, ref.ID)
	}
}

func seriesContext(series *models.Series) TranslateContext {
	return withoutUntrustedIdentity(TranslateContext{
		Title:         series.Title,
		OriginalTitle: series.OriginalTitle.String,
		Year:          yearOf(series.FirstAirDate),
		Genres:        series.Genres,
		Overview:      series.Overview.String,
	}, series.TMDbID.Valid, series.ID, models.SubtitleRunMediaSeries)
}

// yearOf extracts the year from an ISO date. An unparseable or empty date
// yields 0, which TranslateContext treats as "no year" — the prompt simply
// omits it rather than rendering a bogus one.
func yearOf(date string) int {
	if len(date) < 4 {
		return 0
	}
	year, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return year
}

// countryCodes projects TMDb's country objects onto the ISO codes the prompt
// renders. Order is normalized by MetadataHash (sub-1-5b sorts a copy), so the
// caller's order does not split the segment cache.
func countryCodes(countries []models.ProductionCountry) []string {
	if len(countries) == 0 {
		return nil
	}
	out := make([]string, 0, len(countries))
	for _, c := range countries {
		if code := strings.TrimSpace(c.ISO3166_1); code != "" {
			out = append(out, code)
		}
	}
	return out
}

func nullInt64Ptr(v models.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	value := v.Int64
	return &value
}

// compile-time proof the adapter satisfies sub-1-5b's port.
var _ MediaStore = (*repoMediaStore)(nil)
