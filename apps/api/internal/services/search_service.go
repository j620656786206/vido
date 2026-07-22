package services

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// SearchTMDbClient is the narrow slice of *tmdb.Client that SearchService needs.
// It is defined here (not in the tmdb package) and intentionally kept separate
// from tmdb.ClientInterface so adding person search does not force every existing
// ClientInterface mock to grow a method. *tmdb.Client satisfies this interface.
type SearchTMDbClient interface {
	SearchMoviesWithLanguage(ctx context.Context, query string, language string, page int) (*tmdb.SearchResultMovies, error)
	SearchTVShowsWithLanguage(ctx context.Context, query string, language string, page int) (*tmdb.SearchResultTVShows, error)
	// SearchPeopleWithLanguage is used (not the language-defaulting SearchPeople)
	// so person results honor the same zh-TW priority as movies/TV instead of the
	// client's configured default language (Story 11-3 — zh-TW priority).
	SearchPeopleWithLanguage(ctx context.Context, query string, language string, page int) (*tmdb.SearchResultPeople, error)
}

// LocalLibrarySearcher is the narrow slice of *LibraryService the unified
// search needs to surface OWNED items in the dropdown. Owned results survive
// TMDb outages (missing API key, network) — the data is local.
type LocalLibrarySearcher interface {
	SearchLibrary(ctx context.Context, query string, params repository.ListParams, mediaType string) (*LibrarySearchResults, error)
}

// LocalSearchHit is an owned library item surfaced by unified search. ID is the
// LOCAL id (UUID-style), so the frontend routes it to the local detail view —
// which renders without any TMDb dependency.
type LocalSearchHit struct {
	ID            string `json:"id"`
	MediaType     string `json:"media_type"` // "movie" | "tv"
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title,omitempty"`
	ReleaseDate   string `json:"release_date,omitempty"`
	PosterPath    string `json:"poster_path,omitempty"`
}

// UnifiedSearchResult is the response payload of the unified instant-search
// endpoint (GET /api/v1/search). It carries the owned-library section plus the
// three TMDb categories rendered as separate sections in the suggestions
// dropdown: 媒體庫 / 電影 / 影集 / 人物.
type UnifiedSearchResult struct {
	Query       string           `json:"query"`
	Page        int              `json:"page"`
	LocalMovies []LocalSearchHit `json:"local_movies"`
	LocalTV     []LocalSearchHit `json:"local_tv"`
	Movies      []tmdb.Movie     `json:"movies"`
	TVShows     []tmdb.TVShow    `json:"tv_shows"`
	People      []tmdb.Person    `json:"people"`
}

// SearchServiceInterface defines the contract for unified multi-category search.
type SearchServiceInterface interface {
	// Search runs a unified, dual-language (zh-TW + en) search across movies,
	// TV shows, and people, merging/deduplicating by TMDb ID and boosting
	// zh-TW title matches to the top of each media list.
	Search(ctx context.Context, query string, page int) (*UnifiedSearchResult, error)
}

const (
	// searchPrimaryLanguage is the localized language whose titles drive the
	// zh-TW boost ranking (AC #2) and supply preferred metadata on merge (AC #3).
	searchPrimaryLanguage = "zh-TW"
	// searchSecondaryLanguage is queried simultaneously so original-language-only
	// titles still surface when no zh-TW localization exists (AC #3).
	searchSecondaryLanguage = "en"
)

// SearchService implements SearchServiceInterface using the TMDb client directly.
// It bypasses the language-fallback chain because the unified search must query
// BOTH languages simultaneously and merge — the fallback chain instead returns
// the first language that yields localized content (single-language semantics).
type SearchService struct {
	client SearchTMDbClient
	local  LocalLibrarySearcher
}

// Compile-time interface verification.
var _ SearchServiceInterface = (*SearchService)(nil)

// localSearchLimit caps how many owned items each media type contributes to the
// dropdown — it is a suggestions panel, not a browse surface.
const localSearchLimit = 5

// NewSearchService creates a SearchService backed by the given TMDb client and
// local library searcher (nil disables the owned-library section).
func NewSearchService(client SearchTMDbClient, local LocalLibrarySearcher) *SearchService {
	return &SearchService{client: client, local: local}
}

// Search performs the unified dual-language search. The five underlying TMDb
// calls (zh-TW + en for movies and TV, plus people) and the local-library call
// run concurrently; the TMDb client's own rate limiter serializes its calls
// safely. A failure in one category degrades that category to empty rather than
// failing the whole request — the call errors only when EVERY leg (TMDb AND
// local) failed. In particular, a dead/unconfigured TMDb degrades the dropdown
// to owned-library results instead of an error (testsprite-round1 TC092).
func (s *SearchService) Search(ctx context.Context, query string, page int) (*UnifiedSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, tmdb.NewBadRequestError("search query cannot be empty")
	}
	if page < 1 {
		page = 1
	}

	var (
		wg                          sync.WaitGroup
		zhMovies, enMovies          *tmdb.SearchResultMovies
		zhTV, enTV                  *tmdb.SearchResultTVShows
		people                      *tmdb.SearchResultPeople
		localRes                    *LibrarySearchResults
		zhMovieErr, enMovieErr      error
		zhTVErr, enTVErr, peopleErr error
		localErr                    error
	)

	run := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	run(func() {
		zhMovies, zhMovieErr = s.client.SearchMoviesWithLanguage(ctx, query, searchPrimaryLanguage, page)
	})
	run(func() {
		enMovies, enMovieErr = s.client.SearchMoviesWithLanguage(ctx, query, searchSecondaryLanguage, page)
	})
	run(func() { zhTV, zhTVErr = s.client.SearchTVShowsWithLanguage(ctx, query, searchPrimaryLanguage, page) })
	run(func() { enTV, enTVErr = s.client.SearchTVShowsWithLanguage(ctx, query, searchSecondaryLanguage, page) })
	run(func() {
		people, peopleErr = s.client.SearchPeopleWithLanguage(ctx, query, searchPrimaryLanguage, page)
	})
	if s.local != nil {
		run(func() {
			localRes, localErr = s.local.SearchLibrary(ctx, query,
				repository.ListParams{Page: 1, PageSize: localSearchLimit}, "all")
		})
	}
	wg.Wait()

	// Per-category degradation: log a warning for any failed call but keep going
	// with whatever succeeded. Only bail out if literally everything failed.
	errs := []error{zhMovieErr, enMovieErr, zhTVErr, enTVErr, peopleErr}
	if s.local != nil {
		errs = append(errs, localErr)
	}
	var firstErr error
	failed := 0
	for _, err := range errs {
		if err != nil {
			failed++
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if failed == len(errs) {
		slog.Error("Unified search failed across all categories", "query", query, "error", firstErr)
		return nil, firstErr
	}
	if failed > 0 {
		slog.Warn("Unified search partial failure (degraded category to empty)",
			"query", query,
			"failed_calls", failed,
			"first_error", firstErr,
		)
	}

	localMovies, localTV := collectLocalHits(localRes)
	result := &UnifiedSearchResult{
		Query:       query,
		Page:        page,
		LocalMovies: localMovies,
		LocalTV:     localTV,
		Movies:      mergeMovies(zhMovies, enMovies, query),
		TVShows:     mergeTVShows(zhTV, enTV, query),
		People:      collectPeople(people),
	}

	slog.Debug("Unified search completed",
		"query", query,
		"local_movies", len(result.LocalMovies),
		"local_tv", len(result.LocalTV),
		"movies", len(result.Movies),
		"tv_shows", len(result.TVShows),
		"people", len(result.People),
	)

	return result, nil
}

// collectLocalHits maps the library search results onto dropdown hits, keyed by
// LOCAL ids so the detail navigation stays TMDb-independent. A nil result (local
// search disabled or failed) yields empty non-nil slices, keeping the JSON
// arrays stable for the frontend.
func collectLocalHits(res *LibrarySearchResults) (movies, tv []LocalSearchHit) {
	movies, tv = []LocalSearchHit{}, []LocalSearchHit{}
	if res == nil {
		return movies, tv
	}
	for _, r := range res.Results {
		switch {
		case r.Movie != nil:
			movies = append(movies, LocalSearchHit{
				ID:            r.Movie.ID,
				MediaType:     "movie",
				Title:         r.Movie.Title,
				OriginalTitle: r.Movie.OriginalTitle.String,
				ReleaseDate:   r.Movie.ReleaseDate,
				PosterPath:    r.Movie.PosterPath.String,
			})
		case r.Series != nil:
			tv = append(tv, LocalSearchHit{
				ID:            r.Series.ID,
				MediaType:     "tv",
				Title:         r.Series.Title,
				OriginalTitle: r.Series.OriginalTitle.String,
				ReleaseDate:   r.Series.FirstAirDate,
				PosterPath:    r.Series.PosterPath.String,
			})
		}
	}
	return movies, tv
}

// mergeMovies merges the zh-TW and en movie result sets, deduplicating by TMDb
// ID (zh-TW metadata wins — consumed first, AC #3) and boosting items whose
// zh-TW localized title matches the query above original-title-only matches
// (AC #2). Both inputs are nil-safe.
func mergeMovies(zh, en *tmdb.SearchResultMovies, query string) []tmdb.Movie {
	seen := make(map[int]struct{})
	matched := make([]tmdb.Movie, 0)
	rest := make([]tmdb.Movie, 0)

	consume := func(res *tmdb.SearchResultMovies, primary bool) {
		if res == nil {
			return
		}
		for _, m := range res.Results {
			if _, ok := seen[m.ID]; ok {
				continue
			}
			seen[m.ID] = struct{}{}
			if primary && containsFold(m.Title, query) {
				matched = append(matched, m)
			} else {
				rest = append(rest, m)
			}
		}
	}
	consume(zh, true)
	consume(en, false)

	out := make([]tmdb.Movie, 0, len(matched)+len(rest))
	out = append(out, matched...)
	out = append(out, rest...)
	return out
}

// mergeTVShows is the TV-show counterpart of mergeMovies (boosts on the
// localized Name field).
func mergeTVShows(zh, en *tmdb.SearchResultTVShows, query string) []tmdb.TVShow {
	seen := make(map[int]struct{})
	matched := make([]tmdb.TVShow, 0)
	rest := make([]tmdb.TVShow, 0)

	consume := func(res *tmdb.SearchResultTVShows, primary bool) {
		if res == nil {
			return
		}
		for _, t := range res.Results {
			if _, ok := seen[t.ID]; ok {
				continue
			}
			seen[t.ID] = struct{}{}
			if primary && containsFold(t.Name, query) {
				matched = append(matched, t)
			} else {
				rest = append(rest, t)
			}
		}
	}
	consume(zh, true)
	consume(en, false)

	out := make([]tmdb.TVShow, 0, len(matched)+len(rest))
	out = append(out, matched...)
	out = append(out, rest...)
	return out
}

// collectPeople returns a non-nil slice of the person results (people search is
// a single TMDb call — no dual-language merge needed).
func collectPeople(res *tmdb.SearchResultPeople) []tmdb.Person {
	if res == nil || len(res.Results) == 0 {
		return []tmdb.Person{}
	}
	return res.Results
}

// containsFold reports whether substr occurs within s, case-insensitively. For
// CJK queries this is a plain substring test; for Latin queries it folds case
// so "your name" matches a "Your Name" title.
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
