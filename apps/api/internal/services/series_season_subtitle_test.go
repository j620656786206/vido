package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// disc-2026-10-episode-list-subtitle-badge-a AC #1–#4 through GetSeasonEpisodes.

type refreshingEpisodeRepo struct {
	stubEpisodeRepo
	refreshed map[string][2]string // id → {previous, next}
}

func (r *refreshingEpisodeRepo) RefreshSubtitleTracks(_ context.Context, id, previous, next string) (bool, error) {
	if r.refreshed == nil {
		r.refreshed = map[string][2]string{}
	}
	r.refreshed[id] = [2]string{previous, next}
	return true, nil
}

func TestSeriesService_GetSeasonEpisodes_ChineseSubtitle(t *testing.T) {
	series := newSeasonSeries(t, 85421)
	details := &tmdb.SeasonDetails{ID: 1, SeasonNumber: 1, Episodes: []tmdb.EpisodeInfo{
		{EpisodeNumber: 1, Name: "神之火"}, {EpisodeNumber: 2, Name: "瓶中信"}, {EpisodeNumber: 3, Name: "新的血脈"},
	}}
	e1Path, e2Path := "/tv/See/S1/See.S01E01.mkv", "/tv/See/S1/See.S01E02.mkv"
	local := []models.Episode{
		{ID: "e1", SeasonNumber: 1, EpisodeNumber: 1, FilePath: models.NewNullString(e1Path),
			SubtitleStatus: models.SubtitleStatusNotSearched, SubtitleTracks: tracksJSON(t, engEmbedded)},
		{ID: "e2", SeasonNumber: 1, EpisodeNumber: 2, FilePath: models.NewNullString(e2Path),
			SubtitleStatus: models.SubtitleStatusNotSearched, SubtitleTracks: tracksJSON(t, engEmbedded)},
	}
	repo := &refreshingEpisodeRepo{stubEpisodeRepo: stubEpisodeRepo{episodes: local}}
	side := &fakeSidecars{out: map[string]SidecarTracks{
		e1Path: {Tracks: []SubtitleTrack{}},
		e2Path: {Tracks: []SubtitleTrack{sidecar("zh-Hant", "See.S01E02.zh-TW.srt")}},
	}}

	svc := NewSeriesService(seriesRepoReturning(series))
	svc.SetEpisodeDeps(repo, &stubSeasonProvider{details: details})
	svc.SetSidecarReader(side)

	resp, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	require.NoError(t, err)
	require.Len(t, resp.Episodes, 3)

	require.Len(t, side.calls, 1, "one call for the whole season (AC #3)")
	assert.Equal(t, []string{e1Path, e2Path}, side.calls[0])
	assert.Contains(t, side.known[0], e1Path, "stored sidecars are handed over so unchanged files are not reopened")

	assert.Equal(t, models.ChineseSubtitleNone, resp.Episodes[0].ChineseSubtitle)
	require.NotNil(t, resp.Episodes[0].ChineseSubtitleSources)
	assert.Empty(t, *resp.Episodes[0].ChineseSubtitleSources)

	assert.Equal(t, models.ChineseSubtitleZhHant, resp.Episodes[1].ChineseSubtitle)
	assert.Equal(t, []ChineseSubtitleSource{{Kind: ChineseSourceSidecar, Language: "zh-Hant", Label: "zh-TW"}},
		*resp.Episodes[1].ChineseSubtitleSources)

	// No local file → neither field (the client hides the icon).
	assert.Empty(t, resp.Episodes[2].ChineseSubtitle)
	assert.Nil(t, resp.Episodes[2].ChineseSubtitleSources)

	// Only e2's sidecars changed → only e2 is written back, compare-and-set
	// against what this request read (AC #4).
	require.Len(t, repo.refreshed, 1)
	assert.Equal(t, local[1].SubtitleTracks.String, repo.refreshed["e2"][0])
	assert.Equal(t, models.ChineseSubtitleZhHant, models.ChineseSubtitleVerdict("not_searched", "", repo.refreshed["e2"][1]),
		"what is stored now gives the library filter the same answer")
}

func TestSeriesService_GetSeasonEpisodes_NoSidecarReaderUsesStored(t *testing.T) {
	series := newSeasonSeries(t, 85421)
	details := &tmdb.SeasonDetails{ID: 1, SeasonNumber: 1, Episodes: []tmdb.EpisodeInfo{{EpisodeNumber: 1}}}
	local := []models.Episode{{ID: "e1", SeasonNumber: 1, EpisodeNumber: 1, FilePath: models.NewNullString("/m/X.S01E01.mkv"),
		SubtitleStatus: models.SubtitleStatusNotSearched}}
	svc := newSeasonServiceUnderTest(series, local, nil, details, nil)

	resp, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleUnknown, resp.Episodes[0].ChineseSubtitle, "never read and nothing beside it known → unknown")
}
