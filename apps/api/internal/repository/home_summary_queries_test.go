package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// The ux3-1-6 home-summary queries run against the REAL migration chain (the
// newMigratedEpisodeDB helper), so migration 032's columns/indexes are exercised
// by construction — a hand-rolled schema could not catch a predicate referencing
// a column the shipped schema lacks (Rule 15 / bugfix-20-1).

func seedHomeSummaryMovie(t *testing.T, db *sql.DB, id, filePath, subtitleLanguage string, removed bool) {
	t.Helper()
	rm := 0
	if removed {
		rm = 1
	}
	_, err := db.Exec(`INSERT INTO movies (id, title, release_date, file_path, subtitle_language, is_removed)
		VALUES (?, ?, '2020-01-01', ?, ?, ?)`,
		id, "Movie "+id, nullableText(filePath), nullableText(subtitleLanguage), rm)
	require.NoError(t, err)
}

func TestMovieRepository_CountZhHantSubtitle(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewMovieRepository(db)
	ctx := context.Background()

	seedHomeSummaryMovie(t, db, "m-covered", "/media/a.mkv", "zh-Hant", false)
	seedHomeSummaryMovie(t, db, "m-english", "/media/b.mkv", "en", false)
	seedHomeSummaryMovie(t, db, "m-none", "/media/c.mkv", "", false)
	// Fileless: neither covered nor missing — denominator only.
	seedHomeSummaryMovie(t, db, "m-fileless", "", "zh-Hant", false)
	// Removed: out of every count.
	seedHomeSummaryMovie(t, db, "m-removed", "/media/d.mkv", "zh-Hant", true)

	covered, err := repo.CountZhHantSubtitle(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, covered)

	// Sanity: covered + missing never exceeds total (fileless sits in neither).
	missing, err := repo.CountMissingZhHantSubtitle(ctx)
	require.NoError(t, err)
	total, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, missing)
	assert.Equal(t, 4, total)
	assert.LessOrEqual(t, covered+missing, total)
}

func TestSeriesRepository_CountZhHantCovered(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Fully covered: every on-disk episode has zh-Hant.
	seedGenerationSeries(t, db, "s-covered")
	seedGenerationEpisode(t, db, "e1", "s-covered", 1, "/tv/e1.mkv", "zh-Hant")
	seedGenerationEpisode(t, db, "e2", "s-covered", 2, "/tv/e2.mkv", "zh-Hant")

	// Partially covered: one episode still missing → NOT covered.
	seedGenerationSeries(t, db, "s-partial")
	seedGenerationEpisode(t, db, "e3", "s-partial", 1, "/tv/e3.mkv", "zh-Hant")
	seedGenerationEpisode(t, db, "e4", "s-partial", 2, "/tv/e4.mkv", "")

	// Zero on-disk episodes: the vacuous-truth guard — NOT covered.
	seedGenerationSeries(t, db, "s-empty")
	seedGenerationEpisode(t, db, "e5", "s-empty", 1, "", "")

	// Covered but soft-deleted → out of the count.
	seedGenerationSeries(t, db, "s-removed")
	seedGenerationEpisode(t, db, "e6", "s-removed", 1, "/tv/e6.mkv", "zh-Hant")
	_, err := db.Exec(`UPDATE series SET is_removed = 1 WHERE id = 's-removed'`)
	require.NoError(t, err)

	covered, err := repo.CountZhHantCovered(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, covered)
}

func seedHomeSummaryParseJob(t *testing.T, db *sql.DB, id string, status models.ParseJobStatus, mediaID any, completedAt any) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO parse_jobs (id, torrent_hash, file_path, file_name, status, media_id, retry_count, created_at, updated_at, completed_at)
		VALUES (?, ?, '/dl/x.mkv', 'x.mkv', ?, ?, 0, ?, ?, ?)`,
		id, "hash-"+id, string(status), mediaID, time.Now().UTC(), time.Now().UTC(), completedAt)
	require.NoError(t, err)
}

func TestParseJobRepository_CountByStatus(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewParseJobRepository(db)
	ctx := context.Background()

	seedHomeSummaryParseJob(t, db, "p1", models.ParseJobFailed, "media-1", time.Now().UTC())
	seedHomeSummaryParseJob(t, db, "p2", models.ParseJobFailed, nil, time.Now().UTC())
	seedHomeSummaryParseJob(t, db, "p3", models.ParseJobCompleted, "media-2", time.Now().UTC())
	seedHomeSummaryParseJob(t, db, "p4", models.ParseJobPending, nil, nil)

	failed, err := repo.CountByStatus(ctx, models.ParseJobFailed)
	require.NoError(t, err)
	assert.Equal(t, 2, failed)
}

func TestParseJobRepository_CompletedMediaIDsSince(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewParseJobRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	yesterday := now.Add(-30 * time.Hour)

	seedHomeSummaryParseJob(t, db, "p1", models.ParseJobCompleted, "media-a", now)
	// Same media completed twice today → ONE distinct id.
	seedHomeSummaryParseJob(t, db, "p2", models.ParseJobCompleted, "media-a", now)
	seedHomeSummaryParseJob(t, db, "p3", models.ParseJobCompleted, "media-b", yesterday)
	// Completed today but created nothing countable (no media id).
	seedHomeSummaryParseJob(t, db, "p4", models.ParseJobCompleted, nil, now)
	// Failed today → not "processed".
	seedHomeSummaryParseJob(t, db, "p5", models.ParseJobFailed, "media-c", now)
	// Written through the REAL writer (UpdateStatus stamps completed_at in
	// UTC): the lexicographic window must include it — this guards the
	// writer-and-query offset discipline migration 032 established.
	seedHomeSummaryParseJob(t, db, "p6", models.ParseJobPending, "media-d", nil)
	require.NoError(t, repo.UpdateStatus(ctx, "p6", models.ParseJobCompleted, ""))

	ids, err := repo.CompletedMediaIDsSince(ctx, now.Add(-time.Hour))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"media-a", "media-d"}, ids)
}

func completedRun(t *testing.T, repo *SubtitleRunRepository, id, mediaID, mediaType string, completedAt time.Time, spent, budget *float64) {
	t.Helper()
	ctx := context.Background()
	run := &models.SubtitleRun{
		ID: id, MediaID: mediaID, MediaType: mediaType,
		Status: models.SubtitleRunCompleted, StartedAt: completedAt.Add(-time.Minute),
		CompletedAt: &completedAt, SpentUSD: spent, BudgetUSD: budget,
	}
	require.NoError(t, repo.Create(ctx, run))
	require.NoError(t, repo.Update(ctx, run))
}

func TestSubtitleRunRepository_HomeSummaryQueries(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewSubtitleRunRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	spend := func(v float64) *float64 { return &v }

	completedRun(t, repo, "r1", "media-a", models.SubtitleRunMediaMovie, now.Add(-time.Minute), spend(0.42), spend(5))
	completedRun(t, repo, "r2", "media-b", models.SubtitleRunMediaEpisode, now.Add(-2*time.Minute), spend(0.10), spend(5))
	completedRun(t, repo, "r3", "media-c", models.SubtitleRunMediaMovie, now.Add(-30*time.Hour), spend(0.99), spend(5))

	failedAt := now.Add(-3 * time.Minute)
	failed := &models.SubtitleRun{
		ID: "r4", MediaID: "media-d", MediaType: models.SubtitleRunMediaMovie,
		Status: models.SubtitleRunFailed, StartedAt: failedAt.Add(-time.Minute),
		CompletedAt: &failedAt, ErrorMessage: "boom",
	}
	require.NoError(t, repo.Create(ctx, failed))
	require.NoError(t, repo.Update(ctx, failed))

	failedCount, err := repo.CountByStatus(ctx, models.SubtitleRunFailed)
	require.NoError(t, err)
	assert.Equal(t, 1, failedCount)

	refs, err := repo.CompletedMediaRefsSince(ctx, now.Add(-time.Hour))
	require.NoError(t, err)
	assert.ElementsMatch(t, []SubtitleRunMediaRef{
		{MediaID: "media-a", MediaType: models.SubtitleRunMediaMovie},
		{MediaID: "media-b", MediaType: models.SubtitleRunMediaEpisode},
	}, refs)

	// LatestWithSpend picks the newest terminal row carrying a spend — r1.
	latest, err := repo.LatestWithSpend(ctx)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, "r1", latest.ID)
	require.NotNil(t, latest.SpentUSD)
	assert.InDelta(t, 0.42, *latest.SpentUSD, 1e-9)
	require.NotNil(t, latest.BudgetUSD)
	assert.InDelta(t, 5.0, *latest.BudgetUSD, 1e-9)
}

func TestSubtitleRunRepository_LatestWithSpend_AbsentIsNil(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewSubtitleRunRepository(db)
	ctx := context.Background()

	// A pre-032-style row: terminal, but no spend recorded — must NOT surface.
	at := time.Now().UTC()
	run := &models.SubtitleRun{
		ID: "r-old", MediaID: "media-x", MediaType: models.SubtitleRunMediaMovie,
		Status: models.SubtitleRunCompleted, StartedAt: at.Add(-time.Minute), CompletedAt: &at,
	}
	require.NoError(t, repo.Create(ctx, run))
	require.NoError(t, repo.Update(ctx, run))

	latest, err := repo.LatestWithSpend(ctx)
	require.NoError(t, err)
	assert.Nil(t, latest)
}
