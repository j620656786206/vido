package tmdb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/tmdb"
)

// The converter the server wires (subtitle.Converter, an OpenCC process per
// call) behind the TMDb layer, so the batching survives a real process round
// trip. The binary is the repo's stand-in OpenCC script, which runs anywhere.
func TestSimplified_ThroughTheServersConverter(t *testing.T) {
	t.Setenv("VIDO_OPENCC_BIN", "../subtitle/testdata/opencc-helper.sh")
	t.Setenv("VIDO_OPENCC_CONFIG", "../subtitle/testdata/s2twp.json")
	conv, err := subtitle.NewConverter()
	require.NoError(t, err)

	mock := &tmdb.MockClient{GetMovieDetailsResponses: map[string]*tmdb.MovieDetails{
		"zh-TW": {Movie: tmdb.Movie{ID: 1}},
		"zh-CN": {
			Movie:               tmdb.Movie{ID: 1, Title: "这个软件", Overview: "这个软件的质量"},
			ProductionCountries: []tmdb.Country{{ISO31661: "US"}},
		},
	}}
	c := tmdb.NewLanguageFallbackClient(mock, nil)
	c.SetChineseConverter(conv)

	got, _, err := c.GetMovieDetailsWithFallback(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "這個軟體", got.Title, "Taiwan phrase: 软件→軟體")
	assert.Equal(t, "這個軟體的品質", got.Overview, "then the Taiwan lexicon: 質量→品質")

	mock.GetMovieDetailsResponses["zh-CN"].ProductionCountries = []tmdb.Country{{ISO31661: "CN"}}
	got, _, err = c.GetMovieDetailsWithFallback(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "這個軟件的質量", got.Overview, "a mainland film: characters only, its own wording kept")
}
