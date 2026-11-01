package repository

// disc-2026-11-burned-in-subs-b2: the sweep's season query, the undecided
// stamp, the per-kind spend totals and "file changed → ask again". On the
// app's own driver (Rule 28) so stored times are the production text.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
)

func burnedSweepDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open(database.DriverName, filepath.Join(t.TempDir(), "sweep.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func burnedColumns(t *testing.T, db *sql.DB, id string) (sql.NullString, sql.NullTime) {
	t.Helper()
	var v sql.NullString
	var at sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT burned_in_subtitle, burned_in_checked_at FROM episodes WHERE id = ?`, id).Scan(&v, &at))
	return v, at
}

func TestFindSeasonsNeedingBurnedInCheck(t *testing.T) {
	db := burnedSweepDB(t)
	ctx := context.Background()
	series := NewSeriesRepository(db)
	repo := NewEpisodeRepository(db)
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

	seedSeries(t, db, series, "b", "Bravo", subtitleShape{status: "not_searched"})
	seedSeries(t, db, series, "a", "alpha", subtitleShape{status: "not_searched"})
	seedSeries(t, db, series, "gone", "Gone", subtitleShape{status: "not_searched"})
	_, err := db.Exec(`UPDATE series SET is_removed = 1 WHERE id = 'gone'`)
	require.NoError(t, err)

	seedEpisode(t, db, "b", 2, 1, "/tv/b/S02E01.mkv", `[]`) // needs a check
	seedEpisode(t, db, "b", 1, 1, "/tv/b/S01E01.mkv", `[]`) // needs a check
	seedEpisode(t, db, "b", 1, 2, "/tv/b/S01E02.mkv", `[]`) // same season — listed once
	seedEpisode(t, db, "a", 1, 1, "/tv/a/S01E01.mkv", `[]`) // needs a check (sorts first, case-insensitive)
	seedEpisode(t, db, "a", 2, 1, "/tv/a/S02E01.mkv", `[]`) // answered → no
	seedEpisode(t, db, "a", 3, 1, "/tv/a/S03E01.mkv", `[]`) // undecided 2 days ago → no
	seedEpisode(t, db, "a", 4, 1, "/tv/a/S04E01.mkv", `[]`) // undecided 8 days ago → yes
	seedEpisode(t, db, "a", 5, 1, "/tv/a/S05E01.mkv", trEngOnly)
	seedEpisode(t, db, "a", 6, 1, "", `[]`)               // no file
	seedEpisode(t, db, "a", 7, 1, "/tv/a/S07E01.mkv", "") // never probed
	seedEpisode(t, db, "gone", 1, 1, "/tv/g/S01E01.mkv", `[]`)

	_, err = repo.SetSeasonBurnedInSubtitle(ctx, "a", 2, models.BurnedInZhHans, now)
	require.NoError(t, err)
	_, err = repo.MarkSeasonBurnedInChecked(ctx, "a", 3, now.Add(-2*24*time.Hour))
	require.NoError(t, err)
	_, err = repo.MarkSeasonBurnedInChecked(ctx, "a", 4, now.Add(-8*24*time.Hour))
	require.NoError(t, err)

	got, err := repo.FindSeasonsNeedingBurnedInCheck(ctx, now.Add(-7*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, []BurnedInSeasonRef{
		{SeriesID: "a", SeriesTitle: "alpha", Season: 1},
		{SeriesID: "a", SeriesTitle: "alpha", Season: 4},
		{SeriesID: "b", SeriesTitle: "Bravo", Season: 1},
		{SeriesID: "b", SeriesTitle: "Bravo", Season: 2},
	}, got)
}

func TestMarkSeasonBurnedInChecked_StampsOnlyUnansweredTracklessEpisodes(t *testing.T) {
	db := burnedSweepDB(t)
	ctx := context.Background()
	seedSeries(t, db, NewSeriesRepository(db), "s", "Show", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "s", 1, 1, "/tv/s/1.mkv", `[]`)
	seedEpisode(t, db, "s", 1, 2, "/tv/s/2.mkv", trEngOnly)
	seedEpisode(t, db, "s", 2, 1, "/tv/s/3.mkv", `[]`)
	repo := NewEpisodeRepository(db)
	at := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)

	n, err := repo.MarkSeasonBurnedInChecked(ctx, "s", 1, at)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	v, checked := burnedColumns(t, db, "s-s01e01")
	assert.False(t, v.Valid, "the value stays NULL — nothing was decided")
	assert.True(t, checked.Valid)
	_, checked = burnedColumns(t, db, "s-s01e02")
	assert.False(t, checked.Valid)
	_, checked = burnedColumns(t, db, "s-s02e01")
	assert.False(t, checked.Valid)
}

func TestBurnedInIsClearedWhenTheFileChanges(t *testing.T) {
	db := burnedSweepDB(t)
	ctx := context.Background()
	seedSeries(t, db, NewSeriesRepository(db), "s", "Show", subtitleShape{status: "not_searched"})
	repo := NewEpisodeRepository(db)
	for ep := 1; ep <= 4; ep++ {
		seedEpisode(t, db, "s", 1, ep, "/tv/s/"+string(rune('0'+ep))+".mkv", `[]`)
	}
	_, err := db.Exec(`UPDATE episodes SET subtitle_tracks_file_sig = '100:1'`)
	require.NoError(t, err)
	_, err = repo.SetSeasonBurnedInSubtitle(ctx, "s", 1, models.BurnedInZhHans, time.Now())
	require.NoError(t, err)

	// Same file re-read (a script peek): the answer stays.
	require.NoError(t, repo.UpdateSubtitleTracks(ctx, "s-s01e01", `[]`, "100:1"))
	v, at := burnedColumns(t, db, "s-s01e01")
	assert.Equal(t, models.BurnedInZhHans, v.String)
	assert.True(t, at.Valid)

	// New signature = a different file: ask again.
	require.NoError(t, repo.UpdateSubtitleTracks(ctx, "s-s01e02", `[]`, "200:2"))
	v, at = burnedColumns(t, db, "s-s01e02")
	assert.False(t, v.Valid)
	assert.False(t, at.Valid)

	// File gone.
	cleared, err := repo.ClearMissingFile(ctx, "s-s01e03", "/tv/s/3.mkv")
	require.NoError(t, err)
	require.True(t, cleared)
	v, at = burnedColumns(t, db, "s-s01e03")
	assert.False(t, v.Valid)
	assert.False(t, at.Valid)

	// The file moved to another episode row.
	_, err = repo.ReleaseFileFromOtherEpisodes(ctx, "s", "/tv/s/4.mkv", "someone-else")
	require.NoError(t, err)
	v, at = burnedColumns(t, db, "s-s01e04")
	assert.False(t, v.Valid)
	assert.False(t, at.Valid)
}

func TestAISideSpendRepository_TotalsByKindBetween(t *testing.T) {
	db := burnedSweepDB(t)
	repo := NewAISideSpendRepository(db)
	ctx := context.Background()
	from := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 30, 16, 0, 0, 0, time.UTC)
	seed := func(kind models.AISideSpendKind, usd float64, at time.Time) {
		require.NoError(t, repo.Create(ctx, &models.AISideSpend{Kind: kind, Provider: "claude", SpentUSD: usd, CreatedAt: at}))
	}
	seed(models.AISideSpendKindParse, 0.1, from)
	seed(models.AISideSpendKindFansub, 0.2, from.Add(time.Hour))
	seed(models.AISideSpendKindBurnedInCheck, 0.0023, from.Add(time.Hour))
	seed(models.AISideSpendKindBurnedInCheck, 0.0023, from.Add(2*time.Hour))
	seed(models.AISideSpendKindBurnedInCheck, 9, to) // next month

	got, err := repo.TotalsByKindBetween(ctx, from, to)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, 2, got[models.AISideSpendKindBurnedInCheck].Calls)
	assert.True(t, got[models.AISideSpendKindBurnedInCheck].Spent.Equal(decimal.RequireFromString("0.0046")))
	assert.Equal(t, 1, got[models.AISideSpendKindParse].Calls)
	_, ok := got[models.AISideSpendKindKeywords]
	assert.False(t, ok, "a kind with no row is absent")
}
