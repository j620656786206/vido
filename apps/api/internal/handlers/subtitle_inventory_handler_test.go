package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

type invMovies struct {
	movie *models.Movie
	err   error
}

func (f invMovies) FindByID(context.Context, string) (*models.Movie, error) { return f.movie, f.err }

type invEpisodes struct {
	episode *models.Episode
	err     error
}

func (f invEpisodes) FindByID(context.Context, string) (*models.Episode, error) {
	return f.episode, f.err
}

type invProber struct {
	available bool
	tracks    []services.SubtitleTrack
}

func (p invProber) IsAvailable() bool { return p.available }
func (p invProber) Probe(context.Context, string) (*services.MediaTechInfo, error) {
	return &services.MediaTechInfo{SubtitleTracks: p.tracks}, nil
}

// seEpisodeDir mirrors the NAS case that started this story: See S01E02 has
// embedded English tracks and a .zh-TW.srt beside it, and the dialog said
// 「此影片目前沒有任何字幕軌」.
func seeEpisodeDir(t *testing.T) (dir, media string) {
	t.Helper()
	dir = t.TempDir()
	media = filepath.Join(dir, "See.S01E02.mkv")
	require.NoError(t, os.WriteFile(media, []byte("video"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "See.S01E02.zh-TW.srt"),
		[]byte("1\n00:00:01,000 --> 00:00:03,000\n艾肯尼，這是我們的家園，我們團結一致\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "See.S01E02.zh-Hant.srt"),
		[]byte("1\n00:00:01,000 --> 00:00:03,000\n艾肯尼，這是我們的家園，我們團結一致\n"), 0o644))
	return dir, media
}

func getInventory(t *testing.T, h *SubtitleInventoryHandler, path string) (*httptest.ResponseRecorder, subtitle.Inventory) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var resp struct {
		Data subtitle.Inventory `json:"data"`
	}
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	}
	return w, resp.Data
}

func TestSubtitleInventory_Episode(t *testing.T) {
	dir, media := seeEpisodeDir(t)
	episode := &models.Episode{
		ID:           "ep-2",
		FilePath:     models.NewNullString(media),
		SubtitlePath: models.NewNullString(filepath.Join(dir, "See.S01E02.zh-Hant.srt")),
	}
	h := NewSubtitleInventoryHandler(nil, invEpisodes{episode: episode}, invProber{available: true, tracks: []services.SubtitleTrack{
		{Language: "eng", Format: "subrip", StreamIndex: 8, Title: "English"},
		{Language: "eng", Format: "subrip", StreamIndex: 9, Title: "English  [SDH]"},
	}})

	w, inv := getInventory(t, h, "/api/v1/episodes/ep-2/subtitles/inventory")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, subtitle.InventoryOK, inv.Sidecars.Status)
	require.Len(t, inv.Sidecars.Files, 2)
	assert.Equal(t, "See.S01E02.zh-Hant.srt", inv.Sidecars.Files[0].FileName)
	assert.True(t, inv.Sidecars.Files[0].IsVidoOutput)
	assert.Equal(t, subtitle.LangTraditional, inv.Sidecars.Files[1].Language)
	assert.False(t, inv.Sidecars.Files[1].IsVidoOutput)
	require.Len(t, inv.Embedded.Tracks, 2)
	assert.Equal(t, "en", inv.Embedded.Tracks[0].Language)
}

func TestSubtitleInventory_MovieWithoutFFprobeStillListsFiles(t *testing.T) {
	_, media := seeEpisodeDir(t)
	h := NewSubtitleInventoryHandler(invMovies{movie: &models.Movie{FilePath: models.NewNullString(media)}}, nil, nil)

	w, inv := getInventory(t, h, "/api/v1/movies/m1/subtitles/inventory")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, subtitle.InventoryUnavailable, inv.Embedded.Status)
	assert.Len(t, inv.Sidecars.Files, 2)
}

func TestSubtitleInventory_Errors(t *testing.T) {
	notFoundMovie := fmt.Errorf("movie with id x not found: %w", sql.ErrNoRows)
	cases := []struct {
		name string
		h    *SubtitleInventoryHandler
		path string
		code int
	}{
		{"movie missing", NewSubtitleInventoryHandler(invMovies{err: notFoundMovie}, nil, nil), "/api/v1/movies/x/subtitles/inventory", http.StatusNotFound},
		{"episode missing", NewSubtitleInventoryHandler(nil, invEpisodes{err: repository.ErrEpisodeNotFound}, nil), "/api/v1/episodes/x/subtitles/inventory", http.StatusNotFound},
		{"db down", NewSubtitleInventoryHandler(nil, invEpisodes{err: errors.New("locked")}, nil), "/api/v1/episodes/x/subtitles/inventory", http.StatusInternalServerError},
		{"no file path", NewSubtitleInventoryHandler(nil, invEpisodes{episode: &models.Episode{}}, nil), "/api/v1/episodes/x/subtitles/inventory", http.StatusBadRequest},
		{"file gone", NewSubtitleInventoryHandler(nil, invEpisodes{episode: &models.Episode{FilePath: models.NewNullString("/nope/x.mkv")}}, nil), "/api/v1/episodes/x/subtitles/inventory", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := getInventory(t, tc.h, tc.path)
			assert.Equal(t, tc.code, w.Code, w.Body.String())
		})
	}
}
