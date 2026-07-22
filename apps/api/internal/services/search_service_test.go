package services

import (
	"context"
	"errors"
	"testing"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// stubSearchClient is a configurable SearchTMDbClient for unit testing the
// dual-language merge + zh-TW boost logic without hitting the TMDb API.
type stubSearchClient struct {
	movies func(lang string) (*tmdb.SearchResultMovies, error)
	tv     func(lang string) (*tmdb.SearchResultTVShows, error)
	people func() (*tmdb.SearchResultPeople, error)
}

func (s *stubSearchClient) SearchMoviesWithLanguage(_ context.Context, _ string, lang string, _ int) (*tmdb.SearchResultMovies, error) {
	return s.movies(lang)
}

func (s *stubSearchClient) SearchTVShowsWithLanguage(_ context.Context, _ string, lang string, _ int) (*tmdb.SearchResultTVShows, error) {
	return s.tv(lang)
}

func (s *stubSearchClient) SearchPeopleWithLanguage(_ context.Context, _ string, _ string, _ int) (*tmdb.SearchResultPeople, error) {
	return s.people()
}

func movieRes(ms ...tmdb.Movie) *tmdb.SearchResultMovies {
	return &tmdb.SearchResultMovies{Page: 1, Results: ms, TotalResults: len(ms)}
}

func tvRes(ts ...tmdb.TVShow) *tmdb.SearchResultTVShows {
	return &tmdb.SearchResultTVShows{Page: 1, Results: ts, TotalResults: len(ts)}
}

// emptyStub returns a client whose every category succeeds with empty results.
func emptyStub() *stubSearchClient {
	return &stubSearchClient{
		movies: func(string) (*tmdb.SearchResultMovies, error) { return movieRes(), nil },
		tv:     func(string) (*tmdb.SearchResultTVShows, error) { return tvRes(), nil },
		people: func() (*tmdb.SearchResultPeople, error) { return &tmdb.SearchResultPeople{}, nil },
	}
}

func TestSearch_EmptyQuery_ReturnsBadRequest(t *testing.T) {
	svc := NewSearchService(emptyStub(), nil)

	_, err := svc.Search(context.Background(), "   ", 1)
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
	var tErr *tmdb.TMDbError
	if !errors.As(err, &tErr) {
		t.Fatalf("expected *tmdb.TMDbError, got %T", err)
	}
}

func TestSearch_DualLanguageMergeDedup(t *testing.T) {
	// zh-TW returns id=1 (localized title); en returns id=1 (English title) + id=2.
	// Expect: id=1 kept once with zh-TW metadata, id=2 appended from en.
	stub := emptyStub()
	stub.movies = func(lang string) (*tmdb.SearchResultMovies, error) {
		if lang == "zh-TW" {
			return movieRes(tmdb.Movie{ID: 1, Title: "鋼鐵人"}), nil
		}
		return movieRes(
			tmdb.Movie{ID: 1, Title: "Iron Man"},
			tmdb.Movie{ID: 2, Title: "Iron Man 2"},
		), nil
	}

	got, err := NewSearchService(stub, nil).Search(context.Background(), "iron", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Movies) != 2 {
		t.Fatalf("expected 2 deduped movies, got %d: %+v", len(got.Movies), got.Movies)
	}
	// id=1 must carry the zh-TW title (zh-TW metadata wins on merge — AC #3).
	var m1 *tmdb.Movie
	for i := range got.Movies {
		if got.Movies[i].ID == 1 {
			m1 = &got.Movies[i]
		}
	}
	if m1 == nil || m1.Title != "鋼鐵人" {
		t.Fatalf("expected id=1 to keep zh-TW title 鋼鐵人, got %+v", m1)
	}
}

func TestSearch_ZhTitleMatchBoostedAboveOriginalOnly(t *testing.T) {
	// Query matches a zh-TW title. The matching item must rank above a
	// non-matching item even though it appears later in the raw zh-TW list (AC #2).
	stub := emptyStub()
	stub.movies = func(lang string) (*tmdb.SearchResultMovies, error) {
		if lang == "zh-TW" {
			return movieRes(
				tmdb.Movie{ID: 10, Title: "不相關電影"}, // no query match
				tmdb.Movie{ID: 11, Title: "你的名字"},  // zh-TW title match → boost
			), nil
		}
		return movieRes(), nil
	}

	got, err := NewSearchService(stub, nil).Search(context.Background(), "你的名字", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Movies) != 2 {
		t.Fatalf("expected 2 movies, got %d", len(got.Movies))
	}
	if got.Movies[0].ID != 11 {
		t.Fatalf("expected zh-TW-title-match id=11 ranked first, got order %d,%d",
			got.Movies[0].ID, got.Movies[1].ID)
	}
}

func TestSearch_TVDualLanguageBoost(t *testing.T) {
	stub := emptyStub()
	stub.tv = func(lang string) (*tmdb.SearchResultTVShows, error) {
		if lang == "zh-TW" {
			return tvRes(
				tmdb.TVShow{ID: 20, Name: "其他影集"},
				tmdb.TVShow{ID: 21, Name: "進擊的巨人"},
			), nil
		}
		return tvRes(tmdb.TVShow{ID: 22, Name: "Attack on Titan"}), nil
	}

	got, err := NewSearchService(stub, nil).Search(context.Background(), "進擊的巨人", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.TVShows) != 3 {
		t.Fatalf("expected 3 tv shows, got %d", len(got.TVShows))
	}
	if got.TVShows[0].ID != 21 {
		t.Fatalf("expected boosted id=21 first, got %d", got.TVShows[0].ID)
	}
}

func TestSearch_PeopleCategoryIncluded(t *testing.T) {
	stub := emptyStub()
	stub.people = func() (*tmdb.SearchResultPeople, error) {
		return &tmdb.SearchResultPeople{
			Results: []tmdb.Person{{ID: 5655, Name: "Makoto Shinkai", OriginalName: "新海誠", KnownForDepartment: "Directing"}},
		}, nil
	}

	got, err := NewSearchService(stub, nil).Search(context.Background(), "shinkai", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.People) != 1 || got.People[0].ID != 5655 {
		t.Fatalf("expected 1 person id=5655, got %+v", got.People)
	}
}

func TestSearch_PartialFailureDegradesCategory(t *testing.T) {
	// Both movie-language calls fail, but TV + people succeed: the request must
	// still succeed with an empty Movies list (graceful degradation).
	stub := emptyStub()
	stub.movies = func(string) (*tmdb.SearchResultMovies, error) { return nil, errors.New("tmdb 500") }
	stub.tv = func(string) (*tmdb.SearchResultTVShows, error) {
		return tvRes(tmdb.TVShow{ID: 30, Name: "Some Show"}), nil
	}

	got, err := NewSearchService(stub, nil).Search(context.Background(), "show", 1)
	if err != nil {
		t.Fatalf("expected graceful degradation (nil error), got %v", err)
	}
	if len(got.Movies) != 0 {
		t.Fatalf("expected empty movies on failure, got %d", len(got.Movies))
	}
	if got.Movies == nil || got.People == nil {
		t.Fatal("expected non-nil empty slices for JSON [] rendering")
	}
	if len(got.TVShows) != 1 {
		t.Fatalf("expected tv category to survive, got %d", len(got.TVShows))
	}
}

func TestSearch_AllCategoriesFail_ReturnsError(t *testing.T) {
	boom := errors.New("network down")
	stub := &stubSearchClient{
		movies: func(string) (*tmdb.SearchResultMovies, error) { return nil, boom },
		tv:     func(string) (*tmdb.SearchResultTVShows, error) { return nil, boom },
		people: func() (*tmdb.SearchResultPeople, error) { return nil, boom },
	}

	_, err := NewSearchService(stub, nil).Search(context.Background(), "anything", 1)
	if err == nil {
		t.Fatal("expected error when all categories fail, got nil")
	}
}

func TestSearch_EmptyResults_NonNilSlices(t *testing.T) {
	got, err := NewSearchService(emptyStub(), nil).Search(context.Background(), "zzz", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Page != 1 {
		t.Fatalf("expected page normalized to 1, got %d", got.Page)
	}
	if got.Movies == nil || got.TVShows == nil || got.People == nil {
		t.Fatal("expected non-nil empty slices so JSON renders [] not null")
	}
}

// stubLocalSearcher is a configurable LocalLibrarySearcher for the owned-library
// leg (testsprite-round1 TC092 regression lock).
type stubLocalSearcher struct {
	res *LibrarySearchResults
	err error
}

func (s *stubLocalSearcher) SearchLibrary(_ context.Context, _ string, _ repository.ListParams, _ string) (*LibrarySearchResults, error) {
	return s.res, s.err
}

func localFixture() *LibrarySearchResults {
	return &LibrarySearchResults{
		Results: []SearchResult{
			{Type: "movie", Movie: &models.Movie{
				ID: "seed-mv-003", Title: "駭客任務",
				OriginalTitle: models.NewNullString("The Matrix"),
				ReleaseDate:   "1999-03-31",
				PosterPath:    models.NewNullString("/matrix.jpg"),
			}},
			{Type: "series", Series: &models.Series{
				ID: "seed-sr-002", Title: "怪奇物語",
				FirstAirDate: "2016-07-15",
			}},
		},
		TotalCount: 2,
	}
}

// The TC092 shape: TMDb entirely unreachable (no API key) must NOT kill the
// dropdown — owned-library results still come back with a 200.
func TestSearch_TMDbDown_LocalResultsSurvive(t *testing.T) {
	boom := errors.New("TMDB_UNAUTHORIZED")
	stub := &stubSearchClient{
		movies: func(string) (*tmdb.SearchResultMovies, error) { return nil, boom },
		tv:     func(string) (*tmdb.SearchResultTVShows, error) { return nil, boom },
		people: func() (*tmdb.SearchResultPeople, error) { return nil, boom },
	}
	local := &stubLocalSearcher{res: localFixture()}

	got, err := NewSearchService(stub, local).Search(context.Background(), "駭客", 1)
	if err != nil {
		t.Fatalf("expected local-only degradation, got error: %v", err)
	}
	if len(got.LocalMovies) != 1 || got.LocalMovies[0].ID != "seed-mv-003" {
		t.Fatalf("expected the owned movie hit with its LOCAL id, got %+v", got.LocalMovies)
	}
	if got.LocalMovies[0].MediaType != "movie" || got.LocalMovies[0].ReleaseDate != "1999-03-31" {
		t.Fatalf("bad local movie mapping: %+v", got.LocalMovies[0])
	}
	if len(got.LocalTV) != 1 || got.LocalTV[0].ID != "seed-sr-002" || got.LocalTV[0].MediaType != "tv" {
		t.Fatalf("expected the owned series hit, got %+v", got.LocalTV)
	}
	if got.LocalTV[0].ReleaseDate != "2016-07-15" {
		t.Fatalf("series first_air_date must map to release_date, got %+v", got.LocalTV[0])
	}
	if len(got.Movies) != 0 || len(got.TVShows) != 0 {
		t.Fatalf("TMDb categories should degrade to empty, got %d/%d", len(got.Movies), len(got.TVShows))
	}
}

func TestSearch_EveryLegFails_IncludingLocal_ReturnsError(t *testing.T) {
	boom := errors.New("network down")
	stub := &stubSearchClient{
		movies: func(string) (*tmdb.SearchResultMovies, error) { return nil, boom },
		tv:     func(string) (*tmdb.SearchResultTVShows, error) { return nil, boom },
		people: func() (*tmdb.SearchResultPeople, error) { return nil, boom },
	}
	local := &stubLocalSearcher{err: errors.New("db locked")}

	if _, err := NewSearchService(stub, local).Search(context.Background(), "anything", 1); err == nil {
		t.Fatal("expected error when every leg including local fails")
	}
}

func TestSearch_LocalAndTMDbBothSucceed_BothSectionsPresent(t *testing.T) {
	stub := emptyStub()
	stub.movies = func(string) (*tmdb.SearchResultMovies, error) {
		return movieRes(tmdb.Movie{ID: 603, Title: "The Matrix"}), nil
	}
	local := &stubLocalSearcher{res: localFixture()}

	got, err := NewSearchService(stub, local).Search(context.Background(), "matrix", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.LocalMovies) != 1 || len(got.Movies) != 1 {
		t.Fatalf("expected both owned and TMDb sections, got local=%d tmdb=%d", len(got.LocalMovies), len(got.Movies))
	}
}

func TestSearch_NilLocalSearcher_LocalSectionsEmptyNotNil(t *testing.T) {
	got, err := NewSearchService(emptyStub(), nil).Search(context.Background(), "zzz", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.LocalMovies == nil || got.LocalTV == nil {
		t.Fatal("local sections must be [] not null when the local leg is disabled")
	}
}
