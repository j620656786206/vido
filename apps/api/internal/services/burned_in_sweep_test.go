package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// --- fakes ------------------------------------------------------------------

type fakeSeasonFinder struct {
	mu          sync.Mutex
	seasons     []repository.BurnedInSeasonRef
	err         error
	calls       int
	retryBefore time.Time
	block       chan struct{} // when set, each call waits on it
	entered     chan struct{} // when set, each call signals it
}

func (f *fakeSeasonFinder) FindSeasonsNeedingBurnedInCheck(_ context.Context, retryBefore time.Time) ([]repository.BurnedInSeasonRef, error) {
	f.mu.Lock()
	f.calls++
	f.retryBefore = retryBefore
	block, entered := f.block, f.entered
	f.mu.Unlock()
	if entered != nil {
		entered <- struct{}{}
	}
	if block != nil {
		<-block
	}
	return f.seasons, f.err
}

func (f *fakeSeasonFinder) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fakeSeasonCheck books each check's cost into the shared month ledger, the
// way the real service's ai_side_spend rows do.
type fakeSeasonCheck struct {
	configured bool
	cost       float64
	ledger     *fakeMonthSpend
	errAt      map[int]error
	results    map[int]*string
	checked    []string
}

func (c *fakeSeasonCheck) Configured(context.Context) bool { return c.configured }

func (c *fakeSeasonCheck) CheckSeason(_ context.Context, seriesID string, season int) (BurnedInCheckResult, error) {
	i := len(c.checked)
	c.checked = append(c.checked, fmt.Sprintf("%s/%d", seriesID, season))
	if c.ledger != nil {
		c.ledger.add(c.cost)
	}
	if err := c.errAt[i]; err != nil {
		return BurnedInCheckResult{CostUSD: c.cost}, err
	}
	return BurnedInCheckResult{Result: c.results[i], CostUSD: c.cost, Reason: "frames_unavailable"}, nil
}

type fakeMonthSpend struct {
	mu      sync.Mutex
	spent   decimal.Decimal
	other   decimal.Decimal // parse spend — must not count toward the cap
	err     error
	gotFrom time.Time
}

func (m *fakeMonthSpend) add(usd float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spent = m.spent.Add(decimal.NewFromFloat(usd))
}

func (m *fakeMonthSpend) TotalsByKindBetween(_ context.Context, from, _ time.Time) (map[models.AISideSpendKind]repository.AISideSpendTotals, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gotFrom = from
	if m.err != nil {
		return nil, m.err
	}
	return map[models.AISideSpendKind]repository.AISideSpendTotals{
		models.AISideSpendKindBurnedInCheck: {Calls: 1, Spent: m.spent},
		models.AISideSpendKindParse:         {Calls: 1, Spent: m.other},
	}, nil
}

func seasonRefs(n int) []repository.BurnedInSeasonRef {
	out := make([]repository.BurnedInSeasonRef, n)
	for i := range out {
		out[i] = repository.BurnedInSeasonRef{SeriesID: fmt.Sprintf("s%d", i+1), SeriesTitle: "Show", Season: 1}
	}
	return out
}

var burnedSweepNow = time.Date(2026, 11, 15, 12, 0, 0, 0, time.FixedZone("Asia/Taipei", 8*3600))

func newSweep(finder *fakeSeasonFinder, check *fakeSeasonCheck, spend *fakeMonthSpend, capUSD float64) *BurnedInSweeper {
	s := NewBurnedInSweeper(finder, check, spend, capUSD, nil)
	s.SetClock(func() time.Time { return burnedSweepNow })
	return s
}

// --- one round ---------------------------------------------------------------

func TestBurnedInSweep_ChecksEverySeasonInOrder(t *testing.T) {
	finder := &fakeSeasonFinder{seasons: seasonRefs(3)}
	spend := &fakeMonthSpend{}
	zh := models.BurnedInZhHans
	check := &fakeSeasonCheck{configured: true, cost: 0.0023, ledger: spend, results: map[int]*string{1: &zh}}

	res := newSweep(finder, check, spend, 1.0).Run(context.Background())
	assert.Equal(t, []string{"s1/1", "s2/1", "s3/1"}, check.checked)
	assert.Equal(t, 3, res.Seasons)
	assert.Equal(t, 3, res.Checked)
	assert.Equal(t, 1, res.Decided)
	assert.Empty(t, res.Stopped)
	assert.InDelta(t, 0.0069, res.SpentUSD, 1e-9)
	assert.Equal(t, burnedSweepNow.Add(-7*24*time.Hour), finder.retryBefore, "undecided seasons wait a week")
	assert.Equal(t, time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC), spend.gotFrom, "the server-zone calendar month the spend card uses")
}

func TestBurnedInSweep_MonthlyCap(t *testing.T) {
	t.Run("reached before the round: nothing is checked", func(t *testing.T) {
		finder := &fakeSeasonFinder{seasons: seasonRefs(2)}
		spend := &fakeMonthSpend{spent: decimal.RequireFromString("1.00")}
		check := &fakeSeasonCheck{configured: true}
		res := newSweep(finder, check, spend, 1.0).Run(context.Background())
		assert.True(t, res.CapHit)
		assert.Equal(t, "cap_reached", res.Stopped)
		assert.Empty(t, check.checked)
		assert.Zero(t, finder.callCount(), "no season list read either")
	})
	t.Run("reached mid-round: stops before the next season", func(t *testing.T) {
		finder := &fakeSeasonFinder{seasons: seasonRefs(5)}
		spend := &fakeMonthSpend{spent: decimal.RequireFromString("0.995")}
		check := &fakeSeasonCheck{configured: true, cost: 0.0023, ledger: spend}
		res := newSweep(finder, check, spend, 1.0).Run(context.Background())
		assert.Equal(t, []string{"s1/1", "s2/1", "s3/1"}, check.checked, "0.995 → 0.9973 → 0.9996 → 1.0019 ≥ cap")
		assert.True(t, res.CapHit)
		assert.InDelta(t, 1.0019, res.MonthUSD, 1e-9)
	})
	t.Run("other side spend does not count toward the cap", func(t *testing.T) {
		finder := &fakeSeasonFinder{seasons: seasonRefs(1)}
		spend := &fakeMonthSpend{other: decimal.RequireFromString("5")}
		check := &fakeSeasonCheck{configured: true}
		res := newSweep(finder, check, spend, 1.0).Run(context.Background())
		assert.False(t, res.CapHit)
		assert.Len(t, check.checked, 1)
	})
	t.Run("zero cap disables the automatic check", func(t *testing.T) {
		finder := &fakeSeasonFinder{seasons: seasonRefs(1)}
		check := &fakeSeasonCheck{configured: true}
		res := newSweep(finder, check, &fakeMonthSpend{}, 0).Run(context.Background())
		assert.Equal(t, "disabled", res.Stopped)
		assert.Empty(t, check.checked)
	})
	t.Run("unreadable spend stops the round", func(t *testing.T) {
		finder := &fakeSeasonFinder{seasons: seasonRefs(1)}
		check := &fakeSeasonCheck{configured: true}
		res := newSweep(finder, check, &fakeMonthSpend{err: errors.New("db locked")}, 1.0).Run(context.Background())
		assert.Equal(t, "spend_unreadable", res.Stopped)
		assert.Empty(t, check.checked)
	})
}

func TestBurnedInSweep_StopsQuietlyOrOnFailure(t *testing.T) {
	t.Run("no Claude key: quiet, nothing read", func(t *testing.T) {
		finder := &fakeSeasonFinder{seasons: seasonRefs(2)}
		check := &fakeSeasonCheck{configured: false}
		res := newSweep(finder, check, &fakeMonthSpend{}, 1.0).Run(context.Background())
		assert.Equal(t, "no_key", res.Stopped)
		assert.Zero(t, finder.callCount())
		assert.Empty(t, check.checked)
	})
	t.Run("a failed model call ends the round", func(t *testing.T) {
		finder := &fakeSeasonFinder{seasons: seasonRefs(3)}
		check := &fakeSeasonCheck{configured: true, errAt: map[int]error{1: fmt.Errorf("%w: %w", ErrBurnedInModelCall, ai.ErrAIProviderError)}}
		res := newSweep(finder, check, &fakeMonthSpend{}, 1.0).Run(context.Background())
		assert.Equal(t, []string{"s1/1", "s2/1"}, check.checked)
		assert.Equal(t, "model_call_failed", res.Stopped)
		assert.Equal(t, 1, res.Checked)
	})
	t.Run("the key going away mid-round ends it", func(t *testing.T) {
		finder := &fakeSeasonFinder{seasons: seasonRefs(3)}
		check := &fakeSeasonCheck{configured: true, errAt: map[int]error{0: ErrBurnedInNotConfigured}}
		res := newSweep(finder, check, &fakeMonthSpend{}, 1.0).Run(context.Background())
		assert.Equal(t, "no_key", res.Stopped)
		assert.Len(t, check.checked, 1)
	})
	t.Run("listing fails", func(t *testing.T) {
		finder := &fakeSeasonFinder{err: errors.New("db locked")}
		check := &fakeSeasonCheck{configured: true}
		res := newSweep(finder, check, &fakeMonthSpend{}, 1.0).Run(context.Background())
		assert.Equal(t, "list_failed", res.Stopped)
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		finder := &fakeSeasonFinder{seasons: seasonRefs(3)}
		check := &fakeSeasonCheck{configured: true}
		res := newSweep(finder, check, &fakeMonthSpend{}, 1.0).Run(ctx)
		assert.Equal(t, "cancelled", res.Stopped)
		assert.Empty(t, check.checked)
	})
}

// --- scheduling ------------------------------------------------------------------

func TestBurnedInSweep_OnlyOneRoundAtATime(t *testing.T) {
	finder := &fakeSeasonFinder{block: make(chan struct{}), entered: make(chan struct{}, 4)}
	s := newSweep(finder, &fakeSeasonCheck{configured: true}, &fakeMonthSpend{}, 1.0)
	ctx := context.Background()

	s.Trigger(ctx)
	<-finder.entered // the first round is inside its season listing
	assert.True(t, s.runExclusive(ctx).Skipped, "a second round while one runs is skipped, not queued")
	close(finder.block)
	s.Wait()
	assert.Equal(t, 1, finder.callCount())

	// Once it finished, the next call runs.
	assert.False(t, s.runExclusive(ctx).Skipped)
	assert.Equal(t, 2, finder.callCount())
}

func TestBurnedInSweep_StartRunsAfterTheBootDelayThenOnTheInterval(t *testing.T) {
	finder := &fakeSeasonFinder{entered: make(chan struct{}, 16)}
	s := newSweep(finder, &fakeSeasonCheck{configured: true}, &fakeMonthSpend{}, 1.0)
	s.SetSchedule(time.Millisecond, 2*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	for i := 0; i < 3; i++ { // boot round + two interval rounds
		select {
		case <-finder.entered:
		case <-time.After(2 * time.Second):
			t.Fatalf("round %d never ran", i+1)
		}
	}
	cancel()
	s.Wait()
	assert.GreaterOrEqual(t, finder.callCount(), 3)
}

func TestBurnedInSweep_StartStopsBeforeTheBootDelayWhenCancelled(t *testing.T) {
	finder := &fakeSeasonFinder{}
	s := newSweep(finder, &fakeSeasonCheck{configured: true}, &fakeMonthSpend{}, 1.0)
	s.SetSchedule(time.Hour, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	cancel()
	s.Wait()
	require.Zero(t, finder.callCount())
}
