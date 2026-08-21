package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// ─── mocks ──────────────────────────────────────────────────────────────────

type mockNFOMovieGetter struct{ movie *models.Movie }

func nfoTestMovie() *models.Movie {
	return &models.Movie{ID: "movie-1", Title: "Inception", FilePath: models.NewNullString("/media/Inception.mkv")}
}

func (m *mockNFOMovieGetter) GetByID(context.Context, string) (*models.Movie, error) {
	return m.movie, nil
}

type mockNFOSeriesGetter struct {
	series *models.Series
	err    error
}

func (m *mockNFOSeriesGetter) GetByID(context.Context, string) (*models.Series, error) {
	return m.series, m.err
}

type mockNFOEpisodeGetter struct {
	episode *models.Episode
	err     error
}

func (m *mockNFOEpisodeGetter) FindByID(context.Context, string) (*models.Episode, error) {
	return m.episode, m.err
}

// mockNFOLocalizer counts EVERY localization call. The count is the point: the
// confirm gate has to reject before any of these run, because reaching one
// means the user's Claude budget was already spent on a request the server was
// always going to refuse.
type mockNFOLocalizer struct {
	available bool
	calls     int
	err       error
}

func (m *mockNFOLocalizer) IsAvailable() bool { return m.available }
func (m *mockNFOLocalizer) LocalizeMovieNFO(context.Context, models.Movie) (*services.NFOLocalizeResult, error) {
	m.calls++
	return &services.NFOLocalizeResult{Path: "/media/movie.nfo"}, m.err
}
func (m *mockNFOLocalizer) LocalizeTVShowNFO(context.Context, models.Series) (*services.NFOLocalizeResult, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &services.NFOLocalizeResult{Path: "/media/Buffy/tvshow.nfo", Replaced: true, BackupPath: "/media/Buffy/tvshow.nfo.orig"}, nil
}
func (m *mockNFOLocalizer) LocalizeSeriesNFOWithEpisodes(context.Context, models.Series) (*services.NFOSeriesLocalizeResult, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &services.NFOSeriesLocalizeResult{
		Show:      &services.NFOLocalizeResult{Path: "/media/Buffy/tvshow.nfo"},
		Episodes:  []*services.NFOLocalizeResult{{Path: "/media/Buffy/Season01/e1.nfo"}},
		Succeeded: 1,
	}, nil
}
func (m *mockNFOLocalizer) LocalizeEpisodeNFO(context.Context, models.Episode, string) (*services.NFOLocalizeResult, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &services.NFOLocalizeResult{Path: "/media/Buffy/Season01/e1.nfo"}, nil
}

func nfoTestSeries() *models.Series {
	return &models.Series{ID: "series-1", Title: "Buffy", FilePath: models.NewNullString("/media/Buffy")}
}

func nfoTestEpisode() *models.Episode {
	return &models.Episode{
		ID: "episode-1", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 1,
		FilePath: models.NewNullString("/media/Buffy/Season01/e1.mkv"),
	}
}

func newNFORouter(t *testing.T, loc *mockNFOLocalizer, series *mockNFOSeriesGetter, episode *mockNFOEpisodeGetter) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewNFOLocalizerHandler(&mockNFOMovieGetter{movie: nfoTestMovie()}, series, episode, loc)
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func postJSON(t *testing.T, r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(http.MethodPost, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func errorCodeOf(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Error.Code
}

// ─── 🔴 the replace opt-in gate ─────────────────────────────────────────────

func TestNFOHandler_SeriesWithoutConfirmationIs409AndNeverLocalizes(t *testing.T) {
	cases := []struct {
		name string
		body any
	}{
		{"no body at all", nil},
		{"confirm_replace omitted", map[string]any{}},
		{"confirm_replace false", map[string]any{"confirm_replace": false}},
		{"unrelated body", map[string]any{"something": "else"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loc := &mockNFOLocalizer{available: true}
			r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{episode: nfoTestEpisode()})

			w := postJSON(t, r, "/api/v1/series/series-1/localize-nfo", tc.body)

			assert.Equal(t, http.StatusConflict, w.Code)
			assert.Equal(t, "NFO_REPLACE_NOT_CONFIRMED", errorCodeOf(t, w))
			assert.Equal(t, 0, loc.calls,
				"the gate must reject BEFORE any localization — otherwise the refused request still cost money")
		})
	}
}

func TestNFOHandler_EpisodeWithoutConfirmationIs409AndNeverLocalizes(t *testing.T) {
	loc := &mockNFOLocalizer{available: true}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{episode: nfoTestEpisode()})

	w := postJSON(t, r, "/api/v1/episodes/episode-1/localize-nfo", map[string]any{"confirm_replace": false})

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "NFO_REPLACE_NOT_CONFIRMED", errorCodeOf(t, w))
	assert.Equal(t, 0, loc.calls)
}

// The movie route is ADDITIVE — it never replaces anything, so it must NOT have
// grown a confirmation requirement.
func TestNFOHandler_MovieRouteStillNeedsNoConfirmation(t *testing.T) {
	loc := &mockNFOLocalizer{available: true}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{episode: nfoTestEpisode()})

	w := postJSON(t, r, "/api/v1/movies/movie-1/localize-nfo", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, loc.calls)
}

// ─── happy paths ────────────────────────────────────────────────────────────

func TestNFOHandler_SeriesConfirmedLocalizesTheShowFile(t *testing.T) {
	loc := &mockNFOLocalizer{available: true}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{episode: nfoTestEpisode()})

	w := postJSON(t, r, "/api/v1/series/series-1/localize-nfo", map[string]any{"confirm_replace": true})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, loc.calls)
	assert.Contains(t, w.Body.String(), "tvshow.nfo")
	assert.Contains(t, w.Body.String(), "tvshow.nfo.orig")
}

func TestNFOHandler_SeriesWithIncludeEpisodesUsesTheBatch(t *testing.T) {
	loc := &mockNFOLocalizer{available: true}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{episode: nfoTestEpisode()})

	w := postJSON(t, r, "/api/v1/series/series-1/localize-nfo?include_episodes=true", map[string]any{"confirm_replace": true})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"succeeded":1`)
	assert.Contains(t, w.Body.String(), "Season01/e1.nfo")
}

func TestNFOHandler_EpisodeConfirmedLocalizes(t *testing.T) {
	loc := &mockNFOLocalizer{available: true}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{episode: nfoTestEpisode()})

	w := postJSON(t, r, "/api/v1/episodes/episode-1/localize-nfo", map[string]any{"confirm_replace": true})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, loc.calls)
}

// ─── degradations ───────────────────────────────────────────────────────────

func TestNFOHandler_UnavailableLocalizerIs503BeforeTheGate(t *testing.T) {
	loc := &mockNFOLocalizer{available: false}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{episode: nfoTestEpisode()})

	w := postJSON(t, r, "/api/v1/series/series-1/localize-nfo", map[string]any{"confirm_replace": true})

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "NFO_LOCALIZE_DISABLED", errorCodeOf(t, w))
	assert.Equal(t, 0, loc.calls)
}

func TestNFOHandler_MissingSeriesIs404(t *testing.T) {
	loc := &mockNFOLocalizer{available: true}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{err: errors.New("not found")}, &mockNFOEpisodeGetter{})

	w := postJSON(t, r, "/api/v1/series/nope/localize-nfo", map[string]any{"confirm_replace": true})

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, loc.calls)
}

// The interface allows (nil, nil); without an explicit guard that would
// dereference a nil pointer and 500 with a panic instead of a clean 404.
func TestNFOHandler_NilSeriesWithoutErrorIs404(t *testing.T) {
	loc := &mockNFOLocalizer{available: true}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nil, err: nil}, &mockNFOEpisodeGetter{})

	w := postJSON(t, r, "/api/v1/series/nope/localize-nfo", map[string]any{"confirm_replace": true})

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, loc.calls)
}

func TestNFOHandler_SeriesWithoutFolderPathIs400(t *testing.T) {
	series := nfoTestSeries()
	series.FilePath = models.NullString{}
	loc := &mockNFOLocalizer{available: true}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: series}, &mockNFOEpisodeGetter{})

	w := postJSON(t, r, "/api/v1/series/series-1/localize-nfo", map[string]any{"confirm_replace": true})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 0, loc.calls)
}

func TestNFOHandler_LocalizerErrorIs500(t *testing.T) {
	loc := &mockNFOLocalizer{available: true, err: errors.New("disk full")}
	r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{})

	w := postJSON(t, r, "/api/v1/series/series-1/localize-nfo", map[string]any{"confirm_replace": true})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "NFO_LOCALIZE_FAILED", errorCodeOf(t, w))
}

// Without a series/episode getter the TV routes must not exist at all, rather
// than register and nil-panic on the first request.
func TestNFOHandler_TVRoutesAbsentWhenGettersAreNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewNFOLocalizerHandler(&mockNFOMovieGetter{movie: nfoTestMovie()}, nil, nil, &mockNFOLocalizer{available: true})
	h.RegisterRoutes(r.Group("/api/v1"))

	assert.Equal(t, http.StatusNotFound,
		postJSON(t, r, "/api/v1/series/series-1/localize-nfo", map[string]any{"confirm_replace": true}).Code)
	assert.Equal(t, http.StatusNotFound,
		postJSON(t, r, "/api/v1/episodes/episode-1/localize-nfo", map[string]any{"confirm_replace": true}).Code)
}

// PRE-EXISTING FIX regression guard (9R-13a): the movie getter interface allows
// (nil, nil), and the shipped 9R-13 handler dereferenced the result without
// checking — a panic and a stack-trace 500 where a clean 404 belongs.
func TestNFOHandler_NilMovieWithoutErrorIs404NotAPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	loc := &mockNFOLocalizer{available: true}
	h := NewNFOLocalizerHandler(&mockNFOMovieGetter{movie: nil}, nil, nil, loc)
	h.RegisterRoutes(r.Group("/api/v1"))

	w := postJSON(t, r, "/api/v1/movies/nope/localize-nfo", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, 0, loc.calls)
}

// CR M4: presence means yes. `?include_episodes=1` used to fall through to the
// show-only branch and answer 200 — the caller believes 24 episodes were
// localized when none were.
func TestNFOHandler_IncludeEpisodesAcceptsMoreThanTheLiteralTrue(t *testing.T) {
	cases := []struct {
		query string
		batch bool
	}{
		{"?include_episodes=true", true},
		{"?include_episodes=TRUE", true},
		{"?include_episodes=1", true},
		{"?include_episodes", true},
		{"?include_episodes=false", false},
		{"?include_episodes=0", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			loc := &mockNFOLocalizer{available: true}
			r := newNFORouter(t, loc, &mockNFOSeriesGetter{series: nfoTestSeries()}, &mockNFOEpisodeGetter{})

			w := postJSON(t, r, "/api/v1/series/series-1/localize-nfo"+tc.query, map[string]any{"confirm_replace": true})

			require.Equal(t, http.StatusOK, w.Code)
			// The batch response is the only one carrying "succeeded".
			assert.Equal(t, tc.batch, bytes.Contains(w.Body.Bytes(), []byte(`"succeeded"`)))
		})
	}
}

// CR M2: gin PANICS AT BOOT if two registrations disagree on a path parameter's
// name at the same position (`/series/:id` vs `/series/:seriesId`). Every
// existing /series/ and /episodes/ route uses `:id` today, and nothing pinned
// that — a future handler picking a different name would crash the API on
// startup, with only the CI serve-smoke gate to catch it.
func TestNFOHandler_TVRoutesCoexistWithTheExistingSeriesAndEpisodeRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api/v1")

	// The shapes other handlers already register (series_handler.go,
	// douban_rating_handler.go, transcription_handler.go).
	noop := func(c *gin.Context) { c.Status(http.StatusOK) }
	rg.GET("/series/:id", noop)
	rg.GET("/series/:id/seasons", noop)
	rg.GET("/series/:id/seasons/:seasonNumber/episodes", noop)
	rg.GET("/series/:id/douban-rating", noop)
	rg.POST("/episodes/:id/transcribe", noop)

	require.NotPanics(t, func() {
		h := NewNFOLocalizerHandler(&mockNFOMovieGetter{movie: nfoTestMovie()},
			&mockNFOSeriesGetter{series: nfoTestSeries()},
			&mockNFOEpisodeGetter{episode: nfoTestEpisode()},
			&mockNFOLocalizer{available: true})
		h.RegisterRoutes(rg)
	}, "a parameter-name disagreement here is a startup panic, not a test failure")
}
