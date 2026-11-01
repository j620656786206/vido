package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// Run-history categories (disc-2026-11-spend-history-c's table). The first
// four are the chart's colours and the keys of RunHistoryByCategory;
// no_subtitle is the lanes that produced or bought nothing, and is listed but
// never charted.
const (
	RunCategoryChineseTrack     = "chinese_track"
	RunCategoryTranslateEnglish = "translate_english"
	RunCategoryASR              = "asr"
	RunCategoryLegacy           = "legacy"
	RunCategoryNoSubtitle       = "no_subtitle"
)

// Page sizes for GET /api/v1/subtitles/runs.
const (
	RunHistoryDefaultPageSize = 50
	RunHistoryMaxPageSize     = 200
)

// ErrInvalidRunHistoryQuery wraps every malformed query value (month,
// category, page, page_size); the handler answers it with 400
// VALIDATION_INVALID_FORMAT.
var ErrInvalidRunHistoryQuery = errors.New("invalid run history query")

// runHistoryMonthLayout is the `month` query and response format.
const runHistoryMonthLayout = "2006-01"

// SubtitleRunHistory is the GET /api/v1/subtitles/runs body
// (disc-2026-11-spend-history-b-api).
//
// [@contract-v1] — field names approved by Alexyu 2026-11-01; consumed by
// disc-2026-11-spend-history-d-ui (the AI 花費明細 page). Honesty rules
// (PRODUCT.md principle 1) are in the shape:
//
//   - a number that was never recorded is null, never 0 — an item's
//     spent_usd / estimate_usd / route / source / asr_* / batch_id /
//     triggered_by; summary.unpriced_runs counts the non-skipped runs whose
//     spend is null (a skipped run bought nothing);
//   - comparable_* covers COMPLETED runs on a PAID route (翻譯英文字幕,
//     語音辨識＋翻譯, 舊紀錄) that recorded both numbers — a free route
//     (片內中文字幕, 線上字幕) is quoted before the route is known, so its $0
//     says nothing about the quote (disc-2026-11-free-route-records-paid-estimate);
//     comparable_spent_usd / comparable_estimate_usd are null when there are
//     none (comparable_runs = 0);
//   - a day with no runs is a real 0 in `series`.
//
// `summary` and `series` always cover the whole month; `category` filters
// `items` (and `total`) only.
type SubtitleRunHistory struct {
	Month    string            `json:"month"`
	Summary  RunHistorySummary `json:"summary"`
	Series   []RunHistoryDay   `json:"series"`
	Items    []RunHistoryItem  `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int               `json:"total"`
}

// RunHistorySummary is the month's headline. SpentUSD sums every recorded
// spend (failed and stopped runs cost money too); the comparable_* trio is
// only the completed paid-route runs that recorded both their spend and their
// estimate, so the page can say「實際是預估的 N%」 without guessing the rest.
type RunHistorySummary struct {
	SpentUSD     float64 `json:"spent_usd"`
	Runs         int     `json:"runs"`
	UnpricedRuns int     `json:"unpriced_runs"`
	// EstimateUSD sums every estimate the month recorded (failed and free
	// runs included — it is what the user was quoted), null when none was;
	// EstimatedRuns counts them. The page shows it next to SpentUSD
	// (⚖️ Alexyu 2026-11-01: 「總預估多少錢、實際花費多少錢」, not a ratio).
	EstimateUSD           *float64 `json:"estimate_usd"`
	EstimatedRuns         int      `json:"estimated_runs"`
	ComparableRuns        int      `json:"comparable_runs"`
	ComparableSpentUSD    *float64 `json:"comparable_spent_usd"`
	ComparableEstimateUSD *float64 `json:"comparable_estimate_usd"`
}

// RunHistoryDay is one bar of the daily chart. Runs counts the four charted
// categories (a run with a null spend counts here but adds no money);
// no_subtitle runs are not in it. SpentUSD is exactly the sum of ByCategory.
type RunHistoryDay struct {
	Date       string               `json:"date"`
	SpentUSD   float64              `json:"spent_usd"`
	Runs       int                  `json:"runs"`
	ByCategory RunHistoryByCategory `json:"by_category"`
}

// RunHistoryByCategory is one day's money per charted category.
type RunHistoryByCategory struct {
	ChineseTrack     float64 `json:"chinese_track"`
	TranslateEnglish float64 `json:"translate_english"`
	ASR              float64 `json:"asr"`
	Legacy           float64 `json:"legacy"`
}

// RunHistoryItem is one run in the list.
type RunHistoryItem struct {
	ID          string     `json:"id"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Status      string     `json:"status"`
	// FailureReason is models.SubtitleRunFailureReason's code for a failed
	// run — the activity record's classifier — and "" otherwise.
	FailureReason string          `json:"failure_reason"`
	Media         RunHistoryMedia `json:"media"`
	Route         *string         `json:"route"`
	Source        *string         `json:"source"`
	Category      string          `json:"category"`
	SpentUSD      *float64        `json:"spent_usd"`
	EstimateUSD   *float64        `json:"estimate_usd"`
	// TranslationModel is the run's model_id on the lanes that translate
	// (translate, asr) and on a legacy run that cost money — before routes were
	// recorded only a translation could cost anything; null elsewhere, where
	// model_id is only a version tag.
	TranslationModel *string `json:"translation_model"`
	ASRProvider      *string `json:"asr_provider"`
	ASRModel         *string `json:"asr_model"`
	BatchID          *string `json:"batch_id"`
	TriggeredBy      *string `json:"triggered_by"`
}

// RunHistoryMedia names what the run ran on. Title is the movie title or the
// series title (for an episode); null when the item has left the library.
// Season / episode are set only for an episode that still has a title — the
// activity record's rule.
type RunHistoryMedia struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Title   *string `json:"title"`
	Season  *int    `json:"season"`
	Episode *int    `json:"episode"`
}

// RunHistoryQuery is the request. Month "" = the current month; Category ""
// = every category; Page / PageSize 0 = the defaults.
type RunHistoryQuery struct {
	Month    string
	Category string
	Page     int
	PageSize int
}

// runHistorySource is the ledger read this service needs; the concrete
// SubtitleRunRepository satisfies it.
type runHistorySource interface {
	FinishedRunsStartedBetween(ctx context.Context, from, to time.Time) ([]repository.RecentSubtitleRun, error)
}

// SubtitleRunHistoryService lists one month of subtitle runs with what each
// cost, what it was estimated at, and which models ran (the spend history
// page). Pure arithmetic over ledger rows.
type SubtitleRunHistoryService struct {
	runs   runHistorySource
	logger *slog.Logger
	// now is injectable: a month is a calendar month in the server's zone,
	// the same boundary SubtitleSpendService.monthWindow uses.
	now func() time.Time
}

// NewSubtitleRunHistoryService wires the ledger.
func NewSubtitleRunHistoryService(runs runHistorySource, logger *slog.Logger) *SubtitleRunHistoryService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SubtitleRunHistoryService{runs: runs, logger: logger, now: time.Now}
}

// SetClock overrides the clock (tests).
func (s *SubtitleRunHistoryService) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

// History answers one GET /subtitles/runs request. A malformed query value
// returns an error wrapping ErrInvalidRunHistoryQuery.
func (s *SubtitleRunHistoryService) History(ctx context.Context, q RunHistoryQuery) (*SubtitleRunHistory, error) {
	now := s.now()
	loc := now.Location()

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	if q.Month != "" {
		parsed, err := time.ParseInLocation(runHistoryMonthLayout, q.Month, loc)
		if err != nil || parsed.Format(runHistoryMonthLayout) != q.Month {
			return nil, fmt.Errorf("%w: month must be YYYY-MM", ErrInvalidRunHistoryQuery)
		}
		monthStart = parsed
	}
	if q.Category != "" && !isRunHistoryCategory(q.Category) {
		return nil, fmt.Errorf("%w: category must be one of chinese_track, translate_english, asr, legacy, no_subtitle", ErrInvalidRunHistoryQuery)
	}
	page, pageSize := q.Page, q.PageSize
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = RunHistoryDefaultPageSize
	}
	if page < 1 {
		return nil, fmt.Errorf("%w: page must be a positive integer", ErrInvalidRunHistoryQuery)
	}
	if pageSize < 1 || pageSize > RunHistoryMaxPageSize {
		return nil, fmt.Errorf("%w: page_size must be between 1 and %d", ErrInvalidRunHistoryQuery, RunHistoryMaxPageSize)
	}

	monthEnd := monthStart.AddDate(0, 1, 0)
	rows, err := s.runs.FinishedRunsStartedBetween(ctx, monthStart.UTC(), monthEnd.UTC())
	if err != nil {
		return nil, err
	}

	out := &SubtitleRunHistory{
		Month:    monthStart.Format(runHistoryMonthLayout),
		Summary:  summarizeRunHistory(rows),
		Series:   runHistorySeries(rows, monthStart, now),
		Items:    []RunHistoryItem{},
		Page:     page,
		PageSize: pageSize,
	}

	filtered := make([]repository.RecentSubtitleRun, 0, len(rows))
	for _, r := range rows {
		if q.Category == "" || RunCategory(&r.Run) == q.Category {
			filtered = append(filtered, r)
		}
	}
	out.Total = len(filtered)
	if start := (page - 1) * pageSize; start < len(filtered) {
		end := min(start+pageSize, len(filtered))
		for _, r := range filtered[start:end] {
			out.Items = append(out.Items, runHistoryItem(r))
		}
	}
	return out, nil
}

// RunCategory maps a run onto disc-2026-11-spend-history-c's table by its
// route: the two free Chinese-track lanes, translate, asr, an unrecorded
// route (legacy), and everything else — skip, no_text_source, online — as
// no_subtitle.
func RunCategory(run *models.SubtitleRun) string {
	switch run.Route {
	case models.SubtitleRunRouteDeliverDirect, models.SubtitleRunRouteConvertThenDeliver:
		return RunCategoryChineseTrack
	case models.SubtitleRunRouteTranslate:
		return RunCategoryTranslateEnglish
	case models.SubtitleRunRouteASR:
		return RunCategoryASR
	case "":
		return RunCategoryLegacy
	default:
		return RunCategoryNoSubtitle
	}
}

// isPaidRunCategory reports whether a run's category is one the AI bills —
// the only ones whose estimate the accuracy ratio can judge.
func isPaidRunCategory(c string) bool {
	switch c {
	case RunCategoryTranslateEnglish, RunCategoryASR, RunCategoryLegacy:
		return true
	}
	return false
}

func isRunHistoryCategory(c string) bool {
	switch c {
	case RunCategoryChineseTrack, RunCategoryTranslateEnglish, RunCategoryASR, RunCategoryLegacy, RunCategoryNoSubtitle:
		return true
	}
	return false
}

func summarizeRunHistory(rows []repository.RecentSubtitleRun) RunHistorySummary {
	var sum RunHistorySummary
	var spent, estimated, compSpent, compEstimate decimal.Decimal
	for i := range rows {
		run := &rows[i].Run
		sum.Runs++
		if run.EstimateUSD != nil {
			sum.EstimatedRuns++
			estimated = estimated.Add(decimal.NewFromFloat(*run.EstimateUSD))
		}
		if run.SpentUSD == nil {
			// A skipped run bought nothing, so a missing figure there is not
			// a missing price.
			if run.Status != models.SubtitleRunSkipped {
				sum.UnpricedRuns++
			}
			continue
		}
		s := decimal.NewFromFloat(*run.SpentUSD)
		spent = spent.Add(s)
		// Only a completed run did the whole job the estimate priced; a
		// failed or stopped run spent a fraction of it and would drag the
		// 「實際是預估的 N%」 ratio down. A free route is left out too: the
		// quote is taken when the run starts, before the router has found the
		// official Chinese track, so 《基地》's $0 against a $0.32 quote dragged
		// the ratio from 105% to 79% without saying anything about the quote.
		if run.EstimateUSD != nil && run.Status == models.SubtitleRunCompleted && isPaidRunCategory(RunCategory(run)) {
			sum.ComparableRuns++
			compSpent = compSpent.Add(s)
			compEstimate = compEstimate.Add(decimal.NewFromFloat(*run.EstimateUSD))
		}
	}
	sum.SpentUSD = roundUSD(spent).InexactFloat64()
	if sum.EstimatedRuns > 0 {
		e := roundUSD(estimated).InexactFloat64()
		sum.EstimateUSD = &e
	}
	if sum.ComparableRuns > 0 {
		cs, ce := roundUSD(compSpent).InexactFloat64(), roundUSD(compEstimate).InexactFloat64()
		sum.ComparableSpentUSD, sum.ComparableEstimateUSD = &cs, &ce
	}
	return sum
}

// runHistorySeries is one entry per calendar day of the month (server zone),
// through today for the current month and none for a future month. The day
// is its exact spend rounded to cents, and the four categories are split from
// it (allocateCents), so they always add up to the day exactly.
func runHistorySeries(rows []repository.RecentSubtitleRun, monthStart, now time.Time) []RunHistoryDay {
	loc := monthStart.Location()
	type dayAcc struct {
		runs                         int
		chinese, translate, asr, leg decimal.Decimal
	}
	acc := map[string]*dayAcc{}
	for i := range rows {
		run := &rows[i].Run
		cat := RunCategory(run)
		if cat == RunCategoryNoSubtitle {
			continue
		}
		key := run.StartedAt.In(loc).Format(time.DateOnly)
		d := acc[key]
		if d == nil {
			d = &dayAcc{}
			acc[key] = d
		}
		d.runs++
		spent, _ := runSpend(run)
		switch cat {
		case RunCategoryChineseTrack:
			d.chinese = d.chinese.Add(spent)
		case RunCategoryTranslateEnglish:
			d.translate = d.translate.Add(spent)
		case RunCategoryASR:
			d.asr = d.asr.Add(spent)
		case RunCategoryLegacy:
			d.leg = d.leg.Add(spent)
		}
	}

	last := monthStart.AddDate(0, 1, -1)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if today.Before(last) {
		last = today
	}
	series := []RunHistoryDay{}
	for day := monthStart; !day.After(last); day = day.AddDate(0, 0, 1) {
		key := day.Format(time.DateOnly)
		entry := RunHistoryDay{Date: key}
		if d := acc[key]; d != nil {
			// The day is rounded ONCE from its exact sum, then split back
			// into categories so they still add up to it to the cent. Rounding
			// each category first made the chart's day ($2.43) disagree with
			// the summary ($2.44) on a one-day month
			// (disc-2026-11-spend-chart-current-month-one-bar).
			total := roundUSD(d.chinese.Add(d.translate).Add(d.asr).Add(d.leg))
			parts := allocateCents(total, d.chinese, d.translate, d.asr, d.leg)
			c, t, a, l := parts[0], parts[1], parts[2], parts[3]
			entry.Runs = d.runs
			entry.SpentUSD = total.InexactFloat64()
			entry.ByCategory = RunHistoryByCategory{
				ChineseTrack:     c.InexactFloat64(),
				TranslateEnglish: t.InexactFloat64(),
				ASR:              a.InexactFloat64(),
				Legacy:           l.InexactFloat64(),
			}
		}
		series = append(series, entry)
	}
	return series
}

// allocateCents splits a cent-rounded total across exact, non-negative parts
// by the largest-remainder method: each part gets its whole cents, and the
// cents left over go to the parts with the biggest fractions (a tie goes to
// the bigger part). The result always sums to total exactly.
func allocateCents(total decimal.Decimal, parts ...decimal.Decimal) []decimal.Decimal {
	hundred := decimal.NewFromInt(100)
	out := make([]decimal.Decimal, len(parts))
	frac := make([]decimal.Decimal, len(parts))
	given := decimal.Zero
	for i, p := range parts {
		cents := p.Mul(hundred)
		whole := cents.Floor()
		out[i] = whole.Div(hundred)
		frac[i] = cents.Sub(whole)
		given = given.Add(out[i])
	}
	left := total.Sub(given).Mul(hundred).Round(0).IntPart()
	for ; left > 0; left-- {
		best := -1
		for i := range parts {
			if best == -1 || frac[i].GreaterThan(frac[best]) ||
				(frac[i].Equal(frac[best]) && parts[i].GreaterThan(parts[best])) {
				best = i
			}
		}
		out[best] = out[best].Add(decimal.NewFromInt(1).Div(hundred))
		frac[best] = decimal.NewFromInt(-1) // one extra cent per part at most
	}
	return out
}

func runHistoryItem(r repository.RecentSubtitleRun) RunHistoryItem {
	run := r.Run
	item := RunHistoryItem{
		ID:          run.ID,
		StartedAt:   run.StartedAt.UTC(),
		Status:      string(run.Status),
		Media:       RunHistoryMedia{ID: run.MediaID, Type: run.MediaType},
		Route:       optString(run.Route),
		Source:      optString(run.Source),
		Category:    RunCategory(&run),
		ASRProvider: optString(run.ASRProvider),
		ASRModel:    optString(run.ASRModel),
		BatchID:     optString(run.BatchID),
		TriggeredBy: optString(run.TriggeredBy),
	}
	if run.CompletedAt != nil {
		c := run.CompletedAt.UTC()
		item.CompletedAt = &c
	}
	if run.Status == models.SubtitleRunFailed {
		item.FailureReason = models.SubtitleRunFailureReason(run.ErrorMessage)
	}
	if r.MediaTitle != "" {
		title := r.MediaTitle
		item.Media.Title = &title
		if run.MediaType == models.SubtitleRunMediaEpisode && r.SeasonNumber != nil && r.EpisodeNumber != nil {
			sn, en := *r.SeasonNumber, *r.EpisodeNumber
			item.Media.Season, item.Media.Episode = &sn, &en
		}
	}
	if run.SpentUSD != nil {
		v := *run.SpentUSD
		item.SpentUSD = &v
	}
	if run.EstimateUSD != nil {
		v := *run.EstimateUSD
		item.EstimateUSD = &v
	}
	paidLegacy := run.Route == "" && run.SpentUSD != nil && *run.SpentUSD > 0
	if run.Route == models.SubtitleRunRouteTranslate || run.Route == models.SubtitleRunRouteASR || paidLegacy {
		item.TranslationModel = optString(run.ModelID)
	}
	return item
}

// optString is nil for an unrecorded ("") ledger string.
func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
