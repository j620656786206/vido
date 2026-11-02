package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/providers"
)

// disc-2026-11-online-subs-for-episodes

type queryCapturingProvider struct {
	handlerMockProvider
	got providers.SubtitleQuery
}

func (p *queryCapturingProvider) Search(_ context.Context, q providers.SubtitleQuery) ([]providers.SubtitleResult, error) {
	p.got = q
	return p.searchResult, p.searchErr
}

type lookupMovies map[string]*models.Movie

func (m lookupMovies) FindByID(_ context.Context, id string) (*models.Movie, error) {
	if v, ok := m[id]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}

type lookupSeries map[string]*models.Series

func (m lookupSeries) FindByID(_ context.Context, id string) (*models.Series, error) {
	if v, ok := m[id]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}

type lookupEpisodes struct {
	eps        map[string]*models.Episode
	lastID     string
	lastStatus models.SubtitleStatus
	lastPath   string
}

func (l *lookupEpisodes) FindByID(_ context.Context, id string) (*models.Episode, error) {
	if v, ok := l.eps[id]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}

func (l *lookupEpisodes) UpdateEpisodeSubtitleStatus(_ context.Context, id string, status models.SubtitleStatus, path, _ string) error {
	l.lastID, l.lastStatus, l.lastPath = id, status, path
	return nil
}

func lookupFixture() ManualSubtitleLookup {
	return ManualSubtitleLookup{
		Movies: lookupMovies{
			"m-en": {Title: "全面啟動", OriginalTitle: models.NewNullString("Inception"), ReleaseDate: "2010-07-15"},
			"m-jp": {Title: "你的名字", OriginalTitle: models.NewNullString("君の名は。"), ReleaseDate: "2016-08-26"},
			"m-tw": {Title: "陽光普照", ReleaseDate: "2019-10-25"},
		},
		Series: lookupSeries{
			"s-scorpion": {Title: "天蠍行動", OriginalTitle: models.NewNullString("Scorpion"), FirstAirDate: "2014-09-22"},
		},
		Episodes: &lookupEpisodes{eps: map[string]*models.Episode{
			"e-0102": {SeriesID: "s-scorpion", SeasonNumber: 1, EpisodeNumber: 2},
		}},
	}
}

func searchWith(t *testing.T, prov providers.SubtitleProvider, lookup ManualSubtitleLookup, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewSubtitleHandler([]providers.SubtitleProvider{prov},
		subtitle.NewScorer(subtitle.NewDefaultScorerConfig()), nil, subtitle.NewPlacer(subtitle.DefaultPlacerConfig()), nil, nil, nil)
	h.SetMediaLookup(lookup)
	router := gin.New()
	h.RegisterRoutes(router.Group("/api/v1"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/search", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func TestSearch_BuildsQueryFromLibrary(t *testing.T) {
	cases := []struct {
		name string
		body string
		want providers.SubtitleQuery
	}{
		{"movie: original title + year", `{"media_id":"m-en","media_type":"movie"}`, providers.SubtitleQuery{Title: "Inception", Year: 2010}},
		{"movie: kana original falls back to the Chinese title", `{"media_id":"m-jp","media_type":"movie"}`, providers.SubtitleQuery{Title: "你的名字", Year: 2016}},
		{"movie: no original title", `{"media_id":"m-tw","media_type":"movie"}`, providers.SubtitleQuery{Title: "陽光普照", Year: 2019}},
		{"series: original title, no year", `{"media_id":"s-scorpion","media_type":"series"}`, providers.SubtitleQuery{Title: "Scorpion"}},
		{"episode: show title + season + episode", `{"media_id":"e-0102","media_type":"episode"}`, providers.SubtitleQuery{Title: "Scorpion", Season: 1, Episode: 2}},
		{"episode: typed text keeps season/episode", `{"media_id":"e-0102","media_type":"episode","query":"天蝎"}`, providers.SubtitleQuery{Title: "天蝎", Season: 1, Episode: 2}},
		{"movie: typed text wins", `{"media_id":"m-en","media_type":"movie","query":"盗梦空间"}`, providers.SubtitleQuery{Title: "盗梦空间"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prov := &queryCapturingProvider{handlerMockProvider: handlerMockProvider{name: "assrt"}}
			w := searchWith(t, prov, lookupFixture(), tc.body)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Equal(t, tc.want, prov.got)
		})
	}
}

func TestSearch_NoQueryAndUnknownMediaIs400(t *testing.T) {
	prov := &queryCapturingProvider{handlerMockProvider: handlerMockProvider{name: "assrt"}}
	w := searchWith(t, prov, lookupFixture(), `{"media_id":"nope","media_type":"movie"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, prov.got.Title, "never searches for the media id")
}

func TestSearch_EpisodeDropsOtherEpisodes(t *testing.T) {
	prov := &queryCapturingProvider{handlerMockProvider: handlerMockProvider{name: "assrt", searchResult: []providers.SubtitleResult{
		{ID: "pack", Source: "assrt", Filename: "天蝎 第一季 WEB精校版（全）/Scorpion S01E01-E22", Language: "zh-Hant"},
		{ID: "e19", Source: "assrt", Filename: "天蝎 第一季 第十九集/Scorpion S01 E19", Language: "zh-Hant"},
		{ID: "e02", Source: "assrt", Filename: "Scorpion.S01E02.chs.srt", Language: "zh-Hans"},
		{ID: "s2", Source: "assrt", Filename: "Scorpion.S02E02.srt", Language: "zh-Hans"},
	}}}
	w := searchWith(t, prov, lookupFixture(), `{"media_id":"e-0102","media_type":"episode"}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data []SubtitleSearchResultDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	var ids []string
	for _, r := range resp.Data {
		ids = append(ids, r.ID)
	}
	assert.ElementsMatch(t, []string{"pack", "e02"}, ids)
}

func TestDownload_EpisodeRecordsTheEpisode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mediaPath := t.TempDir() + "/Scorpion.S01E02.mkv"
	require.NoError(t, os.WriteFile(mediaPath, []byte("fake"), 0o644))

	prov := &handlerMockProvider{name: "assrt", downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n測試字幕\n")}
	lookup := lookupFixture()
	h := NewSubtitleHandler([]providers.SubtitleProvider{prov},
		subtitle.NewScorer(subtitle.NewDefaultScorerConfig()), nil, subtitle.NewPlacer(subtitle.DefaultPlacerConfig()), nil, nil, nil)
	h.SetMediaLookup(lookup)
	router := gin.New()
	h.RegisterRoutes(router.Group("/api/v1"))

	body, _ := json.Marshal(SubtitleDownloadRequest{
		MediaID: "e-0102", MediaType: "episode", MediaFilePath: mediaPath, SubtitleID: "1", Provider: "assrt",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	eps := lookup.Episodes.(*lookupEpisodes)
	assert.Equal(t, "e-0102", eps.lastID)
	assert.Equal(t, models.SubtitleStatusFound, eps.lastStatus)
	assert.NotEmpty(t, eps.lastPath)
}

func TestCountriesFor_EpisodeUsesItsShow(t *testing.T) {
	h := NewSubtitleHandler(nil, nil, nil, nil, nil, nil, nil)
	h.SetMediaLookup(lookupFixture())
	var gotType, gotID string
	h.SetCountryResolver(func(_ context.Context, mediaType, mediaID string) ([]string, error) {
		gotType, gotID = mediaType, mediaID
		return []string{"US"}, nil
	})

	assert.Equal(t, []string{"US"}, h.countriesFor(context.Background(), "episode", "e-0102"))
	assert.Equal(t, "series", gotType)
	assert.Equal(t, "s-scorpion", gotID)
	assert.Nil(t, h.countriesFor(context.Background(), "episode", "unknown"))
}

// disc-2026-11-online-sub-download-progress

type progressReportingProvider struct{ handlerMockProvider }

func (p *progressReportingProvider) DownloadFor(ctx context.Context, _ string, _ string) ([]byte, error) {
	providers.ReportProgress(ctx, 0.5)
	providers.ReportProgress(ctx, 1)
	return p.downloadData, nil
}

func TestDownload_BroadcastsPercent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mediaPath := t.TempDir() + "/Film.mkv"
	require.NoError(t, os.WriteFile(mediaPath, []byte("fake"), 0o644))

	hub := sse.NewHub()
	var percents []int
	hub.AddObserver(func(ev sse.Event) {
		if m, ok := ev.Data.(map[string]any); ok {
			if p, ok := m["percent"].(int); ok {
				percents = append(percents, p)
			}
		}
	})
	prov := &progressReportingProvider{handlerMockProvider{name: "assrt", downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\nhello\n")}}
	h := NewSubtitleHandler([]providers.SubtitleProvider{prov},
		subtitle.NewScorer(subtitle.NewDefaultScorerConfig()), nil, subtitle.NewPlacer(subtitle.DefaultPlacerConfig()), hub, nil, nil)
	router := gin.New()
	h.RegisterRoutes(router.Group("/api/v1"))

	body, _ := json.Marshal(SubtitleDownloadRequest{MediaID: "m", MediaType: "movie", MediaFilePath: mediaPath, SubtitleID: "1", Provider: "assrt"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []int{5, 45, 85, 95, 100}, percents)
}
