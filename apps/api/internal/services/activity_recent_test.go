package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// disc-2026-10-activity-subtitle-run-time-a — the activity record gains
// finished subtitle-generation runs (kind "subtitle", AC #1 [@contract-v1]).

type fakeRecentRuns struct {
	runs []repository.RecentSubtitleRun
	err  error
}

func (f fakeRecentRuns) RecentTerminalRuns(_ context.Context, limit int) ([]repository.RecentSubtitleRun, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.runs) > limit {
		return f.runs[:limit], nil
	}
	return f.runs, nil
}

func newRecentOnlyService(parse parseJobSource, runs recentRunSource) *ActivityService {
	svc := NewActivityService(fakeScan{}, fakeBatch{}, fakeBatch{}, fakeBatch{}, fakeDownloads{counts: &qbittorrent.DownloadCounts{}}, parse)
	if runs != nil {
		svc.SetRecentRuns(runs)
	}
	return svc
}

func at(s string) time.Time {
	v, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return v.UTC()
}

func doneRun(id string, completed time.Time, status models.SubtitleRunStatus) repository.RecentSubtitleRun {
	return repository.RecentSubtitleRun{
		Run: models.SubtitleRun{
			ID: id, MediaID: "m-" + id, MediaType: models.SubtitleRunMediaMovie, Status: status,
			StartedAt: completed.Add(-time.Minute), CompletedAt: &completed,
		},
		MediaTitle: "片 " + id,
	}
}

func parseDone(id string, completed time.Time) *models.ParseJob {
	return &models.ParseJob{ID: id, Status: models.ParseJobCompleted, FileName: id + ".mkv", UpdatedAt: completed, CompletedAt: &completed}
}

// ─── AC #7: 正式機 shapes on the real migration chain ─────────────────────

const (
	prodSeeInvalidResponse = `subtitle pipeline: episode c5eed5aa-4a23-4826-ad39-0ffe5d8f2a76: translate: SUBTITLE_TRANSLATE_FAILED: LLM subtitle translation failed: cues 151-200: translate chunk [151-200]: AI_INVALID_RESPONSE: Cannot parse AI response`
	prodPeacemakerDenied   = `subtitle pipeline: episode 2c5d56d4-9c3c-487d-871b-7d2d59ca753b: place: placer: write failed: write temp file: open /media/tv/Peacemaker/Season01/.Peacemaker.S01E01.2160p.MAX.WEB-DL.DDPA5.1.HDR.DV.HEVC-SPAMKiNGS.zh-Hant.srt.tmp.2903780514314573825: permission denied`
)

func activityLibraryDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "activity.db")
	db, err := sql.Open(database.DriverName, path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func spend(v float64) *float64 { return &v }

func seedProdShapedRuns(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, repository.NewSeriesRepository(db).Create(ctx, &models.Series{ID: "see", Title: "末日光明", FirstAirDate: "2019-11-01"}))
	require.NoError(t, repository.NewSeriesRepository(db).Create(ctx, &models.Series{ID: "peacemaker", Title: "和平使者", FirstAirDate: "2022-01-13"}))
	require.NoError(t, repository.NewMovieRepository(db).Create(ctx, &models.Movie{ID: "zootopia2", Title: "動物方城市2", ReleaseDate: "2025-11-26"}))
	for _, ep := range []struct {
		id, series string
		s, e       int
	}{{"see-s01e02", "see", 1, 2}, {"see-s01e04", "see", 1, 4}, {"pm-s01e01", "peacemaker", 1, 1}} {
		_, err := db.ExecContext(ctx, `INSERT INTO episodes (id, series_id, season_number, episode_number, subtitle_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'not_searched', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, ep.id, ep.series, ep.s, ep.e)
		require.NoError(t, err)
	}

	runs := repository.NewSubtitleRunRepository(db)
	mk := func(id, mediaID, mediaType string, status models.SubtitleRunStatus, route string, cues int, started, completed string, spent *float64, errMsg string) {
		run := &models.SubtitleRun{
			ID: id, MediaID: mediaID, MediaType: mediaType, Status: status, Route: route,
			CueCount: cues, StartedAt: at(started), SpentUSD: spent, ErrorMessage: errMsg,
		}
		if completed != "" {
			c := at(completed)
			run.CompletedAt = &c
		}
		require.NoError(t, runs.Create(ctx, run))
	}
	// 正式機 run 07621d1c, original values.
	mk("07621d1c", "see-s01e02", "episode", models.SubtitleRunCompleted, models.SubtitleRunRouteTranslate, 568,
		"2026-10-19T01:21:30.067Z", "2026-10-19T01:25:01.473Z", spend(0.197439), "")
	mk("c12acbf5", "see-s01e04", "episode", models.SubtitleRunFailed, models.SubtitleRunRouteTranslate, 0,
		"2026-10-15T03:27:46.919Z", "2026-10-15T03:29:48.661Z", spend(0.126108), prodSeeInvalidResponse)
	mk("c6331c6b", "pm-s01e01", "episode", models.SubtitleRunFailed, models.SubtitleRunRouteTranslate, 0,
		"2026-09-02T17:24:33.461Z", "2026-09-02T17:31:37.709Z", spend(0.196393), prodPeacemakerDenied)
	mk("movie-asr", "zootopia2", "movie", models.SubtitleRunCompleted, models.SubtitleRunRouteASR, 1200,
		"2026-09-01T10:00:00Z", "2026-09-01T10:20:00Z", spend(0), "")
	mk("legacy-null", "zootopia2", "movie", models.SubtitleRunCompleted, "", 900,
		"2026-08-01T10:00:00Z", "2026-08-01T10:10:00Z", nil, "")
	mk("gone", "deleted-movie", "movie", models.SubtitleRunCompleted, models.SubtitleRunRouteTranslate, 300,
		"2026-07-01T10:00:00Z", "2026-07-01T10:05:00Z", spend(0.05), "")
	mk("skipped", "zootopia2", "movie", models.SubtitleRunSkipped, models.SubtitleRunRouteSkip, 0,
		"2026-10-19T02:00:00Z", "2026-10-19T02:00:01Z", nil, "deferred-paid: translate needs a paid call")
	mk("running", "see-s01e04", "episode", models.SubtitleRunRunning, "", 0, "2026-10-19T03:00:00Z", "", nil, "")
}

// recentWire marshals the recent section and returns its events as maps — the
// shape the web client actually parses.
func recentWire(t *testing.T, svc *ActivityService) []map[string]any {
	t.Helper()
	blob, err := json.Marshal(svc.GetActivity(context.Background()).Recent)
	require.NoError(t, err)
	var section struct {
		Status string           `json:"status"`
		Events []map[string]any `json:"events"`
	}
	require.NoError(t, json.Unmarshal(blob, &section))
	require.Equal(t, sectionOK, section.Status)
	return section.Events
}

func TestActivity_RecentSubtitleRuns_ProdShapes(t *testing.T) {
	db := activityLibraryDB(t)
	seedProdShapedRuns(t, db)
	svc := newRecentOnlyService(fakeParse{}, repository.NewSubtitleRunRepository(db))

	events := recentWire(t, svc)
	ids := []string{}
	for _, e := range events {
		ids = append(ids, fmt.Sprint(e["media_title"]))
	}
	require.Len(t, events, 6, "skipped and running never show: %v", ids)

	// 末日光明 S01E02 — 正式機 07621d1c.
	see := events[0]
	assert.Equal(t, "subtitle", see["kind"])
	assert.Equal(t, "completed", see["result"])
	assert.Equal(t, "2026-10-19T01:25:01.473Z", see["at"])
	assert.Equal(t, "episode", see["media_type"])
	assert.Equal(t, "末日光明", see["media_title"])
	assert.Equal(t, 1.0, see["season_number"])
	assert.Equal(t, 2.0, see["episode_number"])
	assert.Equal(t, "translate", see["route"])
	assert.Equal(t, 568.0, see["cue_count"])
	assert.Equal(t, 211.0, see["duration_seconds"], "01:21:30.067 → 01:25:01.473 is 211.406 s")
	assert.Equal(t, 0.197439, see["spent_usd"], "raw, not pre-rounded")
	assert.NotContains(t, see, "failure_reason")
	assert.NotContains(t, see, "detail")

	// 末日光明 S01E04 — AI_INVALID_RESPONSE falls in other (Sally 追認 4).
	e04 := events[1]
	assert.Equal(t, "failed", e04["result"])
	assert.Equal(t, "other", e04["failure_reason"])
	assert.Equal(t, 4.0, e04["episode_number"])
	assert.Equal(t, 0.126108, e04["spent_usd"])
	assert.NotContains(t, e04, "cue_count")
	assert.NotContains(t, e04, "detail", "the raw error (with NAS paths) never goes on the wire")

	// 和平使者 S01E01 — translated, then could not write: not writable ≠ free.
	pm := events[2]
	assert.Equal(t, "和平使者", pm["media_title"])
	assert.Equal(t, "target_not_writable", pm["failure_reason"])
	assert.Equal(t, 0.196393, pm["spent_usd"])

	// The free ASR movie: spent_usd is present and 0 (沒花錢); no episode keys.
	movie := events[3]
	assert.Equal(t, "movie", movie["media_type"])
	assert.Equal(t, "asr", movie["route"])
	v, ok := movie["spent_usd"]
	assert.True(t, ok, "a $0 run must send spent_usd: 0")
	assert.Equal(t, 0.0, v)
	assert.NotContains(t, movie, "season_number")
	assert.NotContains(t, movie, "episode_number")
	assert.Equal(t, 1200.0, movie["duration_seconds"])

	// A pre-032 row: no spend recorded → no key (not recorded ≠ $0); no route.
	legacy := events[4]
	assert.NotContains(t, legacy, "spent_usd")
	assert.NotContains(t, legacy, "route")

	// The deleted movie: still listed, title key present and "".
	gone := events[5]
	title, ok := gone["media_title"]
	assert.True(t, ok, "media_title is always sent")
	assert.Equal(t, "", title)
}

// disc-2026-10-run-timeout-not-cancel AC #5: a queue timeout reads
// queue_timeout and a real cancel reads cancelled, on the real migration chain;
// neither leaks the raw error (sentinel, NAS path) onto the wire.
func TestActivity_RecentSubtitleRuns_QueueTimeoutIsNotCancelled(t *testing.T) {
	db := activityLibraryDB(t)
	ctx := context.Background()
	require.NoError(t, repository.NewMovieRepository(db).Create(ctx, &models.Movie{ID: "big", Title: "大片", ReleaseDate: "2024-01-01"}))
	runs := repository.NewSubtitleRunRepository(db)
	mk := func(id, started, completed, errMsg string) {
		c := at(completed)
		require.NoError(t, runs.Create(ctx, &models.SubtitleRun{
			ID: id, MediaID: "big", MediaType: models.SubtitleRunMediaMovie, Status: models.SubtitleRunFailed,
			StartedAt: at(started), CompletedAt: &c, ErrorMessage: errMsg,
		}))
	}
	const extract = "subtitle pipeline: movie big: route: subtitle route: extract /media/movies/Big (2024)/Big.2160p.mkv: SUBTITLE_EXTRACT_WAIT_ABORTED: gave up waiting for the extraction slot: Big.2160p.mkv"
	mk("queued-out", "2026-10-19T02:00:00Z", "2026-10-19T02:15:00Z",
		models.SubtitleRunQueueTimeoutPrefix+extract+" (queued behind 1 for 15m0s): context deadline exceeded")
	mk("stopped", "2026-10-19T01:00:00Z", "2026-10-19T01:03:00Z",
		models.SubtitleRunCancelledPrefix+extract+" (queued behind 1 for 3m0s): context canceled")

	svc := newRecentOnlyService(fakeParse{}, runs)
	events := recentWire(t, svc)
	require.Len(t, events, 2)
	assert.Equal(t, "queue_timeout", events[0]["failure_reason"], "waiting too long is 處理逾時, not 中途停止")
	assert.Equal(t, "cancelled", events[1]["failure_reason"])

	blob, err := json.Marshal(svc.GetActivity(ctx).Recent)
	require.NoError(t, err)
	assert.NotContains(t, string(blob), "SUBTITLE_EXTRACT_WAIT_ABORTED")
	assert.NotContains(t, string(blob), "/media/")
}

// The raw JSON text, for the two keys whose presence is the whole point.
func TestActivity_RecentSubtitleRuns_SpentZeroAndNullOnTheWire(t *testing.T) {
	completed := at("2026-10-19T01:00:00Z")
	free := doneRun("free", completed, models.SubtitleRunCompleted)
	free.Run.SpentUSD = spend(0)
	unrecorded := doneRun("unrecorded", completed.Add(-time.Hour), models.SubtitleRunCompleted)

	blob, err := json.Marshal(newRecentOnlyService(fakeParse{}, fakeRecentRuns{runs: []repository.RecentSubtitleRun{free, unrecorded}}).
		GetActivity(context.Background()).Recent.Events)
	require.NoError(t, err)
	var raw []json.RawMessage
	require.NoError(t, json.Unmarshal(blob, &raw))
	require.Len(t, raw, 2)
	assert.Contains(t, string(raw[0]), `"spent_usd":0`)
	assert.NotContains(t, string(raw[1]), `spent_usd`)
}

// ─── AC #8: wire keys ─────────────────────────────────────────────────────

func TestActivity_RecentWireKeys(t *testing.T) {
	completed := at("2026-10-19T01:25:01.473Z")
	sn, en := 1, 2
	full := repository.RecentSubtitleRun{
		Run: models.SubtitleRun{
			ID: "full", MediaType: models.SubtitleRunMediaEpisode, Status: models.SubtitleRunCompleted,
			Route: models.SubtitleRunRouteTranslate, CueCount: 568, StartedAt: at("2026-10-19T01:21:30.067Z"),
			CompletedAt: &completed, SpentUSD: spend(0.197439),
		},
		MediaTitle: "末日光明", SeasonNumber: &sn, EpisodeNumber: &en,
	}
	failedAt := completed.Add(-time.Hour)
	bare := repository.RecentSubtitleRun{
		Run: models.SubtitleRun{
			ID: "bare", MediaType: models.SubtitleRunMediaMovie, Status: models.SubtitleRunFailed,
			StartedAt:   failedAt.Add(time.Second), // the clock went backwards: no duration
			CompletedAt: &failedAt, ErrorMessage: "",
		},
	}
	parseAt := completed.Add(-2 * time.Hour)
	svc := newRecentOnlyService(
		fakeParse{all: []*models.ParseJob{parseDone("p1", parseAt)}},
		fakeRecentRuns{runs: []repository.RecentSubtitleRun{full, bare}},
	)
	events := recentWire(t, svc)
	require.Len(t, events, 3)

	keys := func(m map[string]any) []string {
		out := []string{}
		for k := range m {
			out = append(out, k)
		}
		return out
	}
	assert.ElementsMatch(t, []string{
		"kind", "result", "at", "media_type", "media_title", "season_number", "episode_number",
		"route", "cue_count", "duration_seconds", "spent_usd",
	}, keys(events[0]))
	assert.ElementsMatch(t, []string{
		"kind", "result", "at", "media_type", "media_title", "failure_reason",
	}, keys(events[1]), "every optional key absent; media_title still sent")
	assert.Equal(t, "other", events[1]["failure_reason"])

	// The parse event is byte-for-byte the v0 shape.
	blob, err := json.Marshal(svc.GetActivity(context.Background()).Recent.Events[2])
	require.NoError(t, err)
	assert.Equal(t, `{"kind":"parse","result":"completed","detail":"p1.mkv","at":"2026-10-18T23:25:01.473Z"}`, string(blob))
}

// ─── AC #3: merge order and the one cap ───────────────────────────────────

func TestActivity_RecentMergesNewestFirstWithOneCap(t *testing.T) {
	base := at("2026-10-19T00:00:00Z")
	var jobs []*models.ParseJob
	var runs []repository.RecentSubtitleRun
	// Interleaved: parse at even hours, subtitle at odd hours; the parse list
	// arrives OLDEST first to prove the merge sorts rather than trusting
	// either source's order.
	for i := 0; i < 6; i++ {
		jobs = append(jobs, parseDone(fmt.Sprintf("p%d", i), base.Add(time.Duration(2*i)*time.Hour)))
	}
	for i := 5; i >= 0; i-- {
		runs = append(runs, doneRun(fmt.Sprintf("s%d", i), base.Add(time.Duration(2*i+1)*time.Hour), models.SubtitleRunCompleted))
	}
	got := newRecentOnlyService(fakeParse{all: jobs}, fakeRecentRuns{runs: runs}).GetActivity(context.Background()).Recent

	require.Equal(t, sectionOK, got.Status)
	require.Len(t, got.Events, recentEventsMax, "one total cap, not one per source")
	for i := 1; i < len(got.Events); i++ {
		assert.Truef(t, got.Events[i-1].At.After(got.Events[i].At), "event %d not older than %d", i, i-1)
	}
	assert.Equal(t, "subtitle", got.Events[0].Kind, "11:00 subtitle is newest")
	assert.Equal(t, "parse", got.Events[1].Kind, "10:00 parse next")
	// The 8 newest of 12 hourly events reach back to 04:00.
	assert.True(t, got.Events[7].At.Equal(base.Add(4*time.Hour)))
}

func TestActivity_RecentTiesAreStable(t *testing.T) {
	same := at("2026-10-19T00:00:00Z")
	got := newRecentOnlyService(
		fakeParse{all: []*models.ParseJob{parseDone("p-b", same), parseDone("p-a", same)}},
		fakeRecentRuns{runs: []repository.RecentSubtitleRun{
			doneRun("s-b", same, models.SubtitleRunCompleted), doneRun("s-a", same, models.SubtitleRunFailed),
		}},
	).GetActivity(context.Background()).Recent

	order := []string{}
	for _, e := range got.Events {
		order = append(order, e.Kind+":"+e.id)
	}
	assert.Equal(t, []string{"subtitle:s-a", "subtitle:s-b", "parse:p-a", "parse:p-b"}, order)
}

// The second lock: whatever the source hands back, only completed/failed runs
// with a completed_at reach the wire.
func TestActivity_RecentSubtitleOnlyTerminal(t *testing.T) {
	now := at("2026-10-19T00:00:00Z")
	open := doneRun("open", now, models.SubtitleRunFailed)
	open.Run.CompletedAt = nil
	got := newRecentOnlyService(fakeParse{}, fakeRecentRuns{runs: []repository.RecentSubtitleRun{
		doneRun("skipped", now, models.SubtitleRunSkipped),
		doneRun("running", now, models.SubtitleRunRunning),
		doneRun("pending", now, models.SubtitleRunPending),
		open,
		doneRun("ok", now, models.SubtitleRunCompleted),
	}}).GetActivity(context.Background()).Recent
	require.Len(t, got.Events, 1)
	assert.Equal(t, "ok", got.Events[0].id)
}

// A season/episode pair only travels with an episode run, and only whole.
func TestActivity_RecentEpisodeNumbersOnlyForEpisodes(t *testing.T) {
	now := at("2026-10-19T00:00:00Z")
	sn := 1
	half := doneRun("half", now, models.SubtitleRunCompleted)
	half.Run.MediaType = models.SubtitleRunMediaEpisode
	half.SeasonNumber = &sn // episode number missing
	movie := doneRun("movie", now.Add(-time.Minute), models.SubtitleRunCompleted)
	movie.SeasonNumber, movie.EpisodeNumber = &sn, &sn

	got := newRecentOnlyService(fakeParse{}, fakeRecentRuns{runs: []repository.RecentSubtitleRun{half, movie}}).
		GetActivity(context.Background()).Recent
	for _, e := range got.Events {
		assert.Nilf(t, e.SeasonNumber, "%s", e.id)
		assert.Nilf(t, e.EpisodeNumber, "%s", e.id)
	}
}

// ─── AC #6: each source degrades on its own ───────────────────────────────

func TestActivity_RecentSourcesDegradeIndependently(t *testing.T) {
	now := at("2026-10-19T00:00:00Z")
	parseOK := fakeParse{all: []*models.ParseJob{parseDone("p", now)}}
	parseBad := fakeParse{allErr: errors.New("parse db locked")}
	runsOK := fakeRecentRuns{runs: []repository.RecentSubtitleRun{doneRun("s", now, models.SubtitleRunCompleted)}}
	runsBad := fakeRecentRuns{err: errors.New("runs db locked")}

	cases := []struct {
		name       string
		parse      parseJobSource
		runs       recentRunSource
		wantStatus string
		wantKinds  []string
	}{
		{"subtitle source fails → parse only", parseOK, runsBad, sectionOK, []string{"parse"}},
		{"parse source fails → subtitle only", parseBad, runsOK, sectionOK, []string{"subtitle"}},
		{"parse not wired → subtitle only", nil, runsOK, sectionOK, []string{"subtitle"}},
		{"subtitle not wired → today's behaviour", parseOK, nil, sectionOK, []string{"parse"}},
		{"both fail → unavailable", parseBad, runsBad, sectionUnavailable, nil},
		{"parse not wired, subtitle fails → unavailable", nil, runsBad, sectionUnavailable, nil},
		// 正式機 shape: no parse jobs at all. A failing subtitle source must
		// show the error + retry, not an "ok" empty record that renders as
		// nothing.
		{"subtitle fails, parse has nothing → unavailable", fakeParse{}, runsBad, sectionUnavailable, nil},
		{"parse fails, subtitle has nothing → unavailable", parseBad, fakeRecentRuns{}, sectionUnavailable, nil},
		{"nothing failed, nothing finished → ok empty", fakeParse{}, fakeRecentRuns{}, sectionOK, []string{}},
		{"parse fails, subtitle not wired → unavailable as today", parseBad, nil, sectionUnavailable, nil},
		{"neither wired → unavailable", nil, nil, sectionUnavailable, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := newRecentOnlyService(tc.parse, tc.runs).GetActivity(context.Background()).Recent
			assert.Equal(t, tc.wantStatus, got.Status)
			kinds := []string{}
			for _, e := range got.Events {
				kinds = append(kinds, e.Kind)
			}
			if tc.wantKinds == nil {
				assert.Empty(t, got.Events)
				assert.NotEmpty(t, got.Error)
				// No repository error text on the wire once the subtitle
				// source is involved (it stays in the log).
				assert.NotContains(t, got.Error, "runs db locked")
				return
			}
			assert.Equal(t, tc.wantKinds, kinds)
			assert.Empty(t, got.Error)
		})
	}
}

// A failing source is logged (AC #6), including when both fail.
func TestActivity_RecentSourceFailureIsLogged(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	now := at("2026-10-19T00:00:00Z")
	newRecentOnlyService(fakeParse{all: []*models.ParseJob{parseDone("p", now)}}, fakeRecentRuns{err: errors.New("runs db locked")}).
		GetActivity(context.Background())
	assert.Contains(t, buf.String(), "subtitle-run source failed")
	assert.Contains(t, buf.String(), "runs db locked")

	buf.Reset()
	newRecentOnlyService(fakeParse{allErr: errors.New("parse db locked")}, fakeRecentRuns{err: errors.New("runs db locked")}).
		GetActivity(context.Background())
	assert.Contains(t, buf.String(), "parse db locked")
	assert.Contains(t, buf.String(), "runs db locked")

	// A source that is simply not wired is not a failure.
	buf.Reset()
	newRecentOnlyService(fakeParse{all: []*models.ParseJob{parseDone("p", now)}}, nil).GetActivity(context.Background())
	assert.Empty(t, buf.String())
}

// 正式機 stores nine fractional digits; `at` keeps them, in UTC, RFC 3339.
func TestActivity_RecentAtKeepsNanosecondsInUTC(t *testing.T) {
	completed := time.Date(2026, 10, 19, 9, 25, 1, 473263247, time.FixedZone("CST", 8*3600))
	run := doneRun("ns", completed, models.SubtitleRunCompleted)
	events := recentWire(t, newRecentOnlyService(fakeParse{}, fakeRecentRuns{runs: []repository.RecentSubtitleRun{run}}))
	require.Len(t, events, 1)
	assert.Equal(t, "2026-10-19T01:25:01.473263247Z", events[0]["at"])
}

// The series row is gone but its episode survived: no title, so no
// "S01E04 of nothing" either.
func TestActivity_RecentNoEpisodeNumbersWithoutTitle(t *testing.T) {
	sn, en := 1, 4
	run := doneRun("orphan", at("2026-10-19T00:00:00Z"), models.SubtitleRunCompleted)
	run.Run.MediaType = models.SubtitleRunMediaEpisode
	run.MediaTitle = ""
	run.SeasonNumber, run.EpisodeNumber = &sn, &en
	events := recentWire(t, newRecentOnlyService(fakeParse{}, fakeRecentRuns{runs: []repository.RecentSubtitleRun{run}}))
	require.Len(t, events, 1)
	assert.Equal(t, "", events[0]["media_title"])
	assert.NotContains(t, events[0], "season_number")
	assert.NotContains(t, events[0], "episode_number")
}

// Today's unavailable message is kept when only the parse source exists.
func TestActivity_RecentParseErrorMessageUnchanged(t *testing.T) {
	got := newRecentOnlyService(fakeParse{allErr: errors.New("parse db locked")}, nil).GetActivity(context.Background()).Recent
	assert.Equal(t, "parse db locked", got.Error)
}
