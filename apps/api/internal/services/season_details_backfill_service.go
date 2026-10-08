// Package services — season_details_backfill_service.go
//
// disc-2026-10-season-episode-count-unknown: give seasons their TMDb detail.
//
// The scanner creates a season row the moment it sees an episode file —
// before the series is matched — and the code meant to fill it read a dead
// series.seasons JSON column, so nothing ever did: on the NAS all 134 seasons
// had no episode count, name, poster or TMDb id, and every season header said
// 「劇集數未知」 next to 「無圖」. This service fills them from the series'
// TMDb details, once a little after boot and after every library scan.
package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// Implements: disc-2026-10-season-episode-count-unknown AC #1, #3

// SeasonDetailsRepo is the slice of the season repository the backfill needs.
type SeasonDetailsRepo interface {
	FindSeriesNeedingSeasonDetails(ctx context.Context, limit int) ([]repository.SeasonDetailsGap, error)
	FindBySeriesID(ctx context.Context, seriesID string) ([]models.Season, error)
	Update(ctx context.Context, season *models.Season) error
}

// TVDetailsFetcher is *TMDbService's series-details call (cached, rate-limited).
type TVDetailsFetcher interface {
	GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error)
}

// seasonDetailsBackfillBatch bounds one pass; the NAS needs 81 series. Large
// enough that resting (tried) series cannot starve the rest of the list.
const seasonDetailsBackfillBatch = 5000

// SeasonDetailsBackfillService fills empty season rows from TMDb.
type SeasonDetailsBackfillService struct {
	seasons SeasonDetailsRepo
	tmdb    TVDetailsFetcher
	logger  *slog.Logger

	mu      sync.Mutex
	running bool
	rerun   bool
	// tried remembers when a series stayed incomplete after asking TMDb (it
	// lacks that season, a season had not aired, or the call failed), so it
	// is not re-fetched every scan — for triedFor, not for the process' life:
	// an unaired season airs, a transient error passes (CR M1).
	tried map[string]time.Time
	now   func() time.Time
}

// triedFor is how long an incomplete series rests before it is asked again.
const triedFor = 12 * time.Hour

// NewSeasonDetailsBackfillService wires the backfill; nil deps disable it.
func NewSeasonDetailsBackfillService(seasons SeasonDetailsRepo, fetcher TVDetailsFetcher, logger *slog.Logger) *SeasonDetailsBackfillService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SeasonDetailsBackfillService{seasons: seasons, tmdb: fetcher,
		logger: logger.With("service", "season_details_backfill"), tried: map[string]time.Time{}, now: time.Now}
}

// SeasonDetailsBackfillResult is what one pass did.
type SeasonDetailsBackfillResult struct {
	Series  int // series whose seasons were updated
	Seasons int // season rows written
	Failed  int // TMDb or write errors
}

// Run does one pass. Fail-soft per series; never returns an error.
func (s *SeasonDetailsBackfillService) Run(ctx context.Context) SeasonDetailsBackfillResult {
	var res SeasonDetailsBackfillResult
	if s.seasons == nil || s.tmdb == nil {
		return res
	}
	gaps, err := s.seasons.FindSeriesNeedingSeasonDetails(ctx, seasonDetailsBackfillBatch)
	if err != nil {
		s.logger.Warn("season details backfill: listing failed", "error", err)
		return res
	}
	for _, g := range gaps {
		if ctx.Err() != nil {
			return res
		}
		s.mu.Lock()
		at, done := s.tried[g.SeriesID]
		s.mu.Unlock()
		if done && s.now().Sub(at) < triedFor {
			continue
		}

		details, err := s.tmdb.GetTVShowDetails(ctx, int(g.TMDbID))
		if err != nil || details == nil {
			res.Failed++
			s.giveUp(g.SeriesID)
			s.logger.Warn("season details backfill: TMDb fetch failed", "series_id", g.SeriesID, "tmdb_id", g.TMDbID, "error", err)
			continue
		}
		byNumber := make(map[int]tmdb.Season, len(details.Seasons))
		for _, ts := range details.Seasons {
			byNumber[ts.SeasonNumber] = ts
		}
		rows, err := s.seasons.FindBySeriesID(ctx, g.SeriesID)
		if err != nil {
			res.Failed++
			continue
		}
		wrote, incomplete := 0, false
		for i := range rows {
			ts, ok := byNumber[rows[i].SeasonNumber]
			if !ok {
				incomplete = true // TMDb has no such season (a mis-numbered folder)
				continue
			}
			// The series-level count only fills a gap: opening the season
			// writes the exact one (its episode list), and the two must not
			// take turns overwriting each other (CR L3).
			count := 0
			if !rows[i].EpisodeCount.Valid {
				count = ts.EpisodeCount
			}
			if !MergeSeasonDetails(&rows[i], ts.ID, ts.Name, ts.Overview, ts.PosterPath, ts.AirDate, count) {
				continue
			}
			if err := s.seasons.Update(ctx, &rows[i]); err != nil {
				res.Failed++
				s.logger.Warn("season details backfill: write failed", "season_id", rows[i].ID, "error", err)
				continue
			}
			wrote++
		}
		for i := range rows {
			if !rows[i].EpisodeCount.Valid || !rows[i].TMDbID.Valid {
				incomplete = true
			}
		}
		// Only a series that stays incomplete is remembered: one that came out
		// whole leaves the worklist by itself, and must be looked at again if a
		// later scan adds a new season to it.
		if incomplete {
			s.giveUp(g.SeriesID)
		}
		if wrote > 0 {
			res.Series++
			res.Seasons += wrote
		}
	}
	if res.Series+res.Failed > 0 {
		s.logger.Info("season details backfill pass finished", "series", res.Series, "seasons", res.Seasons, "failed", res.Failed)
	}
	return res
}

func (s *SeasonDetailsBackfillService) giveUp(seriesID string) {
	s.mu.Lock()
	s.tried[seriesID] = s.now()
	s.mu.Unlock()
}

// MergeSeasonDetails copies TMDb's season fields onto a row (AC #3: an empty
// TMDb field never blanks a stored one) and reports whether anything changed.
// Shared by the backfill and the season list's write-back.
func MergeSeasonDetails(se *models.Season, tmdbID int, name, overview string, posterPath, airDate *string, episodeCount int) bool {
	changed := false
	setStr := func(dst *models.NullString, v string) {
		if v != "" && (!dst.Valid || dst.String != v) {
			*dst = models.NewNullString(v)
			changed = true
		}
	}
	if tmdbID > 0 && (!se.TMDbID.Valid || se.TMDbID.Int64 != int64(tmdbID)) {
		se.TMDbID = models.NewNullInt64(int64(tmdbID))
		changed = true
	}
	setStr(&se.Name, name)
	setStr(&se.Overview, overview)
	if posterPath != nil {
		setStr(&se.PosterPath, *posterPath)
	}
	if airDate != nil {
		setStr(&se.AirDate, *airDate)
	}
	if episodeCount > 0 && (!se.EpisodeCount.Valid || se.EpisodeCount.Int64 != int64(episodeCount)) {
		se.EpisodeCount = models.NewNullInt64(int64(episodeCount))
		changed = true
	}
	return changed
}

// Trigger starts a pass in the background; a trigger during a pass queues
// exactly one more (a scan may have added seasons mid-pass).
func (s *SeasonDetailsBackfillService) Trigger(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.rerun = true
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()
	go func() {
		for {
			s.Run(ctx)
			s.mu.Lock()
			if !s.rerun || ctx.Err() != nil {
				s.running, s.rerun = false, false
				s.mu.Unlock()
				return
			}
			s.rerun = false
			s.mu.Unlock()
		}
	}()
}

// RunAfter triggers one pass after delay (boot), so TMDb calls wait for the
// app to settle.
func (s *SeasonDetailsBackfillService) RunAfter(ctx context.Context, delay time.Duration) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		s.Trigger(ctx)
	}()
}
