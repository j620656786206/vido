package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// disc-2026-11-spend-history-b-api — the arithmetic over ledger rows. The
// repository's real-SQLite read is in repository/subtitle_run_history_test.go.

type fakeRunHistorySource struct {
	rows     []repository.RecentSubtitleRun
	err      error
	from, to time.Time
}

func (f *fakeRunHistorySource) FinishedRunsStartedBetween(_ context.Context, from, to time.Time) ([]repository.RecentSubtitleRun, error) {
	f.from, f.to = from, to
	return f.rows, f.err
}

var historyTaipei = time.FixedZone("Asia/Taipei", 8*3600)

func newHistoryService(src *fakeRunHistorySource, now time.Time) *SubtitleRunHistoryService {
	svc := NewSubtitleRunHistoryService(src, nil)
	svc.SetClock(func() time.Time { return now })
	return svc
}

func hp(v float64) *float64 { return &v }
func ip(v int) *int         { return &v }

// histRun builds one row; started is a Taipei wall-clock "2006-01-02 15:04".
func histRun(id, route string, status models.SubtitleRunStatus, started string, spent, estimate *float64) repository.RecentSubtitleRun {
	at, err := time.ParseInLocation("2006-01-02 15:04", started, historyTaipei)
	if err != nil {
		panic(err)
	}
	return repository.RecentSubtitleRun{
		Run: models.SubtitleRun{
			ID: id, MediaID: "m-" + id, MediaType: models.SubtitleRunMediaMovie, Status: status,
			Route: route, ModelID: "claude-sonnet-5", StartedAt: at.UTC(), SpentUSD: spent, EstimateUSD: estimate,
		},
		MediaTitle: "片名 " + id,
	}
}

func TestRunCategory_FollowsTheDesignTable(t *testing.T) {
	for route, want := range map[string]string{
		models.SubtitleRunRouteDeliverDirect:      RunCategoryChineseTrack,
		models.SubtitleRunRouteConvertThenDeliver: RunCategoryChineseTrack,
		models.SubtitleRunRouteTranslate:          RunCategoryTranslateEnglish,
		models.SubtitleRunRouteASR:                RunCategoryASR,
		"":                                        RunCategoryLegacy,
		models.SubtitleRunRouteNoTextSource:       RunCategoryNoSubtitle,
		models.SubtitleRunRouteSkip:               RunCategoryNoSubtitle,
		models.SubtitleRunRouteOnline:             RunCategoryNoSubtitle,
	} {
		assert.Equal(t, want, RunCategory(&models.SubtitleRun{Route: route}), "route %q", route)
	}
}

func TestRunHistory_CurrentMonthWindowAndSeriesStopAtToday(t *testing.T) {
	src := &fakeRunHistorySource{rows: []repository.RecentSubtitleRun{
		// Newest first, as the repository returns them.
		histRun("asr", models.SubtitleRunRouteASR, models.SubtitleRunCompleted, "2026-11-03 21:00", hp(3.10), nil),
		histRun("tr", models.SubtitleRunRouteTranslate, models.SubtitleRunFailed, "2026-11-03 08:00", hp(0.814), nil),
		histRun("tr-null", models.SubtitleRunRouteTranslate, models.SubtitleRunCompleted, "2026-11-03 07:00", nil, nil),
		histRun("skip", models.SubtitleRunRouteNoTextSource, models.SubtitleRunSkipped, "2026-11-03 06:00", nil, nil),
		// 00:30 Taipei on Nov 1 is Oct 31 in UTC — it belongs to Nov 1 here.
		histRun("deliver", models.SubtitleRunRouteDeliverDirect, models.SubtitleRunCompleted, "2026-11-01 00:30", hp(0), nil),
	}}
	svc := newHistoryService(src, time.Date(2026, 11, 3, 22, 0, 0, 0, historyTaipei))

	got, err := svc.History(context.Background(), RunHistoryQuery{})
	require.NoError(t, err)

	assert.Equal(t, "2026-11", got.Month)
	assert.Equal(t, time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC), src.from, "month starts at local midnight")
	assert.Equal(t, time.Date(2026, 11, 30, 16, 0, 0, 0, time.UTC), src.to)

	require.Len(t, got.Series, 3, "the current month runs only through today")
	day1, day2, day3 := got.Series[0], got.Series[1], got.Series[2]
	assert.Equal(t, "2026-11-01", day1.Date)
	assert.Equal(t, 1, day1.Runs)
	assert.Equal(t, 0.0, day1.SpentUSD)
	assert.Equal(t, RunHistoryDay{Date: "2026-11-02"}, day2, "a day with no runs is a real zero")
	assert.Equal(t, "2026-11-03", day3.Date)
	assert.Equal(t, 3, day3.Runs, "null-spend runs count; no_subtitle runs do not")
	assert.Equal(t, RunHistoryByCategory{TranslateEnglish: 0.81, ASR: 3.10}, day3.ByCategory)
	assert.Equal(t, 3.91, day3.SpentUSD)
	for _, d := range got.Series {
		c := d.ByCategory
		assert.InDelta(t, d.SpentUSD, c.ChineseTrack+c.TranslateEnglish+c.ASR+c.Legacy, 1e-9, d.Date)
	}

	assert.Equal(t, 5, got.Summary.Runs)
	assert.Equal(t, 1, got.Summary.UnpricedRuns, "the skipped run bought nothing")
	assert.Equal(t, 3.91, got.Summary.SpentUSD, "failed runs' money is counted")
	assert.Equal(t, 0, got.Summary.ComparableRuns)
	assert.Nil(t, got.Summary.ComparableSpentUSD, "no comparable run → null, never 0")
	assert.Nil(t, got.Summary.ComparableEstimateUSD)

	assert.Equal(t, 1, got.Page)
	assert.Equal(t, RunHistoryDefaultPageSize, got.PageSize)
	assert.Equal(t, 5, got.Total)
	require.Len(t, got.Items, 5)
	assert.Equal(t, "asr", got.Items[0].ID)
}

func TestRunHistory_PastMonthHasEveryDayAndFutureMonthNone(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, historyTaipei)

	got, err := newHistoryService(&fakeRunHistorySource{}, now).History(context.Background(), RunHistoryQuery{Month: "2026-02"})
	require.NoError(t, err)
	assert.Equal(t, "2026-02", got.Month)
	require.Len(t, got.Series, 28)
	assert.Equal(t, "2026-02-28", got.Series[27].Date)
	assert.Equal(t, 0, got.Summary.Runs)
	assert.NotNil(t, got.Items, "items is [] on the wire, not null")

	got, err = newHistoryService(&fakeRunHistorySource{}, now).History(context.Background(), RunHistoryQuery{Month: "2026-12"})
	require.NoError(t, err)
	assert.Empty(t, got.Series, "no day of a future month has happened")
	assert.NotNil(t, got.Series)
}

func TestRunHistory_ComparableOnlyCompletedRunsWithBothNumbers(t *testing.T) {
	src := &fakeRunHistorySource{rows: []repository.RecentSubtitleRun{
		histRun("both-a", models.SubtitleRunRouteASR, models.SubtitleRunCompleted, "2026-11-02 10:00", hp(0.70), hp(0.80)),
		histRun("both-b", models.SubtitleRunRouteTranslate, models.SubtitleRunCompleted, "2026-11-02 09:00", hp(0.50), hp(0.55)),
		histRun("spent-only", models.SubtitleRunRouteTranslate, models.SubtitleRunCompleted, "2026-11-02 08:00", hp(0.40), nil),
		histRun("estimate-only", models.SubtitleRunRouteTranslate, models.SubtitleRunCompleted, "2026-11-02 07:00", nil, hp(0.30)),
		// a failed run spent a fraction of its quote: counted in spend, not in the ratio
		histRun("failed", models.SubtitleRunRouteASR, models.SubtitleRunFailed, "2026-11-02 06:00", hp(0.10), hp(0.90)),
		// a skipped run bought nothing: its missing spend is not "unpriced"
		histRun("skipped", "", models.SubtitleRunSkipped, "2026-11-02 05:00", nil, nil),
	}}
	got, err := newHistoryService(src, time.Date(2026, 11, 3, 12, 0, 0, 0, historyTaipei)).History(context.Background(), RunHistoryQuery{})
	require.NoError(t, err)

	assert.Equal(t, 2, got.Summary.ComparableRuns)
	require.NotNil(t, got.Summary.ComparableSpentUSD)
	require.NotNil(t, got.Summary.ComparableEstimateUSD)
	assert.Equal(t, 1.20, *got.Summary.ComparableSpentUSD)
	assert.Equal(t, 1.35, *got.Summary.ComparableEstimateUSD)
	assert.Equal(t, 1.70, got.Summary.SpentUSD)
	assert.Equal(t, 1, got.Summary.UnpricedRuns)
}

func TestRunHistory_CategoryFiltersItemsOnlyAndPages(t *testing.T) {
	src := &fakeRunHistorySource{rows: []repository.RecentSubtitleRun{
		histRun("tr-3", models.SubtitleRunRouteTranslate, models.SubtitleRunCompleted, "2026-11-02 12:00", hp(0.3), nil),
		histRun("asr", models.SubtitleRunRouteASR, models.SubtitleRunCompleted, "2026-11-02 11:00", hp(1), nil),
		histRun("tr-2", models.SubtitleRunRouteTranslate, models.SubtitleRunCompleted, "2026-11-02 10:00", hp(0.2), nil),
		histRun("tr-1", models.SubtitleRunRouteTranslate, models.SubtitleRunCompleted, "2026-11-02 09:00", hp(0.1), nil),
	}}
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, historyTaipei)
	all, err := newHistoryService(src, now).History(context.Background(), RunHistoryQuery{})
	require.NoError(t, err)

	got, err := newHistoryService(src, now).History(context.Background(),
		RunHistoryQuery{Category: RunCategoryTranslateEnglish, Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, got.Total)
	require.Len(t, got.Items, 1)
	assert.Equal(t, "tr-1", got.Items[0].ID)
	assert.Equal(t, all.Summary, got.Summary, "category does not touch the summary")
	assert.Equal(t, all.Series, got.Series, "category does not touch the series")

	beyond, err := newHistoryService(src, now).History(context.Background(), RunHistoryQuery{Page: 9})
	require.NoError(t, err)
	assert.Equal(t, 4, beyond.Total)
	assert.Empty(t, beyond.Items)
	assert.NotNil(t, beyond.Items)
}

func TestRunHistory_ItemShapeKeepsUnrecordedValuesNull(t *testing.T) {
	asr := histRun("asr", models.SubtitleRunRouteASR, models.SubtitleRunCompleted, "2026-11-02 10:00", hp(0.52), hp(0.6))
	asr.Run.MediaType = models.SubtitleRunMediaEpisode
	asr.Run.Source = models.SubtitleRunSourceAudio
	asr.Run.ASRProvider, asr.Run.ASRModel = "elevenlabs", "scribe_v2"
	asr.Run.BatchID, asr.Run.TriggeredBy = "b1", models.SubtitleRunTriggeredManual
	asr.MediaTitle, asr.SeasonNumber, asr.EpisodeNumber = "Lioness", ip(1), ip(1)
	done := asr.Run.StartedAt.Add(5 * time.Minute)
	asr.Run.CompletedAt = &done

	deliver := histRun("deliver", models.SubtitleRunRouteDeliverDirect, models.SubtitleRunCompleted, "2026-11-02 09:00", hp(0), nil)

	failed := histRun("failed", models.SubtitleRunRouteTranslate, models.SubtitleRunFailed, "2026-11-02 08:00", hp(0.12), nil)
	failed.Run.ErrorMessage = models.SubtitleRunCancelledPrefix + "context canceled"

	legacy := histRun("legacy", "", models.SubtitleRunCompleted, "2026-11-02 07:00", nil, nil)
	legacy.MediaTitle = "" // left the library

	orphanEp := histRun("orphan", models.SubtitleRunRouteTranslate, models.SubtitleRunCompleted, "2026-11-02 06:00", hp(0.1), nil)
	orphanEp.Run.MediaType = models.SubtitleRunMediaEpisode
	orphanEp.MediaTitle, orphanEp.SeasonNumber, orphanEp.EpisodeNumber = "", ip(1), ip(4)

	// a September run (before routes were recorded) that cost money could
	// only have been a translation, so it names its model
	paidLegacy := histRun("paid-legacy", "", models.SubtitleRunCompleted, "2026-11-02 05:00", hp(1.19), nil)
	paidLegacy.Run.ModelID = "claude-haiku-4-5"

	src := &fakeRunHistorySource{rows: []repository.RecentSubtitleRun{asr, deliver, failed, legacy, orphanEp, paidLegacy}}
	got, err := newHistoryService(src, time.Date(2026, 11, 3, 12, 0, 0, 0, historyTaipei)).History(context.Background(), RunHistoryQuery{})
	require.NoError(t, err)
	require.Len(t, got.Items, 6)
	byID := map[string]RunHistoryItem{}
	for _, it := range got.Items {
		byID[it.ID] = it
	}

	a := byID["asr"]
	assert.Equal(t, "completed", a.Status)
	assert.Equal(t, "", a.FailureReason)
	assert.Equal(t, RunCategoryASR, a.Category)
	require.NotNil(t, a.Media.Title)
	assert.Equal(t, "Lioness", *a.Media.Title)
	assert.Equal(t, ip(1), a.Media.Season)
	assert.Equal(t, ip(1), a.Media.Episode)
	assert.Equal(t, "episode", a.Media.Type)
	assert.Equal(t, hp(0.52), a.SpentUSD)
	assert.Equal(t, hp(0.6), a.EstimateUSD)
	require.NotNil(t, a.TranslationModel)
	assert.Equal(t, "claude-sonnet-5", *a.TranslationModel, "asr translates, so it names the model")
	assert.Equal(t, "elevenlabs", *a.ASRProvider)
	assert.Equal(t, "scribe_v2", *a.ASRModel)
	assert.Equal(t, "audio", *a.Source)
	assert.Equal(t, "asr", *a.Route)
	assert.Equal(t, "b1", *a.BatchID)
	assert.Equal(t, "manual", *a.TriggeredBy)
	require.NotNil(t, a.CompletedAt)
	assert.Equal(t, time.UTC, a.StartedAt.Location())

	d := byID["deliver"]
	assert.Equal(t, RunCategoryChineseTrack, d.Category)
	assert.Nil(t, d.TranslationModel, "model_id is only a version tag on a free lane")
	require.NotNil(t, d.SpentUSD)
	assert.Equal(t, 0.0, *d.SpentUSD, "a real $0 stays 0")
	assert.Nil(t, d.EstimateUSD)
	assert.Nil(t, d.Source)
	assert.Nil(t, d.ASRProvider)
	assert.Nil(t, d.ASRModel)
	assert.Nil(t, d.BatchID)
	assert.Nil(t, d.TriggeredBy)
	assert.Nil(t, d.CompletedAt)
	assert.Nil(t, d.Media.Season, "a movie has no season")

	f := byID["failed"]
	assert.Equal(t, "failed", f.Status)
	assert.Equal(t, models.SubtitleRunFailureCancelled, f.FailureReason)
	assert.Equal(t, hp(0.12), f.SpentUSD)

	l := byID["legacy"]
	assert.Equal(t, RunCategoryLegacy, l.Category)
	assert.Nil(t, l.Route, "an unrecorded route is null")
	assert.Nil(t, l.TranslationModel)
	assert.Nil(t, l.SpentUSD)
	assert.Nil(t, l.Media.Title, "deleted media → title null")

	pl := byID["paid-legacy"]
	require.NotNil(t, pl.TranslationModel)
	assert.Equal(t, "claude-haiku-4-5", *pl.TranslationModel)

	o := byID["orphan"]
	assert.Nil(t, o.Media.Title)
	assert.Nil(t, o.Media.Season, "numbers only travel with a title")
	assert.Nil(t, o.Media.Episode)
}

func TestRunHistory_RejectsMalformedQueries(t *testing.T) {
	svc := newHistoryService(&fakeRunHistorySource{}, time.Date(2026, 11, 3, 12, 0, 0, 0, historyTaipei))
	for _, q := range []RunHistoryQuery{
		{Month: "2026-1"}, {Month: "2026-13"}, {Month: "26-11"}, {Month: "2026/11"}, {Month: "2026-11-01"},
		{Category: "translate"}, {Category: "ASR"},
		{Page: -1}, {PageSize: -1}, {PageSize: RunHistoryMaxPageSize + 1},
	} {
		_, err := svc.History(context.Background(), q)
		assert.ErrorIs(t, err, ErrInvalidRunHistoryQuery, "%+v", q)
	}
}

func TestRunHistory_LedgerErrorPropagates(t *testing.T) {
	boom := errors.New("db locked")
	svc := newHistoryService(&fakeRunHistorySource{err: boom}, time.Now())
	_, err := svc.History(context.Background(), RunHistoryQuery{})
	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, ErrInvalidRunHistoryQuery)
}
