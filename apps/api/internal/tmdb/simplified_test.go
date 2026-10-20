package tmdb

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── disc-2026-10-metadata-zh-fallback-simplified ───────────────────────────
//
// A fresh library of five Blender films showed 钢铁之泪, 寻龙记 and 大象之梦:
// TMDb has no zh-TW title for them, the chain borrowed zh-CN, and the text was
// stored as sent.

// fakeOpenCC stands in for the OpenCC CLI (simplified_real_test.go runs the
// real one). s2tw swaps characters; s2twp also rewrites a mainland phrase, so
// a test can tell which rule ran. zhtw.Finalize itself is real.
type fakeOpenCC struct {
	err   error
	calls int
}

var fakePairs = []string{"钢", "鋼", "铁", "鐵", "泪", "淚", "寻", "尋", "龙", "龍", "记", "記",
	"梦", "夢", "视", "視", "频", "頻", "华", "華", "说", "說", "这", "這", "个", "個", "电", "電", "剧", "劇", "录", "錄",
	"里", "裡", "杰", "傑", "苏", "蘇"}

var fakeS2TW = strings.NewReplacer(fakePairs...)

var fakeT2S = func() *strings.Replacer {
	rev := make([]string, len(fakePairs))
	for i := 0; i < len(fakePairs); i += 2 {
		rev[i], rev[i+1] = fakePairs[i+1], fakePairs[i]
	}
	return strings.NewReplacer(rev...)
}()

func (f *fakeOpenCC) ConvertS2TW(b []byte) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return b, f.err
	}
	return []byte(fakeS2TW.Replace(string(b))), nil
}

func (f *fakeOpenCC) ConvertT2S(b []byte) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return b, f.err
	}
	return []byte(fakeT2S.Replace(string(b))), nil
}

func (f *fakeOpenCC) ConvertS2TWP(b []byte) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return b, f.err
	}
	return []byte(fakeS2TW.Replace(strings.ReplaceAll(string(b), "视频", "影片"))), nil
}

func zhFallbackClient(mock *MockClient, conv *fakeOpenCC) *LanguageFallbackClient {
	c := NewLanguageFallbackClient(mock, nil)
	if conv != nil {
		c.SetChineseConverter(conv)
	}
	return c
}

func TestMovieDetails_BorrowedSimplifiedIsTraditional(t *testing.T) {
	tos := &MovieDetails{
		Movie:               Movie{ID: 133701, Title: "钢铁之泪", Overview: "这个视频说的是梦"},
		Tagline:             "钢铁",
		ProductionCountries: []Country{{ISO31661: "NL"}},
	}
	mock := &MockClient{GetMovieDetailsResponses: map[string]*MovieDetails{
		"zh-TW": {Movie: Movie{ID: 133701}},
		"zh-CN": tos,
	}}

	got, _, err := zhFallbackClient(mock, &fakeOpenCC{}).GetMovieDetailsWithFallback(context.Background(), 133701)
	require.NoError(t, err)

	assert.Equal(t, "鋼鐵之淚", got.Title)
	assert.Equal(t, "這個影片說的是夢", got.Overview, "a Dutch film gets Taiwan phrases (视频→影片)")
	assert.Equal(t, "鋼鐵", got.Tagline)
	assert.Equal(t, "钢铁之泪", tos.Title, "the client's response is never written into")
}

func TestMovieDetails_MainlandFilmKeepsItsOwnWording(t *testing.T) {
	mock := &MockClient{GetMovieDetailsResponses: map[string]*MovieDetails{
		"zh-TW": {Movie: Movie{ID: 1}},
		"zh-CN": {
			Movie:               Movie{ID: 1, Title: "流浪地球", Overview: "这个视频"},
			ProductionCountries: []Country{{ISO31661: "CN"}},
		},
	}}

	got, _, err := zhFallbackClient(mock, &fakeOpenCC{}).GetMovieDetailsWithFallback(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "這個視頻", got.Overview, "a mainland film keeps 视频 — characters only (project_cn_subtitle_policy)")
}

func TestMovieDetails_ZhTWAndEnglishAreNeverConverted(t *testing.T) {
	conv := &fakeOpenCC{}
	mock := &MockClient{GetMovieDetailsResponses: map[string]*MovieDetails{
		// zh-TW has the title; TMDb's zh-TW text can itself carry a mainland
		// phrase, and s2twp must not get a chance to touch it.
		"zh-TW": {Movie: Movie{ID: 2, Title: "台灣片名 视频", Overview: ""}},
		"zh-CN": {Movie: Movie{ID: 2, Title: "大陆片名", Overview: "这个剧"}},
	}}

	got, _, err := zhFallbackClient(mock, conv).GetMovieDetailsWithFallback(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, "台灣片名 视频", got.Title, "the zh-TW value wins untouched")
	assert.Equal(t, "這個劇", got.Overview, "only the borrowed overview is converted")

	conv.calls = 0
	mock = &MockClient{GetMovieDetailsResponses: map[string]*MovieDetails{
		"zh-TW": {Movie: Movie{ID: 3}},
		"zh-CN": {Movie: Movie{ID: 3}},
		"en":    {Movie: Movie{ID: 3, Title: "Sintel", Overview: "A girl and a dragon."}},
	}}
	got, _, err = zhFallbackClient(mock, conv).GetMovieDetailsWithFallback(context.Background(), 3)
	require.NoError(t, err)
	assert.Equal(t, "Sintel", got.Title)
	assert.Zero(t, conv.calls, "an English answer never reaches OpenCC")
}

// TMDb users type Traditional into zh-CN too; s2twp over it would break a
// name (蘇里→蘇裡), so a field that is not Simplified is left as sent.
func TestMovieDetails_TraditionalTypedIntoZhCNIsLeftAlone(t *testing.T) {
	mock := &MockClient{GetMovieDetailsResponses: map[string]*MovieDetails{
		"zh-TW": {Movie: Movie{ID: 4}},
		"zh-CN": {Movie: Movie{ID: 4, Title: "鋼鐵之淚", Overview: "傑克蘇里"}},
	}}

	got, _, err := zhFallbackClient(mock, &fakeOpenCC{}).GetMovieDetailsWithFallback(context.Background(), 4)
	require.NoError(t, err)
	assert.Equal(t, "鋼鐵之淚", got.Title)
	assert.Equal(t, "傑克蘇里", got.Overview)
}

func TestTVShowDetails_BorrowedSimplifiedIsTraditional(t *testing.T) {
	mock := &MockClient{GetTVShowDetailsResponses: map[string]*TVShowDetails{
		"zh-TW": {TVShow: TVShow{ID: 9}},
		"zh-CN": {TVShow: TVShow{ID: 9, Name: "梦华录", Overview: "这个电视剧"}, Tagline: "梦"},
	}}

	got, _, err := zhFallbackClient(mock, &fakeOpenCC{}).GetTVShowDetailsWithFallback(context.Background(), 9)
	require.NoError(t, err)
	assert.Equal(t, "夢華錄", got.Name)
	assert.Equal(t, "這個電視劇", got.Overview)
	assert.Equal(t, "夢", got.Tagline)
}

func TestSeasonDetails_BorrowedEpisodeNamesAreTraditional(t *testing.T) {
	zhCN := &SeasonDetails{Name: "第一季", Overview: "这个", Episodes: []EpisodeInfo{
		{EpisodeNumber: 1, Name: "寻龙", Overview: "视频"},
		{EpisodeNumber: 2, Name: "梦", Overview: ""},
	}}
	mock := &MockClient{GetSeasonDetailsResponses: map[string]*SeasonDetails{
		"zh-TW": {Episodes: []EpisodeInfo{{EpisodeNumber: 1}, {EpisodeNumber: 2}}},
		"zh-CN": zhCN,
	}}

	got, _, err := zhFallbackClient(mock, &fakeOpenCC{}).GetSeasonDetailsWithFallback(context.Background(), 9, 1)
	require.NoError(t, err)
	assert.Equal(t, "這個", got.Overview)
	assert.Equal(t, "尋龍", got.Episodes[0].Name)
	assert.Equal(t, "視頻", got.Episodes[0].Overview, "a season names no country: characters only, never a rewritten phrase")
	assert.Equal(t, "夢", got.Episodes[1].Name)
	assert.Equal(t, "寻龙", zhCN.Episodes[0].Name, "the client's episodes are never written into")
}

func TestSearchMovies_ListBorrowedFromZhCNIsTraditional(t *testing.T) {
	mock := &MockClient{SearchMoviesResponses: map[string]*SearchResultMovies{
		"zh-TW": {Results: []Movie{{ID: 1, Title: "", Overview: ""}}},
		"zh-CN": {Results: []Movie{
			{ID: 1, Title: "寻龙记", Overview: "视频", OriginalLanguage: "en"},
			{ID: 2, Title: "流浪地球", Overview: "视频", OriginalLanguage: "zh"},
		}},
	}}

	got, lang, err := zhFallbackClient(mock, &fakeOpenCC{}).SearchMoviesWithFallback(context.Background(), "sintel", 1)
	require.NoError(t, err)
	assert.Equal(t, "zh-CN", lang)
	assert.Equal(t, "尋龍記", got.Results[0].Title)
	assert.Equal(t, "影片", got.Results[0].Overview, "a foreign film gets Taiwan phrases")
	assert.Equal(t, "視頻", got.Results[1].Overview, "a Chinese-language film keeps its wording")
}

func TestTVLists_OriginCountryDecidesTheWording(t *testing.T) {
	mock := &MockClient{TVSimilarResponses: map[string]*SearchResultTVShows{
		"zh-CN": {Results: []TVShow{
			{ID: 1, Name: "梦", Overview: "视频", OriginCountry: []string{"HK"}},
			{ID: 2, Name: "梦", Overview: "视频", OriginCountry: []string{"US"}},
		}},
	}}

	got, _, err := zhFallbackClient(mock, &fakeOpenCC{}).GetTVSimilarWithFallback(context.Background(), 5)
	require.NoError(t, err)
	assert.Equal(t, "視頻", got.Results[0].Overview)
	assert.Equal(t, "影片", got.Results[1].Overview)
}

func TestSimplified_NoConverterOrFailureKeepsTheText(t *testing.T) {
	newMock := func() *MockClient {
		return &MockClient{GetMovieDetailsResponses: map[string]*MovieDetails{
			"zh-TW": {Movie: Movie{ID: 1}},
			"zh-CN": {Movie: Movie{ID: 1, Title: "钢铁之泪", Overview: "这个"}},
		}}
	}

	got, _, err := zhFallbackClient(newMock(), nil).GetMovieDetailsWithFallback(context.Background(), 1)
	require.NoError(t, err, "a missing converter never fails the lookup")
	assert.Equal(t, "钢铁之泪", got.Title)

	got, _, err = zhFallbackClient(newMock(), &fakeOpenCC{err: errors.New("opencc: exit status 1")}).
		GetMovieDetailsWithFallback(context.Background(), 1)
	require.NoError(t, err, "a failed conversion never fails the lookup")
	assert.Equal(t, "钢铁之泪", got.Title, "on failure the original text is kept, not half a conversion")
}

func TestIsSimplifiedLanguage(t *testing.T) {
	for lang, want := range map[string]bool{
		"zh-CN": true, "zh-cn": true, "zh-SG": true, "zh-Hans": true,
		"zh-TW": false, "zh-HK": false, "en": false, "": false,
	} {
		assert.Equal(t, want, isSimplifiedLanguage(lang), lang)
	}
}
