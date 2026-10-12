package models

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-10-genre-names-simplified

func TestCanonicalGenres(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"TMDb Simplified (what zh-TW returns)", []string{"剧情", "动作冒险", "动画"}, []string{"劇情", "動作冒險", "動畫"}},
		{"TMDb English-only TV genres", []string{"Sci-Fi & Fantasy", "War & Politics"}, []string{"科幻奇幻", "戰爭政治"}},
		{"en-US names, any case", []string{"Drama", "SCIENCE FICTION", "tv movie"}, []string{"劇情", "科幻", "電視電影"}},
		{"old editor keys", []string{"drama", "sci-fi", "documentary"}, []string{"劇情", "科幻", "紀錄"}},
		{"Taiwan names stay", []string{"劇情", "犯罪", "真人秀"}, []string{"劇情", "犯罪", "真人秀"}},
		{"unknown names kept, trimmed", []string{" 武俠 ", "传记"}, []string{"武俠", "传记"}},
		{"duplicates collapse to the first", []string{"剧情", "劇情", "Drama", "动作"}, []string{"劇情", "動作"}},
		{"empty names dropped", []string{"", "  ", "喜剧"}, []string{"喜劇"}},
		{"nil", nil, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanonicalGenres(tt.in)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenresJSON_StoresTaiwanNames(t *testing.T) {
	m := Movie{Genres: []string{"剧情", "Sci-Fi & Fantasy", "劇情"}}
	js, err := m.GenresJSON()
	require.NoError(t, err)
	assert.Equal(t, `["劇情","科幻奇幻"]`, js)
	assert.Equal(t, []string{"劇情", "科幻奇幻"}, m.Genres, "the model matches what is stored")

	s := Series{Genres: []string{"动画", "War & Politics"}}
	js, err = s.GenresJSON()
	require.NoError(t, err)
	assert.Equal(t, `["動畫","戰爭政治"]`, js)

	var empty Series
	js, err = empty.GenresJSON()
	require.NoError(t, err)
	assert.Equal(t, `[]`, js)
}

// Every alias belongs to an id the table names.
func TestGenreAliases_AllIDsKnown(t *testing.T) {
	for id := range genreAliases {
		_, ok := GenreNamesByTMDbID[id]
		assert.True(t, ok, "alias id %d has no Taiwan name", id)
	}
}

// The web app's GENRE_MAP and this table are one list in two languages: the
// editor offers the web names and the library stores these, so a drift would
// split one genre into two again.
func TestGenreNames_MatchWebGenreMap(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	src, err := os.ReadFile(filepath.Join(filepath.Dir(here), "../../../web/src/lib/genres.ts"))
	require.NoError(t, err)

	block := regexp.MustCompile(`(?s)GENRE_MAP[^{]*\{(.*?)\n\};`).FindSubmatch(src)
	require.NotNil(t, block, "GENRE_MAP not found in genres.ts")
	web := map[int]string{}
	for _, m := range regexp.MustCompile(`(?m)^\s*(\d+):\s*'([^']+)'`).FindAllSubmatch(block[1], -1) {
		id, err := strconv.Atoi(string(m[1]))
		require.NoError(t, err)
		web[id] = string(m[2])
	}
	require.Len(t, web, 27)
	assert.Equal(t, web, GenreNamesByTMDbID)
}
