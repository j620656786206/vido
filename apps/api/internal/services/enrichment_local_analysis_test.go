package services

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// 2026-08-31 內測實測(Alexyu Synology):沒有 TMDB key 時,metadata 失敗的
// 提前 return 餓死了整條本地分析 — 技術徽章、內嵌字幕軌、以及一個就躺在影片
// 旁邊的 .srt 全部沒被偵測(「TMDB 變相必填」)。這批測試釘住脫鉤後的行為:
// 本地分析(ffprobe + 外掛字幕偵測)不依賴任何外部服務。

// newMovieWithSidecar lays out <tmp>/Movie (2005)/movie.mkv + movie.srt and
// returns a scanner-shaped row pointing at the mkv.
func newMovieWithSidecar(t *testing.T) *models.Movie {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "Movie (2005)")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	mkv := filepath.Join(dir, "Movie.2005.1080p.mkv")
	require.NoError(t, os.WriteFile(mkv, []byte("not a real mkv"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.2005.1080p.srt"), []byte("1\n00:00:01,000 --> 00:00:02,000\nhi\n"), 0o644))

	return &models.Movie{
		ID:          "m-local-1",
		Title:       "Movie.2005.1080p.mkv",
		FilePath:    models.NewNullString(mkv),
		ParseStatus: models.ParseStatusPending,
	}
}

func TestApplyFFprobeTechInfo_SidecarDetectionNeedsNoProbe(t *testing.T) {
	// ffprobeService nil = the binary/service is entirely absent. The sidecar
	// .srt is a pure filesystem fact and must still be recorded.
	s := &EnrichmentService{logger: slog.Default()}
	movie := newMovieWithSidecar(t)

	s.applyFFprobeTechInfo(context.Background(), movie)

	require.True(t, movie.SubtitleTracks.Valid, "sidecar subtitle must be detected without ffprobe")
	assert.Contains(t, movie.SubtitleTracks.String, `"format":"srt"`)
	assert.Contains(t, movie.SubtitleTracks.String, `"external":true`)
}

func TestPersistFailedWithLocalAnalysis_KeepsLocalFindings(t *testing.T) {
	// The metadata line failed (no TMDB key) — the persisted row must still
	// carry everything the box could see with its own eyes.
	repo := &mockMovieRepoForNFO{}
	s := &EnrichmentService{movieRepo: repo, logger: slog.Default()}
	movie := newMovieWithSidecar(t)

	err := s.persistFailedWithLocalAnalysis(context.Background(), movie)

	require.NoError(t, err)
	require.NotNil(t, repo.updatedMovie, "the narrow enriched-metadata writer must be used")
	assert.Equal(t, models.ParseStatusFailed, repo.updatedMovie.ParseStatus)
	assert.True(t, repo.updatedMovie.SubtitleTracks.Valid,
		"metadata failure must not starve subtitle-track detection")
	assert.Contains(t, repo.updatedMovie.SubtitleTracks.String, `"format":"srt"`)
}
