package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/tmdb"
)

// sub-7-2b AC #2 — a TMDb SEARCH hit carries no genres and no countries (the
// provider never mapped genre_ids), so every title matched through the normal
// scan path translated with `Genres: []` and no countries. After the match,
// one details call fills both. These tests pin that the call happens exactly
// when it should, and that its failure never costs the match itself.

// facetsTMDb is mockTMDbServiceForNFO plus a configurable TV details answer
// and call counters.
type facetsTMDb struct {
	mockTMDbServiceForNFO
	tvDetails  *tmdb.TVShowDetails
	tvErr      error
	movieCalls int
	tvCalls    int
}

func (f *facetsTMDb) GetMovieDetails(ctx context.Context, id int) (*tmdb.MovieDetails, error) {
	f.movieCalls++
	return f.mockTMDbServiceForNFO.GetMovieDetails(ctx, id)
}

func (f *facetsTMDb) GetTVShowDetails(_ context.Context, _ int) (*tmdb.TVShowDetails, error) {
	f.tvCalls++
	if f.tvErr != nil {
		return nil, f.tvErr
	}
	if f.tvDetails == nil {
		return &tmdb.TVShowDetails{}, nil
	}
	return f.tvDetails, nil
}

func fightClubSearch(source models.MetadataSource) *mockPQMetadataService {
	return &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: source,
		Items:  []metadata.MetadataItem{{ID: "550", Title: "Fight Club", TitleZhTW: "鬥陣俱樂部"}},
	}}
}

func fightClubParser() *mockPQParserService {
	return &mockPQParserService{result: &parser.ParseResult{
		Status: parser.ParseStatusSuccess, MediaType: parser.MediaTypeMovie, CleanedTitle: "Fight Club", Year: 1999,
	}}
}

func TestEnrichMovie_SearchPath_FillsGenresAndCountriesFromDetails(t *testing.T) {
	repo := &mockMovieRepoForNFO{}
	tmdbFake := &facetsTMDb{mockTMDbServiceForNFO: mockTMDbServiceForNFO{getMovieDetailsResp: &tmdb.MovieDetails{
		Movie:               tmdb.Movie{ID: 550, Title: "鬥陣俱樂部"},
		Genres:              []tmdb.Genre{{ID: 18, Name: "劇情"}, {ID: 53, Name: "驚悚"}},
		ProductionCountries: []tmdb.Country{{ISO31661: "US", Name: "United States of America"}, {ISO31661: "DE", Name: "Germany"}},
	}}}
	svc := NewEnrichmentService(repo, fightClubParser(), fightClubSearch(models.MetadataSourceTMDb), nil, tmdbFake, nil, nil, nil)

	require.NoError(t, svc.enrichMovie(context.Background(), &models.Movie{ID: "movie-1", Title: "Fight.Club.1999.mkv"}))

	require.NotNil(t, repo.updatedMovie)
	assert.Equal(t, 1, tmdbFake.movieCalls)
	assert.Equal(t, []string{"劇情", "驚悚"}, repo.updatedMovie.Genres, "genres come from details — the search hit has none")
	countries, err := repo.updatedMovie.GetProductionCountries()
	require.NoError(t, err)
	assert.Equal(t, []models.ProductionCountry{{ISO3166_1: "US", Name: "United States of America"}, {ISO3166_1: "DE", Name: "Germany"}}, countries)
	assert.Equal(t, models.ParseStatusSuccess, repo.updatedMovie.ParseStatus)
}

func TestEnrichMovie_SearchPath_DetailsFailureStillLandsTheMatch(t *testing.T) {
	repo := &mockMovieRepoForNFO{}
	tmdbFake := &facetsTMDb{mockTMDbServiceForNFO: mockTMDbServiceForNFO{getMovieDetailsErr: errors.New("TMDB_TIMEOUT")}}
	svc := NewEnrichmentService(repo, fightClubParser(), fightClubSearch(models.MetadataSourceTMDb), nil, tmdbFake, nil, nil, nil)

	require.NoError(t, svc.enrichMovie(context.Background(), &models.Movie{ID: "movie-1", Title: "Fight.Club.1999.mkv"}))

	require.NotNil(t, repo.updatedMovie, "the match is persisted; the details call is context, not a gate")
	assert.Equal(t, "鬥陣俱樂部", repo.updatedMovie.Title)
	assert.Equal(t, models.ParseStatusSuccess, repo.updatedMovie.ParseStatus)
	assert.Empty(t, repo.updatedMovie.Genres)
	assert.False(t, repo.updatedMovie.ProductionCountriesJSON.Valid)
}

func TestEnrichMovie_SearchPath_NonTMDbMatchMakesNoDetailsCall(t *testing.T) {
	repo := &mockMovieRepoForNFO{}
	tmdbFake := &facetsTMDb{}
	svc := NewEnrichmentService(repo, fightClubParser(), fightClubSearch(models.MetadataSourceDouban), nil, tmdbFake, nil, nil, nil)

	require.NoError(t, svc.enrichMovie(context.Background(), &models.Movie{ID: "movie-1", Title: "Fight.Club.1999.mkv"}))

	require.NotNil(t, repo.updatedMovie)
	assert.Equal(t, 0, tmdbFake.movieCalls, "a Douban subject id is numeric too — it must never turn into a TMDb details call")
}

func TestEnrichMovie_SearchPath_NoTMDbServiceIsStillFine(t *testing.T) {
	repo := &mockMovieRepoForNFO{}
	svc := NewEnrichmentService(repo, fightClubParser(), fightClubSearch(models.MetadataSourceTMDb), nil, nil, nil, nil, nil)

	require.NoError(t, svc.enrichMovie(context.Background(), &models.Movie{ID: "movie-1", Title: "Fight.Club.1999.mkv"}))
	require.NotNil(t, repo.updatedMovie)
}

func breakingBadSearch() *mockPQMetadataService {
	return &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceTMDb,
		Items:  []metadata.MetadataItem{{ID: "1396", Title: "Breaking Bad", TitleZhTW: "絕命毒師"}},
	}}
}

func TestEnrichSeries_FillsGenresAndCountriesFromDetails(t *testing.T) {
	var events []string
	seriesRepo := &recordingSeriesRepo{events: &events}
	tmdbFake := &facetsTMDb{tvDetails: &tmdb.TVShowDetails{
		TVShow:              tmdb.TVShow{ID: 1396, Name: "絕命毒師", OriginCountry: []string{"US"}},
		Genres:              []tmdb.Genre{{ID: 18, Name: "劇情"}, {ID: 80, Name: "犯罪"}},
		ProductionCountries: []tmdb.Country{{ISO31661: "US", Name: "United States of America"}},
	}}
	svc := NewEnrichmentService(&mockMovieRepoForNFO{}, &mockPQParserService{result: &parser.ParseResult{CleanedTitle: "Breaking Bad"}},
		breakingBadSearch(), nil, tmdbFake, nil, nil, nil)
	svc.SetSeriesRepo(seriesRepo)

	require.NoError(t, svc.enrichSeries(context.Background(), &models.Series{ID: "series-1", Title: "Breaking Bad"}))

	require.NotNil(t, seriesRepo.updated)
	assert.Equal(t, 1, tmdbFake.tvCalls)
	assert.Equal(t, []string{"劇情", "犯罪"}, seriesRepo.updated.Genres)
	assert.Equal(t, []models.ProductionCountry{{ISO3166_1: "US", Name: "United States of America"}}, seriesRepo.updated.ProductionCountries)
	assert.True(t, seriesRepo.updated.ProductionCountriesJSON.Valid, "the JSON column is what the repository writes")
}

// TMDb's TV details frequently ship an empty production_countries while
// origin_country is set (e.g. most CN dramas). That is exactly the case
// sub-7-4's CN rule needs, so the fallback is not optional.
func TestEnrichSeries_FallsBackToOriginCountry(t *testing.T) {
	var events []string
	seriesRepo := &recordingSeriesRepo{events: &events}
	tmdbFake := &facetsTMDb{tvDetails: &tmdb.TVShowDetails{
		TVShow: tmdb.TVShow{ID: 1396, Name: "慶餘年", OriginCountry: []string{"CN"}},
	}}
	svc := NewEnrichmentService(&mockMovieRepoForNFO{}, &mockPQParserService{result: &parser.ParseResult{CleanedTitle: "Joy of Life"}},
		breakingBadSearch(), nil, tmdbFake, nil, nil, nil)
	svc.SetSeriesRepo(seriesRepo)

	require.NoError(t, svc.enrichSeries(context.Background(), &models.Series{ID: "series-1", Title: "Joy of Life"}))

	require.NotNil(t, seriesRepo.updated)
	assert.Equal(t, []models.ProductionCountry{{ISO3166_1: "CN"}}, seriesRepo.updated.ProductionCountries)
}

func TestEnrichSeries_DetailsFailureStillLandsTheMatch(t *testing.T) {
	var events []string
	seriesRepo := &recordingSeriesRepo{events: &events}
	tmdbFake := &facetsTMDb{tvErr: errors.New("TMDB_RATE_LIMIT")}
	svc := NewEnrichmentService(&mockMovieRepoForNFO{}, &mockPQParserService{result: &parser.ParseResult{CleanedTitle: "Breaking Bad"}},
		breakingBadSearch(), nil, tmdbFake, nil, nil, nil)
	svc.SetSeriesRepo(seriesRepo)

	require.NoError(t, svc.enrichSeries(context.Background(), &models.Series{ID: "series-1", Title: "Breaking Bad"}))

	require.NotNil(t, seriesRepo.updated)
	assert.Equal(t, "絕命毒師", seriesRepo.updated.Title)
	assert.Equal(t, models.ParseStatusSuccess, seriesRepo.updated.ParseStatus)
	assert.False(t, seriesRepo.updated.ProductionCountriesJSON.Valid)
}

func TestSeriesCountriesFromDetails(t *testing.T) {
	assert.Nil(t, seriesCountriesFromDetails(nil))
	assert.Nil(t, seriesCountriesFromDetails(&tmdb.TVShowDetails{}))
	assert.Equal(t, []models.ProductionCountry{{ISO3166_1: "KR"}},
		seriesCountriesFromDetails(&tmdb.TVShowDetails{TVShow: tmdb.TVShow{OriginCountry: []string{"", "KR"}}}))
	assert.Equal(t, []models.ProductionCountry{{ISO3166_1: "JP", Name: "Japan"}},
		seriesCountriesFromDetails(&tmdb.TVShowDetails{
			TVShow:              tmdb.TVShow{OriginCountry: []string{"KR"}},
			ProductionCountries: []tmdb.Country{{ISO31661: "JP", Name: "Japan"}, {ISO31661: ""}},
		}), "production_countries wins when present; blank codes are dropped")
}
