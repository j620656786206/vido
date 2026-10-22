package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// seeded runs the seed into a temp dir and reopens the database the way the
// app does, so every assertion reads through the production read path.
func seeded(t *testing.T) (*repository.Repositories, *repository.AISideSpendRepository, string) {
	return seededWith(t, false)
}

func seededWith(t *testing.T, failureFixtures bool) (*repository.Repositories, *repository.AISideSpendRepository, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("DB_PATH", filepath.Join(dataDir, "vido.db"))
	require.NoError(t, run(dataDir, "", false, failureFixtures))

	cfg, err := config.LoadDatabaseConfig()
	require.NoError(t, err)
	db, err := database.Initialize(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return repository.NewRepositoriesWithCache(db.Conn()), repository.NewAISideSpendRepository(db.Conn()), filepath.Join(dataDir, "media")
}

func TestSeed_MovieSubtitleVerdictsAreSpread(t *testing.T) {
	repos, _, _ := seeded(t)
	ctx := context.Background()

	want := map[string]models.ChineseSubtitle{
		"seed-mv-001": models.ChineseSubtitleZhHant,
		"seed-mv-002": models.ChineseSubtitleNone,
		"seed-mv-003": models.ChineseSubtitleZhHans,
		"seed-mv-004": models.ChineseSubtitleZhHant,
		"seed-mv-005": models.ChineseSubtitleNone,
		"seed-mv-006": models.ChineseSubtitleNone,
		"seed-mv-007": models.ChineseSubtitleZhHans,
		"seed-mv-008": models.ChineseSubtitleNone,
		"seed-mv-009": models.ChineseSubtitleNone,
		"seed-mv-010": models.ChineseSubtitleUnknown,
		"seed-mv-011": models.ChineseSubtitleZhHant,
		"seed-mv-012": models.ChineseSubtitleNone,
	}
	for id, verdict := range want {
		m, err := repos.Movies.FindByID(ctx, id)
		require.NoError(t, err, id)
		assert.Equal(t, verdict, m.ChineseSubtitle, id)
	}

	// The badge the list shows, not just the verdict: 006 names its reason
	// (無字幕源), 012 was skipped, 002 is plain 缺中文.
	m006, err := repos.Movies.FindByID(ctx, "seed-mv-006")
	require.NoError(t, err)
	assert.Equal(t, models.SubtitleRankNoSourceOrStopped, models.SubtitleTriageRank(m006.ChineseSubtitle, m006.SubtitleStatus))
	m012, err := repos.Movies.FindByID(ctx, "seed-mv-012")
	require.NoError(t, err)
	assert.Equal(t, models.SubtitleStatusSkipped, m012.SubtitleStatus)
	m002, err := repos.Movies.FindByID(ctx, "seed-mv-002")
	require.NoError(t, err)
	assert.Equal(t, models.SubtitleRankMissing, models.SubtitleTriageRank(m002.ChineseSubtitle, m002.SubtitleStatus))
}

func TestSeed_SeriesRollUpFromEpisodes(t *testing.T) {
	repos, _, _ := seeded(t)
	ctx := context.Background()

	for id, verdict := range map[string]models.ChineseSubtitle{
		"seed-sr-001": models.ChineseSubtitleNone,   // S01E02 is English only
		"seed-sr-002": models.ChineseSubtitleZhHant, // every episode has 繁中
	} {
		s, err := repos.Series.FindByID(ctx, id)
		require.NoError(t, err, id)
		assert.Equal(t, verdict, s.ChineseSubtitle, id)
	}
}

func TestSeed_EpisodeTracksCarryTheFileSignature(t *testing.T) {
	repos, _, mediaRoot := seeded(t)
	ctx := context.Background()

	ep, err := repos.Episodes.FindByID(ctx, "seed-sr-001-s01e01")
	require.NoError(t, err)
	require.True(t, ep.FilePath.Valid)
	info, err := os.Stat(filepath.Join(mediaRoot, "tv", "進擊的巨人", "S01E01.mkv"))
	require.NoError(t, err)
	// Equal signature = the background sweep skips the episode instead of
	// re-probing the placeholder file.
	assert.Equal(t, services.FileSignature(info), ep.SubtitleTracksFileSig.String)
}

func TestSeed_SubtitleActivityThisMonth(t *testing.T) {
	repos, sideSpend, _ := seeded(t)
	ctx := context.Background()

	recent, err := repos.SubtitleRuns.RecentTerminalRuns(ctx, 20)
	require.NoError(t, err)
	require.Len(t, recent, 5)
	reasons := map[string]bool{}
	for _, r := range recent {
		require.NotNil(t, r.Run.CompletedAt, r.Run.ID)
		assert.True(t, r.Run.CompletedAt.After(r.Run.StartedAt), r.Run.ID)
		if r.Run.Status == models.SubtitleRunFailed {
			reasons[models.SubtitleRunFailureReason(r.Run.ErrorMessage)] = true
		}
	}
	assert.Equal(t, map[string]bool{
		models.SubtitleRunFailureTargetNotWritable: true,
		models.SubtitleRunFailureQueueTimeout:      true,
		models.SubtitleRunFailureCreditExhausted:   true,
	}, reasons)

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 1, 0)
	completed, err := repos.SubtitleRuns.CompletedRunsBetween(ctx, from, to)
	require.NoError(t, err)
	assert.Len(t, completed, 2)

	totals, err := sideSpend.TotalsBetween(ctx, from, to)
	require.NoError(t, err)
	assert.Equal(t, 2, totals.Calls)
	assert.Equal(t, "0.021", totals.Spent.String())
}

// --failure-fixtures still composes with the activity runs: its run has no
// completed_at (it feeds the home attention cell, not the activity record),
// so the record keeps the same five rows.
func TestSeed_FailureFixturesComposeWithActivityRuns(t *testing.T) {
	repos, _, _ := seededWith(t, true)
	ctx := context.Background()

	_, err := repos.SubtitleRuns.FindByID(ctx, "seed-failure-run-001")
	require.NoError(t, err)
	recent, err := repos.SubtitleRuns.RecentTerminalRuns(ctx, 20)
	require.NoError(t, err)
	assert.Len(t, recent, 5)
}

func TestThisMonthAgo_ClampsToTheCurrentMonth(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	firstOfMonth := time.Date(2026, 11, 1, 0, 30, 0, 0, loc)
	got := thisMonthAgo(firstOfMonth, 5*time.Hour)
	assert.Equal(t, time.Date(2026, 11, 1, 0, 1, 0, 0, loc).UTC(), got)

	midMonth := time.Date(2026, 10, 22, 15, 0, 0, 0, loc)
	assert.Equal(t, midMonth.Add(-5*time.Hour).UTC(), thisMonthAgo(midMonth, 5*time.Hour))
}

func TestSeed_SetupCompletedAndRemovedStayHidden(t *testing.T) {
	repos, _, _ := seeded(t)
	ctx := context.Background()

	done, err := repos.Settings.GetBool(ctx, "setup_completed")
	require.NoError(t, err)
	assert.True(t, done)

	m, err := repos.Movies.FindByID(ctx, "seed-mv-201")
	require.NoError(t, err)
	assert.True(t, m.IsRemoved)
}
