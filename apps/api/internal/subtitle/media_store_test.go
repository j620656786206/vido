package subtitle

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// ─── Repo fakes ────────────────────────────────────────────────────────────

type generationWrite struct {
	id       string
	status   models.SubtitleStatus
	path     string
	language string
}

type fakeMovieRepo struct {
	movie  *models.Movie
	err    error
	writes []generationWrite
}

func (r *fakeMovieRepo) FindByID(context.Context, string) (*models.Movie, error) {
	return r.movie, r.err
}

func (r *fakeMovieRepo) UpdateSubtitleGenerationStatus(_ context.Context, id string, status models.SubtitleStatus, path, language string) error {
	r.writes = append(r.writes, generationWrite{id, status, path, language})
	return nil
}

type fakeSeriesRepo struct {
	series *models.Series
	err    error
	writes []generationWrite
}

func (r *fakeSeriesRepo) FindByID(context.Context, string) (*models.Series, error) {
	return r.series, r.err
}

func (r *fakeSeriesRepo) UpdateSubtitleGenerationStatus(_ context.Context, id string, status models.SubtitleStatus, path, language string) error {
	r.writes = append(r.writes, generationWrite{id, status, path, language})
	return nil
}

type fakeEpisodeRepo struct {
	episode *models.Episode
	err     error
	writes  []generationWrite
}

func (r *fakeEpisodeRepo) FindByID(context.Context, string) (*models.Episode, error) {
	return r.episode, r.err
}

func (r *fakeEpisodeRepo) UpdateEpisodeSubtitleStatus(_ context.Context, id string, status models.SubtitleStatus, path, language string) error {
	r.writes = append(r.writes, generationWrite{id, status, path, language})
	return nil
}

// ─── Load ──────────────────────────────────────────────────────────────────

func TestMediaStore_LoadMovie(t *testing.T) {
	tmdbID := int64(603)
	movies := &fakeMovieRepo{movie: &models.Movie{
		ID:                  "m1",
		Title:               "駭客任務",
		OriginalTitle:       models.NewNullString("The Matrix"),
		ReleaseDate:         "1999-03-31",
		Genres:              []string{"Action", "Sci-Fi"},
		Overview:            models.NewNullString("A hacker learns…"),
		TMDbID:              models.NewNullInt64(tmdbID),
		FilePath:            models.NewNullString("/media/matrix.mkv"),
		ProductionCountries: []models.ProductionCountry{{ISO3166_1: "US"}, {ISO3166_1: "AU"}},
	}}

	store := NewMediaStore(movies, nil, nil)
	item, err := store.Load(context.Background(), MediaRef{ID: "m1", MediaType: models.SubtitleRunMediaMovie})
	require.NoError(t, err)

	assert.Equal(t, "/media/matrix.mkv", item.FilePath)
	require.NotNil(t, item.TMDbID)
	assert.Equal(t, tmdbID, *item.TMDbID)
	assert.Empty(t, item.ShowKey, "a movie shares its prompt prefix with nothing — the D10 gate must bypass it")
	assert.Equal(t, "駭客任務", item.Context.Title)
	assert.Equal(t, "The Matrix", item.Context.OriginalTitle)
	assert.Equal(t, 1999, item.Context.Year)
	assert.Equal(t, []string{"Action", "Sci-Fi"}, item.Context.Genres)
	assert.Equal(t, []string{"US", "AU"}, item.Context.Countries)
}

// TestMediaStore_LoadEpisodeUsesTheParentSeriesMetadata — the FR26 prompt
// context is SHOW-level. Keying it per episode would give every episode of a
// show a different MetadataHash and silently split the segment cache, which is
// the exact non-determinism sub-1-5b AC #3.1 exists to prevent.
func TestMediaStore_LoadEpisodeUsesTheParentSeriesMetadata(t *testing.T) {
	episodes := &fakeEpisodeRepo{episode: &models.Episode{
		ID:       "ep-1",
		SeriesID: "s-42",
		FilePath: models.NewNullString("/media/s01e01.mkv"),
	}}
	series := &fakeSeriesRepo{series: &models.Series{
		ID:            "s-42",
		Title:         "怪奇物語",
		OriginalTitle: models.NewNullString("Stranger Things"),
		FirstAirDate:  "2016-07-15",
		Genres:        []string{"Drama"},
		// sub-6-7: identity fields only reach the prompt for a TMDb-matched
		// show; this fixture models a matched one.
		TMDbID: models.NewNullInt64(66732),
	}}

	store := NewMediaStore(nil, series, episodes)
	item, err := store.Load(context.Background(), MediaRef{ID: "ep-1", MediaType: models.SubtitleRunMediaEpisode})
	require.NoError(t, err)

	assert.Equal(t, "/media/s01e01.mkv", item.FilePath, "the EPISODE's own file is what gets extracted")
	assert.Equal(t, "s-42", item.ShowKey,
		"the D10 gate keys on the SERIES id — an episode-keyed gate would serialize nothing")
	assert.Equal(t, "Stranger Things", item.Context.OriginalTitle)
	assert.Equal(t, 2016, item.Context.Year)

	// Two episodes of one show MUST hash identically, or the cache splits.
	episodes.episode = &models.Episode{ID: "ep-2", SeriesID: "s-42",
		FilePath: models.NewNullString("/media/s01e02.mkv")}
	other, err := store.Load(context.Background(), MediaRef{ID: "ep-2", MediaType: models.SubtitleRunMediaEpisode})
	require.NoError(t, err)
	assert.Equal(t, MetadataHash(item.Context), MetadataHash(other.Context),
		"every episode of a show must share one MetadataHash, or the segment cache splits per episode")
}

// TestMediaStore_LoadEpisodeSurvivesAnUnmatchedSeries — metadata is optional
// context for the prompt. Refusing to translate an episode because its series
// row is unmatched would be a worse outcome than translating with less context.
func TestMediaStore_LoadEpisodeSurvivesAnUnmatchedSeries(t *testing.T) {
	episodes := &fakeEpisodeRepo{episode: &models.Episode{
		ID: "ep-1", SeriesID: "s-missing",
		FilePath: models.NewNullString("/media/s01e01.mkv"),
	}}
	series := &fakeSeriesRepo{err: errors.New("no such series")}

	item, err := NewMediaStore(nil, series, episodes).
		Load(context.Background(), MediaRef{ID: "ep-1", MediaType: models.SubtitleRunMediaEpisode})

	require.NoError(t, err)
	assert.Equal(t, "/media/s01e01.mkv", item.FilePath)
	assert.Equal(t, "s-missing", item.ShowKey, "the gate key survives even when the metadata does not")
	assert.Empty(t, item.Context.Title)
}

func TestMediaStore_LoadRejectsAnUnknownMediaType(t *testing.T) {
	_, err := NewMediaStore(nil, nil, nil).
		Load(context.Background(), MediaRef{ID: "x", MediaType: "tv"})
	require.Error(t, err, "`tv` is the TMDB vocabulary — the internal one is movie|series|episode (sub-1-2 AC #1)")
	assert.Contains(t, err.Error(), "tv")
}

func TestMediaStore_LoadMissingRowIsAnError(t *testing.T) {
	_, err := NewMediaStore(&fakeMovieRepo{movie: nil}, nil, nil).
		Load(context.Background(), MediaRef{ID: "gone", MediaType: models.SubtitleRunMediaMovie})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMediaStore_YearOfHandlesJunkDates(t *testing.T) {
	for _, date := range []string{"", "20", "abcd-01-01", "n/a"} {
		assert.Zero(t, yearOf(date), "an unparseable date must yield no year, never a bogus one: %q", date)
	}
	assert.Equal(t, 1999, yearOf("1999-03-31"))
}

// ─── SetSubtitleStatus ─────────────────────────────────────────────────────

// TestMediaStore_WritesGoThroughTheGenerationSafeWriters is the
// `backlog-subtitle-status-writer-search-columns` guard at the adapter level:
// every media type must land on a writer that leaves the search columns alone.
func TestMediaStore_WritesGoThroughTheGenerationSafeWriters(t *testing.T) {
	movies := &fakeMovieRepo{}
	series := &fakeSeriesRepo{}
	episodes := &fakeEpisodeRepo{}
	store := NewMediaStore(movies, series, episodes)
	ctx := context.Background()

	require.NoError(t, store.SetSubtitleStatus(ctx,
		MediaRef{ID: "m1", MediaType: models.SubtitleRunMediaMovie},
		models.SubtitleStatusFound, "/media/m1.zh-Hant.srt", "zh-Hant"))
	require.NoError(t, store.SetSubtitleStatus(ctx,
		MediaRef{ID: "s1", MediaType: models.SubtitleRunMediaSeries},
		models.SubtitleStatusTranslating, "", ""))
	require.NoError(t, store.SetSubtitleStatus(ctx,
		MediaRef{ID: "ep1", MediaType: models.SubtitleRunMediaEpisode},
		models.SubtitleStatusNotSearched, "", ""))

	require.Len(t, movies.writes, 1)
	assert.Equal(t, generationWrite{"m1", models.SubtitleStatusFound, "/media/m1.zh-Hant.srt", "zh-Hant"}, movies.writes[0])
	require.Len(t, series.writes, 1)
	assert.Equal(t, models.SubtitleStatusTranslating, series.writes[0].status)
	require.Len(t, episodes.writes, 1)
	assert.Equal(t, models.SubtitleStatusNotSearched, episodes.writes[0].status)
}

func TestMediaStore_SetSubtitleStatusRejectsAnUnknownMediaType(t *testing.T) {
	err := NewMediaStore(nil, nil, nil).SetSubtitleStatus(context.Background(),
		MediaRef{ID: "x", MediaType: "film"}, models.SubtitleStatusFound, "", "")
	require.Error(t, err)
}

func TestMediaStore_UnwiredRepoFailsByName(t *testing.T) {
	err := NewMediaStore(nil, nil, nil).SetSubtitleStatus(context.Background(),
		MediaRef{ID: "m1", MediaType: models.SubtitleRunMediaMovie}, models.SubtitleStatusFound, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "movie repository")
}

// ─── sub-6-7: a filename-shaped title is not show context ───────────────────

func TestMediaStore_UnmatchedMovieWithFilenameTitleSendsNoIdentity(t *testing.T) {
	movies := &fakeMovieRepo{movie: &models.Movie{
		ID:          "m-unmatched",
		Title:       "[bitsearch.to] Wake.Up.Dead.Man.2025.2160p.WEB-DL.DV.HDR-NAHOM.mkv",
		ReleaseDate: "2025-01-01",
		Overview:    models.NewNullString("parsed-from-nothing"),
		Genres:      []string{"Mystery"}, // CR M4: genres survive, so the hash is compared against a genres-only context
		FilePath:    models.NewNullString("/media/wake.mkv"),
		// TMDbID left invalid: the scanner never matched this file.
	}}

	item, err := NewMediaStore(movies, nil, nil).
		Load(context.Background(), MediaRef{ID: "m-unmatched", MediaType: models.SubtitleRunMediaMovie})
	require.NoError(t, err)

	assert.Nil(t, item.TMDbID)
	assert.Empty(t, item.Context.Title, "a release filename must never reach the prompt as the film's title")
	assert.Empty(t, item.Context.OriginalTitle)
	assert.Zero(t, item.Context.Year, "unmatched: the year came from the same filename parse")
	assert.Empty(t, item.Context.Overview)
	assert.Equal(t, []string{"Mystery"}, item.Context.Genres)
	assert.Equal(t, MetadataHash(TranslateContext{Genres: []string{"Mystery"}}), MetadataHash(item.Context),
		"the cache key must not carry the release name — only the fields that were kept")
}

func TestMediaStore_UnmatchedMovieWithCleanTitleKeepsIt(t *testing.T) {
	// CR M4: the parser (or the metadata editor) produced a real title on a row
	// TMDb never matched — that is context worth sending, not noise.
	movies := &fakeMovieRepo{movie: &models.Movie{
		ID: "m-clean", Title: "Wake Up Dead Man", ReleaseDate: "2025-01-01",
		Overview: models.NewNullString("A detective returns."),
		FilePath: models.NewNullString("/media/wake.mkv"),
	}}
	item, err := NewMediaStore(movies, nil, nil).
		Load(context.Background(), MediaRef{ID: "m-clean", MediaType: models.SubtitleRunMediaMovie})
	require.NoError(t, err)
	assert.Equal(t, "Wake Up Dead Man", item.Context.Title)
	assert.Equal(t, 2025, item.Context.Year)
	assert.Equal(t, "A detective returns.", item.Context.Overview)
}

func TestMediaStore_MatchedMovieWithFilenameShapedTitleKeepsTMDbFields(t *testing.T) {
	// CR H3: the row matched, so Year/Overview are TMDb's and stay; only the
	// scraped title line is dropped.
	movies := &fakeMovieRepo{movie: &models.Movie{
		ID:          "m-shaped",
		Title:       "Predator.Badlands.2025.2160p.MA.WEB-DL.DV.HDR.TYMBLE",
		ReleaseDate: "2025-11-07",
		Overview:    models.NewNullString("A young Predator…"),
		Genres:      []string{"Action"},
		TMDbID:      models.NewNullInt64(1),
		FilePath:    models.NewNullString("/media/predator.mkv"),
	}}

	item, err := NewMediaStore(movies, nil, nil).
		Load(context.Background(), MediaRef{ID: "m-shaped", MediaType: models.SubtitleRunMediaMovie})
	require.NoError(t, err)

	assert.Empty(t, item.Context.Title)
	assert.Empty(t, item.Context.OriginalTitle)
	assert.Equal(t, 2025, item.Context.Year)
	assert.Equal(t, "A young Predator…", item.Context.Overview)
	assert.Equal(t, []string{"Action"}, item.Context.Genres)
}

func TestMediaStore_MatchedMovieKeepsItsRealTitle(t *testing.T) {
	movies := &fakeMovieRepo{movie: &models.Movie{
		ID: "m-ok", Title: "Dune: Part Two", ReleaseDate: "2024-03-01",
		TMDbID: models.NewNullInt64(693134), FilePath: models.NewNullString("/media/dune2.mkv"),
	}}
	item, err := NewMediaStore(movies, nil, nil).
		Load(context.Background(), MediaRef{ID: "m-ok", MediaType: models.SubtitleRunMediaMovie})
	require.NoError(t, err)
	assert.Equal(t, "Dune: Part Two", item.Context.Title)
	assert.Equal(t, 2024, item.Context.Year)
}

func TestMediaStore_BracketTitledFilmIsNotAFilename(t *testing.T) {
	movies := &fakeMovieRepo{movie: &models.Movie{
		ID: "m-rec", Title: "[REC]", ReleaseDate: "2007-11-23",
		TMDbID: models.NewNullInt64(8329), FilePath: models.NewNullString("/media/rec.mkv"),
	}}
	item, err := NewMediaStore(movies, nil, nil).
		Load(context.Background(), MediaRef{ID: "m-rec", MediaType: models.SubtitleRunMediaMovie})
	require.NoError(t, err)
	assert.Equal(t, "[REC]", item.Context.Title, "CR H2: a bracket alone is not evidence of a filename")
}

func TestMediaStore_LoadSeriesAppliesTheSameRule(t *testing.T) {
	// CR M8: the series row's own Load path (not only episode→series).
	series := &fakeSeriesRepo{series: &models.Series{
		ID: "s-shaped", Title: "Some.Show.S01.2160p.WEB-DL", FirstAirDate: "2024-01-01",
		Overview: models.NewNullString("TMDb overview"), TMDbID: models.NewNullInt64(99),
		FilePath: models.NewNullString("/media/show"),
	}}
	item, err := NewMediaStore(nil, series, nil).
		Load(context.Background(), MediaRef{ID: "s-shaped", MediaType: models.SubtitleRunMediaSeries})
	require.NoError(t, err)
	assert.Equal(t, "s-shaped", item.ShowKey)
	assert.Empty(t, item.Context.Title)
	assert.Equal(t, 2024, item.Context.Year, "matched series keeps TMDb's year")
	assert.Equal(t, "TMDb overview", item.Context.Overview)
}

func TestMediaStore_UnmatchedSeriesWithFilenameTitleSendsNoIdentity(t *testing.T) {
	episodes := &fakeEpisodeRepo{episode: &models.Episode{
		ID: "ep-1", SeriesID: "s-unmatched", FilePath: models.NewNullString("/media/s01e01.mkv"),
	}}
	series := &fakeSeriesRepo{series: &models.Series{
		ID: "s-unmatched", Title: "Some.Show.S01.2160p.WEB-DL", FirstAirDate: "2024-01-01",
		// TMDbID invalid.
	}}
	item, err := NewMediaStore(nil, series, episodes).
		Load(context.Background(), MediaRef{ID: "ep-1", MediaType: models.SubtitleRunMediaEpisode})
	require.NoError(t, err)
	assert.Equal(t, "s-unmatched", item.ShowKey, "the gate key is not identity — it survives")
	assert.Empty(t, item.Context.Title)
	assert.Zero(t, item.Context.Year)
}
