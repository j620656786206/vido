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
