package services

// Story dsr-6d-a: per-item states, the last terminal snapshot, the 202
// progress snapshot, the error terminal and episode series titles.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
)

// ─── Helpers ────────────────────────────────────────────────────────────────

func threeMovies() *fakeCandidateFinder {
	return &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
		genMovie(uuidC, "C", "/m/c.mkv"),
	}}
}

func terminalItems(t *testing.T, events []map[string]interface{}) []GenerationBatchItemState {
	t.Helper()
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	items, ok := last["items"].([]GenerationBatchItemState)
	require.True(t, ok, "a terminal broadcast carries the full queue, got %T", last["items"])
	return items
}

func statusesByID(items []GenerationBatchItemState) map[string]GenerationBatchItemStatus {
	out := make(map[string]GenerationBatchItemStatus, len(items))
	for _, it := range items {
		out[it.MediaID] = it.Status
	}
	return out
}

// assertInvariants pins AC #1: on every terminal the counts ARE the per-item tallies.
func assertInvariants(t *testing.T, prog *GenerationBatchProgress) {
	t.Helper()
	require.NotNil(t, prog)
	var done, failed, paused int
	for _, it := range prog.Items {
		switch it.Status {
		case GenerationBatchItemDone:
			done++
		case GenerationBatchItemFailed:
			failed++
		case GenerationBatchItemPaused:
			paused++
		case GenerationBatchItemQueued, GenerationBatchItemRunning:
			t.Errorf("terminal snapshot still has %s item %s", it.Status, it.MediaID)
		}
		if it.Status != GenerationBatchItemFailed {
			assert.Equal(t, GenerationBatchItemReasonNone, it.Reason, "reason is only set on failed items (%s)", it.MediaID)
		}
	}
	assert.Equal(t, prog.SuccessCount, done, "success_count == done items")
	assert.Equal(t, prog.FailCount, failed, "fail_count == failed items")
	assert.Equal(t, prog.PausedCount, paused, "paused_count == paused items")
	assert.Len(t, prog.Items, prog.TotalItems, "len(items) == total_items")
}

func lastSnapshot(t *testing.T, p *GenerationBatchProcessor) *GenerationBatchProgress {
	t.Helper()
	progress, last := p.Snapshot()
	require.Nil(t, progress, "no batch should be running")
	require.NotNil(t, last, "a finished batch leaves its terminal snapshot")
	return last
}

// ─── AC #1: per-item states and reasons ─────────────────────────────────────

func TestGenerationBatchItems_AllSucceed(t *testing.T) {
	p, client := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, threeMovies(), 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)

	items := terminalItems(t, eventsUntilTerminal(t, client))
	for _, it := range items {
		assert.Equal(t, GenerationBatchItemDone, it.Status, it.MediaID)
	}
	last := lastSnapshot(t, p)
	assert.Equal(t, GenerationBatchStatusComplete, last.Status)
	assertInvariants(t, last)
}

func TestGenerationBatchItems_FailureReasons(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		errs: map[string]error{
			uuidA: fmt.Errorf("media %s: only non-target tracks: %w", uuidA, ErrGenerationItemSkipped),
			uuidB: fmt.Errorf("media %s 已在管線佇列處理中: %w", uuidB, ErrTranscriptionInProgress),
			uuidC: errors.New("ffmpeg exploded"),
			uuidD: ErrTranscriptionDisabled,
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
		genMovie(uuidC, "C", "/m/c.mkv"),
		genMovie(uuidD, "D", "/m/d.mkv"),
		genMovie(uuidSeven, "Seven", "/m/7.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)
	eventsUntilTerminal(t, client)

	last := lastSnapshot(t, p)
	want := map[string]GenerationBatchItemReason{
		uuidA:     GenerationBatchItemReasonSkipped,
		uuidB:     GenerationBatchItemReasonBusyElsewhere,
		uuidC:     GenerationBatchItemReasonError,
		uuidD:     GenerationBatchItemReasonSkipped, // legacy "not configured" is the same class as a pipeline skip
		uuidSeven: GenerationBatchItemReasonNone,
	}
	for _, it := range last.Items {
		require.Contains(t, want, it.MediaID, "unexpected queue entry")
		assert.Equal(t, want[it.MediaID], it.Reason, it.MediaID)
		if it.MediaID == uuidSeven {
			assert.Equal(t, GenerationBatchItemDone, it.Status)
		} else {
			assert.Equal(t, GenerationBatchItemFailed, it.Status, it.MediaID)
		}
	}
	assert.Equal(t, 4, last.FailCount, "fail_count semantics unchanged: skips and busy items still count as failures")
	assertInvariants(t, last)
}

func TestGenerationBatchItems_RunningBroadcastsCarryTheChangedItem(t *testing.T) {
	runner := &fakeGenerationRunner{available: true, errs: map[string]error{uuidB: errors.New("boom")}}
	p, client := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)

	events := eventsUntilTerminal(t, client)
	require.Len(t, events, 7, "3 items × (running + outcome) + terminal")

	type change struct {
		id     string
		status GenerationBatchItemStatus
		reason GenerationBatchItemReason
	}
	var got []change
	for _, ev := range events[:len(events)-1] {
		assert.Nil(t, ev["items"], "running broadcasts must not carry the whole queue")
		ch, ok := ev["changed_item"].(GenerationBatchItemState)
		require.True(t, ok, "running broadcasts carry changed_item, got %T", ev["changed_item"])
		got = append(got, change{ch.MediaID, ch.Status, ch.Reason})
	}
	assert.Equal(t, []change{
		{uuidA, GenerationBatchItemRunning, ""},
		{uuidA, GenerationBatchItemDone, ""},
		{uuidB, GenerationBatchItemRunning, ""},
		{uuidB, GenerationBatchItemFailed, GenerationBatchItemReasonError},
		{uuidC, GenerationBatchItemRunning, ""},
		{uuidC, GenerationBatchItemDone, ""},
	}, got)

	terminal := events[len(events)-1]
	assert.Nil(t, terminal["changed_item"], "the terminal broadcast carries the full queue instead")
	assert.Len(t, terminalItems(t, events), 3)
}

// AC #2: 11 original keys + items + changed_item on EVERY broadcast.
func TestGenerationBatchItems_SSEKeySet(t *testing.T) {
	p, client := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, threeMovies(), 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)

	want := []string{
		"batch_id", "total_items", "current_index", "current_media_id",
		"current_item", "success_count", "fail_count", "paused_count",
		"status", "spent_usd", "budget_usd", "items", "changed_item",
	}
	for _, ev := range eventsUntilTerminal(t, client) {
		assert.Len(t, ev, len(want))
		for _, k := range want {
			assert.Contains(t, ev, k)
		}
	}
}

func TestGenerationBatchItems_BudgetPreCheckPausesTheRest(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(ctx context.Context, mediaID string) error {
			if mediaID == uuidA {
				ai.BudgetFromContext(ctx).RecordLLM("claude-sonnet-5", 1_000_000, 0)
			}
			return nil
		},
	}
	p, client := newTestGenerationProcessor(t, runner, threeMovies(), 1.0)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)
	eventsUntilTerminal(t, client)

	last := lastSnapshot(t, p)
	assert.Equal(t, GenerationBatchStatusBudgetCeiling, last.Status)
	assert.Equal(t, map[string]GenerationBatchItemStatus{
		uuidA: GenerationBatchItemDone,
		uuidB: GenerationBatchItemPaused,
		uuidC: GenerationBatchItemPaused,
	}, statusesByID(last.Items))
	assertInvariants(t, last)
}

func TestGenerationBatchItems_BudgetMidItemPausesItAndTheRest(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		errs:      map[string]error{uuidB: fmt.Errorf("translate: %w", ai.ErrBudgetExceeded)},
	}
	p, client := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)
	eventsUntilTerminal(t, client)

	last := lastSnapshot(t, p)
	assert.Equal(t, map[string]GenerationBatchItemStatus{
		uuidA: GenerationBatchItemDone,
		uuidB: GenerationBatchItemPaused,
		uuidC: GenerationBatchItemPaused,
	}, statusesByID(last.Items))
	assertInvariants(t, last)
}

func TestGenerationBatchItems_CancelMidItemCancelsItAndTheRest(t *testing.T) {
	started := make(chan struct{})
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(ctx context.Context, mediaID string) error {
			if mediaID == uuidB {
				close(started)
				<-ctx.Done()
				return ctx.Err()
			}
			return nil
		},
	}
	p, client := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	<-started
	p.Cancel()
	waitUntilIdle(t, p)
	eventsUntilTerminal(t, client)

	last := lastSnapshot(t, p)
	assert.Equal(t, GenerationBatchStatusCancelled, last.Status)
	assert.Equal(t, map[string]GenerationBatchItemStatus{
		uuidA: GenerationBatchItemDone,
		uuidB: GenerationBatchItemCancelled,
		uuidC: GenerationBatchItemCancelled,
	}, statusesByID(last.Items))
	assertInvariants(t, last)
}

func TestGenerationBatchItems_CancelBeforeTheNextItem(t *testing.T) {
	var p *GenerationBatchProcessor
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(_ context.Context, mediaID string) error {
			if mediaID == uuidA {
				p.Cancel() // the ctx check before item B trips
			}
			return nil
		},
	}
	proc, sseClient := newTestGenerationProcessor(t, runner, threeMovies(), 5)
	p = proc

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)
	eventsUntilTerminal(t, sseClient)

	last := lastSnapshot(t, p)
	assert.Equal(t, GenerationBatchStatusCancelled, last.Status)
	assert.Equal(t, GenerationBatchItemDone, statusesByID(last.Items)[uuidA],
		"A finished before the cancel landed (the runner returned nil)")
	assert.Equal(t, GenerationBatchItemCancelled, statusesByID(last.Items)[uuidC])
	assertInvariants(t, last)
}

// The queue exists from the moment Start returns — the 202 must never carry an
// empty queue because the goroutine has not run yet.
func TestGenerationBatchItems_QueueIsFilledBeforeStartReturns(t *testing.T) {
	release := make(chan struct{})
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(_ context.Context, _ string) error {
			<-release
			return nil
		},
	}
	p, _ := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	batchID, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)

	snap := p.SnapshotFor(batchID)
	require.NotNil(t, snap)
	require.Len(t, snap.Items, 3)
	for i, it := range snap.Items {
		if i == 0 {
			assert.Contains(t, []GenerationBatchItemStatus{GenerationBatchItemQueued, GenerationBatchItemRunning}, it.Status)
			continue
		}
		assert.Equal(t, GenerationBatchItemQueued, it.Status)
	}

	// Let the batch finish inside the test: an orphaned process goroutine would
	// otherwise broadcast into a hub that is already shutting down.
	close(release)
	waitUntilIdle(t, p)
}

// Snapshots are deep copies: mutating one never reaches the processor.
func TestGenerationBatchItems_SnapshotsDoNotShareTheQueue(t *testing.T) {
	release := make(chan struct{})
	runner := &fakeGenerationRunner{available: true, onCall: func(_ context.Context, _ string) error {
		<-release
		return nil
	}}
	p, _ := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	batchID, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	snap := p.SnapshotFor(batchID)
	require.NotNil(t, snap)
	// The processor never rewrites a title, so a shared backing array would
	// leak this edit into every later snapshot.
	snap.Items[1].SeriesTitle = "edited by a reader"
	close(release)
	waitUntilIdle(t, p)

	fresh := p.SnapshotFor(batchID)
	require.NotNil(t, fresh)
	assert.Equal(t, "", fresh.Items[1].SeriesTitle)
	assert.Equal(t, GenerationBatchItemDone, fresh.Items[1].Status)
}

// ─── AC #4: panic → error terminal ──────────────────────────────────────────

func TestGenerationBatchItems_PanicEndsTheBatchWithError(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(_ context.Context, mediaID string) error {
			if mediaID == uuidB {
				panic("nil map write in some adapter")
			}
			return nil
		},
	}
	p, client := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)

	events := eventsUntilTerminal(t, client)
	assert.Equal(t, GenerationBatchStatusError, events[len(events)-1]["status"])

	last := lastSnapshot(t, p)
	assert.Equal(t, GenerationBatchStatusError, last.Status)
	assert.Equal(t, map[string]GenerationBatchItemStatus{
		uuidA: GenerationBatchItemDone,
		uuidB: GenerationBatchItemFailed,
		uuidC: GenerationBatchItemCancelled,
	}, statusesByID(last.Items))
	for _, it := range last.Items {
		if it.MediaID == uuidB {
			assert.Equal(t, GenerationBatchItemReasonError, it.Reason)
		}
	}
	assert.Equal(t, 1, last.FailCount, "the item that panicked counts as failed")
	assertInvariants(t, last)
	assert.False(t, p.IsRunning())

	// The processor is usable again.
	runner.onCall = nil
	_, _, err = p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)
}

// ─── AC #3 / #5 / #6: last, SnapshotFor, DismissLast ────────────────────────

func TestGenerationBatchItems_LastLifecycle(t *testing.T) {
	gate := make(chan struct{})
	var gated atomic.Bool
	runner := &fakeGenerationRunner{available: true, onCall: func(_ context.Context, _ string) error {
		if gated.Load() {
			<-gate
		}
		return nil
	}}
	p, _ := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	progress, last := p.Snapshot()
	assert.Nil(t, progress)
	assert.Nil(t, last, "nothing has finished yet")

	firstID, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)

	got := lastSnapshot(t, p)
	assert.Equal(t, firstID, got.BatchID)
	assert.Equal(t, got, p.SnapshotFor(firstID), "SnapshotFor falls back to last for a finished batch")
	assert.Nil(t, p.SnapshotFor("some-other-batch"))

	// A new batch clears last while it runs.
	gated.Store(true)
	secondID, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	progress, last = p.Snapshot()
	require.NotNil(t, progress)
	assert.Equal(t, secondID, progress.BatchID)
	assert.Nil(t, last, "last is null while a batch runs")
	assert.Nil(t, p.SnapshotFor(firstID), "the previous batch's snapshot is gone")

	dismissed, running := p.DismissLast()
	assert.False(t, dismissed, "nothing is dismissed while running")
	assert.True(t, running)

	close(gate)
	waitUntilIdle(t, p)

	dismissed, running = p.DismissLast()
	assert.True(t, dismissed)
	assert.False(t, running)
	_, last = p.Snapshot()
	assert.Nil(t, last)

	dismissed, running = p.DismissLast()
	assert.False(t, dismissed, "idempotent")
	assert.False(t, running)
}

// A 409 or an empty scope must not wipe the last result.
func TestGenerationBatchItems_RejectedStartsKeepLast(t *testing.T) {
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, threeMovies(), 5)
	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)
	require.NotNil(t, lastSnapshot(t, p))

	empty := &fakeCandidateFinder{}
	p.finder = empty
	id, items, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	assert.Empty(t, id)
	assert.Empty(t, items)
	_, last := p.Snapshot()
	assert.NotNil(t, last, "an empty scope starts nothing, so it must not clear last")
}

// ActivityProgress keeps working and does not need the queue.
func TestGenerationBatchItems_ActivityProgressUnchanged(t *testing.T) {
	release := make(chan struct{})
	runner := &fakeGenerationRunner{available: true, onCall: func(_ context.Context, _ string) error {
		<-release
		return nil
	}}
	p, _ := newTestGenerationProcessor(t, runner, threeMovies(), 5)
	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		active, _, current, total, item := p.ActivityProgress()
		return active && current == 1 && total == 3 && item == "A"
	}, time.Second, time.Millisecond)
	close(release)
	waitUntilIdle(t, p)
}

// ─── AC #7: series titles ───────────────────────────────────────────────────

type countingSeriesResolver struct {
	mu     sync.Mutex
	calls  map[string]int
	titles map[string]string
	err    error
}

func (c *countingSeriesResolver) FindByID(_ context.Context, id string) (*models.Series, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.calls == nil {
		c.calls = map[string]int{}
	}
	c.calls[id]++
	if c.err != nil {
		return nil, c.err
	}
	title, ok := c.titles[id]
	if !ok {
		return nil, fmt.Errorf("series with id %s not found", id)
	}
	return &models.Series{ID: id, Title: title}, nil
}

func (c *countingSeriesResolver) callsFor(id string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[id]
}

func episodesProcessor(t *testing.T, resolver CandidateSeriesTitleResolver) (*GenerationBatchProcessor, *fakeEpisodeFinder) {
	t.Helper()
	ep1 := genEpisode(uuidA, 4, 7, "第七章", "/tv/s4e7.mkv")
	ep2 := genEpisode(uuidB, 4, 8, "第八章", "/tv/s4e8.mkv")
	ep3 := genEpisode(uuidC, 1, 1, "", "/tv/other.mkv")
	ep3.SeriesID = "series-2"
	ep4 := genEpisode(uuidD, 1, 2, "", "/tv/orphan.mkv")
	ep4.SeriesID = ""
	episodes := &fakeEpisodeFinder{byID: map[string]*models.Episode{
		uuidA: &ep1, uuidB: &ep2, uuidC: &ep3, uuidD: &ep4,
	}}
	finder := &fakeCandidateFinder{
		byID: map[string]*models.Movie{uuidSeven: func() *models.Movie { m := genMovie(uuidSeven, "Movie", "/m/7.mkv"); return &m }()},
	}
	p, _ := newTestGenerationProcessorWithEpisodes(t, &fakeGenerationRunner{available: true}, finder, episodes, 5)
	if resolver != nil {
		p.SetSeriesTitleResolver(resolver)
	}
	return p, episodes
}

func TestGenerationBatchItems_EpisodesCarryTheSeriesTitleOncePerSeries(t *testing.T) {
	resolver := &countingSeriesResolver{titles: map[string]string{"series-1": "怪奇物語", "series-2": "黑鏡"}}
	p, _ := episodesProcessor(t, resolver)

	_, items, err := p.Start(context.Background(), "selected", []string{uuidA, uuidB, uuidC, uuidD, uuidSeven}, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)

	byID := map[string]GenerationBatchItem{}
	for _, it := range items {
		byID[it.MediaID] = it
	}
	assert.Equal(t, "怪奇物語", byID[uuidA].SeriesTitle)
	assert.Equal(t, "怪奇物語", byID[uuidB].SeriesTitle)
	assert.Equal(t, "S04E07 第七章", byID[uuidA].Title, "title itself is unchanged")
	assert.Equal(t, "黑鏡", byID[uuidC].SeriesTitle)
	assert.Equal(t, "", byID[uuidD].SeriesTitle, "an episode without a series id skips the lookup")
	assert.Equal(t, "", byID[uuidSeven].SeriesTitle, "movies have no series title")
	assert.Equal(t, 1, resolver.callsFor("series-1"), "one lookup per series within a batch")
	assert.Equal(t, 0, resolver.callsFor(""), "no lookup for an empty series id")

	// The memo does not outlive the batch.
	_, _, err = p.Start(context.Background(), "selected", []string{uuidA}, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)
	assert.Equal(t, 2, resolver.callsFor("series-1"), "a new batch looks the series up again")

	last := lastSnapshot(t, p)
	assert.Equal(t, "怪奇物語", last.Items[0].SeriesTitle, "item states carry the series title too")
}

func TestGenerationBatchItems_SeriesLookupFailureDegrades(t *testing.T) {
	p, _ := episodesProcessor(t, &countingSeriesResolver{err: errors.New("database is locked")})

	_, items, err := p.Start(context.Background(), "selected", []string{uuidA}, 0, "")
	require.NoError(t, err, "a display-only lookup must never fail the batch start")
	require.Len(t, items, 1)
	assert.Equal(t, "", items[0].SeriesTitle)
	waitUntilIdle(t, p)
}

func TestGenerationBatchItems_NoResolverMeansEmptySeriesTitle(t *testing.T) {
	p, _ := episodesProcessor(t, nil)

	_, items, err := p.Start(context.Background(), "selected", []string{uuidA}, 0, "")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "", items[0].SeriesTitle)
	waitUntilIdle(t, p)
}

// ─── CR H1: the re-entrancy guards ──────────────────────────────────────────

// A late finish for a batch that already ended must not touch the one running
// now — the recover path can re-enter finish after a new batch has started.
func TestGenerationBatchItems_FinishIgnoresAStaleBatch(t *testing.T) {
	release := make(chan struct{})
	runner := &fakeGenerationRunner{available: true, onCall: func(_ context.Context, _ string) error {
		<-release
		return nil
	}}
	p, client := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	liveID, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	drainEvents(t, client)

	p.finish("a-batch-that-ended-long-ago", GenerationBatchStatusError, 0, GenerationBatchItem{}, ai.NewBudget(5))
	p.markItem("a-batch-that-ended-long-ago", 2, GenerationBatchItem{MediaID: uuidC}, GenerationBatchItemFailed, GenerationBatchItemReasonError, ai.NewBudget(5))

	live := p.SnapshotFor(liveID)
	require.NotNil(t, live, "the live batch must still be running")
	assert.Equal(t, GenerationBatchStatusRunning, live.Status)
	assert.Equal(t, 0, live.FailCount, "a stale finish must not invent failures")
	assert.Equal(t, GenerationBatchItemQueued, live.Items[2].Status)
	assert.Empty(t, drainEvents(t, client), "a stale call broadcasts nothing")

	close(release)
	waitUntilIdle(t, p)
}

// finish is idempotent: a second call for the same batch (a panic inside
// finish re-entering through the recover) changes nothing.
func TestGenerationBatchItems_FinishIsIdempotent(t *testing.T) {
	p, client := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, threeMovies(), 5)

	batchID, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)
	eventsUntilTerminal(t, client)
	before := lastSnapshot(t, p)

	p.finish(batchID, GenerationBatchStatusCancelled, 0, GenerationBatchItem{}, ai.NewBudget(5))

	after := lastSnapshot(t, p)
	assert.Equal(t, before, after, "the terminal snapshot is written once")
	assert.Equal(t, GenerationBatchStatusComplete, after.Status)
	assert.Empty(t, drainEvents(t, client), "no second terminal event")
}

// CR M1: two Starts that both pass the quick check — the loser gets 409 and
// leaves no batch behind.
func TestGenerationBatchItems_SecondStartLosesTheDoubleCheck(t *testing.T) {
	enumerating := make(chan struct{})
	releaseEnumeration := make(chan struct{})
	release := make(chan struct{})
	finder := &blockingCandidateFinder{
		fakeCandidateFinder: threeMovies(),
		entered:             enumerating,
		release:             releaseEnumeration,
	}
	runner := &fakeGenerationRunner{available: true, onCall: func(_ context.Context, _ string) error {
		<-release
		return nil
	}}
	p, _ := newTestGenerationProcessor(t, runner, finder.fakeCandidateFinder, 5)
	p.finder = finder

	type startResult struct {
		id  string
		err error
	}
	results := make(chan startResult, 2)
	go func() {
		id, _, err := p.Start(context.Background(), "missing", nil, 0, "")
		results <- startResult{id, err}
	}()
	<-enumerating // the first Start is past the quick check, inside collectItems
	go func() {
		id, _, err := p.Start(context.Background(), "missing", nil, 0, "")
		results <- startResult{id, err}
	}()
	<-enumerating
	close(releaseEnumeration)

	var started, conflicts int
	for i := 0; i < 2; i++ {
		r := <-results
		if errors.Is(r.err, ErrGenerationBatchRunning) {
			conflicts++
			assert.Empty(t, r.id)
			continue
		}
		require.NoError(t, r.err)
		started++
	}
	assert.Equal(t, 1, started, "single-flight: exactly one batch starts")
	assert.Equal(t, 1, conflicts)
	assert.True(t, p.IsRunning())

	close(release)
	waitUntilIdle(t, p)
	assert.Len(t, lastSnapshot(t, p).Items, 3, "only one queue was ever enumerated")
}

// blockingCandidateFinder lets a test hold two Start calls inside collectItems
// at the same time, so both reach the second (locked) check.
type blockingCandidateFinder struct {
	*fakeCandidateFinder
	entered chan struct{}
	release chan struct{}
}

func (f *blockingCandidateFinder) FindMissingZhHantSubtitle(ctx context.Context) ([]models.Movie, error) {
	f.entered <- struct{}{}
	<-f.release
	return f.fakeCandidateFinder.FindMissingZhHantSubtitle(ctx)
}

// drainEvents returns every generation_batch_progress payload available right
// now, without waiting for a terminal one.
func drainEvents(t *testing.T, client *sse.Client) []map[string]interface{} {
	t.Helper()
	var out []map[string]interface{}
	for {
		select {
		case ev, ok := <-client.Events:
			if !ok {
				return out
			}
			if data, isMap := ev.Data.(map[string]interface{}); isMap && ev.Type == sse.EventGenerationBatchProgress {
				out = append(out, data)
			}
		case <-time.After(50 * time.Millisecond):
			return out
		}
	}
}
