package repository

// disc-2026-10-episode-list-subtitle-badge-a AC #4 — subtitle_tracks must
// survive a real round trip through the repository (the bugfix-20-1 class: a
// column in the table and the struct but missing from SELECT/scan reads as
// NULL forever, i.e. every episode "unknown").

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

func TestEpisodeRepository_SubtitleTracksRoundTrip(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()
	repo := NewEpisodeRepository(db)
	ctx := context.Background()
	createTestSeries(t, db, "series-1")

	ep := &models.Episode{ID: "ep-1", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 1,
		FilePath: models.NewNullString("/media/See/S01E01.mkv")}
	require.NoError(t, repo.Create(ctx, ep))

	found, err := repo.FindByID(ctx, "ep-1")
	require.NoError(t, err)
	assert.False(t, found.SubtitleTracks.Valid, "never read = NULL, not []")

	require.NoError(t, repo.UpdateSubtitleTracks(ctx, "ep-1", `[{"language":"eng"}]`, "100:200"))
	found, err = repo.FindByID(ctx, "ep-1")
	require.NoError(t, err)
	assert.Equal(t, `[{"language":"eng"}]`, found.SubtitleTracks.String)
	assert.Equal(t, "100:200", found.SubtitleTracksFileSig.String)
	assert.Equal(t, ep.UpdatedAt.Unix(), found.UpdatedAt.Unix(), "a look at the file is not a change to the row")

	// A full Update (the scanner's upsert) must not wipe what the sweep read.
	found.Title = models.NewNullString("神之火")
	require.NoError(t, repo.Update(ctx, found))
	again, err := repo.FindByID(ctx, "ep-1")
	require.NoError(t, err)
	assert.Equal(t, `[{"language":"eng"}]`, again.SubtitleTracks.String)

	assert.Error(t, repo.UpdateSubtitleTracks(ctx, "missing", "[]", ""))
}

func TestEpisodeRepository_RefreshSubtitleTracksIsCompareAndSet(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()
	repo := NewEpisodeRepository(db)
	ctx := context.Background()
	createTestSeries(t, db, "series-1")

	require.NoError(t, repo.Create(ctx, &models.Episode{ID: "ep-1", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 1}))
	require.NoError(t, repo.UpdateSubtitleTracks(ctx, "ep-1", `[]`, "1:1"))

	wrote, err := repo.RefreshSubtitleTracks(ctx, "ep-1", `["stale"]`, `[{"language":"zh-Hant"}]`)
	require.NoError(t, err)
	assert.False(t, wrote, "a stale copy must not overwrite the current value")

	wrote, err = repo.RefreshSubtitleTracks(ctx, "ep-1", `[]`, `[{"language":"zh-Hant"}]`)
	require.NoError(t, err)
	assert.True(t, wrote)
	found, err := repo.FindByID(ctx, "ep-1")
	require.NoError(t, err)
	assert.Equal(t, `[{"language":"zh-Hant"}]`, found.SubtitleTracks.String)
	assert.Equal(t, "1:1", found.SubtitleTracksFileSig.String, "a sidecar refresh keeps the probe's signature")
}

func TestEpisodeRepository_FindWithFiles(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()
	repo := NewEpisodeRepository(db)
	ctx := context.Background()
	createTestSeries(t, db, "series-1")

	require.NoError(t, repo.Create(ctx, &models.Episode{ID: "b", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 2, FilePath: models.NewNullString("/m/2.mkv")}))
	require.NoError(t, repo.Create(ctx, &models.Episode{ID: "a", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 1, FilePath: models.NewNullString("/m/1.mkv")}))
	require.NoError(t, repo.Create(ctx, &models.Episode{ID: "c", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 3}))

	eps, err := repo.FindWithFiles(ctx)
	require.NoError(t, err)
	require.Len(t, eps, 2)
	assert.Equal(t, "a", eps[0].ID)
	assert.Equal(t, "b", eps[1].ID)
}
