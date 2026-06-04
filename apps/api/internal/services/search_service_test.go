package services

import (
	"context"
	"errors"
	"testing"

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
	svc := NewSearchService(emptyStub())

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

	got, err := NewSearchService(stub).Search(context.Background(), "iron", 1)
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

	got, err := NewSearchService(stub).Search(context.Background(), "你的名字", 1)
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

	got, err := NewSearchService(stub).Search(context.Background(), "進擊的巨人", 1)
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

	got, err := NewSearchService(stub).Search(context.Background(), "shinkai", 1)
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

	got, err := NewSearchService(stub).Search(context.Background(), "show", 1)
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

	_, err := NewSearchService(stub).Search(context.Background(), "anything", 1)
	if err == nil {
		t.Fatal("expected error when all categories fail, got nil")
	}
}

func TestSearch_EmptyResults_NonNilSlices(t *testing.T) {
	got, err := NewSearchService(emptyStub()).Search(context.Background(), "zzz", 0)
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
