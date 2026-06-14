package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// SeasonDetailsProvider is the narrow slice of the TMDb service that the season
// accordion needs: fetch a season's full episode list (cached 24h). Implemented
// by *TMDbService.
type SeasonDetailsProvider interface {
	GetSeasonDetails(ctx context.Context, tvID int, seasonNumber int) (*tmdb.SeasonDetails, error)
}

// ErrSeasonDepsNotConfigured is returned when GetSeasonEpisodes is called before
// the episode repo / TMDb season provider have been wired via SetEpisodeDeps.
var ErrSeasonDepsNotConfigured = errors.New("series season dependencies not configured")

// ErrSeriesNotLinkedToTMDb is returned when a series has no tmdb_id and therefore
// cannot resolve its canonical episode list.
var ErrSeriesNotLinkedToTMDb = errors.New("series is not linked to TMDb")

// MergedEpisode is a single episode row for the detail-page accordion: TMDb
// episode metadata merged with local file/subtitle enrichment. JSON tags are
// snake_case (Rule 18) — the web client transforms to camelCase via fetchApi.
type MergedEpisode struct {
	EpisodeNumber int     `json:"episode_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview,omitempty"`
	AirDate       *string `json:"air_date,omitempty"`
	Runtime       *int    `json:"runtime,omitempty"`
	StillPath     *string `json:"still_path,omitempty"`
	VoteAverage   float64 `json:"vote_average"`

	// Local enrichment — only meaningful when HasLocalFile is true (AC #5/#6).
	HasLocalFile     bool   `json:"has_local_file"`
	SubtitleStatus   string `json:"subtitle_status,omitempty"`
	SubtitleLanguage string `json:"subtitle_language,omitempty"`
	FilePath         string `json:"file_path,omitempty"`
}

// SeasonEpisodesResponse is the payload for the season-episodes endpoint:
// the season summary (from the series' cached SeasonsJSON) plus merged episodes.
type SeasonEpisodesResponse struct {
	Season   models.SeasonSummary `json:"season"`
	Episodes []MergedEpisode      `json:"episodes"`
}

// GetSeasons returns the cached season summaries for a series, read from the
// series' SeasonsJSON column (no TMDb call) (Story 12-2 Task 4.1).
func (s *SeriesService) GetSeasons(ctx context.Context, seriesID string) ([]models.SeasonSummary, error) {
	if seriesID == "" {
		return nil, fmt.Errorf("series id cannot be empty")
	}
	if s.seasonRepo == nil {
		return nil, ErrSeasonDepsNotConfigured
	}

	// bugfix-20-1: read the canonical `seasons` table (populated by the parse
	// pipeline, parse_queue_service.go), NOT the dead `series.seasons` JSON column
	// — which FindByID never even SELECTs, so the old path always returned [].
	seasons, err := s.seasonRepo.FindBySeriesID(ctx, seriesID)
	if err != nil {
		slog.Error("Failed to load seasons", "error", err, "series_id", seriesID)
		return nil, fmt.Errorf("failed to load seasons: %w", err)
	}

	summaries := make([]models.SeasonSummary, 0, len(seasons))
	for i := range seasons {
		summaries = append(summaries, seasonToSummary(&seasons[i]))
	}
	return summaries, nil
}

// seasonToSummary maps a persisted Season (the `seasons` table) to the
// SeasonSummary shape the accordion consumes. SeasonSummary.ID is the TMDb season
// id (the FE keys off it). bugfix-20-1.
func seasonToSummary(se *models.Season) models.SeasonSummary {
	summary := models.SeasonSummary{
		SeasonNumber: se.SeasonNumber,
		Name:         se.Name.String,
		Overview:     se.Overview.String,
		PosterPath:   se.PosterPath.String,
		AirDate:      se.AirDate.String,
	}
	if se.TMDbID.Valid {
		summary.ID = int(se.TMDbID.Int64)
	}
	if se.EpisodeCount.Valid {
		summary.EpisodeCount = int(se.EpisodeCount.Int64)
	}
	return summary
}

// GetSeasonEpisodes merges the canonical TMDb episode list for a season with the
// local episode records (file path + subtitle status) and returns the combined
// view for the accordion (Story 12-2 Tasks 4.2/4.3).
//
// TMDb provides the authoritative episode list; local records (only present for
// scanned files) enrich matching episodes by (season_number, episode_number).
func (s *SeriesService) GetSeasonEpisodes(ctx context.Context, seriesID string, seasonNumber int) (*SeasonEpisodesResponse, error) {
	if seriesID == "" {
		return nil, fmt.Errorf("series id cannot be empty")
	}
	if s.episodeRepo == nil || s.seasonProvider == nil {
		return nil, ErrSeasonDepsNotConfigured
	}

	series, err := s.repo.FindByID(ctx, seriesID)
	if err != nil {
		slog.Error("Failed to get series for season episodes", "error", err, "series_id", seriesID)
		return nil, err
	}

	if !series.TMDbID.Valid || series.TMDbID.Int64 <= 0 {
		return nil, ErrSeriesNotLinkedToTMDb
	}
	tmdbID := int(series.TMDbID.Int64)

	// Resolve the season summary from the cached SeasonsJSON (for the response
	// header). Falls back to a minimal summary if not present.
	season := s.resolveSeasonSummary(series, seasonNumber)

	// Canonical episode list from TMDb (cached). A failure here surfaces to the
	// handler as a retry-able error inside the accordion body (AC #7).
	details, err := s.seasonProvider.GetSeasonDetails(ctx, tmdbID, seasonNumber)
	if err != nil {
		slog.Warn("Failed to fetch TMDb season details",
			"error", err, "series_id", seriesID, "tmdb_id", tmdbID, "season_number", seasonNumber)
		return nil, fmt.Errorf("failed to fetch season episodes: %w", err)
	}

	// Local episodes for subtitle/file enrichment, indexed by episode number.
	localByNumber := s.localEpisodesByNumber(ctx, seriesID, seasonNumber)

	episodes := make([]MergedEpisode, 0, len(details.Episodes))
	for _, te := range details.Episodes {
		merged := MergedEpisode{
			EpisodeNumber: te.EpisodeNumber,
			Name:          te.Name,
			Overview:      te.Overview,
			AirDate:       te.AirDate,
			Runtime:       te.Runtime,
			StillPath:     te.StillPath,
			VoteAverage:   te.VoteAverage,
		}

		if local, ok := localByNumber[te.EpisodeNumber]; ok && local.FilePath.Valid && local.FilePath.String != "" {
			merged.HasLocalFile = true
			merged.FilePath = local.FilePath.String
			merged.SubtitleStatus = string(local.SubtitleStatus)
			if local.SubtitleLanguage.Valid {
				merged.SubtitleLanguage = local.SubtitleLanguage.String
			}
		}

		episodes = append(episodes, merged)
	}

	return &SeasonEpisodesResponse{Season: season, Episodes: episodes}, nil
}

// resolveSeasonSummary returns the cached SeasonSummary for the given season
// number, or a minimal summary (just the number) when the series has no cached
// seasons data.
func (s *SeriesService) resolveSeasonSummary(series *models.Series, seasonNumber int) models.SeasonSummary {
	seasons, err := series.GetSeasons()
	if err != nil {
		slog.Warn("Failed to parse seasons JSON; using minimal summary",
			"error", err, "series_id", series.ID)
	}
	for _, sm := range seasons {
		if sm.SeasonNumber == seasonNumber {
			return sm
		}
	}
	return models.SeasonSummary{SeasonNumber: seasonNumber}
}

// localEpisodesByNumber loads the local episode records for a season and indexes
// them by episode number. A repository error is logged and treated as "no local
// records" so episode display degrades to TMDb-only rather than failing.
func (s *SeriesService) localEpisodesByNumber(ctx context.Context, seriesID string, seasonNumber int) map[int]models.Episode {
	localEpisodes, err := s.episodeRepo.FindBySeasonNumber(ctx, seriesID, seasonNumber)
	if err != nil {
		slog.Warn("Failed to load local episodes; showing TMDb data only",
			"error", err, "series_id", seriesID, "season_number", seasonNumber)
		return map[int]models.Episode{}
	}

	byNumber := make(map[int]models.Episode, len(localEpisodes))
	for _, ep := range localEpisodes {
		byNumber[ep.EpisodeNumber] = ep
	}
	return byNumber
}
