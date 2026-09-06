package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/tmdb"
)

// sub-7-3 AC #1 / #6(f): the two enrichment call points (movie, series) hand
// the matched TMDb id to the credits fetcher, persist the cast through the
// credits-only writer right after the row write, and then resolve the row's
// glossary scope once — Resolve is where the drawer is seeded
// (seed-on-first-resolve), so the resolver must be called AFTER the row is
// written (it needs to see tmdb_id).

type fakeGlossarySeeder struct {
	credits   *models.Credits
	pairs     []CastPair
	fetchErr  error
	fetchArgs []string // "movie/550"
	seedArgs  []string // "scope|mediaID|npairs"
	events    *[]string
}

func (f *fakeGlossarySeeder) FetchCredits(_ context.Context, mediaType string, tmdbID int64) (*models.Credits, []CastPair, error) {
	f.fetchArgs = append(f.fetchArgs, mediaType+"/"+itoa64(tmdbID))
	if f.events != nil {
		*f.events = append(*f.events, "fetch")
	}
	if f.fetchErr != nil {
		return nil, nil, f.fetchErr
	}
	return f.credits, f.pairs, nil
}

func (f *fakeGlossarySeeder) SeedFromCredits(_ context.Context, scope, mediaID string, pairs []CastPair) SeedResult {
	f.seedArgs = append(f.seedArgs, scope+"|"+mediaID+"|"+itoa64(int64(len(pairs))))
	if f.events != nil {
		*f.events = append(*f.events, "seed")
	}
	return SeedResult{Seeded: len(pairs)}
}

type fakeScopeResolver struct {
	scope  string
	err    error
	calls  []string
	events *[]string
}

func (f *fakeScopeResolver) Resolve(_ context.Context, mediaID string) (string, error) {
	f.calls = append(f.calls, mediaID)
	if f.events != nil {
		*f.events = append(*f.events, "resolve")
	}
	return f.scope, f.err
}

// recordingMovieRepo wraps the NFO mock so the test can see WHEN the write
// happened relative to the seed call.
type recordingMovieRepo struct {
	mockMovieRepoForNFO
	events *[]string
}

func (m *recordingMovieRepo) UpdateEnrichedMetadata(ctx context.Context, movie *models.Movie) error {
	*m.events = append(*m.events, "write")
	return m.mockMovieRepoForNFO.UpdateEnrichedMetadata(ctx, movie)
}

func (m *recordingMovieRepo) UpdateCredits(ctx context.Context, id string, credits *models.Credits) error {
	*m.events = append(*m.events, "credits")
	return m.mockMovieRepoForNFO.UpdateCredits(ctx, id, credits)
}

type recordingSeriesRepo struct {
	mockPQSeriesRepo
	updated       *models.Series
	creditsWrites []string
	lastCredits   *models.Credits
	events        *[]string
}

func (m *recordingSeriesRepo) UpdateEnrichedMetadata(_ context.Context, s *models.Series) error {
	m.updated = s
	*m.events = append(*m.events, "write")
	return nil
}

func (m *recordingSeriesRepo) UpdateCredits(_ context.Context, id string, credits *models.Credits) error {
	m.creditsWrites = append(m.creditsWrites, id)
	m.lastCredits = credits
	*m.events = append(*m.events, "credits")
	return nil
}

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

var seedTestCredits = &models.Credits{Cast: []models.CastMember{{ID: 17419, Name: "布萊恩·克蘭斯頓", Character: "華特·懷特"}}}
var seedTestPairs = []CastPair{{Kind: "character", Src: "Walter White", Zh: "華特·懷特"}}

func writeSeedTestNFO(t *testing.T, tmdbID string) *models.Movie {
	t.Helper()
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.2004.mkv")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.2004.nfo"),
		[]byte(`<movie><title>功夫</title><uniqueid type="tmdb">`+tmdbID+`</uniqueid></movie>`), 0o644))
	return &models.Movie{ID: "movie-1", Title: "Movie.2004.mkv", FilePath: models.NewNullString(videoPath)}
}

func TestEnrichMovie_NFOPath_PersistsCreditsAndSeedsAfterWrite(t *testing.T) {
	var events []string
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs, events: &events}
	scopes := &fakeScopeResolver{scope: "tmdb:movie:10196", events: &events}
	repo := &recordingMovieRepo{events: &events}
	mockTMDb := &mockTMDbServiceForNFO{getMovieDetailsResp: &tmdb.MovieDetails{Movie: tmdb.Movie{ID: 10196, Title: "功夫"}}}

	svc := NewEnrichmentService(repo, nil, nil, NewNFOReaderService(nil), mockTMDb, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, scopes)

	movie := writeSeedTestNFO(t, "10196")
	require.NoError(t, svc.enrichMovie(context.Background(), movie))

	assert.Equal(t, []string{"movie/10196"}, seeder.fetchArgs)
	assert.Empty(t, seeder.seedArgs, "enrichment never seeds directly — the resolver does, on first sight of the scope")
	assert.Equal(t, []string{"movie-1"}, scopes.calls, "the row's scope is resolved once, keyed by the LOCAL id")
	assert.Equal(t, []string{"fetch", "write", "credits", "resolve"}, events,
		"row write first, then the credits-only writer, then the resolve that seeds")

	// The zh-TW cast goes through the narrow credits writer, never the wide copy.
	assert.Equal(t, []string{"movie-1"}, repo.creditsWrites)
	require.NotNil(t, repo.lastCredits)
	require.Len(t, repo.lastCredits.Cast, 1)
	assert.Equal(t, "布萊恩·克蘭斯頓", repo.lastCredits.Cast[0].Name)
	require.NotNil(t, repo.updatedMovie)
	assert.False(t, repo.updatedMovie.CreditsJSON.Valid, "credits are not smuggled through UpdateEnrichedMetadata")
	assert.Equal(t, int64(10196), repo.updatedMovie.TMDbID.Int64)
}

func TestEnrichMovie_SearchPath_PersistsCreditsEvenWithoutResolver(t *testing.T) {
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	repo := &mockMovieRepoForNFO{}
	parserSvc := &mockPQParserService{result: &parser.ParseResult{Status: parser.ParseStatusSuccess, MediaType: parser.MediaTypeMovie, CleanedTitle: "Fight Club", Year: 1999}}
	metaSvc := &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceTMDb,
		Items:  []metadata.MetadataItem{{ID: "550", Title: "Fight Club", TitleZhTW: "鬥陣俱樂部"}},
	}}
	svc := NewEnrichmentService(repo, parserSvc, metaSvc, nil, nil, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, nil)

	movie := &models.Movie{ID: "movie-2", Title: "Fight.Club.1999.mkv"}
	require.NoError(t, svc.enrichMovie(context.Background(), movie))

	assert.Equal(t, []string{"movie/550"}, seeder.fetchArgs)
	assert.Empty(t, seeder.seedArgs, "no resolver → nothing seeds (and nothing guesses a scope)")
	assert.Equal(t, []string{"movie-2"}, repo.creditsWrites)
	assert.Len(t, repo.lastCredits.Cast, 1)
}

func TestEnrichMovie_SearchPath_TVMatchFetchesAggregateCredits(t *testing.T) {
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	repo := &mockMovieRepoForNFO{}
	parserSvc := &mockPQParserService{result: &parser.ParseResult{Status: parser.ParseStatusSuccess, MediaType: parser.MediaTypeTVShow, CleanedTitle: "Breaking Bad"}}
	metaSvc := &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceTMDb,
		Items:  []metadata.MetadataItem{{ID: "1396", Title: "Breaking Bad"}},
	}}
	scopes := &fakeScopeResolver{scope: "tmdb:movie:1396"}
	svc := NewEnrichmentService(repo, parserSvc, metaSvc, nil, nil, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, scopes)

	require.NoError(t, svc.enrichMovie(context.Background(), &models.Movie{ID: "movie-3", Title: "Breaking.Bad.S01E01.mkv"}))
	assert.Equal(t, []string{"tv/1396"}, seeder.fetchArgs, "a TV match on a movie row still asks for the show's aggregate credits")
	assert.Equal(t, []string{"movie-3"}, repo.creditsWrites, "the cast is stored on the row")
	assert.Equal(t, []string{"movie-3"}, scopes.calls, "which drawer (if any) gets seeded is the resolver's call, not enrichment's")
}

func TestEnrichMovie_NumericNonTMDbIDNeverReachesTheSeeder(t *testing.T) {
	// applyMetadataToMovie stores ANY numeric provider id in TMDbID — a Douban
	// subject id is numeric — so the gate has to be on the SOURCE, not the id.
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	repo := &mockMovieRepoForNFO{}
	parserSvc := &mockPQParserService{result: &parser.ParseResult{Status: parser.ParseStatusSuccess, MediaType: parser.MediaTypeMovie, CleanedTitle: "讓子彈飛"}}
	metaSvc := &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceDouban,
		Items:  []metadata.MetadataItem{{ID: "1292052", TitleZhTW: "讓子彈飛"}},
	}}
	svc := NewEnrichmentService(repo, parserSvc, metaSvc, nil, nil, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, &fakeScopeResolver{scope: "tmdb:movie:1292052"})

	require.NoError(t, svc.enrichMovie(context.Background(), &models.Movie{ID: "movie-7", Title: "x.mkv"}))
	assert.Equal(t, int64(1292052), repo.updatedMovie.TMDbID.Int64, "pre-existing behaviour: the numeric id is stored")
	assert.Empty(t, seeder.fetchArgs, "…but it is not a TMDb id, so no credits call")
	assert.Empty(t, seeder.seedArgs)
	assert.Empty(t, repo.creditsWrites)
}

func TestEnrichMovie_NFOWithoutIDsDoesNotSeedFromStaleTMDbID(t *testing.T) {
	// A reparse of a WRONG earlier match keeps tmdb_id on the row; an NFO that
	// carries no ids must not turn that stale id into the wrong show's cast.
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	repo := &mockMovieRepoForNFO{}
	scopes := &fakeScopeResolver{scope: "tmdb:movie:999"}
	svc := NewEnrichmentService(repo, nil, nil, NewNFOReaderService(nil), &mockTMDbServiceForNFO{}, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, scopes)

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.nfo"), []byte(`<movie><title>功夫</title></movie>`), 0o644))
	movie := &models.Movie{ID: "movie-8", Title: "Movie.mkv", FilePath: models.NewNullString(videoPath), TMDbID: models.NewNullInt64(999)}

	require.NoError(t, svc.enrichMovie(context.Background(), movie))
	assert.Empty(t, seeder.fetchArgs, "no TMDb match in THIS pass → no credits call")
	assert.Empty(t, scopes.calls, "…and no resolve either — nothing new landed")
	assert.Equal(t, models.ParseStatusSuccess, repo.updatedMovie.ParseStatus, "NFO data still applied")
}

func TestEnrichMovie_NonTMDbMatchDoesNotTouchTheSeeder(t *testing.T) {
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	repo := &mockMovieRepoForNFO{}
	parserSvc := &mockPQParserService{result: &parser.ParseResult{Status: parser.ParseStatusSuccess, MediaType: parser.MediaTypeMovie, CleanedTitle: "讓子彈飛"}}
	metaSvc := &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceDouban,
		Items:  []metadata.MetadataItem{{ID: "douban-3742360", TitleZhTW: "讓子彈飛"}},
	}}
	svc := NewEnrichmentService(repo, parserSvc, metaSvc, nil, nil, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, &fakeScopeResolver{scope: "local:movie-4"})

	require.NoError(t, svc.enrichMovie(context.Background(), &models.Movie{ID: "movie-4", Title: "x.mkv"}))
	assert.Empty(t, seeder.fetchArgs, "no TMDb id → nothing to fetch")
	assert.Empty(t, seeder.seedArgs)
	assert.Equal(t, models.ParseStatusSuccess, repo.updatedMovie.ParseStatus)
}

func TestEnrichMovie_CreditsFetchFailureStillEnrichesTheRow(t *testing.T) {
	seeder := &fakeGlossarySeeder{fetchErr: errors.New("tmdb 500")}
	repo := &mockMovieRepoForNFO{}
	mockTMDb := &mockTMDbServiceForNFO{getMovieDetailsResp: &tmdb.MovieDetails{Movie: tmdb.Movie{ID: 10196, Title: "功夫"}}}
	svc := NewEnrichmentService(repo, nil, nil, NewNFOReaderService(nil), mockTMDb, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, &fakeScopeResolver{scope: "tmdb:movie:10196"})

	scopes := &fakeScopeResolver{scope: "tmdb:movie:10196"}
	svc.SetGlossarySeeder(seeder, scopes)
	require.NoError(t, svc.enrichMovie(context.Background(), writeSeedTestNFO(t, "10196")))
	assert.Equal(t, []string{"movie/10196"}, seeder.fetchArgs)
	require.NotNil(t, repo.updatedMovie, "the row is still written")
	assert.Equal(t, models.ParseStatusSuccess, repo.updatedMovie.ParseStatus)
	assert.Empty(t, repo.creditsWrites, "no credits stored")
	assert.Equal(t, []string{"movie-1"}, scopes.calls, "the resolve still happens — the seeder retries the fetch on its own schedule")
}

func TestEnrichMovie_ManualSourceOutranksMatch_CreditsKeptButGlossaryStillSeeds(t *testing.T) {
	// Ownership follows metadata_source, exactly like title/poster: a row the
	// user edited (source=manual) keeps its cast; the glossary is still seeded
	// because glossary terms have their own provenance and never overwrite.
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	repo := &mockMovieRepoForNFO{}
	parserSvc := &mockPQParserService{result: &parser.ParseResult{Status: parser.ParseStatusSuccess, MediaType: parser.MediaTypeMovie, CleanedTitle: "Fight Club"}}
	metaSvc := &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceTMDb,
		Items:  []metadata.MetadataItem{{ID: "550", Title: "Fight Club"}},
	}}
	scopes := &fakeScopeResolver{scope: "tmdb:movie:550"}
	svc := NewEnrichmentService(repo, parserSvc, metaSvc, nil, nil, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, scopes)

	movie := &models.Movie{ID: "movie-5", Title: "Fight.Club.mkv", MetadataSource: models.NewNullString(string(models.MetadataSourceManual))}
	require.NoError(t, svc.enrichMovie(context.Background(), movie))

	assert.Equal(t, []string{"movie/550"}, seeder.fetchArgs)
	assert.Empty(t, repo.creditsWrites, "manual outranks tmdb: the user's cast stays")
	assert.Equal(t, []string{"movie-5"}, scopes.calls, "the glossary is still resolved (and seeded there): terms have their own provenance")
}

func TestEnrichMovie_RematchOverwritesEarlierTMDbCast(t *testing.T) {
	// A pending row that was matched before (source=tmdb) and is being
	// re-matched — e.g. the first match was the wrong film — takes the NEW
	// film's cast; tmdb does not outrank tmdb.
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	repo := &mockMovieRepoForNFO{}
	parserSvc := &mockPQParserService{result: &parser.ParseResult{Status: parser.ParseStatusSuccess, MediaType: parser.MediaTypeMovie, CleanedTitle: "Fight Club"}}
	metaSvc := &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceTMDb,
		Items:  []metadata.MetadataItem{{ID: "550", Title: "Fight Club"}},
	}}
	svc := NewEnrichmentService(repo, parserSvc, metaSvc, nil, nil, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, nil)

	movie := &models.Movie{ID: "movie-6", Title: "Fight.Club.mkv", MetadataSource: models.NewNullString(string(models.MetadataSourceTMDb))}
	require.NoError(t, movie.SetCredits(&models.Credits{Cast: []models.CastMember{{Name: "上一次比對錯的片的演員"}}}))
	require.NoError(t, svc.enrichMovie(context.Background(), movie))

	assert.Equal(t, []string{"movie-6"}, repo.creditsWrites)
	assert.Equal(t, "布萊恩·克蘭斯頓", repo.lastCredits.Cast[0].Name)
}

func TestEnrichMovie_ResolverErrorIsLoggedNotFatal(t *testing.T) {
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	repo := &mockMovieRepoForNFO{}
	mockTMDb := &mockTMDbServiceForNFO{getMovieDetailsResp: &tmdb.MovieDetails{Movie: tmdb.Movie{ID: 10196}}}
	svc := NewEnrichmentService(repo, nil, nil, NewNFOReaderService(nil), mockTMDb, nil, nil, nil)
	svc.SetGlossarySeeder(seeder, &fakeScopeResolver{err: errors.New("db locked")})

	require.NoError(t, svc.enrichMovie(context.Background(), writeSeedTestNFO(t, "10196")))
	assert.Equal(t, []string{"movie-1"}, repo.creditsWrites, "the cast itself is still stored")
	assert.Equal(t, models.ParseStatusSuccess, repo.updatedMovie.ParseStatus)
}

func TestEnrichSeries_PersistsCreditsAndSeedsAfterWrite(t *testing.T) {
	var events []string
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs, events: &events}
	scopes := &fakeScopeResolver{scope: "tmdb:tv:1396", events: &events}
	seriesRepo := &recordingSeriesRepo{events: &events}
	parserSvc := &mockPQParserService{result: &parser.ParseResult{CleanedTitle: "Breaking Bad"}}
	metaSvc := &mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceTMDb,
		Items:  []metadata.MetadataItem{{ID: "1396", Title: "Breaking Bad", TitleZhTW: "絕命毒師"}},
	}}
	svc := NewEnrichmentService(&mockMovieRepoForNFO{}, parserSvc, metaSvc, nil, nil, nil, nil, nil)
	svc.SetSeriesRepo(seriesRepo)
	svc.SetGlossarySeeder(seeder, scopes)

	require.NoError(t, svc.enrichSeries(context.Background(), &models.Series{ID: "series-1", Title: "Breaking Bad"}))

	assert.Equal(t, []string{"tv/1396"}, seeder.fetchArgs)
	assert.Empty(t, seeder.seedArgs)
	assert.Equal(t, []string{"series-1"}, scopes.calls)
	assert.Equal(t, []string{"fetch", "write", "credits", "resolve"}, events)
	require.NotNil(t, seriesRepo.updated)
	assert.Equal(t, []string{"series-1"}, seriesRepo.creditsWrites)
	require.Len(t, seriesRepo.lastCredits.Cast, 1)
	assert.Equal(t, "華特·懷特", seriesRepo.lastCredits.Cast[0].Character)
	assert.Equal(t, "絕命毒師", seriesRepo.updated.Title)
}

func TestEnrichSeries_NoMatchNeverCallsSeeder(t *testing.T) {
	seeder := &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs}
	seriesRepo := &recordingSeriesRepo{events: &[]string{}}
	parserSvc := &mockPQParserService{result: &parser.ParseResult{CleanedTitle: "Nope"}}
	metaSvc := &mockPQMetadataService{searchResult: &metadata.SearchResult{}}
	svc := NewEnrichmentService(&mockMovieRepoForNFO{}, parserSvc, metaSvc, nil, nil, nil, nil, nil)
	scopes := &fakeScopeResolver{}
	svc.SetSeriesRepo(seriesRepo)
	svc.SetGlossarySeeder(seeder, scopes)

	require.NoError(t, svc.enrichSeries(context.Background(), &models.Series{ID: "series-2", Title: "Nope"}))
	assert.Empty(t, seeder.fetchArgs)
	assert.Empty(t, scopes.calls)
	assert.Equal(t, models.ParseStatusFailed, seriesRepo.updated.ParseStatus)
}

func TestEnrichMovie_WithoutSeederIsUnchanged(t *testing.T) {
	repo := &mockMovieRepoForNFO{}
	mockTMDb := &mockTMDbServiceForNFO{getMovieDetailsResp: &tmdb.MovieDetails{Movie: tmdb.Movie{ID: 10196}}}
	svc := NewEnrichmentService(repo, nil, nil, NewNFOReaderService(nil), mockTMDb, nil, nil, nil)
	require.NoError(t, svc.enrichMovie(context.Background(), writeSeedTestNFO(t, "10196")))
	assert.Empty(t, repo.creditsWrites)
}
