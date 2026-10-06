package handlers

// Story disc-2026-10-subtitle-filter-disagrees-with-badges AC #5 (handler leg,
// Rule 28 real shape): GET /library?chinese_subtitle=missing and
// /library/search through the REAL handler → LibraryService → repositories on
// a migrated sqlite file. Every returned item must carry the badge verdict the
// filter selected on, and a bad value is the documented 400.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

type chineseSubtitleWireItem struct {
	Type  string `json:"type"`
	Movie *struct {
		ID              string `json:"id"`
		ChineseSubtitle string `json:"chinese_subtitle"`
	} `json:"movie"`
	Series *struct {
		ID              string `json:"id"`
		ChineseSubtitle string `json:"chinese_subtitle"`
	} `json:"series"`
}

func TestLibraryChineseSubtitleFilter_RealHandlerPath(t *testing.T) {
	db := setupRouteCTestDB(t)
	ctx := context.Background()
	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	libSvc := services.NewLibraryService(movieRepo, seriesRepo, repository.NewEpisodeRepository(db))
	router := setupLibraryTestRouter(NewLibraryHandler(libSvc))

	set := func(table, id string, status string, lang, tracks interface{}) {
		_, err := db.ExecContext(ctx,
			"UPDATE "+table+" SET subtitle_status = ?, subtitle_language = ?, subtitle_tracks = ? WHERE id = ?",
			status, lang, tracks, id)
		require.NoError(t, err)
	}
	movie := func(id, status string, lang, tracks interface{}) {
		require.NoError(t, movieRepo.Create(ctx, &models.Movie{ID: id, Title: "Shape " + id, ReleaseDate: "2020-01-01"}))
		set("movies", id, status, lang, tracks)
	}
	movie("m-chi", "not_searched", nil, `[{"language":"chi","format":"subrip","external":false,"stream_index":2}]`)
	movie("m-eng", "not_searched", nil, `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`)
	movie("m-notfound", "not_found", nil, nil)
	movie("m-null", "not_searched", nil, nil)
	require.NoError(t, seriesRepo.Create(ctx, &models.Series{ID: "s-yue", Title: "Shape s-yue", FirstAirDate: "2020-01-01"}))
	set("series", "s-yue", "not_searched", nil, `[{"language":"yue","format":"subrip","external":false,"stream_index":2}]`)

	get := func(url string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", url, nil)
		router.ServeHTTP(w, req)
		return w
	}
	idsWithVerdict := func(t *testing.T, items []chineseSubtitleWireItem) []string {
		var ids []string
		for _, it := range items {
			switch {
			case it.Movie != nil:
				assert.Equal(t, "none", it.Movie.ChineseSubtitle, "movie %s", it.Movie.ID)
				ids = append(ids, it.Movie.ID)
			case it.Series != nil:
				assert.Equal(t, "none", it.Series.ChineseSubtitle, "series %s", it.Series.ID)
				ids = append(ids, it.Series.ID)
			}
		}
		sort.Strings(ids)
		return ids
	}
	want := []string{"m-eng", "m-notfound", "s-yue"}

	t.Run("/library?chinese_subtitle=missing returns exactly the 缺中文 titles", func(t *testing.T) {
		w := get("/api/v1/library?chinese_subtitle=missing")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Data struct {
				Items []chineseSubtitleWireItem `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, want, idsWithVerdict(t, body.Data.Items))
	})

	t.Run("/library/search applies the same filter", func(t *testing.T) {
		w := get("/api/v1/library/search?q=Shape&chinese_subtitle=missing")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Data struct {
				Results []chineseSubtitleWireItem `json:"results"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, want, idsWithVerdict(t, body.Data.Results))
	})

	t.Run("every list item carries chinese_subtitle, even unfiltered", func(t *testing.T) {
		w := get("/api/v1/library")
		require.Equal(t, http.StatusOK, w.Code)
		// Decode loosely: items[i]["movie"|"series"]["chinese_subtitle"].
		var loose struct {
			Data struct {
				Items []map[string]json.RawMessage `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loose))
		require.Len(t, loose.Data.Items, 5)
		for _, it := range loose.Data.Items {
			var typ string
			require.NoError(t, json.Unmarshal(it["type"], &typ))
			var obj map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(it[typ], &obj))
			_, ok := obj["chinese_subtitle"]
			assert.True(t, ok, "%s item lacks chinese_subtitle", typ)
		}
	})

	t.Run("bad value is the documented 400", func(t *testing.T) {
		for _, url := range []string{"/api/v1/library?chinese_subtitle=lots", "/api/v1/library/search?q=Shape&chinese_subtitle=lots"} {
			w := get(url)
			assert.Equal(t, http.StatusBadRequest, w.Code, url)
			assert.Contains(t, w.Body.String(), "VALIDATION_INVALID_FORMAT", url)
			assert.Contains(t, w.Body.String(), `chinese_subtitle contains unknown value \"lots\"`, url)
		}
	})
}
