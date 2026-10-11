// Package services — context_facets_backfill_service.go
//
// disc-2026-10-production-countries-never-filled: give every matched title its
// production countries and genres.
//
// Both come from TMDb's details call and both feed the subtitle pipeline: the
// countries decide whether a CN/HK/MO title keeps its own wording when
// Simplified Chinese is converted (zhtw.KeepsOwnWording), and both go into the
// translation prompt's description of the title. On the NAS every one of 136
// titles had no countries (the enrichment writer never wrote the column) and
// almost none had genres (rows matched before sub-7-2b never fetched details,
// and nothing re-matches a matched row).
//
// This service walks those rows once a little after boot and fills only what
// is empty, through FillContextFacets — so a match or an edit that lands while
// it is asking TMDb is never overwritten. New matches store their countries
// themselves (EnrichmentService.persistCountries), so it does not need to run
// after scans.
package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// Implements: disc-2026-10-production-countries-never-filled AC #3–#5

// MovieContextFacetsRepo is the slice of the movie repository the backfill needs.
type MovieContextFacetsRepo interface {
	FindMissingContextFacets(ctx context.Context, afterID string, limit int) ([]models.Movie, error)
	FillContextFacets(ctx context.Context, id, genresJSON string, countriesJSON models.NullString) (bool, error)
}

// SeriesContextFacetsRepo is the series counterpart.
type SeriesContextFacetsRepo interface {
	FindMissingContextFacets(ctx context.Context, afterID string, limit int) ([]models.Series, error)
	FillContextFacets(ctx context.Context, id, genresJSON string, countriesJSON models.NullString) (bool, error)
}

// ContextFacetsFetcher is the TMDb details calls (TMDbService: 24h cache and
// the client's own rate limiter).
type ContextFacetsFetcher interface {
	GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error)
	GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error)
}

// contextFacetsBatch is one page; a pass pages through every listed row.
const contextFacetsBatch = 200

// ContextFacetsBackfillService fills missing countries and genres on matched rows.
type ContextFacetsBackfillService struct {
	movies MovieContextFacetsRepo
	series SeriesContextFacetsRepo
	fetch  ContextFacetsFetcher
	logger *slog.Logger
	// tried remembers rows TMDb had nothing more for, so they are asked once
	// per process rather than once per pass.
	tried map[string]struct{}
}

// NewContextFacetsBackfillService wires the backfill; a nil fetcher disables it,
// a nil repo skips that table.
func NewContextFacetsBackfillService(movies MovieContextFacetsRepo, series SeriesContextFacetsRepo, fetch ContextFacetsFetcher, logger *slog.Logger) *ContextFacetsBackfillService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ContextFacetsBackfillService{movies: movies, series: series, fetch: fetch,
		logger: logger.With("service", "context_facets_backfill"), tried: map[string]struct{}{}}
}

// ContextFacetsBackfillResult is what one pass did, for the log line and tests.
type ContextFacetsBackfillResult struct {
	Movies, Series int // rows that got at least one facet
	Empty          int // TMDb had nothing to fill (not retried this process)
	Failed         int // fetch, list or write errors (retried next pass)
}

// Run does one pass over both tables. Fail-soft per row; never returns an error.
func (s *ContextFacetsBackfillService) Run(ctx context.Context) ContextFacetsBackfillResult {
	var res ContextFacetsBackfillResult
	if s.fetch == nil {
		return res
	}
	if s.movies != nil {
		s.walkMovies(ctx, &res)
	}
	if s.series != nil {
		s.walkSeries(ctx, &res)
	}
	if res.Movies+res.Series+res.Empty+res.Failed > 0 {
		s.logger.Info("context facets backfill pass finished",
			"movies_filled", res.Movies, "series_filled", res.Series, "tmdb_has_none", res.Empty, "failed", res.Failed)
	}
	return res
}

func (s *ContextFacetsBackfillService) walkMovies(ctx context.Context, res *ContextFacetsBackfillResult) {
	after := ""
	for {
		page, err := s.movies.FindMissingContextFacets(ctx, after, contextFacetsBatch)
		if err != nil {
			res.Failed++
			s.logger.Warn("context facets backfill: listing movies failed", "error", err)
			return
		}
		for _, m := range page {
			if ctx.Err() != nil {
				return
			}
			if _, done := s.tried[m.ID]; done || !m.TMDbID.Valid {
				continue
			}
			details, err := s.fetch.GetMovieDetails(ctx, int(m.TMDbID.Int64))
			if err != nil || details == nil {
				res.Failed++
				s.logger.Warn("context facets backfill: TMDb fetch failed", "id", m.ID, "media_type", "movie", "tmdb_id", m.TMDbID.Int64, "error", err)
				continue
			}
			s.fill(ctx, res, &res.Movies, "movie", m.ID, genreNames(details.Genres), tmdbCountriesToModel(details.ProductionCountries), s.movies.FillContextFacets)
		}
		if len(page) < contextFacetsBatch {
			return
		}
		after = page[len(page)-1].ID
	}
}

func (s *ContextFacetsBackfillService) walkSeries(ctx context.Context, res *ContextFacetsBackfillResult) {
	after := ""
	for {
		page, err := s.series.FindMissingContextFacets(ctx, after, contextFacetsBatch)
		if err != nil {
			res.Failed++
			s.logger.Warn("context facets backfill: listing series failed", "error", err)
			return
		}
		for _, sr := range page {
			if ctx.Err() != nil {
				return
			}
			if _, done := s.tried[sr.ID]; done || !sr.TMDbID.Valid {
				continue
			}
			details, err := s.fetch.GetTVShowDetails(ctx, int(sr.TMDbID.Int64))
			if err != nil || details == nil {
				res.Failed++
				s.logger.Warn("context facets backfill: TMDb fetch failed", "id", sr.ID, "media_type", "tv", "tmdb_id", sr.TMDbID.Int64, "error", err)
				continue
			}
			// TV details often carry only origin_country — same rule as a match.
			s.fill(ctx, res, &res.Series, "tv", sr.ID, genreNames(details.Genres), seriesCountriesFromDetails(details), s.series.FillContextFacets)
		}
		if len(page) < contextFacetsBatch {
			return
		}
		after = page[len(page)-1].ID
	}
}

func (s *ContextFacetsBackfillService) fill(ctx context.Context, res *ContextFacetsBackfillResult, filled *int, mediaType, id string, genres []string, countries []models.ProductionCountry,
	write func(context.Context, string, string, models.NullString) (bool, error)) {
	// Serialise through the models so the stored shape is exactly what a
	// match writes (Movie.GenresJSON / Movie.SetProductionCountries).
	var m models.Movie
	m.Genres = genres
	genresJSON, err := m.GenresJSON()
	if err == nil && len(countries) > 0 {
		err = m.SetProductionCountries(countries)
	}
	if err != nil {
		res.Failed++
		s.logger.Warn("context facets backfill: encode failed", "id", id, "media_type", mediaType, "error", err)
		return
	}
	written, err := write(ctx, id, genresJSON, m.ProductionCountriesJSON)
	if err != nil {
		res.Failed++
		s.logger.Warn("context facets backfill: write failed", "id", id, "media_type", mediaType, "error", err)
		return
	}
	if !written {
		// TMDb had nothing for the column(s) still empty.
		res.Empty++
		s.tried[id] = struct{}{}
		return
	}
	*filled++
}

func genreNames(genres []tmdb.Genre) []string {
	out := make([]string, 0, len(genres))
	for _, g := range genres {
		if g.Name != "" {
			out = append(out, g.Name)
		}
	}
	return out
}

// RunAfter runs one pass in the background once the app has settled, then
// stops. Called from main at boot; idempotent, so every boot may run one.
// The returned stop cancels the pass and waits for it to return — main calls
// it before db.Close() so an in-flight write never meets a closed DB (CR L2).
func (s *ContextFacetsBackfillService) RunAfter(ctx context.Context, delay time.Duration) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		s.Run(ctx)
	}()
	return func() {
		cancel()
		<-done
	}
}
