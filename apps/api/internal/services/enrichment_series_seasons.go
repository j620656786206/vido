package services

import (
	"context"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// seasonRowsRepo is the slice of the season repository a re-match needs.
type seasonRowsRepo interface {
	FindBySeriesID(ctx context.Context, seriesID string) ([]models.Season, error)
	Update(ctx context.Context, season *models.Season) error
}

// SetSeasonRepository wires the season rows, so a series that becomes a
// different show also gets that show's seasons
// (disc-2026-10-rematch-keeps-old-season-rows). nil = season rows are left as
// they are.
func (s *EnrichmentService) SetSeasonRepository(repo seasonRowsRepo) {
	s.seasons = repo
}

// resetSeasonsForNewShow rewrites the series' season rows when its TMDb id
// changed. Each row becomes the new show's season — replaced, not merged: a
// field the new show does not have must not keep the old show's value. A
// season the new show does not have keeps its row (its episode files hang off
// it) with every TMDb field cleared. Fail-soft: a TMDb failure leaves the rows
// alone rather than wiping them, and the match itself stands.
func (s *EnrichmentService) resetSeasonsForNewShow(ctx context.Context, seriesID string, previous, current models.NullInt64) {
	if s.seasons == nil || tmdbIDOf(previous) == tmdbIDOf(current) {
		return
	}
	byNumber := map[int]tmdb.Season{}
	if id := tmdbIDOf(current); id > 0 {
		if s.tmdbService == nil {
			return
		}
		details, err := s.tmdbService.GetTVShowDetails(ctx, int(id))
		if err != nil || details == nil {
			s.logger.Warn("TMDb details for the new match failed; seasons left as they were",
				"id", seriesID, "tmdb_id", id, "error", err)
			return
		}
		for _, ts := range details.Seasons {
			byNumber[ts.SeasonNumber] = ts
		}
	}

	rows, err := s.seasons.FindBySeriesID(ctx, seriesID)
	if err != nil {
		s.logger.Warn("season rows could not be read; seasons left as they were", "id", seriesID, "error", err)
		return
	}
	for i := range rows {
		row := &rows[i]
		ts, ok := byNumber[row.SeasonNumber]
		if !ok {
			ts = tmdb.Season{} // not a season of the new show: clear it
		}
		row.TMDbID = positiveInt64(int64(ts.ID))
		row.Name = nonEmptyString(ts.Name)
		row.Overview = nonEmptyString(ts.Overview)
		row.PosterPath = nonEmptyString(derefString(ts.PosterPath))
		row.AirDate = nonEmptyString(derefString(ts.AirDate))
		row.EpisodeCount = positiveInt64(int64(ts.EpisodeCount))
		row.VoteAverage = models.NullFloat64{} // the old show's rating
		if err := s.seasons.Update(ctx, row); err != nil {
			s.logger.Warn("season row reset failed", "id", seriesID, "season", row.SeasonNumber, "error", err)
		}
	}
	s.logger.Info("season rows reset for the new match", "id", seriesID,
		"from_tmdb_id", tmdbIDOf(previous), "to_tmdb_id", tmdbIDOf(current), "seasons", len(rows))
}

func tmdbIDOf(id models.NullInt64) int64 {
	if !id.Valid || id.Int64 <= 0 {
		return 0
	}
	return id.Int64
}

func positiveInt64(v int64) models.NullInt64 {
	if v <= 0 {
		return models.NullInt64{}
	}
	return models.NewNullInt64(v)
}

func nonEmptyString(v string) models.NullString {
	if v == "" {
		return models.NullString{}
	}
	return models.NewNullString(v)
}
