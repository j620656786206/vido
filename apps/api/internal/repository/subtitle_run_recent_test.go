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

// disc-2026-10-activity-subtitle-run-time-a AC #5 / #7: the activity record's
// subtitle source, on the REAL migration chain (Rule 28) with foreign keys on
// like the app DSN. The rows are the 正式機 shapes (2026-10-19 read-only query).

func mustUTC(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339Nano, s)
	require.NoError(t, err)
	return v.UTC()
}

func f64p(v float64) *float64 { return &v }

type recentRunSeed struct {
	id, mediaID, mediaType string
	status                 models.SubtitleRunStatus
	route                  string
	cues                   int
	started                string
	completed              string // "" = NULL
	spent                  *float64
	errMsg                 string
}

func seedRecentRuns(t *testing.T, db *sql.DB, seeds []recentRunSeed) {
	t.Helper()
	repo := NewSubtitleRunRepository(db)
	for _, s := range seeds {
		run := &models.SubtitleRun{
			ID: s.id, MediaID: s.mediaID, MediaType: s.mediaType, Status: s.status,
			Route: s.route, CueCount: s.cues, StartedAt: mustUTC(t, s.started),
			SpentUSD: s.spent, ErrorMessage: s.errMsg,
		}
		if s.completed != "" {
			c := mustUTC(t, s.completed)
			run.CompletedAt = &c
		}
		require.NoError(t, repo.Create(context.Background(), run))
	}
}

const (
	prodSeeInvalidResponse = `subtitle pipeline: episode c5eed5aa-4a23-4826-ad39-0ffe5d8f2a76: translate: SUBTITLE_TRANSLATE_FAILED: LLM subtitle translation failed: cues 151-200: translate chunk [151-200]: AI_INVALID_RESPONSE: Cannot parse AI response`
	prodPeacemakerDenied   = `subtitle pipeline: episode 2c5d56d4-9c3c-487d-871b-7d2d59ca753b: place: placer: write failed: write temp file: open /media/tv/Peacemaker/Season01/.Peacemaker.S01E01.2160p.MAX.WEB-DL.DDPA5.1.HDR.DV.HEVC-SPAMKiNGS.zh-Hant.srt.tmp.2903780514314573825: permission denied`
)

// seedActivityLibrary writes the AC #7 library and runs. Newest first by
// completed_at: see-e02, see-e04, peacemaker, movie-asr, legacy-null, gone.
func seedActivityLibrary(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	series := NewSeriesRepository(db)
	movies := NewMovieRepository(db)
	require.NoError(t, series.Create(ctx, &models.Series{ID: "see", Title: "末日光明", FirstAirDate: "2019-11-01"}))
	require.NoError(t, series.Create(ctx, &models.Series{ID: "peacemaker", Title: "和平使者", FirstAirDate: "2022-01-13"}))
	require.NoError(t, movies.Create(ctx, &models.Movie{ID: "zootopia2", Title: "動物方城市2", ReleaseDate: "2025-11-26"}))
	for _, ep := range []struct {
		id, series string
		s, e       int
	}{{"see-s01e02", "see", 1, 2}, {"see-s01e04", "see", 1, 4}, {"pm-s01e01", "peacemaker", 1, 1}} {
		_, err := db.ExecContext(ctx, `INSERT INTO episodes (id, series_id, season_number, episode_number, subtitle_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'not_searched', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, ep.id, ep.series, ep.s, ep.e)
		require.NoError(t, err)
	}

	seedRecentRuns(t, db, []recentRunSeed{
		// 正式機 run 07621d1c, original values.
		{id: "07621d1c", mediaID: "see-s01e02", mediaType: "episode", status: models.SubtitleRunCompleted,
			route: models.SubtitleRunRouteTranslate, cues: 568,
			started: "2026-10-19T01:21:30.067Z", completed: "2026-10-19T01:25:01.473Z", spent: f64p(0.197439)},
		{id: "c12acbf5", mediaID: "see-s01e04", mediaType: "episode", status: models.SubtitleRunFailed,
			route:   models.SubtitleRunRouteTranslate,
			started: "2026-10-15T03:27:46.919Z", completed: "2026-10-15T03:29:48.661Z", spent: f64p(0.126108), errMsg: prodSeeInvalidResponse},
		{id: "c6331c6b", mediaID: "pm-s01e01", mediaType: "episode", status: models.SubtitleRunFailed,
			route:   models.SubtitleRunRouteTranslate,
			started: "2026-09-02T17:24:33.461Z", completed: "2026-09-02T17:31:37.709Z", spent: f64p(0.196393), errMsg: prodPeacemakerDenied},
		{id: "movie-asr", mediaID: "zootopia2", mediaType: "movie", status: models.SubtitleRunCompleted,
			route: models.SubtitleRunRouteASR, cues: 1200,
			started: "2026-09-01T10:00:00Z", completed: "2026-09-01T10:20:00Z", spent: f64p(0)},
		// A pre-migration-032/041 row: no spend recorded, no route.
		{id: "legacy-null", mediaID: "zootopia2", mediaType: "movie", status: models.SubtitleRunCompleted,
			cues: 900, started: "2026-08-01T10:00:00Z", completed: "2026-08-01T10:10:00Z"},
		// The movie was deleted from the library after this run.
		{id: "gone", mediaID: "deleted-movie", mediaType: "movie", status: models.SubtitleRunCompleted,
			route: models.SubtitleRunRouteTranslate, cues: 300,
			started: "2026-07-01T10:00:00Z", completed: "2026-07-01T10:05:00Z", spent: f64p(0.05)},
		// Must never show: skipped (deferred-paid), running, pending, and a
		// failed row that somehow lacks completed_at.
		{id: "skipped", mediaID: "zootopia2", mediaType: "movie", status: models.SubtitleRunSkipped,
			route: models.SubtitleRunRouteSkip, started: "2026-10-19T02:00:00Z", completed: "2026-10-19T02:00:01Z",
			errMsg: "deferred-paid: translate needs a paid call"},
		{id: "running", mediaID: "see-s01e04", mediaType: "episode", status: models.SubtitleRunRunning,
			started: "2026-10-19T03:00:00Z"},
		{id: "pending", mediaID: "see-s01e04", mediaType: "episode", status: models.SubtitleRunPending,
			started: "2026-10-19T03:00:00Z"},
		{id: "failed-open", mediaID: "see-s01e04", mediaType: "episode", status: models.SubtitleRunFailed,
			started: "2026-10-19T04:00:00Z", errMsg: "boom"},
	})
}

func TestSubtitleRunRepository_RecentTerminalRuns(t *testing.T) {
	db := newMigratedLibraryDB(t)
	seedActivityLibrary(t, db)
	repo := NewSubtitleRunRepository(db)

	got, err := repo.RecentTerminalRuns(context.Background(), 8)
	require.NoError(t, err)

	ids := make([]string, len(got))
	for i, r := range got {
		ids[i] = r.Run.ID
	}
	assert.Equal(t, []string{"07621d1c", "c12acbf5", "c6331c6b", "movie-asr", "legacy-null", "gone"}, ids,
		"completed+failed with a completed_at only, newest first")

	see := got[0]
	assert.Equal(t, "末日光明", see.MediaTitle)
	require.NotNil(t, see.SeasonNumber)
	require.NotNil(t, see.EpisodeNumber)
	assert.Equal(t, 1, *see.SeasonNumber)
	assert.Equal(t, 2, *see.EpisodeNumber)
	// The run itself comes back whole through scanSubtitleRun.
	assert.Equal(t, 568, see.Run.CueCount)
	assert.Equal(t, models.SubtitleRunRouteTranslate, see.Run.Route)
	require.NotNil(t, see.Run.SpentUSD)
	assert.Equal(t, 0.197439, *see.Run.SpentUSD)
	require.NotNil(t, see.Run.CompletedAt)
	assert.True(t, see.Run.CompletedAt.Equal(mustUTC(t, "2026-10-19T01:25:01.473Z")))
	assert.True(t, see.Run.StartedAt.Equal(mustUTC(t, "2026-10-19T01:21:30.067Z")))

	assert.Equal(t, "和平使者", got[2].MediaTitle)
	assert.Equal(t, prodPeacemakerDenied, got[2].Run.ErrorMessage)

	movie := got[3]
	assert.Equal(t, "動物方城市2", movie.MediaTitle)
	assert.Nil(t, movie.SeasonNumber, "a movie has no season")
	assert.Nil(t, movie.EpisodeNumber)
	require.NotNil(t, movie.Run.SpentUSD)
	assert.Equal(t, 0.0, *movie.Run.SpentUSD)

	assert.Nil(t, got[4].Run.SpentUSD, "NULL spend stays nil — not recorded is not $0")
	assert.Equal(t, "", got[4].Run.Route)

	assert.Equal(t, "", got[5].MediaTitle, "a deleted item still returns, untitled")
}

func TestSubtitleRunRepository_RecentTerminalRuns_LimitAndTies(t *testing.T) {
	db := newMigratedLibraryDB(t)
	seedActivityLibrary(t, db)
	repo := NewSubtitleRunRepository(db)

	got, err := repo.RecentTerminalRuns(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "07621d1c", got[0].Run.ID)

	none, err := repo.RecentTerminalRuns(context.Background(), 0)
	require.NoError(t, err)
	assert.Empty(t, none)

	// Two runs finishing in the same instant come back in id order, poll after poll.
	seedRecentRuns(t, db, []recentRunSeed{
		{id: "tie-b", mediaID: "zootopia2", mediaType: "movie", status: models.SubtitleRunCompleted,
			started: "2026-10-20T00:00:00Z", completed: "2026-10-20T00:01:00Z"},
		{id: "tie-a", mediaID: "zootopia2", mediaType: "movie", status: models.SubtitleRunFailed,
			started: "2026-10-20T00:00:00Z", completed: "2026-10-20T00:01:00Z"},
	})
	got, err = repo.RecentTerminalRuns(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, "tie-a", got[0].Run.ID)
	assert.Equal(t, "tie-b", got[1].Run.ID)
}

// A whole-series run (media_type "series") is titled from the series row and
// carries no episode numbers.
func TestSubtitleRunRepository_RecentTerminalRuns_SeriesRun(t *testing.T) {
	db := newMigratedLibraryDB(t)
	seedActivityLibrary(t, db)
	seedRecentRuns(t, db, []recentRunSeed{
		{id: "series-run", mediaID: "see", mediaType: "series", status: models.SubtitleRunCompleted,
			started: "2026-10-21T00:00:00Z", completed: "2026-10-21T00:02:00Z"},
	})
	got, err := NewSubtitleRunRepository(db).RecentTerminalRuns(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "末日光明", got[0].MediaTitle)
	assert.Nil(t, got[0].SeasonNumber)
	assert.Nil(t, got[0].EpisodeNumber)
}

// The query plan rides the completed_at index (AC #3) — the page polls every 15 s.
func TestSubtitleRunRepository_RecentTerminalRuns_UsesCompletedAtIndex(t *testing.T) {
	db := newMigratedLibraryDB(t)
	rows, err := db.Query(`EXPLAIN QUERY PLAN `+recentTerminalRunsQuery,
		models.SubtitleRunCompleted, models.SubtitleRunFailed, 8)
	require.NoError(t, err)
	defer rows.Close()
	var plan string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notused, &detail))
		plan += detail + "\n"
	}
	require.NoError(t, rows.Err())
	assert.Contains(t, plan, "idx_subtitle_runs_completed_at", plan)
	assert.NotContains(t, plan, "USE TEMP B-TREE FOR ORDER BY", "a full sort of every run would defeat LIMIT: "+plan)
}
