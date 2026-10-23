package repository

// disc-2026-10-series-wrong-version-c: a series whose folder is gone and whose
// episodes all lost their files leaves the library. Real migration chain.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

func TestSeriesRemovedFolder_ListsOnlyShowsWithoutEpisodeFiles(t *testing.T) {
	db := newMigratedLibraryDB(t)
	series := NewSeriesRepository(db)
	ctx := context.Background()

	seedSeries(t, db, series, "old", "將夜", subtitleShape{status: "not_searched"})
	seedSeries(t, db, series, "new", "將夜", subtitleShape{status: "not_searched"})
	seedSeries(t, db, series, "nopath", "No Path", subtitleShape{status: "not_searched"})
	seedSeries(t, db, series, "gone", "Already Gone", subtitleShape{status: "not_searched"})
	_, err := db.Exec(`UPDATE series SET file_path = '/tv/將夜 (2026)', library_id = NULL WHERE id = 'old'`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE series SET file_path = '/tv/將夜' WHERE id = 'new'`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE series SET file_path = '/tv/Gone', is_removed = 1 WHERE id = 'gone'`)
	require.NoError(t, err)
	seedEpisode(t, db, "old", 1, 1, "", "") // TMDb placeholders, no files
	seedEpisode(t, db, "old", 1, 2, "", "")
	seedEpisode(t, db, "new", 1, 1, "/tv/將夜/Season01/E01.mp4", "")

	refs, err := series.FindFoldersForRemovalCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, []SeriesFolderRef{{ID: "old", FilePath: "/tv/將夜 (2026)"}}, refs,
		"a show with a file, one with no folder path, and one already removed are not candidates")
}

func TestSeriesRemovedFolder_MarkRechecksAndTouchesOnlyTheFlag(t *testing.T) {
	db := newMigratedLibraryDB(t)
	series := NewSeriesRepository(db)
	ctx := context.Background()

	seedSeries(t, db, series, "old", "將夜", subtitleShape{status: "found"})
	_, err := db.Exec(`UPDATE series SET file_path = '/tv/將夜 (2026)', tmdb_id = 83612 WHERE id = 'old'`)
	require.NoError(t, err)

	// An episode came back while the scan stat-ed: keep the show.
	seedEpisode(t, db, "old", 1, 1, "/tv/將夜 (2026)/E01.mp4", "")
	marked, err := series.MarkRemovedIfNoFiles(ctx, "old")
	require.NoError(t, err)
	assert.False(t, marked)

	_, err = db.Exec(`UPDATE episodes SET file_path = NULL WHERE series_id = 'old'`)
	require.NoError(t, err)
	marked, err = series.MarkRemovedIfNoFiles(ctx, "old")
	require.NoError(t, err)
	assert.True(t, marked)

	got, err := series.FindByID(ctx, "old")
	require.NoError(t, err)
	assert.True(t, got.IsRemoved)
	assert.Equal(t, int64(83612), got.TMDbID.Int64, "TMDb data stays")
	assert.Equal(t, models.SubtitleStatusFound, got.SubtitleStatus, "subtitle state stays")
	var episodes int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM episodes WHERE series_id = 'old'`).Scan(&episodes))
	assert.Equal(t, 1, episodes, "episodes are not deleted")

	marked, err = series.MarkRemovedIfNoFiles(ctx, "old")
	require.NoError(t, err)
	assert.False(t, marked, "already removed: nothing to mark")

	require.NoError(t, series.RestoreRemoved(ctx, "old"))
	got, err = series.FindByID(ctx, "old")
	require.NoError(t, err)
	assert.False(t, got.IsRemoved)
}
