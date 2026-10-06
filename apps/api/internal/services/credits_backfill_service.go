// Package services — credits_backfill_service.go
//
// disc-2026-10-series-credits-empty: fill the cast on matched rows that never
// got one.
//
// Story sub-7-3 (2026-09-07) started storing TMDb cast on a row — but only at
// the moment the row is matched, and nothing ever re-matches a row that is
// already matched. Every title matched before that date (on the NAS: all 81
// series and all 55 movies) therefore still has no cast, which is why the
// speech-recognition name prompt (#697) had three actor names and no
// character names to give for See S01E02.
//
// This service walks those rows and fetches their cast once, through the same
// fetcher enrichment uses. It never overwrites: a row with any cast at all —
// TMDb's or the user's from the Metadata Editor — is not in its query.
package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/vido/api/internal/models"
)

// Implements: disc-2026-10-series-credits-empty + sub-7-3 (cast on the row)

// MovieCreditsBackfillRepo is the two movie-repo calls the backfill needs.
type MovieCreditsBackfillRepo interface {
	FindMissingCredits(ctx context.Context, limit int) ([]models.Movie, error)
	UpdateCredits(ctx context.Context, id string, credits *models.Credits) error
}

// SeriesCreditsBackfillRepo is the series counterpart.
type SeriesCreditsBackfillRepo interface {
	FindMissingCredits(ctx context.Context, limit int) ([]models.Series, error)
	UpdateCredits(ctx context.Context, id string, credits *models.Credits) error
}

// creditsBackfillBatch bounds one pass; the NAS library needs 136 rows, a
// large one a few passes on consecutive starts. TMDb's own limiter in the
// client paces the calls.
const creditsBackfillBatch = 200

// CreditsBackfillService fills missing cast on matched rows.
type CreditsBackfillService struct {
	movies MovieCreditsBackfillRepo
	series SeriesCreditsBackfillRepo
	fetch  GlossaryCreditsFetcher
	logger *slog.Logger
	// tried remembers rows whose TMDb cast came back empty, so a title TMDb
	// genuinely has no cast for is asked once per process, not once per pass.
	tried map[string]struct{}
}

// NewCreditsBackfillService wires the backfill; any nil dependency disables it.
func NewCreditsBackfillService(movies MovieCreditsBackfillRepo, series SeriesCreditsBackfillRepo, fetch GlossaryCreditsFetcher, logger *slog.Logger) *CreditsBackfillService {
	if logger == nil {
		logger = slog.Default()
	}
	return &CreditsBackfillService{movies: movies, series: series, fetch: fetch,
		logger: logger.With("service", "credits_backfill"), tried: map[string]struct{}{}}
}

// CreditsBackfillResult is what one pass did, for the log line and tests.
type CreditsBackfillResult struct {
	Movies, Series int // rows that now have a cast
	Empty          int // TMDb had no cast for them (not retried this process)
	Failed         int // fetch or write errors (retried next pass)
}

// Run does one pass over both tables. Fail-soft per row: one TMDb miss is
// logged and the walk continues; the pass itself never returns an error.
func (s *CreditsBackfillService) Run(ctx context.Context) CreditsBackfillResult {
	var res CreditsBackfillResult
	if s.fetch == nil {
		return res
	}
	if s.movies != nil {
		movies, err := s.movies.FindMissingCredits(ctx, creditsBackfillBatch)
		if err != nil {
			s.logger.Warn("credits backfill: listing movies failed", "error", err)
		}
		for _, m := range movies {
			if ctx.Err() != nil {
				return res
			}
			if !m.TMDbID.Valid {
				continue
			}
			s.fill(ctx, &res, &res.Movies, "movie", m.ID, m.TMDbID.Int64, s.movies.UpdateCredits)
		}
	}
	if s.series != nil {
		list, err := s.series.FindMissingCredits(ctx, creditsBackfillBatch)
		if err != nil {
			s.logger.Warn("credits backfill: listing series failed", "error", err)
		}
		for _, sr := range list {
			if ctx.Err() != nil {
				return res
			}
			if !sr.TMDbID.Valid {
				continue
			}
			s.fill(ctx, &res, &res.Series, "tv", sr.ID, sr.TMDbID.Int64, s.series.UpdateCredits)
		}
	}
	if res.Movies+res.Series+res.Empty+res.Failed > 0 {
		s.logger.Info("credits backfill pass finished",
			"movies_filled", res.Movies, "series_filled", res.Series, "tmdb_has_no_cast", res.Empty, "failed", res.Failed)
	}
	return res
}

func (s *CreditsBackfillService) fill(ctx context.Context, res *CreditsBackfillResult, filled *int, mediaType, id string, tmdbID int64, write func(context.Context, string, *models.Credits) error) {
	if _, done := s.tried[id]; done {
		return
	}
	credits, _, err := s.fetch.FetchCredits(ctx, mediaType, tmdbID)
	if err != nil && credits == nil {
		res.Failed++
		s.logger.Warn("credits backfill: TMDb fetch failed", "id", id, "media_type", mediaType, "tmdb_id", tmdbID, "error", err)
		return
	}
	if credits == nil || len(credits.Cast) == 0 {
		res.Empty++
		s.tried[id] = struct{}{}
		return
	}
	if err := write(ctx, id, credits); err != nil {
		res.Failed++
		s.logger.Warn("credits backfill: write failed", "id", id, "media_type", mediaType, "error", err)
		return
	}
	*filled++
}

// RunAfter runs one pass in the background once the app has settled (the
// scanner and enrichment get the first minutes), then stops. Called from
// main at boot; a pass is cheap and idempotent, so every boot may run one.
func (s *CreditsBackfillService) RunAfter(ctx context.Context, delay time.Duration) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		s.Run(ctx)
	}()
}
