package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// sub-7-6b — the monthly aggregation is arithmetic over ledger rows with the
// honesty rules baked into the shape: absent is never 0, estimates are named,
// and unknown lanes are reported apart.

type fakeSpendRuns struct {
	runs    []models.SubtitleRun
	batch   map[string][]models.SubtitleRun
	err     error
	gotFrom time.Time
	gotTo   time.Time
}

func (f *fakeSpendRuns) CompletedRunsBetween(_ context.Context, from, to time.Time) ([]models.SubtitleRun, error) {
	f.gotFrom, f.gotTo = from, to
	return f.runs, f.err
}

func (f *fakeSpendRuns) RunsByBatchID(_ context.Context, id string) ([]models.SubtitleRun, error) {
	return f.batch[id], f.err
}

type fakeSpendMovies struct{ byID map[string]*models.Movie }

func (f fakeSpendMovies) FindByID(_ context.Context, id string) (*models.Movie, error) {
	if m, ok := f.byID[id]; ok {
		return m, nil
	}
	return nil, errors.New("not found")
}

type fakeSpendEpisodes struct{ byID map[string]*models.Episode }

func (f fakeSpendEpisodes) FindByID(_ context.Context, id string) (*models.Episode, error) {
	if e, ok := f.byID[id]; ok {
		return e, nil
	}
	return nil, errors.New("not found")
}

func f64(v float64) *float64 { return &v }
func iptr(v int) *int        { return &v }

func run(route, model string, spent *float64, cues int, hits *int) models.SubtitleRun {
	return models.SubtitleRun{
		ID: route + "-" + model, MediaID: "m", MediaType: models.SubtitleRunMediaMovie,
		Status: models.SubtitleRunCompleted, Route: route, ModelID: model,
		SpentUSD: spent, CueCount: cues, CacheHitCues: hits,
	}
}

func TestSpendSummary_SplitsPaidLanesAndGroupsByModel(t *testing.T) {
	src := &fakeSpendRuns{runs: []models.SubtitleRun{
		run(models.SubtitleRunRouteTranslate, "claude-sonnet-5", f64(0.50), 800, nil),
		run(models.SubtitleRunRouteTranslate, "claude-haiku-4-5", f64(0.10), 600, nil),
		run(models.SubtitleRunRouteASR, "claude-sonnet-5", f64(1.25), 900, nil),
		run(models.SubtitleRunRouteSkip, "claude-sonnet-5", f64(0), 0, nil), // bought nothing, saved nothing
	}}
	svc := NewSubtitleSpendService(src, nil, nil, nil)

	got, err := svc.MonthSummary(context.Background(), "")
	require.NoError(t, err)

	assert.Equal(t, 2, got.TranslatedRuns)
	assert.Equal(t, 0.60, got.TranslatedUSD)
	assert.Equal(t, 1, got.ASRRuns)
	assert.Equal(t, 1.25, got.ASRUSD)
	assert.Equal(t, 0, got.UnpricedRuns)
	assert.Equal(t, 0, got.SkippedDeliverCount)
	require.Len(t, got.ByModel, 2)
	assert.Equal(t, SpendByModel{ModelID: "claude-sonnet-5", Runs: 2, USD: 1.75}, got.ByModel[0], "most expensive model first")
	assert.Equal(t, SpendByModel{ModelID: "claude-haiku-4-5", Runs: 1, USD: 0.10}, got.ByModel[1])
	assert.Nil(t, got.CacheSavedUSDEstimate, "no run measured its cache split → null, not $0")
	assert.Equal(t, 0, got.CacheMeasuredRuns)
	assert.Nil(t, got.ByBatch, "no batch_id asked → no by_batch key")
}

func TestSpendSummary_AbsentSpendIsCountedNotZeroed(t *testing.T) {
	src := &fakeSpendRuns{runs: []models.SubtitleRun{
		run(models.SubtitleRunRouteTranslate, "claude-sonnet-5", nil, 100, nil), // stamped before migration 032
		{ID: "old", MediaID: "m", MediaType: models.SubtitleRunMediaMovie, Status: models.SubtitleRunCompleted,
			ModelID: "claude-haiku-4-5", SpentUSD: f64(0.30)}, // pre-041: no route
	}}
	got, err := NewSubtitleSpendService(src, nil, nil, nil).MonthSummary(context.Background(), "")
	require.NoError(t, err)

	assert.Equal(t, 1, got.TranslatedRuns)
	assert.Equal(t, 0.0, got.TranslatedUSD)
	assert.Equal(t, 1, got.UnpricedRuns, "a run with no recorded spend is COUNTED, never priced at $0")
	assert.Equal(t, 1, got.UnroutedRuns, "a pre-ledger row is reported apart, never folded into translated")
	assert.Equal(t, 0.30, got.UnroutedUSD)
	require.Len(t, got.ByModel, 1, "the unrouted row's money is not attributed to a lane's model table")
}

func TestSpendSummary_CacheSavedIsPerCuePriceTimesHits(t *testing.T) {
	src := &fakeSpendRuns{runs: []models.SubtitleRun{
		// $0.90 bought 900 model cues (1,000 − 100 cached) → $0.001/cue → 100 hits saved $0.10.
		run(models.SubtitleRunRouteTranslate, "claude-sonnet-5", f64(0.90), 1000, iptr(100)),
		// Measured but cold: contributes to measured_runs, saves nothing.
		run(models.SubtitleRunRouteTranslate, "claude-haiku-4-5", f64(0.20), 500, iptr(0)),
		// Fully cached ($0 spent, all hits): measured, but no per-cue price to derive — saves nothing, honestly.
		run(models.SubtitleRunRouteTranslate, "claude-haiku-4-5", f64(0), 300, iptr(300)),
		// Not measured (deliver lane): no split.
		run(models.SubtitleRunRouteDeliverDirect, "claude-haiku-4-5", f64(0), 0, nil),
	}}
	got, err := NewSubtitleSpendService(src, nil, nil, nil).MonthSummary(context.Background(), "")
	require.NoError(t, err)

	assert.Equal(t, 3, got.CacheMeasuredRuns)
	assert.Equal(t, 400, got.CacheHitCues)
	require.NotNil(t, got.CacheSavedUSDEstimate)
	assert.Equal(t, 0.10, *got.CacheSavedUSDEstimate)
}

func TestSpendSummary_SkippedSavedClimbsTheRuntimeLadderAndFlagsTheAssumption(t *testing.T) {
	movies := fakeSpendMovies{byID: map[string]*models.Movie{
		"probed": {DurationSeconds: models.NewNullInt64(120 * 60)}, // 120 min measured
		"tmdb":   {Runtime: models.NewNullInt64(90)},               // 90 min editorial
	}}
	episodes := fakeSpendEpisodes{byID: map[string]*models.Episode{
		"ep": {DurationSeconds: models.NewNullInt64(45 * 60)},
	}}
	deliver := func(id, mediaType, model string) models.SubtitleRun {
		return models.SubtitleRun{ID: id, MediaID: id, MediaType: mediaType, Status: models.SubtitleRunCompleted,
			Route: models.SubtitleRunRouteDeliverDirect, ModelID: model, SpentUSD: f64(0)}
	}

	t.Run("every runtime known → no assumption flag", func(t *testing.T) {
		src := &fakeSpendRuns{runs: []models.SubtitleRun{
			deliver("probed", models.SubtitleRunMediaMovie, "claude-sonnet-5"),
			deliver("tmdb", models.SubtitleRunMediaMovie, "claude-haiku-4-5"),
			deliver("ep", models.SubtitleRunMediaEpisode, "claude-sonnet-5"),
		}}
		got, err := NewSubtitleSpendService(src, movies, episodes, nil).MonthSummary(context.Background(), "")
		require.NoError(t, err)

		assert.Equal(t, 3, got.SkippedDeliverCount)
		assert.False(t, got.SkippedSavedRuntimeAssumed)
		// 120×0.00804 + 90×0.00301 + 45×0.00804 — the SAME per-minute rates the consent estimate quotes.
		want := roundUSD(estimateUSD(RouteExtract, 120, decimal.Zero, "claude-sonnet-5")).
			Add(roundUSD(estimateUSD(RouteExtract, 90, decimal.Zero, "claude-haiku-4-5"))).
			Add(roundUSD(estimateUSD(RouteExtract, 45, decimal.Zero, "claude-sonnet-5")))
		assert.InDelta(t, want.InexactFloat64(), got.SkippedSavedUSDEstimate, 0.011, "cents-rounded sum of per-item estimates")
		assert.Greater(t, got.SkippedSavedUSDEstimate, 1.0)
	})

	t.Run("an unknown runtime falls to 45 min AND raises the flag the UI renders as ≈", func(t *testing.T) {
		src := &fakeSpendRuns{runs: []models.SubtitleRun{
			deliver("probed", models.SubtitleRunMediaMovie, "claude-sonnet-5"),
			deliver("nobody-knows", models.SubtitleRunMediaMovie, "claude-sonnet-5"),
		}}
		got, err := NewSubtitleSpendService(src, movies, episodes, nil).MonthSummary(context.Background(), "")
		require.NoError(t, err)
		assert.True(t, got.SkippedSavedRuntimeAssumed)
		assert.Equal(t, 2, got.SkippedDeliverCount)
	})

	t.Run("no readers wired → every skipped item is assumed", func(t *testing.T) {
		src := &fakeSpendRuns{runs: []models.SubtitleRun{deliver("probed", models.SubtitleRunMediaMovie, "")}}
		got, err := NewSubtitleSpendService(src, nil, nil, nil).MonthSummary(context.Background(), "")
		require.NoError(t, err)
		assert.True(t, got.SkippedSavedRuntimeAssumed)
		assert.Greater(t, got.SkippedSavedUSDEstimate, 0.0, "a blank model prices on the calibration anchor, never $0")
	})
}

func TestSpendSummary_MonthWindowIsTheServerZoneCalendarMonth(t *testing.T) {
	taipei := time.FixedZone("Asia/Taipei", 8*3600)
	src := &fakeSpendRuns{}
	svc := NewSubtitleSpendService(src, nil, nil, nil)
	svc.SetClock(func() time.Time { return time.Date(2026, 10, 1, 0, 30, 0, 0, taipei) })

	got, err := svc.MonthSummary(context.Background(), "")
	require.NoError(t, err)

	assert.Equal(t, "month", got.Period)
	assert.Equal(t, time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC), src.gotFrom, "Oct 1 00:00 Taipei = Sep 30 16:00 UTC — half an hour into the month is STILL this month")
	assert.Equal(t, time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC), src.gotTo)
	assert.Equal(t, src.gotFrom, got.From)
	assert.Equal(t, src.gotTo, got.To)
}

func TestBatchReceipt_SumsEveryStatusButCountsCuesOnCompleted(t *testing.T) {
	src := &fakeSpendRuns{batch: map[string][]models.SubtitleRun{"b1": {
		{ID: "1", Status: models.SubtitleRunCompleted, Route: models.SubtitleRunRouteTranslate, ModelID: "claude-sonnet-5",
			SpentUSD: f64(0.40), CueCount: 500, CacheHitCues: iptr(50)},
		{ID: "2", Status: models.SubtitleRunCompleted, Route: models.SubtitleRunRouteDeliverDirect, ModelID: "claude-sonnet-5",
			SpentUSD: f64(0), CueCount: 700},
		{ID: "3", Status: models.SubtitleRunFailed, Route: models.SubtitleRunRouteASR, ModelID: "claude-sonnet-5",
			SpentUSD: f64(0.13)}, // a budget-paused ASR item: the money is gone
	}}}
	svc := NewSubtitleSpendService(src, nil, nil, nil)

	got, err := svc.MonthSummary(context.Background(), "b1")
	require.NoError(t, err)
	require.NotNil(t, got.ByBatch)
	r := got.ByBatch
	assert.Equal(t, "b1", r.BatchID)
	assert.Equal(t, 3, r.Runs)
	assert.Equal(t, 2, r.CompletedRuns)
	assert.Equal(t, 0.53, r.USD, "failed runs cost money too")
	assert.Equal(t, 1200, r.CueCount)
	require.NotNil(t, r.CacheHitCues)
	assert.Equal(t, 50, *r.CacheHitCues)
	assert.Equal(t, 1, r.CacheMeasuredRuns)
	assert.Equal(t, "claude-sonnet-5", r.ModelID)
	assert.Empty(t, r.ModelIDs)

	unknown, err := svc.BatchReceipt(context.Background(), "nope")
	require.NoError(t, err)
	assert.Equal(t, 0, unknown.Runs)
	assert.Nil(t, unknown.CacheHitCues)
}

func TestBatchReceipt_NamesEveryModelWhenTheBatchChangedMidway(t *testing.T) {
	src := &fakeSpendRuns{batch: map[string][]models.SubtitleRun{"b2": {
		{ID: "1", Status: models.SubtitleRunCompleted, ModelID: "claude-sonnet-5", SpentUSD: f64(0.1)},
		{ID: "2", Status: models.SubtitleRunCompleted, ModelID: "claude-haiku-4-5", SpentUSD: f64(0.1)},
	}}}
	got, err := NewSubtitleSpendService(src, nil, nil, nil).BatchReceipt(context.Background(), "b2")
	require.NoError(t, err)
	assert.Equal(t, "", got.ModelID)
	assert.Equal(t, []string{"claude-haiku-4-5", "claude-sonnet-5"}, got.ModelIDs)
}

func TestSpendSummary_LedgerErrorSurfaces(t *testing.T) {
	src := &fakeSpendRuns{err: errors.New("db locked")}
	_, err := NewSubtitleSpendService(src, nil, nil, nil).MonthSummary(context.Background(), "")
	require.Error(t, err)
}
