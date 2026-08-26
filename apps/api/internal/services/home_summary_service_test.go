package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// --- fakes over the narrow home-summary sources ---

type fakeHomeMovies struct {
	total, covered int
	totalErr       error
	coveredErr     error
}

func (f *fakeHomeMovies) Count(context.Context) (int, error) { return f.total, f.totalErr }
func (f *fakeHomeMovies) CountZhHantSubtitle(context.Context) (int, error) {
	return f.covered, f.coveredErr
}

type fakeHomeSeries struct {
	total, covered int
	coveredErr     error
}

func (f *fakeHomeSeries) Count(context.Context) (int, error) { return f.total, nil }
func (f *fakeHomeSeries) CountZhHantCovered(context.Context) (int, error) {
	return f.covered, f.coveredErr
}

type fakeHomeParse struct {
	failed    int
	failedErr error
	ids       []string
	idsErr    error
}

func (f *fakeHomeParse) CountByStatus(context.Context, models.ParseJobStatus) (int, error) {
	return f.failed, f.failedErr
}
func (f *fakeHomeParse) CompletedMediaIDsSince(context.Context, time.Time) ([]string, error) {
	return f.ids, f.idsErr
}

type fakeHomeRuns struct {
	failed    int
	failedErr error
	refs      []repository.SubtitleRunMediaRef
	latest    *models.SubtitleRun
	latestErr error
}

func (f *fakeHomeRuns) CountByStatus(context.Context, models.SubtitleRunStatus) (int, error) {
	return f.failed, f.failedErr
}
func (f *fakeHomeRuns) CompletedMediaRefsSince(context.Context, time.Time) ([]repository.SubtitleRunMediaRef, error) {
	return f.refs, nil
}
func (f *fakeHomeRuns) LatestWithSpend(context.Context) (*models.SubtitleRun, error) {
	return f.latest, f.latestErr
}

type fakeHomeScan struct{ active bool }

func (f *fakeHomeScan) IsScanActive() bool        { return f.active }
func (f *fakeHomeScan) GetProgress() ScanProgress { return ScanProgress{} }

type fakeHomeBatch struct{ active bool }

func (f *fakeHomeBatch) ActivityProgress() (bool, int, int, int, string) {
	return f.active, 50, 1, 2, "item"
}

type fakeLiveSpend struct{ prog *GenerationBatchProgress }

func (f *fakeLiveSpend) GetProgress() *GenerationBatchProgress { return f.prog }

func newHomeService(movies *fakeHomeMovies, series *fakeHomeSeries, parse *fakeHomeParse, runs *fakeHomeRuns, live *fakeLiveSpend) *HomeSummaryService {
	s := NewHomeSummaryService(movies, series, parse, runs,
		&fakeHomeScan{}, &fakeHomeBatch{}, &fakeHomeBatch{}, &fakeHomeBatch{}, live)
	s.now = func() time.Time { return time.Date(2026, 8, 26, 14, 30, 0, 0, time.UTC) }
	return s
}

func TestHomeSummary_AllCellsOK(t *testing.T) {
	s := newHomeService(
		&fakeHomeMovies{total: 30, covered: 20},
		&fakeHomeSeries{total: 25, covered: 22},
		&fakeHomeParse{failed: 1, ids: []string{"a", "b"}},
		&fakeHomeRuns{failed: 1, refs: []repository.SubtitleRunMediaRef{{MediaID: "c", MediaType: "movie"}}},
		&fakeLiveSpend{},
	)
	sum := s.GetSummary(context.Background())

	assert.Equal(t, sectionOK, sum.Coverage.Status)
	assert.Equal(t, 42, sum.Coverage.Covered)
	assert.Equal(t, 55, sum.Coverage.Total)
	assert.Equal(t, sectionOK, sum.ProcessedToday.Status)
	assert.Equal(t, 3, sum.ProcessedToday.Count)
	assert.Equal(t, sectionOK, sum.Attention.Status)
	assert.Equal(t, 2, sum.Attention.FailedCount)
	assert.Equal(t, sectionOK, sum.InFlight.Status)
	assert.Equal(t, 0, sum.InFlight.Count)
}

func TestHomeSummary_ProcessedTodayDedupesAcrossSources(t *testing.T) {
	// "a" was parsed AND subtitled today — one 部, not two.
	s := newHomeService(
		&fakeHomeMovies{}, &fakeHomeSeries{},
		&fakeHomeParse{ids: []string{"a", "b"}},
		&fakeHomeRuns{refs: []repository.SubtitleRunMediaRef{{MediaID: "a", MediaType: "movie"}}},
		&fakeLiveSpend{},
	)
	sum := s.GetSummary(context.Background())
	assert.Equal(t, 2, sum.ProcessedToday.Count)
}

func TestHomeSummary_CoverageFailSoft(t *testing.T) {
	s := newHomeService(
		&fakeHomeMovies{coveredErr: errors.New("db locked")},
		&fakeHomeSeries{},
		&fakeHomeParse{ids: []string{"a"}},
		&fakeHomeRuns{},
		&fakeLiveSpend{},
	)
	sum := s.GetSummary(context.Background())
	assert.Equal(t, sectionUnavailable, sum.Coverage.Status)
	assert.Equal(t, "db locked", sum.Coverage.Error)
	// Siblings still serve.
	assert.Equal(t, sectionOK, sum.ProcessedToday.Status)
	assert.Equal(t, sectionOK, sum.Attention.Status)
	assert.Equal(t, sectionOK, sum.InFlight.Status)
}

func TestHomeSummary_NilSourcesDegradeTheirOwnCells(t *testing.T) {
	s := NewHomeSummaryService(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	sum := s.GetSummary(context.Background())
	assert.Equal(t, sectionUnavailable, sum.Coverage.Status)
	assert.Equal(t, sectionUnavailable, sum.ProcessedToday.Status)
	assert.Equal(t, sectionUnavailable, sum.Attention.Status)
	// In-flight is in-memory: nil sources mean "nothing running", a valid 0.
	assert.Equal(t, sectionOK, sum.InFlight.Status)
	assert.Equal(t, 0, sum.InFlight.Count)
}

func TestHomeSummary_SpendPrecedence_LiveBatchWins(t *testing.T) {
	spent := 0.42
	s := newHomeService(
		&fakeHomeMovies{}, &fakeHomeSeries{}, &fakeHomeParse{},
		&fakeHomeRuns{latest: &models.SubtitleRun{SpentUSD: &spent, BudgetUSD: &spent}},
		&fakeLiveSpend{prog: &GenerationBatchProgress{SpentUSD: 1.2, BudgetUSD: 5}},
	)
	cell := s.GetSummary(context.Background()).Attention
	assert.Equal(t, "live_batch", cell.SpendSource)
	assert.InDelta(t, 1.2, *cell.SpentUSD, 1e-9)
	assert.InDelta(t, 5.0, *cell.BudgetUSD, 1e-9)
}

func TestHomeSummary_SpendPrecedence_LastRunWhenIdle(t *testing.T) {
	spent, budget := 0.42, 5.0
	s := newHomeService(
		&fakeHomeMovies{}, &fakeHomeSeries{}, &fakeHomeParse{},
		&fakeHomeRuns{latest: &models.SubtitleRun{SpentUSD: &spent, BudgetUSD: &budget}},
		&fakeLiveSpend{},
	)
	cell := s.GetSummary(context.Background()).Attention
	assert.Equal(t, "last_run", cell.SpendSource)
	assert.InDelta(t, 0.42, *cell.SpentUSD, 1e-9)
}

func TestHomeSummary_SpendAbsentIsNotZero(t *testing.T) {
	s := newHomeService(&fakeHomeMovies{}, &fakeHomeSeries{}, &fakeHomeParse{}, &fakeHomeRuns{}, &fakeLiveSpend{})
	cell := s.GetSummary(context.Background()).Attention
	assert.Nil(t, cell.SpentUSD)
	assert.Nil(t, cell.BudgetUSD)
	assert.Empty(t, cell.SpendSource)
}

func TestHomeSummary_SpendResolutionFailureKeepsFailedCount(t *testing.T) {
	s := newHomeService(
		&fakeHomeMovies{}, &fakeHomeSeries{},
		&fakeHomeParse{failed: 2},
		&fakeHomeRuns{failed: 1, latestErr: errors.New("db locked")},
		&fakeLiveSpend{},
	)
	cell := s.GetSummary(context.Background()).Attention
	assert.Equal(t, sectionOK, cell.Status)
	assert.Equal(t, 3, cell.FailedCount)
	assert.Nil(t, cell.SpentUSD)
}

func TestHomeSummary_InFlightCountsActiveSources(t *testing.T) {
	s := NewHomeSummaryService(
		&fakeHomeMovies{}, &fakeHomeSeries{}, &fakeHomeParse{}, &fakeHomeRuns{},
		&fakeHomeScan{active: true}, &fakeHomeBatch{active: true}, &fakeHomeBatch{}, &fakeHomeBatch{active: true},
		&fakeLiveSpend{},
	)
	sum := s.GetSummary(context.Background())
	assert.Equal(t, 3, sum.InFlight.Count)
}
