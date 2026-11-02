package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// disc-2026-11-burned-in-subs-c1 AC #1/#2: the season list carries the
// burned-in answer, and a Chinese answer is listed as the last source, kind
// "burned", in the same language vocabulary as every other source.
func TestSeriesService_GetSeasonEpisodes_BurnedIn(t *testing.T) {
	series := newSeasonSeries(t, 85421)
	var tmdbEps []tmdb.EpisodeInfo
	for i := 1; i <= 5; i++ {
		tmdbEps = append(tmdbEps, tmdb.EpisodeInfo{EpisodeNumber: i})
	}
	details := &tmdb.SeasonDetails{ID: 1, SeasonNumber: 1, Episodes: tmdbEps}
	ep := func(n int, burned string, tracks models.NullString) models.Episode {
		e := models.Episode{ID: "e" + string(rune('0'+n)), SeasonNumber: 1, EpisodeNumber: n,
			FilePath:       models.NewNullString("/tv/x/" + string(rune('0'+n)) + ".mkv"),
			SubtitleStatus: models.SubtitleStatusNotSearched, SubtitleTracks: tracks}
		if burned != "" {
			e.BurnedInSubtitle = models.NewNullString(burned)
		}
		return e
	}
	empty := models.NewNullString("[]")
	withZhTW := tracksJSON(t, sidecar("zh-Hant", "4.zh-TW.srt"))
	local := []models.Episode{
		ep(1, models.BurnedInZhHans, empty),
		ep(2, models.BurnedInNone, empty),
		ep(3, "", empty), // never checked
		ep(4, models.BurnedInZhHans, withZhTW),
		ep(5, models.BurnedInZh, empty),
	}
	svc := NewSeriesService(seriesRepoReturning(series))
	svc.SetEpisodeDeps(&stubEpisodeRepo{episodes: local}, &stubSeasonProvider{details: details})

	resp, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	require.NoError(t, err)
	require.Len(t, resp.Episodes, 5)
	got := resp.Episodes

	assert.Equal(t, "zh-Hans", got[0].BurnedInSubtitle)
	assert.Equal(t, models.ChineseSubtitleZhHans, got[0].ChineseSubtitle)
	assert.Equal(t, []ChineseSubtitleSource{{Kind: ChineseSourceBurned, Language: "zh-Hans"}}, *got[0].ChineseSubtitleSources)

	assert.Equal(t, "none", got[1].BurnedInSubtitle, "none is exposed — the dialog needs it to hide the row")
	assert.Empty(t, *got[1].ChineseSubtitleSources, "none is not a source")

	assert.Empty(t, got[2].BurnedInSubtitle)
	assert.Empty(t, *got[2].ChineseSubtitleSources)

	assert.Equal(t, []ChineseSubtitleSource{
		{Kind: ChineseSourceSidecar, Language: "zh-Hant", Label: "zh-TW"},
		{Kind: ChineseSourceBurned, Language: "zh-Hans"},
	}, *got[3].ChineseSubtitleSources, "burned comes last")

	assert.Equal(t, []ChineseSubtitleSource{{Kind: ChineseSourceBurned, Language: "zh-unknown"}}, *got[4].ChineseSubtitleSources,
		"untold script uses the sources' zh-unknown, like an untagged chi track")

	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	var wire struct {
		Episodes []map[string]any `json:"episodes"`
	}
	require.NoError(t, json.Unmarshal(raw, &wire))
	assert.Equal(t, "zh-Hans", wire.Episodes[0]["burned_in_subtitle"])
	assert.NotContains(t, wire.Episodes[2], "burned_in_subtitle", "never checked → key absent")
}

// The episode model serialises the answer (single-episode reads) but never
// the check time.
func TestEpisodeJSON_BurnedInSubtitle(t *testing.T) {
	e := models.Episode{ID: "e1", BurnedInSubtitle: models.NewNullString("zh-Hant"),
		BurnedInCheckedAt: models.NewNullTime(time.Now())}
	raw, err := json.Marshal(e)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"burned_in_subtitle":"zh-Hant"`)
	assert.NotContains(t, string(raw), "burned_in_checked_at")

	raw, err = json.Marshal(models.Episode{ID: "e2"})
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "burned_in_subtitle", "never checked → key absent")
}
