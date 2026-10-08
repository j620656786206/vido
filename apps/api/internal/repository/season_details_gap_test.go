package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-10-season-episode-count-unknown AC #4: only matched, not-removed
// series with an incomplete season are listed, once each.
func TestSeasonRepository_FindSeriesNeedingSeasonDetails(t *testing.T) {
	db := newMigratedLibraryDB(t)
	ctx := context.Background()
	exec := func(q string, args ...interface{}) {
		t.Helper()
		_, err := db.Exec(q, args...)
		require.NoError(t, err)
	}
	for _, s := range []struct {
		id      string
		tmdb    interface{}
		removed int
	}{{"see", 85421, 0}, {"done", 1396, 0}, {"unmatched", nil, 0}, {"gone", 999, 1}} {
		exec(`INSERT INTO series (id, title, first_air_date, tmdb_id, is_removed, created_at, updated_at)
			VALUES (?, ?, '2020-01-01', ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, s.id, s.id, s.tmdb, s.removed)
	}
	season := func(id, series string, n int, count, tmdb interface{}) {
		exec(`INSERT INTO seasons (id, series_id, season_number, episode_count, tmdb_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, id, series, n, count, tmdb)
	}
	season("see1", "see", 1, nil, nil) // bare, as the scanner writes it
	season("see2", "see", 2, nil, nil)
	season("done1", "done", 1, 7, 3572)
	season("un1", "unmatched", 1, nil, nil)
	season("gone1", "gone", 1, nil, nil)

	got, err := NewSeasonRepository(db).FindSeriesNeedingSeasonDetails(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, []SeasonDetailsGap{{SeriesID: "see", TMDbID: 85421}}, got)
}
