package services

import (
	"context"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// HomeSummaryService composes the four readout-band cells behind the Home v3
// header (GET /api/v1/home-summary — Story ux3-1-6, tech-spec D1/D2/D3). Like
// StatusSummaryService/ActivityService it is fail-soft per cell (ADR B1/F3): a
// broken source marks ONLY its own cell "unavailable" and the endpoint never
// returns an error envelope — the band renders its healthy cells and hides the
// number on the broken one (the brief's honesty rule: 量不到的格不顯示數字).
type HomeSummaryService struct {
	movies homeMovieCountSource
	series homeSeriesCountSource
	parse  homeParseJobSource
	runs   homeSubtitleRunSource
	// The in-flight cell reuses the SAME source wiring as /activity's
	// active_jobs (tech-spec D2: one counting path with the nav badge — the
	// concrete instances in main.go are identical).
	scan          scanStateSource
	batch         batchJobSource
	generation    batchJobSource
	transcription batchJobSource
	// liveSpend resolves the attention cell's spend while a generation batch is
	// running (tech-spec D3 precedence: live batch wins over the last persisted
	// run).
	liveSpend liveSpendSource

	// now is injectable for deterministic day-anchor tests.
	now func() time.Time
}

// Narrow interfaces (testability) — satisfied by the concrete repositories and
// GenerationBatchProcessor in main.go.
type homeMovieCountSource interface {
	Count(ctx context.Context) (int, error)
	CountZhHantSubtitle(ctx context.Context) (int, error)
}
type homeSeriesCountSource interface {
	Count(ctx context.Context) (int, error)
	CountZhHantCovered(ctx context.Context) (int, error)
}
type homeParseJobSource interface {
	CountByStatus(ctx context.Context, status models.ParseJobStatus) (int, error)
	CompletedMediaIDsSince(ctx context.Context, since time.Time) ([]string, error)
}
type homeSubtitleRunSource interface {
	CountByStatus(ctx context.Context, status models.SubtitleRunStatus) (int, error)
	CompletedMediaRefsSince(ctx context.Context, since time.Time) ([]repository.SubtitleRunMediaRef, error)
	LatestWithSpend(ctx context.Context) (*models.SubtitleRun, error)
}
type liveSpendSource interface {
	GetProgress() *GenerationBatchProgress
}

// NewHomeSummaryService wires the sources the four cells read.
func NewHomeSummaryService(
	movies homeMovieCountSource,
	series homeSeriesCountSource,
	parse homeParseJobSource,
	runs homeSubtitleRunSource,
	scan scanStateSource,
	batch batchJobSource,
	generation batchJobSource,
	transcription batchJobSource,
	liveSpend liveSpendSource,
) *HomeSummaryService {
	return &HomeSummaryService{
		movies: movies, series: series, parse: parse, runs: runs,
		scan: scan, batch: batch, generation: generation, transcription: transcription,
		liveSpend: liveSpend,
		now:       time.Now,
	}
}

// Spend sources for the attention cell (tech-spec D3). The wire value tells the
// client which copy applies (執行中 vs 最近一次執行).
const (
	spendSourceLiveBatch = "live_batch"
	spendSourceLastRun   = "last_run"
)

// HomeSummary is the GET /api/v1/home-summary payload `[@contract-v1]`
// (tech-spec D1). snake_case on the wire; the web client camelCases at the
// fetchApi boundary (Rule 18). All human copy lives on the client.
type HomeSummary struct {
	Coverage       CoverageCell       `json:"coverage"`
	ProcessedToday ProcessedTodayCell `json:"processed_today"`
	Attention      AttentionCell      `json:"attention"`
	InFlight       InFlightCell       `json:"in_flight"`
}

// CoverageCell is 繁中字幕 N/M by 部 (movies + series). Fileless items count in
// Total only — Covered ≤ Total by construction.
type CoverageCell struct {
	Status  string `json:"status"`
	Covered int    `json:"covered"`
	Total   int    `json:"total"`
	Error   string `json:"error,omitempty"`
}

// ProcessedTodayCell is the distinct media completed since server-local
// start-of-day across parse jobs ∪ subtitle runs.
type ProcessedTodayCell struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Error  string `json:"error,omitempty"`
}

// AttentionCell is the exception readout: durable failure count, plus AI spend
// when a datum exists. The spend trio is absent (not zero) when nothing has
// been recorded yet — absent ≠ $0 (tech-spec D3).
type AttentionCell struct {
	Status      string   `json:"status"`
	FailedCount int      `json:"failed_count"`
	SpentUSD    *float64 `json:"spent_usd,omitempty"`
	BudgetUSD   *float64 `json:"budget_usd,omitempty"`
	SpendSource string   `json:"spend_source,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// InFlightCell mirrors the nav badge's number (len of /activity active jobs).
type InFlightCell struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Error  string `json:"error,omitempty"`
}

// GetSummary composes all four cells. It NEVER returns an error — each cell
// degrades independently (fail-soft).
func (s *HomeSummaryService) GetSummary(ctx context.Context) HomeSummary {
	return HomeSummary{
		Coverage:       s.coverageCell(ctx),
		ProcessedToday: s.processedTodayCell(ctx),
		Attention:      s.attentionCell(ctx),
		InFlight:       s.inFlightCell(),
	}
}

func (s *HomeSummaryService) coverageCell(ctx context.Context) CoverageCell {
	if s.movies == nil || s.series == nil {
		return CoverageCell{Status: sectionUnavailable, Error: "service unavailable"}
	}
	movieTotal, err := s.movies.Count(ctx)
	if err != nil {
		return CoverageCell{Status: sectionUnavailable, Error: err.Error()}
	}
	seriesTotal, err := s.series.Count(ctx)
	if err != nil {
		return CoverageCell{Status: sectionUnavailable, Error: err.Error()}
	}
	movieCovered, err := s.movies.CountZhHantSubtitle(ctx)
	if err != nil {
		return CoverageCell{Status: sectionUnavailable, Error: err.Error()}
	}
	seriesCovered, err := s.series.CountZhHantCovered(ctx)
	if err != nil {
		return CoverageCell{Status: sectionUnavailable, Error: err.Error()}
	}
	return CoverageCell{
		Status:  sectionOK,
		Covered: movieCovered + seriesCovered,
		Total:   movieTotal + seriesTotal,
	}
}

// processedTodayCell dedupes on the bare media id: parse jobs carry untyped
// media ids while runs carry (id, type), and ids are UUIDs — collision-free
// across tables — so the bare id is the exact dedupe grain.
func (s *HomeSummaryService) processedTodayCell(ctx context.Context) ProcessedTodayCell {
	if s.parse == nil || s.runs == nil {
		return ProcessedTodayCell{Status: sectionUnavailable, Error: "service unavailable"}
	}
	// Server-local calendar day (tech-spec D2): 「今天」 means the operator's
	// today, and both repositories normalize the comparison via datetime().
	now := s.now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	parseIDs, err := s.parse.CompletedMediaIDsSince(ctx, startOfDay)
	if err != nil {
		return ProcessedTodayCell{Status: sectionUnavailable, Error: err.Error()}
	}
	runRefs, err := s.runs.CompletedMediaRefsSince(ctx, startOfDay)
	if err != nil {
		return ProcessedTodayCell{Status: sectionUnavailable, Error: err.Error()}
	}

	seen := make(map[string]struct{}, len(parseIDs)+len(runRefs))
	for _, id := range parseIDs {
		seen[id] = struct{}{}
	}
	for _, ref := range runRefs {
		seen[ref.MediaID] = struct{}{}
	}
	return ProcessedTodayCell{Status: sectionOK, Count: len(seen)}
}

func (s *HomeSummaryService) attentionCell(ctx context.Context) AttentionCell {
	if s.parse == nil || s.runs == nil {
		return AttentionCell{Status: sectionUnavailable, Error: "service unavailable"}
	}
	parseFailed, err := s.parse.CountByStatus(ctx, models.ParseJobFailed)
	if err != nil {
		return AttentionCell{Status: sectionUnavailable, Error: err.Error()}
	}
	runsFailed, err := s.runs.CountByStatus(ctx, models.SubtitleRunFailed)
	if err != nil {
		return AttentionCell{Status: sectionUnavailable, Error: err.Error()}
	}
	cell := AttentionCell{Status: sectionOK, FailedCount: parseFailed + runsFailed}

	// Spend precedence (tech-spec D3): live batch → latest persisted run →
	// absent. A spend-resolution failure never degrades the cell — the failure
	// count is the cell's core datum and stays served.
	if s.liveSpend != nil {
		if prog := s.liveSpend.GetProgress(); prog != nil {
			spent, budget := prog.SpentUSD, prog.BudgetUSD
			cell.SpentUSD, cell.BudgetUSD, cell.SpendSource = &spent, &budget, spendSourceLiveBatch
			return cell
		}
	}
	if run, err := s.runs.LatestWithSpend(ctx); err == nil && run != nil && run.SpentUSD != nil {
		cell.SpentUSD, cell.BudgetUSD, cell.SpendSource = run.SpentUSD, run.BudgetUSD, spendSourceLastRun
	}
	return cell
}

// inFlightCell counts the same jobs /activity lists — an empty count is a valid
// OK state, and the in-memory sources cannot error.
func (s *HomeSummaryService) inFlightCell() InFlightCell {
	count := 0
	if s.scan != nil && s.scan.IsScanActive() {
		count++
	}
	for _, src := range []batchJobSource{s.batch, s.generation, s.transcription} {
		if src == nil {
			continue
		}
		if active, _, _, _, _ := src.ActivityProgress(); active {
			count++
		}
	}
	return InFlightCell{Status: sectionOK, Count: count}
}
