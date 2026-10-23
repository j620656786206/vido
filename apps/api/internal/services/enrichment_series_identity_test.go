package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/tmdb"
)

// ─── disc-2026-10-series-wrong-version-a ────────────────────────────────────

// everNightSearch is what TMDb answers for 「將夜」: the 2018 drama first, the
// 2026 animation second.
type recordingSearch struct {
	mockPQMetadataService
	reqs []*SearchMetadataRequest
}

func (r *recordingSearch) SearchMetadata(ctx context.Context, req *SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
	r.reqs = append(r.reqs, req)
	return r.mockPQMetadataService.SearchMetadata(ctx, req)
}

func everNightSearch() *recordingSearch {
	return &recordingSearch{mockPQMetadataService: mockPQMetadataService{searchResult: &metadata.SearchResult{
		Source: models.MetadataSourceTMDb,
		Items: []metadata.MetadataItem{
			{ID: "83612", Title: "將夜", Year: 2018, MediaType: metadata.MediaTypeTV},
			{ID: "282136", Title: "將夜", Year: 2026, MediaType: metadata.MediaTypeTV},
		},
	}}}
}

const everNightFile = "[将夜].Ever.Night.2026.S01E01.2160p.WEB-DL.HDR10.H265.10bit.AAC-UBWEB.mp4"

type seriesIdentityEnv struct {
	svc    *EnrichmentService
	repo   *recordingSeriesRepo
	tmdb   *facetsTMDb
	search *recordingSearch
	root   string // the library root
}

func newSeriesIdentityEnv(t *testing.T, cleaned string) *seriesIdentityEnv {
	t.Helper()
	var events []string
	env := &seriesIdentityEnv{
		repo:   &recordingSeriesRepo{events: &events},
		tmdb:   &facetsTMDb{tvDetails: &tmdb.TVShowDetails{TVShow: tmdb.TVShow{ID: 282136, Name: "將夜", FirstAirDate: "2026-04-23"}}},
		search: everNightSearch(),
		root:   t.TempDir(),
	}
	env.svc = NewEnrichmentService(&mockMovieRepoForNFO{}, &mockPQParserService{result: &parser.ParseResult{CleanedTitle: cleaned}},
		env.search, NewNFOReaderService(nil), env.tmdb, nil, nil, nil)
	env.svc.SetSeriesRepo(env.repo)
	env.svc.SetLibraryRoots(func(context.Context) ([]string, error) { return []string{env.root}, nil })
	return env
}

// showDir makes <root>/<name>, optionally with a tvshow.nfo and files in
// Season01.
func (env *seriesIdentityEnv) showDir(t *testing.T, name, nfo string, files ...string) string {
	t.Helper()
	dir := filepath.Join(env.root, name)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "Season01"), 0o755))
	if nfo != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "tvshow.nfo"), []byte(nfo), 0o644))
	}
	for _, f := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Season01", f), []byte("v"), 0o644))
	}
	return dir
}

func seriesAt(dir, title string) *models.Series {
	return &models.Series{ID: "series-1", Title: title, FilePath: models.NewNullString(dir)}
}

func TestEnrichSeries_TVShowNFOTMDbIDWinsOverSearch(t *testing.T) {
	env := newSeriesIdentityEnv(t, "將夜")
	dir := env.showDir(t, "將夜", `<tvshow><title>將夜</title><uniqueid default="true" type="tmdb">282136</uniqueid></tvshow>`)
	s := seriesAt(dir, "將夜")
	s.TMDbID = models.NewNullInt64(83612) // the wrong earlier match
	s.MetadataSource = models.NewNullString(string(models.MetadataSourceTMDb))

	require.NoError(t, env.svc.enrichSeries(context.Background(), s))

	assert.Empty(t, env.search.reqs, "an NFO id is not searched")
	require.NotNil(t, env.repo.updated)
	assert.Equal(t, int64(282136), env.repo.updated.TMDbID.Int64)
	assert.Equal(t, string(models.MetadataSourceNFO), env.repo.updated.MetadataSource.String)
	assert.Equal(t, models.ParseStatusSuccess, env.repo.updated.ParseStatus)
	assert.Equal(t, "2026-04-23", env.repo.updated.FirstAirDate)
	require.NotNil(t, env.repo.lastCredits, "the 2018 cast must not stay on the 2026 show")
	assert.Empty(t, env.repo.lastCredits.Cast)
}

func TestEnrichSeries_TVShowNFOIMDbIDFindsTheShow(t *testing.T) {
	env := newSeriesIdentityEnv(t, "將夜")
	env.tmdb.findByExtResp = &tmdb.FindByExternalIDResponse{TVResults: []tmdb.TVShow{{ID: 282136}}}
	dir := env.showDir(t, "將夜", `<tvshow><uniqueid type="imdb">tt32221651</uniqueid></tvshow>`)

	require.NoError(t, env.svc.enrichSeries(context.Background(), seriesAt(dir, "將夜")))

	assert.Empty(t, env.search.reqs)
	require.NotNil(t, env.repo.updated)
	assert.Equal(t, int64(282136), env.repo.updated.TMDbID.Int64)
}

func TestEnrichSeries_NFOInTheLibraryRootIsNotThisShows(t *testing.T) {
	env := newSeriesIdentityEnv(t, "將夜")
	require.NoError(t, os.WriteFile(filepath.Join(env.root, "tvshow.nfo"),
		[]byte(`<tvshow><uniqueid type="tmdb">999</uniqueid><year>2026</year></tvshow>`), 0o644))

	require.NoError(t, env.svc.enrichSeries(context.Background(), seriesAt(env.root, "將夜")))

	assert.Equal(t, 1, env.tmdb.tvCalls, "only the facets call of the search match — no NFO lookup")
	require.Len(t, env.search.reqs, 1)
	assert.Zero(t, env.search.reqs[0].Year, "the root's NFO year is not this show's either")
	assert.Equal(t, int64(83612), env.repo.updated.TMDbID.Int64)
}

func TestEnrichSeries_UnreadableRootsDoNotTrustTheFolder(t *testing.T) {
	env := newSeriesIdentityEnv(t, "將夜")
	dir := env.showDir(t, "將夜", `<tvshow><uniqueid type="tmdb">282136</uniqueid></tvshow>`)
	env.svc.SetLibraryRoots(func(context.Context) ([]string, error) { return nil, errors.New("db locked") })

	require.NoError(t, env.svc.enrichSeries(context.Background(), seriesAt(dir, "將夜")))

	require.Len(t, env.search.reqs, 1)
	assert.Equal(t, int64(83612), env.repo.updated.TMDbID.Int64)
}

func TestEnrichSeries_BrokenNFOOrTMDbFailureFallsBackToSearch(t *testing.T) {
	t.Run("broken nfo", func(t *testing.T) {
		env := newSeriesIdentityEnv(t, "將夜")
		dir := env.showDir(t, "將夜", `<tvshow><uniqueid type="tmdb">`)
		require.NoError(t, env.svc.enrichSeries(context.Background(), seriesAt(dir, "將夜")))
		require.Len(t, env.search.reqs, 1)
		assert.Equal(t, models.ParseStatusSuccess, env.repo.updated.ParseStatus)
	})
	t.Run("tmdb cannot resolve the id", func(t *testing.T) {
		env := newSeriesIdentityEnv(t, "將夜")
		env.tmdb.tvErr = errors.New("TMDB_NOT_FOUND")
		dir := env.showDir(t, "將夜", `<tvshow><uniqueid type="tmdb">282136</uniqueid></tvshow>`)
		require.NoError(t, env.svc.enrichSeries(context.Background(), seriesAt(dir, "將夜")))
		require.Len(t, env.search.reqs, 1)
		assert.Equal(t, models.ParseStatusSuccess, env.repo.updated.ParseStatus)
	})
}

func TestEnrichSeries_ManualRowNeverReadsTheNFO(t *testing.T) {
	env := newSeriesIdentityEnv(t, "將夜")
	dir := env.showDir(t, "將夜", `<tvshow><uniqueid type="tmdb">282136</uniqueid></tvshow>`)
	s := seriesAt(dir, "我選的")
	s.TMDbID = models.NewNullInt64(83612)
	s.MetadataSource = models.NewNullString(string(models.MetadataSourceManual))

	require.NoError(t, env.svc.enrichSeries(context.Background(), s))

	assert.Zero(t, env.tmdb.tvCalls)
	assert.Empty(t, env.search.reqs)
	require.NotNil(t, env.repo.updated)
	assert.Equal(t, int64(83612), env.repo.updated.TMDbID.Int64)
}

func TestEnrichSeries_YearPicksTheRightRemake(t *testing.T) {
	cases := []struct {
		name   string
		folder string
		nfo    string
		title  string
		files  []string
		want   int64
		wantQ  string
		wantYr int
	}{
		{"nfo year", "將夜", `<tvshow><title>將夜</title><year>2026</year></tvshow>`, "將夜", nil, 282136, "將夜", 2026},
		{"folder name", "將夜 (2026)", "", "將夜", nil, 282136, "將夜", 2026},
		{"series title", "將夜", "", "Ever Night 2026", nil, 282136, "Ever Night", 2026},
		{"episode file name", "將夜", "", "將夜", []string{everNightFile}, 282136, "將夜", 2026},
		{"no year anywhere: the first hit, as before", "將夜", "", "將夜", []string{"[将夜].S01E01.mp4"}, 83612, "將夜", 0},
		{"a year no hit has: the first hit", "將夜 (2031)", "", "將夜", nil, 83612, "將夜", 2031},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newSeriesIdentityEnv(t, tc.title)
			dir := env.showDir(t, tc.folder, tc.nfo, tc.files...)

			require.NoError(t, env.svc.enrichSeries(context.Background(), seriesAt(dir, tc.title)))

			require.Len(t, env.search.reqs, 1)
			assert.Equal(t, tc.wantQ, env.search.reqs[0].Query)
			assert.Equal(t, tc.wantYr, env.search.reqs[0].Year)
			require.NotNil(t, env.repo.updated)
			assert.Equal(t, tc.want, env.repo.updated.TMDbID.Int64)
		})
	}
}

func TestYearFromEpisodeFiles(t *testing.T) {
	t.Run("a show named after a year is not a year", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "1923.S01E01.mkv"), []byte("v"), 0o644))
		assert.Zero(t, yearFromEpisodeFiles(dir))
	})
	t.Run("files beside the folder before season folders; non-video ignored", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Show.2019.S01E01.srt"), []byte("s"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Show.2020.S01E01.mkv"), []byte("v"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "Season 02"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Season 02", "Show.2021.S02E01.mkv"), []byte("v"), 0o644))
		assert.Equal(t, 2020, yearFromEpisodeFiles(dir))
	})
	t.Run("only season folders are entered", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "extras"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "extras", "Show.2019.S01E01.mkv"), []byte("v"), 0o644))
		assert.Zero(t, yearFromEpisodeFiles(dir))
	})
}

func TestPickSeriesMatch_OnlyTradesForTheSameNamedShow(t *testing.T) {
	items := []metadata.MetadataItem{
		{ID: "1", Title: "The Show", Year: 2019},
		{ID: "2", Title: "Another Show", Year: 2021},
		{ID: "3", Title: "Something", OriginalTitle: "The Show", Year: 2021},
	}
	assert.Equal(t, "3", pickSeriesMatch(items, 2021).ID, "a same-named version that aired that year")
	assert.Equal(t, "1", pickSeriesMatch(items[:2], 2021).ID,
		"a later season's year must not trade the show for an unrelated one")
	assert.Equal(t, "1", pickSeriesMatch(items, 2019).ID)
	assert.Equal(t, "1", pickSeriesMatch(items, 0).ID)
}
