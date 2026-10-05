package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/providers"
)

// backlog-lexicon-on-non-llm-convert-paths: the two manual paths finish a
// converted subtitle the same way every automatic path does — script, then the
// Taiwan vocabulary, mainland titles keeping their own words.

const lexiconSimplifiedSRT = "1\n00:00:01,000 --> 00:00:03,000\n这个软件的质量很好\n"

func fixedCountries(codes []string, err error) MediaCountryResolver {
	return func(context.Context, string, string) ([]string, error) { return codes, err }
}

func newLexiconHandler(t *testing.T, resolver MediaCountryResolver) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mediaPath := filepath.Join(t.TempDir(), "movie.mkv")
	require.NoError(t, os.WriteFile(mediaPath, []byte("fake"), 0644))

	conv, err := subtitle.NewConverter()
	require.NoError(t, err)
	prov := &handlerMockProvider{name: "assrt", downloadData: []byte(lexiconSimplifiedSRT)}
	handler := NewSubtitleHandler(
		[]providers.SubtitleProvider{prov},
		subtitle.NewScorer(subtitle.NewDefaultScorerConfig()), conv,
		subtitle.NewPlacer(subtitle.DefaultPlacerConfig()), nil, &mockStatusUpdater{}, nil,
	)
	if resolver != nil {
		handler.SetCountryResolver(resolver)
	}

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, mediaPath
}

func placedSubtitle(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Data struct {
			SubtitlePath string `json:"subtitle_path"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	got, err := os.ReadFile(resp.Data.SubtitlePath)
	require.NoError(t, err)
	return string(got)
}

func TestSubtitleHandler_Download_ConvertedSubtitleGetsTaiwanVocabulary(t *testing.T) {
	cases := []struct {
		name     string
		resolver MediaCountryResolver
		want     string
	}{
		{"non-mainland → script + vocabulary", fixedCountries([]string{"US"}, nil), "這個軟體的品質很好"},
		{"mainland → script only", fixedCountries([]string{"CN"}, nil), "這個軟體的質量很好"},
		{"lookup fails → treated as non-mainland, download still succeeds", fixedCountries(nil, errors.New("db down")), "這個軟體的品質很好"},
		{"no resolver wired → non-mainland", nil, "這個軟體的品質很好"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, mediaPath := newLexiconHandler(t, tc.resolver)
			convert := true
			w := postJSON(t, router, "/api/v1/subtitles/download", SubtitleDownloadRequest{
				MediaID: "movie-1", MediaType: "movie", MediaFilePath: mediaPath,
				SubtitleID: "sub-1", Provider: "assrt", ConvertToTraditional: &convert,
			})
			assert.Contains(t, placedSubtitle(t, w), tc.want)
		})
	}
}

func TestSubtitleHandler_Download_ToggleOffKeepsTheSubtitleUntouched(t *testing.T) {
	router, mediaPath := newLexiconHandler(t, fixedCountries([]string{"US"}, nil))
	convert := false
	w := postJSON(t, router, "/api/v1/subtitles/download", SubtitleDownloadRequest{
		MediaID: "movie-1", MediaType: "movie", MediaFilePath: mediaPath,
		SubtitleID: "sub-1", Provider: "assrt", ConvertToTraditional: &convert,
	})
	assert.Contains(t, placedSubtitle(t, w), "这个软件的质量很好")
}

func TestSubtitleHandler_Convert_ConvertedSubtitleGetsTaiwanVocabulary(t *testing.T) {
	cases := []struct {
		name    string
		country string
		want    string
	}{
		{"non-mainland → script + vocabulary", "US", "這個軟體的品質很好"},
		{"mainland → script only (an explicit convert is still honoured)", "CN", "這個軟體的質量很好"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, mediaPath := newLexiconHandler(t, fixedCountries([]string{tc.country}, nil))
			writeSidecar(t, mediaPath, "zh-Hans", "srt", lexiconSimplifiedSRT)
			w := postJSON(t, router, "/api/v1/subtitles/convert", SubtitleConvertRequest{
				MediaID: "movie-1", MediaType: "movie", MediaFilePath: mediaPath,
			})
			assert.Contains(t, placedSubtitle(t, w), tc.want)
		})
	}
}

// --- RepoCountryResolver ---

type fakeMovieFinder struct {
	movie *models.Movie
	err   error
}

func (f fakeMovieFinder) FindByID(context.Context, string) (*models.Movie, error) {
	return f.movie, f.err
}

type fakeSeriesFinder struct {
	series *models.Series
	err    error
}

func (f fakeSeriesFinder) FindByID(context.Context, string) (*models.Series, error) {
	return f.series, f.err
}

func TestRepoCountryResolver(t *testing.T) {
	movie := &models.Movie{ProductionCountries: []models.ProductionCountry{{ISO3166_1: "US"}, {ISO3166_1: " CN "}}}
	series := &models.Series{ProductionCountries: []models.ProductionCountry{{ISO3166_1: "TW"}}}
	resolve := NewRepoCountryResolver(fakeMovieFinder{movie: movie}, fakeSeriesFinder{series: series})

	got, err := resolve(context.Background(), "movie", "m1")
	require.NoError(t, err)
	assert.Equal(t, []string{"US", "CN"}, got)

	got, err = resolve(context.Background(), "series", "s1")
	require.NoError(t, err)
	assert.Equal(t, []string{"TW"}, got)

	_, err = NewRepoCountryResolver(fakeMovieFinder{err: errors.New("boom")}, nil)(context.Background(), "movie", "m1")
	assert.Error(t, err)

	got, err = NewRepoCountryResolver(fakeMovieFinder{}, nil)(context.Background(), "movie", "missing")
	require.NoError(t, err)
	assert.Nil(t, got, "a missing row has no countries")

	got, err = NewRepoCountryResolver(nil, nil)(context.Background(), "series", "s1")
	require.NoError(t, err)
	assert.Nil(t, got, "an unwired repo has no countries")
}
