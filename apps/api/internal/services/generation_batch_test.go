package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
)

// ─── Fakes ──────────────────────────────────────────────────────────────────

// fakeGenerationRunner is a narrow generationRunner fake (Rule 11).
type fakeGenerationRunner struct {
	mu        sync.Mutex
	calls     []int64
	errs      map[int64]error
	available bool
	// onCall lets a test spend from the ctx budget / observe ctx mid-item.
	onCall func(ctx context.Context, mediaID int64) error
}

func (f *fakeGenerationRunner) IsAvailable() bool { return f.available }

func (f *fakeGenerationRunner) RunTranscription(ctx context.Context, mediaID int64, _ string, _ string, _ ...TranscriptionOption) error {
	f.mu.Lock()
	f.calls = append(f.calls, mediaID)
	f.mu.Unlock()
	if f.onCall != nil {
		if err := f.onCall(ctx, mediaID); err != nil {
			return err
		}
	}
	if f.errs != nil {
		return f.errs[mediaID]
	}
	return nil
}

func (f *fakeGenerationRunner) callIDs() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]int64, len(f.calls))
	copy(out, f.calls)
	return out
}

// fakeCandidateFinder is a narrow generationCandidateFinder fake.
type fakeCandidateFinder struct {
	movies  []models.Movie
	count   int
	findErr error
	byID    map[string]*models.Movie
}

func (f *fakeCandidateFinder) FindMissingZhHantSubtitle(_ context.Context) ([]models.Movie, error) {
	return f.movies, f.findErr
}
func (f *fakeCandidateFinder) CountMissingZhHantSubtitle(_ context.Context) (int, error) {
	return f.count, f.findErr
}
func (f *fakeCandidateFinder) FindByID(_ context.Context, id string) (*models.Movie, error) {
	if m, ok := f.byID[id]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("movie with id %s not found", id)
}

func genMovie(id, title, filePath string) models.Movie {
	return models.Movie{ID: id, Title: title, FilePath: models.NewNullString(filePath)}
}

// waitUntilIdle polls until no batch is running (terminal state reached).
func waitUntilIdle(t *testing.T, p *GenerationBatchProcessor) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !p.IsRunning() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("batch did not reach a terminal state in time")
}

// drainEvents collects all generation_batch_progress payloads currently queued.
func drainEvents(client *sse.Client) []map[string]interface{} {
	var out []map[string]interface{}
	for {
		select {
		case ev := <-client.Events:
			if ev.Type != sse.EventGenerationBatchProgress {
				continue
			}
			if data, ok := ev.Data.(map[string]interface{}); ok {
				out = append(out, data)
			}
		default:
			return out
		}
	}
}

func newTestGenerationProcessor(t *testing.T, runner *fakeGenerationRunner, finder *fakeCandidateFinder, budgetUSD float64) (*GenerationBatchProcessor, *sse.Client) {
	t.Helper()
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })
	client := hub.Register()
	p := NewGenerationBatchProcessor(runner, finder, hub, budgetUSD, nil)
	return p, client
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestGenerationBatch_InitialState(t *testing.T) {
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, &fakeCandidateFinder{}, 5)
	assert.False(t, p.IsRunning())
	assert.Nil(t, p.GetProgress())
	active, _, _, _, _ := p.ActivityProgress()
	assert.False(t, active)
}

func TestGenerationBatch_IsAvailable(t *testing.T) {
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: false}, &fakeCandidateFinder{}, 5)
	assert.False(t, p.IsAvailable())
	p2, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, &fakeCandidateFinder{}, 5)
	assert.True(t, p2.IsAvailable())
}

// AC 5: sequential order over the enumerated queue, terminal complete.
func TestGenerationBatch_MissingScope_SequentialComplete(t *testing.T) {
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie("1", "Alpha", "/media/a.mkv"),
		genMovie("2", "Bravo", "/media/b.mkv"),
		genMovie("3", "Charlie", "/media/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	batchID, items, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, batchID)
	require.Len(t, items, 3)
	assert.Equal(t, int64(1), items[0].MediaID)
	assert.Equal(t, "Alpha", items[0].Title)

	waitUntilIdle(t, p)
	assert.Equal(t, []int64{1, 2, 3}, runner.callIDs(), "items must run sequentially in queue order")

	time.Sleep(50 * time.Millisecond) // let SSE fan-out drain
	events := drainEvents(client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusComplete, last["status"])
	assert.Equal(t, 3, last["success_count"])
	assert.Equal(t, 0, last["fail_count"])
	assert.Equal(t, 0, last["paused_count"])
}

// AC 9 [@contract-v1]: exact SSE payload keys.
func TestGenerationBatch_SSEPayloadFields(t *testing.T) {
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{movies: []models.Movie{genMovie("7", "Alpha", "/m/a.mkv")}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	waitUntilIdle(t, p)
	time.Sleep(50 * time.Millisecond)

	events := drainEvents(client)
	require.NotEmpty(t, events)
	wantKeys := []string{
		"batch_id", "total_items", "current_index", "current_media_id",
		"current_item", "success_count", "fail_count", "paused_count",
		"status", "spent_usd", "budget_usd",
	}
	for _, ev := range events {
		assert.Len(t, ev, len(wantKeys))
		for _, k := range wantKeys {
			assert.Contains(t, ev, k, "payload must carry %q", k)
		}
	}
	last := events[len(events)-1]
	assert.Equal(t, int64(7), last["current_media_id"])
	assert.Equal(t, 5.0, last["budget_usd"], "cost line rides the batch SSE (no 9R-17 needed)")
}

// AC 5: a failing item increments fail_count and the loop continues —
// including the per-media 409 (ErrTranscriptionInProgress) skip.
func TestGenerationBatch_PerItemFailContinue(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		errs: map[int64]error{
			2: errors.New("ffmpeg exploded"),
			3: ErrTranscriptionInProgress, // user ran it from the detail dialog mid-batch
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie("1", "A", "/m/a.mkv"),
		genMovie("2", "B", "/m/b.mkv"),
		genMovie("3", "C", "/m/c.mkv"),
		genMovie("4", "D", "/m/d.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []int64{1, 2, 3, 4}, runner.callIDs(), "loop must continue past failures")

	time.Sleep(50 * time.Millisecond)
	events := drainEvents(client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusComplete, last["status"])
	assert.Equal(t, 2, last["success_count"])
	assert.Equal(t, 2, last["fail_count"])
}

// AC 2: cancel — in-flight item's ctx cancelled, queued items never start.
func TestGenerationBatch_CancelMidItem(t *testing.T) {
	started := make(chan struct{})
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(ctx context.Context, mediaID int64) error {
			if mediaID == 1 {
				close(started)
				<-ctx.Done() // block until the batch is cancelled
				return ctx.Err()
			}
			return nil
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie("1", "A", "/m/a.mkv"),
		genMovie("2", "B", "/m/b.mkv"),
		genMovie("3", "C", "/m/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	<-started
	p.Cancel()
	waitUntilIdle(t, p)

	assert.Equal(t, []int64{1}, runner.callIDs(), "queued items must never start after cancel")

	time.Sleep(50 * time.Millisecond)
	events := drainEvents(client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusCancelled, last["status"])
}

// Cancel is idempotent and safe when nothing runs.
func TestGenerationBatch_CancelIdle(t *testing.T) {
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, &fakeCandidateFinder{}, 5)
	assert.NotPanics(t, func() { p.Cancel() })
}

// AC 7: pre-item budget check — exhausted envelope pauses the remainder
// (paused, NOT failed) and the batch ends budget_ceiling.
func TestGenerationBatch_BudgetCeiling_PreCheck(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(ctx context.Context, mediaID int64) error {
			if mediaID == 1 {
				// Spend $3 (> $1 ceiling) from the SHARED batch budget.
				ai.BudgetFromContext(ctx).RecordLLM("claude-sonnet-5", 1_000_000, 0)
			}
			return nil
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie("1", "A", "/m/a.mkv"),
		genMovie("2", "B", "/m/b.mkv"),
		genMovie("3", "C", "/m/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 1.0)

	_, _, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []int64{1}, runner.callIDs(), "items after the ceiling hit must not start")

	time.Sleep(50 * time.Millisecond)
	events := drainEvents(client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusBudgetCeiling, last["status"])
	assert.Equal(t, 1, last["success_count"], "completed items stay done")
	assert.Equal(t, 0, last["fail_count"], "paused is NOT failed")
	assert.Equal(t, 2, last["paused_count"])
	spent, ok := last["spent_usd"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 3.0, spent, 0.001)
}

// AC 7: mid-item ErrBudgetExceeded — that item AND the remaining queue are paused.
func TestGenerationBatch_BudgetCeiling_MidItem(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		errs: map[int64]error{
			2: fmt.Errorf("translate: %w", ai.ErrBudgetExceeded),
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie("1", "A", "/m/a.mkv"),
		genMovie("2", "B", "/m/b.mkv"),
		genMovie("3", "C", "/m/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []int64{1, 2}, runner.callIDs())

	time.Sleep(50 * time.Millisecond)
	events := drainEvents(client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusBudgetCeiling, last["status"])
	assert.Equal(t, 1, last["success_count"])
	assert.Equal(t, 0, last["fail_count"], "the interrupted item is paused, not failed")
	assert.Equal(t, 2, last["paused_count"], "interrupted item + remaining queue")
}

// 409 single-flight: a second Start while running is rejected.
func TestGenerationBatch_SingleFlight(t *testing.T) {
	started := make(chan struct{})
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(ctx context.Context, mediaID int64) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{genMovie("1", "A", "/m/a.mkv")}}
	p, _ := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	<-started

	_, _, err = p.Start(context.Background(), "missing", nil)
	assert.ErrorIs(t, err, ErrGenerationBatchRunning)

	prog := p.GetProgress()
	require.NotNil(t, prog)
	assert.Equal(t, GenerationBatchStatusRunning, prog.Status)
	assert.Equal(t, 5.0, prog.BudgetUSD)

	// AC 10: surfaces as an activity source while running.
	active, _, cur, total, item := p.ActivityProgress()
	assert.True(t, active)
	assert.Equal(t, 1, cur)
	assert.Equal(t, 1, total)
	assert.Equal(t, "A", item)

	p.Cancel()
	waitUntilIdle(t, p)
}

// AC 1: empty missing scope — nothing to do is not an error, no batch starts.
func TestGenerationBatch_EmptyMissingScope(t *testing.T) {
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, &fakeCandidateFinder{}, 5)

	batchID, items, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	assert.Empty(t, batchID)
	assert.NotNil(t, items)
	assert.Empty(t, items)
	assert.False(t, p.IsRunning())
}

// AC 8 + ID-type ruling: selected scope resolves int64 wire ids against the
// string-keyed movie repo; queue preserves the caller's order.
func TestGenerationBatch_SelectedScope(t *testing.T) {
	m7 := genMovie("7", "Seven", "/m/7.mkv")
	m9 := genMovie("9", "Nine", "/m/9.mkv")
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{byID: map[string]*models.Movie{"7": &m7, "9": &m9}}
	p, _ := newTestGenerationProcessor(t, runner, finder, 5)

	_, items, err := p.Start(context.Background(), "selected", []int64{9, 7})
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, int64(9), items[0].MediaID)
	assert.Equal(t, "Nine", items[0].Title)

	waitUntilIdle(t, p)
	assert.Equal(t, []int64{9, 7}, runner.callIDs())
}

// AC 8 ruling: a selected id that is not a movie (or has no file) REJECTS the
// request — documented in Swagger, no silent filtering.
func TestGenerationBatch_SelectedScope_InvalidIDRejected(t *testing.T) {
	m7 := genMovie("7", "Seven", "/m/7.mkv")
	noFile := models.Movie{ID: "8", Title: "NoFile"}
	finder := &fakeCandidateFinder{byID: map[string]*models.Movie{"7": &m7, "8": &noFile}}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, finder, 5)

	// Unknown id (e.g. a series id)
	_, _, err := p.Start(context.Background(), "selected", []int64{7, 999})
	assert.ErrorIs(t, err, ErrGenerationSelectionInvalid)
	assert.False(t, p.IsRunning())

	// Movie without a media file
	_, _, err = p.Start(context.Background(), "selected", []int64{8})
	assert.ErrorIs(t, err, ErrGenerationSelectionInvalid)
	assert.False(t, p.IsRunning())
}

// AC 3: preview returns the count without starting anything.
func TestGenerationBatch_PreviewMissing(t *testing.T) {
	finder := &fakeCandidateFinder{count: 38}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, finder, 5)

	n, err := p.PreviewMissing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 38, n)
	assert.False(t, p.IsRunning())
}

// Enumeration failure surfaces as a start error (500 at the handler).
func TestGenerationBatch_EnumerationError(t *testing.T) {
	finder := &fakeCandidateFinder{findErr: errors.New("db locked")}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrGenerationBatchRunning)
	assert.False(t, p.IsRunning())
}

// Rows with non-numeric ids or no file are skipped fail-soft in missing scope.
func TestGenerationBatch_MissingScope_SkipsMalformedRows(t *testing.T) {
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie("1", "A", "/m/a.mkv"),
		genMovie("not-a-number", "Weird", "/m/w.mkv"),
		{ID: "3", Title: "NoFile"},
	}}
	p, _ := newTestGenerationProcessor(t, runner, finder, 5)

	_, items, err := p.Start(context.Background(), "missing", nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, int64(1), items[0].MediaID)
	waitUntilIdle(t, p)
}
