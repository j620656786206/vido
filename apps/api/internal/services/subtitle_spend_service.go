package services

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/vido/api/internal/models"
)

// SubtitleSpendSummary is the GET /api/v1/subtitles/spend body (sub-7-6b).
//
// [@contract-v1] — consumed by sub-7-6c (the activity hub's「本月 AI 花費」
// block, the F8 batch receipt, the home readout). Money is USD as float64,
// rounded to cents. The honesty rules (PRODUCT.md principle 1) are encoded in
// the shape, not left to the reader:
//
//   - a figure that was never recorded is a COUNT of such runs, never $0 —
//     `unpriced_runs` (completed runs whose spend was not stamped) and
//     `unrouted_runs` / `unrouted_usd` (runs older than migration 041, whose
//     lane is unknown, so their money is reported apart from translated/asr);
//   - the two "saved" numbers are ESTIMATES and are named so;
//     `skipped_saved_runtime_assumed` is true when at least one skipped item's
//     runtime fell back to the 45-minute assumption (the ONE condition the UI
//     renders as ≈ — Sally 2026-09-05);
//   - `cache_saved_usd_estimate` is null (not 0) when no run in the period
//     measured its cache split (`cache_measured_runs` = 0).
type SubtitleSpendSummary struct {
	Period string    `json:"period"`
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`

	TranslatedRuns int     `json:"translated_runs"`
	TranslatedUSD  float64 `json:"translated_usd"`
	ASRRuns        int     `json:"asr_runs"`
	ASRUSD         float64 `json:"asr_usd"`
	UnpricedRuns   int     `json:"unpriced_runs"`
	UnroutedRuns   int     `json:"unrouted_runs"`
	UnroutedUSD    float64 `json:"unrouted_usd"`

	SkippedDeliverCount        int     `json:"skipped_deliver_count"`
	SkippedSavedUSDEstimate    float64 `json:"skipped_saved_usd_estimate"`
	SkippedSavedRuntimeAssumed bool    `json:"skipped_saved_runtime_assumed"`

	CacheHitCues          int      `json:"cache_hit_cues"`
	CacheMeasuredRuns     int      `json:"cache_measured_runs"`
	CacheSavedUSDEstimate *float64 `json:"cache_saved_usd_estimate"`

	ByModel []SpendByModel `json:"by_model"`
	// ByBatch is present only when the request named a batch_id.
	ByBatch *SubtitleBatchReceipt `json:"by_batch,omitempty"`
}

// SpendByModel is one row of the per-model table: the paid lanes (translate
// and asr) grouped by the model that billed them.
type SpendByModel struct {
	ModelID string  `json:"model_id"`
	Runs    int     `json:"runs"`
	USD     float64 `json:"usd"`
}

// SubtitleBatchReceipt is「本次批次」: summed from the ledger rows stamped with
// the batch id, so it survives a reload of the F8 dialog. Every status counts
// in `runs`; only `completed` runs count in `completed_runs` and cue totals.
// `cache_hit_cues` is null when no run of the batch measured its split.
type SubtitleBatchReceipt struct {
	BatchID           string   `json:"batch_id"`
	Runs              int      `json:"runs"`
	CompletedRuns     int      `json:"completed_runs"`
	USD               float64  `json:"usd"`
	CueCount          int      `json:"cue_count"`
	CacheHitCues      *int     `json:"cache_hit_cues"`
	CacheMeasuredRuns int      `json:"cache_measured_runs"`
	ModelID           string   `json:"model_id"`
	ModelIDs          []string `json:"model_ids,omitempty"`
}

// spendRunSource is the ledger read this service needs; the concrete
// SubtitleRunRepository satisfies it.
type spendRunSource interface {
	CompletedRunsBetween(ctx context.Context, from, to time.Time) ([]models.SubtitleRun, error)
	RunsByBatchID(ctx context.Context, batchID string) ([]models.SubtitleRun, error)
}

// SubtitleSpendService aggregates the subtitle_runs ledger (sub-7-6a) into
// the monthly figures and the per-batch receipt. Pure arithmetic over rows —
// it never calls a provider.
type SubtitleSpendService struct {
	runs     spendRunSource
	movies   SubtitleStateReader        // runtime ladder for skipped movies (nil-safe)
	episodes EpisodeSubtitleStateReader // runtime ladder for skipped episodes (nil-safe)
	logger   *slog.Logger
	// now is injectable: 「本月」 is a calendar boundary in the server's zone.
	now func() time.Time
}

// NewSubtitleSpendService wires the ledger and the two runtime readers. Either
// reader may be nil — a skipped item then falls to the assumed runtime, and
// the summary says so.
func NewSubtitleSpendService(runs spendRunSource, movies SubtitleStateReader, episodes EpisodeSubtitleStateReader, logger *slog.Logger) *SubtitleSpendService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SubtitleSpendService{runs: runs, movies: movies, episodes: episodes, logger: logger, now: time.Now}
}

// SetClock overrides the clock (tests; Rule 23's backend twin).
func (s *SubtitleSpendService) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

// monthWindow is the current calendar month in the server's zone, as UTC
// instants [from, to) — the same「today means the operator's today」 choice
// HomeSummaryService.processedTodayCell makes.
func (s *SubtitleSpendService) monthWindow() (from, to time.Time) {
	now := s.now()
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to = from.AddDate(0, 1, 0)
	return from.UTC(), to.UTC()
}

// MonthSummary aggregates this month's completed runs. batchID, when
// non-empty, adds the batch receipt (any status, any month) to the body.
func (s *SubtitleSpendService) MonthSummary(ctx context.Context, batchID string) (*SubtitleSpendSummary, error) {
	from, to := s.monthWindow()
	runs, err := s.runs.CompletedRunsBetween(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := s.summarize(ctx, runs)
	out.Period = "month"
	out.From, out.To = from, to

	if batchID != "" {
		receipt, err := s.BatchReceipt(ctx, batchID)
		if err != nil {
			return nil, err
		}
		out.ByBatch = receipt
	}
	return out, nil
}

// summarize is the pure aggregation over completed runs.
func (s *SubtitleSpendService) summarize(ctx context.Context, runs []models.SubtitleRun) *SubtitleSpendSummary {
	out := &SubtitleSpendSummary{ByModel: []SpendByModel{}}
	var translated, asr, unrouted, skippedSaved, cacheSaved decimal.Decimal
	byModel := map[string]*struct {
		runs int
		usd  decimal.Decimal
	}{}

	for i := range runs {
		run := &runs[i]
		spent, priced := runSpend(run)

		switch run.Route {
		case models.SubtitleRunRouteTranslate, models.SubtitleRunRouteASR:
			if run.Route == models.SubtitleRunRouteTranslate {
				out.TranslatedRuns++
				translated = translated.Add(spent)
			} else {
				out.ASRRuns++
				asr = asr.Add(spent)
			}
			if !priced {
				out.UnpricedRuns++
			}
			m := byModel[run.ModelID]
			if m == nil {
				m = &struct {
					runs int
					usd  decimal.Decimal
				}{}
				byModel[run.ModelID] = m
			}
			m.runs++
			m.usd = m.usd.Add(spent)

			// Cache saved: only runs that MEASURED their split, and only when
			// the model actually translated something to derive a per-cue price.
			if run.CacheHitCues != nil {
				out.CacheMeasuredRuns++
				out.CacheHitCues += *run.CacheHitCues
				if priced && run.CueCount > *run.CacheHitCues && *run.CacheHitCues > 0 {
					billedCues := decimal.NewFromInt(int64(run.CueCount - *run.CacheHitCues))
					perCue := spent.Div(billedCues)
					cacheSaved = cacheSaved.Add(perCue.Mul(decimal.NewFromInt(int64(*run.CacheHitCues))))
				}
			}

		case models.SubtitleRunRouteDeliverDirect, models.SubtitleRunRouteConvertThenDeliver:
			// The free lanes: an existing Chinese track was delivered instead of
			// paying for a translation. "Saved" = what the translate lane would
			// have charged for this runtime on this model.
			out.SkippedDeliverCount++
			minutes, known := s.runtimeMinutes(ctx, run)
			if !known {
				out.SkippedSavedRuntimeAssumed = true
			}
			skippedSaved = skippedSaved.Add(estimateUSD(RouteExtract, minutes, decimal.Zero, spendModelFor(run)))

		case "":
			// Pre-041 rows: their lane is unknown, so their money is reported
			// on its own line rather than silently folded into "translated".
			out.UnroutedRuns++
			unrouted = unrouted.Add(spent)
			if !priced {
				out.UnpricedRuns++
			}

		default:
			// skip / no_text_source: nothing was bought and nothing was saved.
		}
	}

	out.TranslatedUSD = roundUSD(translated).InexactFloat64()
	out.ASRUSD = roundUSD(asr).InexactFloat64()
	out.UnroutedUSD = roundUSD(unrouted).InexactFloat64()
	out.SkippedSavedUSDEstimate = roundUSD(skippedSaved).InexactFloat64()
	if out.CacheMeasuredRuns > 0 {
		v := roundUSD(cacheSaved).InexactFloat64()
		out.CacheSavedUSDEstimate = &v
	}

	for id, m := range byModel {
		out.ByModel = append(out.ByModel, SpendByModel{ModelID: id, Runs: m.runs, USD: roundUSD(m.usd).InexactFloat64()})
	}
	sort.Slice(out.ByModel, func(i, j int) bool {
		if out.ByModel[i].USD != out.ByModel[j].USD {
			return out.ByModel[i].USD > out.ByModel[j].USD
		}
		return out.ByModel[i].ModelID < out.ByModel[j].ModelID
	})
	return out
}

// BatchReceipt sums one consent batch's ledger rows.
func (s *SubtitleSpendService) BatchReceipt(ctx context.Context, batchID string) (*SubtitleBatchReceipt, error) {
	runs, err := s.runs.RunsByBatchID(ctx, batchID)
	if err != nil {
		return nil, err
	}
	receipt := &SubtitleBatchReceipt{BatchID: batchID}
	var usd decimal.Decimal
	var hits int
	modelSet := map[string]struct{}{}
	for i := range runs {
		run := &runs[i]
		receipt.Runs++
		spent, _ := runSpend(run)
		usd = usd.Add(spent) // failed / paused runs cost money too
		if run.ModelID != "" {
			modelSet[run.ModelID] = struct{}{}
		}
		if run.Status != models.SubtitleRunCompleted {
			continue
		}
		receipt.CompletedRuns++
		receipt.CueCount += run.CueCount
		if run.CacheHitCues != nil {
			receipt.CacheMeasuredRuns++
			hits += *run.CacheHitCues
		}
	}
	receipt.USD = roundUSD(usd).InexactFloat64()
	if receipt.CacheMeasuredRuns > 0 {
		receipt.CacheHitCues = &hits
	}
	ids := make([]string, 0, len(modelSet))
	for id := range modelSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 1 {
		receipt.ModelID = ids[0]
	} else if len(ids) > 1 {
		receipt.ModelIDs = ids // a batch that changed model mid-way names them all
	}
	return receipt, nil
}

// runSpend returns the run's recorded spend and whether it was recorded at all
// (absent is not $0).
func runSpend(run *models.SubtitleRun) (decimal.Decimal, bool) {
	if run.SpentUSD == nil {
		return decimal.Zero, false
	}
	return decimal.NewFromFloat(*run.SpentUSD), true
}

// spendModelFor prices a skipped item on the model that run WOULD have used:
// the row's model, or the calibration anchor when a pre-sub-6-5 row left it
// blank (generous, never a downward surprise — translationRatePerMinute).
func spendModelFor(run *models.SubtitleRun) string {
	if run.ModelID != "" {
		return run.ModelID
	}
	return translationCalibrationModel
}

// runtimeMinutes climbs the same ladder the consent estimate uses
// (candidateRow.runtimeMinutes): measured container duration → TMDb runtime
// → the 45-minute assumption. known=false ONLY on the assumption.
func (s *SubtitleSpendService) runtimeMinutes(ctx context.Context, run *models.SubtitleRun) (float64, bool) {
	switch run.MediaType {
	case models.SubtitleRunMediaMovie:
		if s.movies == nil {
			break
		}
		movie, err := s.movies.FindByID(ctx, run.MediaID)
		if err != nil || movie == nil {
			break
		}
		if movie.DurationSeconds.Valid && movie.DurationSeconds.Int64 > 0 {
			return float64(movie.DurationSeconds.Int64) / 60.0, true
		}
		if movie.Runtime.Valid && movie.Runtime.Int64 > 0 {
			return float64(movie.Runtime.Int64), true
		}
	case models.SubtitleRunMediaEpisode:
		if s.episodes == nil {
			break
		}
		episode, err := s.episodes.FindByID(ctx, run.MediaID)
		if err != nil || episode == nil {
			break
		}
		if episode.DurationSeconds.Valid && episode.DurationSeconds.Int64 > 0 {
			return float64(episode.DurationSeconds.Int64) / 60.0, true
		}
	}
	return unknownRuntimeMinutes, false
}
