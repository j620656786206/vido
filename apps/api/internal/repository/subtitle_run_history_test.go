package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
)

// disc-2026-11-spend-history-b-api AC #5 (Rule 28): the spend history read on
// the REAL migration chain, the app's own driver (sqlite-utc — the time text
// the started_at window compares in production) and foreign keys on like the
// app DSN. Movie, episode, deleted media and a failed run.

func newAppDriverLibraryDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	db, err := sql.Open(database.DriverName, path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func TestSubtitleRunRepository_FinishedRunsStartedBetween(t *testing.T) {
	db := newAppDriverLibraryDB(t)
	ctx := context.Background()
	require.NoError(t, NewSeriesRepository(db).Create(ctx, &models.Series{ID: "lioness", Title: "Lioness", FirstAirDate: "2023-07-23"}))
	require.NoError(t, NewMovieRepository(db).Create(ctx, &models.Movie{ID: "zootopia2", Title: "動物方城市2", ReleaseDate: "2025-11-26"}))
	_, err := db.ExecContext(ctx, `INSERT INTO episodes (id, series_id, season_number, episode_number, subtitle_status, created_at, updated_at)
		VALUES ('lioness-s01e01', 'lioness', 1, 1, 'not_searched', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)

	repo := NewSubtitleRunRepository(db)
	at := func(s string) time.Time { return mustUTC(t, s) }
	done := func(s string) *time.Time { v := at(s); return &v }
	runs := []*models.SubtitleRun{
		// An ASR run on an episode with every history column recorded.
		{ID: "ep-asr", MediaID: "lioness-s01e01", MediaType: "episode", Status: models.SubtitleRunCompleted,
			ModelID: "claude-sonnet-5", Route: models.SubtitleRunRouteASR, Source: models.SubtitleRunSourceAudio,
			StartedAt: at("2026-10-28T13:33:46Z"), CompletedAt: done("2026-10-28T13:38:25Z"),
			SpentUSD: f64p(0.52), EstimateUSD: f64p(0.6), ASRProvider: "elevenlabs", ASRModel: "scribe_v2",
			BatchID: "batch-1", TriggeredBy: models.SubtitleRunTriggeredManual},
		// A failed translate run on a movie.
		{ID: "movie-failed", MediaID: "zootopia2", MediaType: "movie", Status: models.SubtitleRunFailed,
			ModelID: "claude-sonnet-5", Route: models.SubtitleRunRouteTranslate,
			StartedAt: at("2026-10-15T03:27:46Z"), CompletedAt: done("2026-10-15T03:29:48Z"),
			SpentUSD: f64p(0.126108), ErrorMessage: "cancelled: context canceled"},
		// The movie was deleted after this run.
		{ID: "gone", MediaID: "deleted-movie", MediaType: "movie", Status: models.SubtitleRunCompleted,
			Route: models.SubtitleRunRouteDeliverDirect, StartedAt: at("2026-10-02T10:00:00Z"), CompletedAt: done("2026-10-02T10:00:05Z")},
		// A skipped run is history too (沒有可用字幕).
		{ID: "skipped", MediaID: "zootopia2", MediaType: "movie", Status: models.SubtitleRunSkipped,
			Route: models.SubtitleRunRouteNoTextSource, StartedAt: at("2026-10-01T00:00:00Z"), CompletedAt: done("2026-10-01T00:00:01Z")},
		// Excluded: still running; started exactly at the window's end; started before it.
		{ID: "running", MediaID: "zootopia2", MediaType: "movie", Status: models.SubtitleRunRunning, StartedAt: at("2026-10-20T00:00:00Z")},
		{ID: "at-end", MediaID: "zootopia2", MediaType: "movie", Status: models.SubtitleRunCompleted, StartedAt: at("2026-11-01T00:00:00Z")},
		{ID: "before", MediaID: "zootopia2", MediaType: "movie", Status: models.SubtitleRunCompleted, StartedAt: at("2026-09-30T23:59:59Z")},
	}
	for _, r := range runs {
		require.NoError(t, repo.Create(ctx, r))
	}

	// A local-zone bound must compare the same as its UTC instant.
	taipei := time.FixedZone("CST", 8*3600)
	got, err := repo.FinishedRunsStartedBetween(ctx, at("2026-10-01T00:00:00Z").In(taipei), at("2026-11-01T00:00:00Z").In(taipei))
	require.NoError(t, err)

	ids := make([]string, len(got))
	for i, r := range got {
		ids[i] = r.Run.ID
	}
	assert.Equal(t, []string{"ep-asr", "movie-failed", "gone", "skipped"}, ids,
		"completed, failed and skipped runs started in [from, to), newest started first")

	ep := got[0]
	assert.Equal(t, "Lioness", ep.MediaTitle, "an episode carries its series title")
	require.NotNil(t, ep.SeasonNumber)
	require.NotNil(t, ep.EpisodeNumber)
	assert.Equal(t, 1, *ep.SeasonNumber)
	assert.Equal(t, 1, *ep.EpisodeNumber)
	require.NotNil(t, ep.Run.EstimateUSD)
	assert.Equal(t, 0.6, *ep.Run.EstimateUSD)
	assert.Equal(t, "elevenlabs", ep.Run.ASRProvider)
	assert.Equal(t, "scribe_v2", ep.Run.ASRModel)
	assert.Equal(t, models.SubtitleRunSourceAudio, ep.Run.Source)
	assert.Equal(t, "batch-1", ep.Run.BatchID)
	assert.Equal(t, models.SubtitleRunTriggeredManual, ep.Run.TriggeredBy)

	movie := got[1]
	assert.Equal(t, "動物方城市2", movie.MediaTitle)
	assert.Nil(t, movie.SeasonNumber)
	assert.Equal(t, models.SubtitleRunFailed, movie.Run.Status)
	assert.Equal(t, "cancelled: context canceled", movie.Run.ErrorMessage)
	assert.Nil(t, movie.Run.EstimateUSD, "unrecorded estimate reads back nil, not 0")

	gone := got[2]
	assert.Equal(t, "", gone.MediaTitle, "deleted media keeps its run, title empty")
	assert.Nil(t, gone.Run.SpentUSD, "unrecorded spend reads back nil, not 0")
	assert.Equal(t, "", gone.Run.Source)
	assert.Equal(t, "", gone.Run.ASRProvider)
}

func TestSubtitleRunRepository_FinishedRunsStartedBetween_EmptyIsNotAnError(t *testing.T) {
	repo := NewSubtitleRunRepository(newAppDriverLibraryDB(t))
	got, err := repo.FinishedRunsStartedBetween(context.Background(), time.Now().AddDate(0, -1, 0), time.Now())
	require.NoError(t, err)
	assert.Empty(t, got)
}
