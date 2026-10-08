package repository

// disc-2026-10-episode-rows-outlive-deleted-files: an episode whose file was
// deleted from the NAS goes back to "known episode, no local file" — and the
// series' Chinese-subtitle verdict stops counting it. Real migration chain.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

func TestEpisodeRemovedFile_ListsFilesWithTheirSeriesLibrary(t *testing.T) {
	db := newMigratedLibraryDB(t)
	series := NewSeriesRepository(db)
	episodes := NewEpisodeRepository(db)
	ctx := context.Background()

	seedSeries(t, db, series, "see", "See", subtitleShape{status: "not_searched"})
	_, err := db.Exec(`INSERT INTO media_libraries (id, name, content_type, created_at, updated_at) VALUES ('lib-tv', 'TV', 'series', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE series SET library_id = 'lib-tv' WHERE id = 'see'`)
	require.NoError(t, err)
	seedEpisode(t, db, "see", 1, 1, "/tv/See/S1/E01.mkv", trEngOnly)
	seedEpisode(t, db, "see", 1, 2, "", "") // TMDb placeholder, no file

	refs, err := episodes.FindFilesForRemovalCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, []EpisodeFileRef{{ID: "see-s01e01", FilePath: "/tv/See/S1/E01.mkv", LibraryID: "lib-tv"}}, refs)
}

func TestEpisodeRemovedFile_ClearDropsTheFileAndTheRollupStopsCountingIt(t *testing.T) {
	db := newMigratedLibraryDB(t)
	series := NewSeriesRepository(db)
	episodes := NewEpisodeRepository(db)
	ctx := context.Background()

	seedSeries(t, db, series, "see", "See", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "see", 1, 1, "/tv/See/S1/E01.mkv", trEngOnly) // deleted from the NAS
	seedEpisode(t, db, "see", 1, 2, "/tv/See/S1/E02.mkv", trZhTW)
	got, err := series.FindByID(ctx, "see")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleNone, got.ChineseSubtitle, "before: the deleted English-only episode drags the series to 「缺中文」")

	// Delivery state another flow wrote — must survive the clear.
	_, err = db.Exec(`UPDATE episodes SET subtitle_tracks_file_sig = '10:1', subtitle_status = 'not_found' WHERE id = 'see-s01e01'`)
	require.NoError(t, err)

	// A stale path (the episode was re-ingested elsewhere meanwhile) clears nothing.
	cleared, err := episodes.ClearMissingFile(ctx, "see-s01e01", "/tv/See/old/E01.mkv")
	require.NoError(t, err)
	assert.False(t, cleared)

	cleared, err = episodes.ClearMissingFile(ctx, "see-s01e01", "/tv/See/S1/E01.mkv")
	require.NoError(t, err)
	assert.True(t, cleared)

	ep, err := episodes.FindByID(ctx, "see-s01e01")
	require.NoError(t, err)
	assert.False(t, ep.FilePath.Valid, "back to 「沒有本地檔」")
	assert.False(t, ep.SubtitleTracks.Valid, "tracks were read from the file that is gone")
	assert.False(t, ep.SubtitleTracksFileSig.Valid)
	assert.Equal(t, models.SubtitleStatus("not_found"), ep.SubtitleStatus, "delivery state belongs to other flows — kept")

	got, err = series.FindByID(ctx, "see")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleZhHant, got.ChineseSubtitle, "after: only the episode still on disk counts")

	refs, err := episodes.FindFilesForRemovalCheck(ctx)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "see-s01e02", refs[0].ID)
}

// AC #5: the file comes back — the scan's Upsert (matched by series + season
// + episode) writes the path onto the SAME row, so nothing is restored by hand.
func TestEpisodeRemovedFile_FileBackIsWrittenOntoTheSameRow(t *testing.T) {
	db := newMigratedLibraryDB(t)
	series := NewSeriesRepository(db)
	episodes := NewEpisodeRepository(db)
	ctx := context.Background()

	seedSeries(t, db, series, "see", "See", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "see", 1, 1, "/tv/See/S1/E01.mkv", trEngOnly)
	cleared, err := episodes.ClearMissingFile(ctx, "see-s01e01", "/tv/See/S1/E01.mkv")
	require.NoError(t, err)
	require.True(t, cleared)

	created, err := episodes.Upsert(ctx, &models.Episode{
		SeriesID: "see", SeasonNumber: 1, EpisodeNumber: 1,
		FilePath: models.NewNullString("/tv/See/S1/See.S01E01.renamed.mkv"),
	})
	require.NoError(t, err)
	assert.False(t, created)
	back, err := episodes.FindByID(ctx, "see-s01e01")
	require.NoError(t, err)
	assert.Equal(t, "/tv/See/S1/See.S01E01.renamed.mkv", back.FilePath.String)
}
