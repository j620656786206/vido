package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// Implements: disc-2026-11-burned-in-subs-b2 AC #1, #3, #4

// Sweep cadence and retry window (disc-2026-11-burned-in-subs-b2, ruling 1A).
const (
	// BurnedInBootDelay lets the scanner and the subtitle-track sweep settle
	// before the boot round — which, on an existing library, is the one-off
	// check of everything already there.
	BurnedInBootDelay = 2 * time.Minute
	// BurnedInSweepInterval: the subtitle-track sweep keeps writing new
	// episodes as '[]' between scans, so a round also runs on a clock.
	BurnedInSweepInterval = 6 * time.Hour
	// burnedInRetryAfter: an undecided season is asked again only after this.
	burnedInRetryAfter = 7 * 24 * time.Hour
)

// BurnedInSeasonFinder lists the seasons a round should check.
type BurnedInSeasonFinder interface {
	FindSeasonsNeedingBurnedInCheck(ctx context.Context, retryBefore time.Time) ([]repository.BurnedInSeasonRef, error)
}

// BurnedInSeasonCheck is the part of BurnedInSubtitleService a round drives.
type BurnedInSeasonCheck interface {
	Configured(ctx context.Context) bool
	CheckSeason(ctx context.Context, seriesID string, season int) (BurnedInCheckResult, error)
}

// BurnedInSpendTotals reads this month's ai_side_spend by kind.
type BurnedInSpendTotals interface {
	TotalsByKindBetween(ctx context.Context, from, to time.Time) (map[models.AISideSpendKind]repository.AISideSpendTotals, error)
}

// BurnedInSweepResult is what one round did.
type BurnedInSweepResult struct {
	Seasons   int // seasons found needing a check
	Checked   int // CheckSeason calls that returned without error
	Decided   int // of those, how many wrote a value
	Skipped   bool
	CapHit    bool
	Stopped   string // why the round ended early ("" = ran to the end)
	SpentUSD  float64
	MonthUSD  float64 // this month's burned-in spend when the round ended
	CapUSD    float64
	TookRound time.Duration
}

// BurnedInSweeper runs the burned-in check over the library by itself: once
// after boot, after every scan, and every few hours, one season at a time,
// never two rounds at once, and never past the monthly cap. CheckSeason stays
// the unit of work; this type only decides WHEN and WHETHER.
type BurnedInSweeper struct {
	finder BurnedInSeasonFinder
	check  BurnedInSeasonCheck
	spend  BurnedInSpendTotals
	capUSD decimal.Decimal
	logger *slog.Logger

	now       func() time.Time
	bootDelay time.Duration
	interval  time.Duration

	running atomic.Bool
	wg      sync.WaitGroup
}

// NewBurnedInSweeper builds the sweeper. capUSD <= 0 disables it (Run
// returns at once); main does not start one then.
func NewBurnedInSweeper(finder BurnedInSeasonFinder, check BurnedInSeasonCheck, spend BurnedInSpendTotals, capUSD float64, logger *slog.Logger) *BurnedInSweeper {
	if logger == nil {
		logger = slog.Default()
	}
	limit := decimal.Zero
	if capUSD > 0 {
		limit = decimal.NewFromFloat(capUSD)
	}
	return &BurnedInSweeper{
		finder: finder, check: check, spend: spend, capUSD: limit,
		logger: logger.With("component", "burned_in_sweep"),
		now:    time.Now, bootDelay: BurnedInBootDelay, interval: BurnedInSweepInterval,
	}
}

// SetSchedule overrides the boot delay and interval (tests).
func (s *BurnedInSweeper) SetSchedule(bootDelay, interval time.Duration) {
	s.bootDelay, s.interval = bootDelay, interval
}

// SetClock overrides the clock (tests): it decides the month and the retry
// window.
func (s *BurnedInSweeper) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

// Start runs a round after the boot delay, then one every interval, until ctx
// ends. Wait blocks until it has returned.
func (s *BurnedInSweeper) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.bootDelay):
		}
		s.runExclusive(ctx)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runExclusive(ctx)
			}
		}
	}()
}

// Trigger starts a round in the background — the scan-complete hook. A
// round already running means this one is skipped, not queued: the running
// round reads the season list fresh, and the clock round follows anyway.
func (s *BurnedInSweeper) Trigger(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runExclusive(ctx)
	}()
}

// Wait blocks until every round started by Start or Trigger has returned.
// main cancels their ctx first, then waits, then closes the database.
func (s *BurnedInSweeper) Wait() {
	s.wg.Wait()
}

// runExclusive runs one round unless one is already running.
func (s *BurnedInSweeper) runExclusive(ctx context.Context) BurnedInSweepResult {
	if !s.running.CompareAndSwap(false, true) {
		s.logger.Debug("burned-in sweep: a round is already running — skipped")
		return BurnedInSweepResult{Skipped: true}
	}
	defer s.running.Store(false)
	return s.Run(ctx)
}

// Run does one round, in season order, until the list ends, the monthly cap
// is reached, or something stops it (no key, a failed model call, ctx).
// Never returns an error: a background round has nobody to return it to.
func (s *BurnedInSweeper) Run(ctx context.Context) BurnedInSweepResult {
	res := BurnedInSweepResult{CapUSD: s.capUSD.InexactFloat64()}
	if !s.capUSD.IsPositive() {
		res.Stopped = "disabled"
		return res
	}
	if !s.check.Configured(ctx) {
		s.logger.Info("burned-in sweep: no Claude API key — round skipped")
		res.Stopped = "no_key"
		return res
	}
	started := s.now()
	if s.capReached(ctx, &res) {
		return res
	}

	seasons, err := s.finder.FindSeasonsNeedingBurnedInCheck(ctx, s.now().Add(-burnedInRetryAfter))
	if err != nil {
		s.logger.Warn("burned-in sweep: listing seasons failed", "error", err)
		res.Stopped = "list_failed"
		return res
	}
	res.Seasons = len(seasons)
	if len(seasons) == 0 {
		return res
	}
	s.logger.Info("burned-in sweep: round started", "seasons", len(seasons), "month_usd", res.MonthUSD, "cap_usd", res.CapUSD)

	spent := decimal.Zero
	for i, ref := range seasons {
		if ctx.Err() != nil {
			res.Stopped = "cancelled"
			break
		}
		// Re-read before every season, not once per round: the manual
		// trigger spends from the same month while a round runs.
		if i > 0 && s.capReached(ctx, &res) {
			break
		}
		out, err := s.check.CheckSeason(ctx, ref.SeriesID, ref.Season)
		spent = spent.Add(decimal.NewFromFloat(out.CostUSD))
		log := s.logger.With("series_id", ref.SeriesID, "series", ref.SeriesTitle, "season", ref.Season)
		if err != nil {
			switch {
			case errors.Is(err, ErrBurnedInNotConfigured):
				log.Info("burned-in sweep: Claude key went away — round ended")
				res.Stopped = "no_key"
			case errors.Is(err, ErrBurnedInModelCall):
				// A rejected key or an outage fails every season the same
				// way; the next round tries again.
				log.Warn("burned-in sweep: model call failed — round ended", "error", err)
				res.Stopped = "model_call_failed"
			default:
				log.Warn("burned-in sweep: season check failed — round ended", "error", err)
				res.Stopped = "check_failed"
			}
			break
		}
		res.Checked++
		if out.Result != nil {
			res.Decided++
			log.Debug("burned-in sweep: season decided", "result", *out.Result, "episodes_updated", out.EpisodesUpdated)
		} else {
			log.Debug("burned-in sweep: season undecided", "reason", out.Reason)
		}
	}
	res.SpentUSD = spent.InexactFloat64()
	res.TookRound = s.now().Sub(started)
	s.logger.Info("burned-in sweep: round finished",
		"seasons", res.Seasons, "checked", res.Checked, "decided", res.Decided,
		"spent_usd", res.SpentUSD, "cap_hit", res.CapHit, "stopped", res.Stopped,
		"took", res.TookRound.Round(time.Second).String())
	return res
}

// capReached reads this month's burned-in spend into res and reports whether
// the cap is reached; a failed read stops the round (an unknown spend must
// not be treated as zero).
func (s *BurnedInSweeper) capReached(ctx context.Context, res *BurnedInSweepResult) bool {
	from, to := CalendarMonthWindow(s.now())
	byKind, err := s.spend.TotalsByKindBetween(ctx, from, to)
	if err != nil {
		s.logger.Warn("burned-in sweep: reading this month's spend failed — round ended", "error", err)
		res.Stopped = "spend_unreadable"
		return true
	}
	month := byKind[models.AISideSpendKindBurnedInCheck].Spent
	res.MonthUSD = month.InexactFloat64()
	if month.LessThan(s.capUSD) {
		return false
	}
	s.logger.Warn("burned-in sweep: monthly cap reached — automatic checks resume next month",
		"month_usd", res.MonthUSD, "cap_usd", res.CapUSD, "env", "VIDO_BURNED_IN_MONTHLY_CAP_USD")
	res.CapHit = true
	res.Stopped = "cap_reached"
	return true
}
