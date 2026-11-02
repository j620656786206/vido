package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-11-backfill-subtitle-cue-counts AC #3: the SQL half of the movie
// cue-count backfill lists track lists stored before counts were read, and the
// write is a compare-and-set. Real migration chain (Rule 28).
func TestFindMoviesMissingCueCounts(t *testing.T) {
	db := newMigratedLibraryDB(t)
	ctx := context.Background()
	repo := NewMovieRepository(db)
	old := `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`
	counted := `[{"language":"eng","format":"subrip","external":false,"stream_index":2,"cue_count":1609}]`
	unstated := `[{"language":"eng","format":"subrip","external":false,"stream_index":2,"cue_count":-1}]`
	seedMovie(t, db, repo, "m-old", "Old", subtitleShape{status: "not_searched", tracks: strp(old)})
	seedMovie(t, db, repo, "m-counted", "Counted", subtitleShape{status: "not_searched", tracks: strp(counted)})
	seedMovie(t, db, repo, "m-unstated", "Unstated", subtitleShape{status: "not_searched", tracks: strp(unstated)})
	seedMovie(t, db, repo, "m-empty", "Empty", subtitleShape{status: "not_searched", tracks: strp(`[]`)})
	seedMovie(t, db, repo, "m-null", "Null", subtitleShape{status: "not_searched"})
	seedMovie(t, db, repo, "m-removed", "Removed", subtitleShape{status: "not_searched", tracks: strp(old)})
	_, err := db.Exec(`UPDATE movies SET is_removed = 1 WHERE id = 'm-removed'`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE movies SET file_path = '/m/' || id || '.mkv'`)
	require.NoError(t, err)

	got, err := repo.FindMoviesMissingCueCounts(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "m-old", got[0].ID)

	written, err := repo.ReplaceSubtitleTracks(ctx, "m-old", "stale value", counted)
	require.NoError(t, err)
	assert.False(t, written, "a row changed since it was read is not overwritten")
	written, err = repo.ReplaceSubtitleTracks(ctx, "m-old", old, counted)
	require.NoError(t, err)
	assert.True(t, written)

	got, err = repo.FindMoviesMissingCueCounts(ctx, "", 10)
	require.NoError(t, err)
	assert.Empty(t, got, "after one re-read the movie leaves the list")
}
