package repository

// disc-2026-11-burned-in-subs-b1 AC #5/#6: one season check writes every
// episode of that season whose tracks were read empty, and the value moves the
// episode and series verdicts. Real migration chain (Rule 28).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

func TestSetSeasonBurnedInSubtitle_WritesOnlyTracklessEpisodesOfThatSeason(t *testing.T) {
	db := newMigratedLibraryDB(t)
	ctx := context.Background()
	seedSeries(t, db, NewSeriesRepository(db), "s", "Show", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "s", 1, 1, "/tv/s/S01E01.mkv", `[]`)
	seedEpisode(t, db, "s", 1, 2, "/tv/s/S01E02.mkv", `[]`)
	seedEpisode(t, db, "s", 1, 3, "/tv/s/S01E03.mkv", trEngOnly) // has a track: left alone
	seedEpisode(t, db, "s", 1, 4, "/tv/s/S01E04.mkv", "")        // never read: left alone
	seedEpisode(t, db, "s", 2, 1, "/tv/s/S02E01.mkv", `[]`)      // other season: left alone

	repo := NewEpisodeRepository(db)
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	n, err := repo.SetSeasonBurnedInSubtitle(ctx, "s", 1, models.BurnedInZhHans, at)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	season1, err := repo.FindBySeasonNumber(ctx, "s", 1)
	require.NoError(t, err)
	got := map[int]models.Episode{}
	for _, e := range season1 {
		got[e.EpisodeNumber] = e
	}
	assert.Equal(t, models.BurnedInZhHans, got[1].BurnedInSubtitle.String, "read back through the select list (Rule 15)")
	assert.True(t, got[1].BurnedInCheckedAt.Valid)
	assert.True(t, got[1].BurnedInCheckedAt.Time.Equal(at))
	assert.Equal(t, models.BurnedInZhHans, got[2].BurnedInSubtitle.String)
	assert.False(t, got[3].BurnedInSubtitle.Valid)
	assert.False(t, got[4].BurnedInSubtitle.Valid)

	season2, err := repo.FindBySeasonNumber(ctx, "s", 2)
	require.NoError(t, err)
	require.Len(t, season2, 1)
	assert.False(t, season2[0].BurnedInSubtitle.Valid)

	_, err = repo.SetSeasonBurnedInSubtitle(ctx, "s", 1, "zh_hans", at)
	assert.Error(t, err, "only the four stored values are accepted")
}

func TestBurnedInSubtitle_MovesSeriesVerdict(t *testing.T) {
	db := newMigratedLibraryDB(t)
	ctx := context.Background()
	seriesRepo := NewSeriesRepository(db)
	seedSeries(t, db, seriesRepo, "burned", "Burned", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "burned", 1, 1, "/tv/b/S01E01.mkv", `[]`)
	seedEpisode(t, db, "burned", 1, 2, "/tv/b/S01E02.mkv", `[]`)

	got, err := seriesRepo.FindByID(ctx, "burned")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleNone, got.ChineseSubtitle, "no tracks, never checked → missing Chinese")

	episodes := NewEpisodeRepository(db)
	for _, tc := range []struct {
		burned string
		want   models.ChineseSubtitle
	}{
		{models.BurnedInZhHans, models.ChineseSubtitleZhHans},
		{models.BurnedInZhHant, models.ChineseSubtitleZhHant},
		{models.BurnedInZh, models.ChineseSubtitleZh},
		{models.BurnedInNone, models.ChineseSubtitleNone},
	} {
		_, err := episodes.SetSeasonBurnedInSubtitle(ctx, "burned", 1, tc.burned, time.Now())
		require.NoError(t, err)
		got, err := seriesRepo.FindByID(ctx, "burned")
		require.NoError(t, err)
		assert.Equal(t, tc.want, got.ChineseSubtitle, "burned %s", tc.burned)
	}
}

// disc-2026-11-burned-in-subs-b2 AC #5 (ruling 2A): an episode with Chinese
// burned into the picture — any script — leaves the generation queue; one
// checked and found without burned Chinese stays.
func TestBurnedInChinese_LeavesTheGenerationQueue(t *testing.T) {
	db := newMigratedLibraryDB(t)
	ctx := context.Background()
	seedSeries(t, db, NewSeriesRepository(db), "q", "Queue", subtitleShape{status: "not_searched"})
	repo := NewEpisodeRepository(db)
	for season, burned := range map[int]string{
		1: models.BurnedInZhHant, 2: models.BurnedInZhHans, 3: models.BurnedInZh, 4: models.BurnedInNone,
	} {
		seedEpisode(t, db, "q", season, 1, fmt.Sprintf("/tv/q/S%02dE01.mkv", season), `[]`)
		_, err := repo.SetSeasonBurnedInSubtitle(ctx, "q", season, burned, time.Now())
		require.NoError(t, err)
	}
	seedEpisode(t, db, "q", 5, 1, "/tv/q/S05E01.mkv", `[]`) // never checked

	missing, err := repo.FindMissingZhHantSubtitle(ctx)
	require.NoError(t, err)
	var ids []string
	for _, e := range missing {
		ids = append(ids, e.ID)
	}
	assert.Equal(t, []string{"q-s04e01", "q-s05e01"}, ids, "only burned none and never-checked still need a subtitle")
	count, err := repo.CountMissingZhHantSubtitle(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "the preview count drops the same rows")
}
