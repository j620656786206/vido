package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/sse"
)

// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS —
// mirror the prod creation path (uuid.New().String()); do NOT invent numeric
// ids (numeric fixtures hid the int64 contract bug through three gate layers).
const (
	uuidA     = "0a54a9e2-3a67-4f3e-9f8e-a1c2d3e4f501"
	uuidB     = "1b65baf3-4b78-4a4f-8a9f-b2d3e4f5a602"
	uuidC     = "2c76cba4-5c89-4b5a-9baf-c3e4f5a6b703"
	uuidD     = "3d87dcb5-6d9a-4c6b-8cba-d4f5a6b7c804"
	uuidSeven = "7e98edc6-7eab-4d7c-9dcb-e5a6b7c8d905"
	uuidEight = "8fa9fed7-8fbc-4e8d-8edc-f6b7c8d9e006"
	uuidNine  = "9ab0afe8-9acd-4f9e-9fed-a7c8d9e0f107"
)

// ─── Fakes ──────────────────────────────────────────────────────────────────

// fakeGenerationRunner is a narrow GenerationRunner fake (Rule 11).
type fakeGenerationRunner struct {
	mu         sync.Mutex
	calls      []string
	mediaTypes []string
	errs       map[string]error
	available  bool
	// onCall lets a test spend from the ctx budget / observe ctx mid-item.
	onCall func(ctx context.Context, mediaID string) error
}

func (f *fakeGenerationRunner) IsAvailable() bool { return f.available }

func (f *fakeGenerationRunner) ExecuteGeneration(ctx context.Context, mediaID, mediaType, _, _ string) error {
	f.mu.Lock()
	f.calls = append(f.calls, mediaID)
	f.mediaTypes = append(f.mediaTypes, mediaType)
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

func (f *fakeGenerationRunner) callIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *fakeGenerationRunner) callMediaTypes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.mediaTypes))
	copy(out, f.mediaTypes)
	return out
}

// fakeCandidateFinder is a narrow generationCandidateFinder fake.
type fakeCandidateFinder struct {
	movies  []models.Movie
	count   int
	findErr error
	byID    map[string]*models.Movie
	byIDErr error // non-nil = simulate a REAL lookup failure (not not-found)
}

func (f *fakeCandidateFinder) FindMissingZhHantSubtitle(_ context.Context) ([]models.Movie, error) {
	return f.movies, f.findErr
}
func (f *fakeCandidateFinder) CountMissingZhHantSubtitle(_ context.Context) (int, error) {
	return f.count, f.findErr
}
func (f *fakeCandidateFinder) FindByID(_ context.Context, id string) (*models.Movie, error) {
	if f.byIDErr != nil {
		return nil, f.byIDErr
	}
	if m, ok := f.byID[id]; ok {
		return m, nil
	}
	// Mirror the real repo's not-found shape (movie_repository.go wraps
	// sql.ErrNoRows) — collectItems classifies on it (CR M2).
	return nil, fmt.Errorf("movie with id %s not found: %w", id, sql.ErrNoRows)
}

// fakeEpisodeFinder is a narrow generationEpisodeFinder fake (sub-4-2).
type fakeEpisodeFinder struct {
	byID    map[string]*models.Episode
	byIDErr error // non-nil = simulate a REAL lookup failure (not not-found)
	// count is the episode half of the preview (sub-5-1 AC #7).
	count    int
	countErr error
}

func (f *fakeEpisodeFinder) CountMissingZhHantSubtitle(_ context.Context) (int, error) {
	if f.countErr != nil {
		return 0, f.countErr
	}
	return f.count, nil
}

func (f *fakeEpisodeFinder) FindByID(_ context.Context, id string) (*models.Episode, error) {
	if f.byIDErr != nil {
		return nil, f.byIDErr
	}
	if e, ok := f.byID[id]; ok {
		return e, nil
	}
	// Mirror the real repo's not-found shape (episode_repository.go wraps
	// repository.ErrEpisodeNotFound) — toEpisodeItem classifies on it (CR M2).
	return nil, fmt.Errorf("episode with id %s: %w", id, repository.ErrEpisodeNotFound)
}

func genMovie(id, title, filePath string) models.Movie {
	return models.Movie{ID: id, Title: title, FilePath: models.NewNullString(filePath)}
}

func genEpisode(id string, season, episode int, title, filePath string) models.Episode {
	return models.Episode{
		ID:            id,
		SeriesID:      "series-1",
		SeasonNumber:  season,
		EpisodeNumber: episode,
		Title:         models.NewNullString(title),
		FilePath:      models.NewNullString(filePath),
	}
}

// waitUntilIdle polls until no batch is running (terminal state reached).
func waitUntilIdle(t *testing.T, p *GenerationBatchProcessor) {
	t.Helper()
	require.Eventually(t, func() bool { return !p.IsRunning() }, 3*time.Second, 5*time.Millisecond,
		"batch did not reach a terminal state in time")
}

// eventsUntilTerminal reads generation_batch_progress payloads from the client
// until the FIRST terminal status (anything but `running`) and returns every
// payload seen up to and including it. Bounded by a 2 s deadline that fails
// the test with the statuses seen so far.
//
// This replaces the former `time.Sleep(50ms)` + non-blocking drain
// (preexisting-fail-generation-batch-cancel-mid-item-flake). Two things made
// the sleep necessary and unreliable: `finish` clears activeBatch — so
// waitUntilIdle returns — BEFORE it broadcasts the terminal event, and the
// Hub's fan-out goroutine races the test thread. Waiting for the event the
// test is about to assert on is the only synchronisation that cannot lose.
func eventsUntilTerminal(t *testing.T, client *sse.Client) []map[string]interface{} {
	t.Helper()
	var out []map[string]interface{}
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev, ok := <-client.Events:
			if !ok {
				t.Fatalf("SSE client closed before a terminal status; saw %v", statusesOf(out))
			}
			if ev.Type != sse.EventGenerationBatchProgress {
				continue
			}
			data, ok := ev.Data.(map[string]interface{})
			if !ok {
				continue
			}
			out = append(out, data)
			if status, _ := data["status"].(string); status != GenerationBatchStatusRunning {
				return out
			}
		case <-deadline:
			t.Fatalf("no terminal generation_batch_progress event within 2s; saw %v", statusesOf(out))
		}
	}
}

func statusesOf(events []map[string]interface{}) []interface{} {
	out := make([]interface{}, 0, len(events))
	for _, e := range events {
		out = append(out, e["status"])
	}
	return out
}

func newTestGenerationProcessor(t *testing.T, runner *fakeGenerationRunner, finder *fakeCandidateFinder, budgetUSD float64) (*GenerationBatchProcessor, *sse.Client) {
	t.Helper()
	return newTestGenerationProcessorWithEpisodes(t, runner, finder, nil, budgetUSD)
}

func newTestGenerationProcessorWithEpisodes(t *testing.T, runner *fakeGenerationRunner, finder *fakeCandidateFinder, episodes *fakeEpisodeFinder, budgetUSD float64) (*GenerationBatchProcessor, *sse.Client) {
	t.Helper()
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })
	client := hub.Register()
	// Register only ENQUEUES the client; Hub.Run registers it later and picks
	// randomly between register and broadcast when both are ready — so a batch
	// started right away could fan its events out to zero clients. Wait for
	// the registration to land before handing the processor back.
	require.Eventually(t, func() bool { return hub.ClientCount() == 1 }, 2*time.Second, time.Millisecond,
		"SSE client never registered")
	var epFinder generationEpisodeFinder
	if episodes != nil {
		epFinder = episodes
	}
	p := NewGenerationBatchProcessor(runner, finder, epFinder, hub, budgetUSD, nil)
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
		genMovie(uuidA, "Alpha", "/media/a.mkv"),
		genMovie(uuidB, "Bravo", "/media/b.mkv"),
		genMovie(uuidC, "Charlie", "/media/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	batchID, items, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, batchID)
	require.Len(t, items, 3)
	assert.Equal(t, uuidA, items[0].MediaID)
	assert.Equal(t, "Alpha", items[0].Title)

	waitUntilIdle(t, p)
	assert.Equal(t, []string{uuidA, uuidB, uuidC}, runner.callIDs(), "items must run sequentially in queue order")

	events := eventsUntilTerminal(t, client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusComplete, last["status"])
	assert.Equal(t, 3, last["success_count"])
	assert.Equal(t, 0, last["fail_count"])
	assert.Equal(t, 0, last["paused_count"])
}

// AC 9 [@contract-v2]: exact SSE payload keys; current_media_id is a UUID STRING.
func TestGenerationBatch_SSEPayloadFields(t *testing.T) {
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{movies: []models.Movie{genMovie(uuidSeven, "Alpha", "/m/a.mkv")}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	events := eventsUntilTerminal(t, client)
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
	assert.Equal(t, uuidSeven, last["current_media_id"], "media ids are UUID strings on the wire (9R-18)")
	assert.Equal(t, 5.0, last["budget_usd"], "cost line rides the batch SSE (no 9R-17 needed)")
}

// AC 5: a failing item increments fail_count and the loop continues —
// including the per-media 409 (ErrTranscriptionInProgress) skip.
func TestGenerationBatch_PerItemFailContinue(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		errs: map[string]error{
			uuidB: errors.New("ffmpeg exploded"),
			uuidC: ErrTranscriptionInProgress, // user ran it from the detail dialog mid-batch
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
		genMovie(uuidC, "C", "/m/c.mkv"),
		genMovie(uuidD, "D", "/m/d.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []string{uuidA, uuidB, uuidC, uuidD}, runner.callIDs(), "loop must continue past failures")

	events := eventsUntilTerminal(t, client)
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
		onCall: func(ctx context.Context, mediaID string) error {
			if mediaID == uuidA {
				close(started)
				<-ctx.Done() // block until the batch is cancelled
				return ctx.Err()
			}
			return nil
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
		genMovie(uuidC, "C", "/m/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	<-started
	p.Cancel()
	waitUntilIdle(t, p)

	assert.Equal(t, []string{uuidA}, runner.callIDs(), "queued items must never start after cancel")

	events := eventsUntilTerminal(t, client)
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
		onCall: func(ctx context.Context, mediaID string) error {
			if mediaID == uuidA {
				// Spend $3 (> $1 ceiling) from the SHARED batch budget.
				ai.BudgetFromContext(ctx).RecordLLM("claude-sonnet-5", 1_000_000, 0)
			}
			return nil
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
		genMovie(uuidC, "C", "/m/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 1.0)

	_, _, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []string{uuidA}, runner.callIDs(), "items after the ceiling hit must not start")

	events := eventsUntilTerminal(t, client)
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
		errs: map[string]error{
			uuidB: fmt.Errorf("translate: %w", ai.ErrBudgetExceeded),
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
		genMovie(uuidC, "C", "/m/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []string{uuidA, uuidB}, runner.callIDs())

	events := eventsUntilTerminal(t, client)
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
		onCall: func(ctx context.Context, mediaID string) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{genMovie(uuidA, "A", "/m/a.mkv")}}
	p, _ := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	<-started

	_, _, err = p.Start(context.Background(), "missing", nil, 0)
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

	batchID, items, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	assert.Empty(t, batchID)
	assert.NotNil(t, items)
	assert.Empty(t, items)
	assert.False(t, p.IsRunning())
}

// AC 8 + ID-type ruling (9R-18): selected scope passes the string UUID wire ids
// straight to the string-keyed movie repo; queue preserves the caller's order.
func TestGenerationBatch_SelectedScope(t *testing.T) {
	m7 := genMovie(uuidSeven, "Seven", "/m/7.mkv")
	m9 := genMovie(uuidNine, "Nine", "/m/9.mkv")
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{byID: map[string]*models.Movie{uuidSeven: &m7, uuidNine: &m9}}
	p, _ := newTestGenerationProcessor(t, runner, finder, 5)

	_, items, err := p.Start(context.Background(), "selected", []string{uuidNine, uuidSeven}, 0)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, uuidNine, items[0].MediaID)
	assert.Equal(t, "Nine", items[0].Title)

	waitUntilIdle(t, p)
	assert.Equal(t, []string{uuidNine, uuidSeven}, runner.callIDs())
}

// AC 8 ruling: a selected id that is not a movie (or has no file) REJECTS the
// request — documented in Swagger, no silent filtering.
func TestGenerationBatch_SelectedScope_InvalidIDRejected(t *testing.T) {
	m7 := genMovie(uuidSeven, "Seven", "/m/7.mkv")
	noFile := models.Movie{ID: uuidEight, Title: "NoFile"}
	finder := &fakeCandidateFinder{byID: map[string]*models.Movie{uuidSeven: &m7, uuidEight: &noFile}}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, finder, 5)

	// Unknown id (e.g. a series id)
	_, _, err := p.Start(context.Background(), "selected", []string{uuidSeven, "9ff0c000-dead-4bee-8f00-000000000999"}, 0)
	assert.ErrorIs(t, err, ErrGenerationSelectionInvalid)
	assert.False(t, p.IsRunning())

	// Movie without a media file
	_, _, err = p.Start(context.Background(), "selected", []string{uuidEight}, 0)
	assert.ErrorIs(t, err, ErrGenerationSelectionInvalid)
	assert.False(t, p.IsRunning())
}

// AC 3: preview returns the count without starting anything. sub-5-1 AC #7:
// a nil episode finder degrades the including-episodes count to movies-only.
func TestGenerationBatch_PreviewMissing(t *testing.T) {
	finder := &fakeCandidateFinder{count: 38}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, finder, 5)

	movies, includingEpisodes, err := p.PreviewMissing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 38, movies)
	assert.Equal(t, 38, includingEpisodes, "nil episode finder → movies-only degrade")
	assert.False(t, p.IsRunning())
}

// sub-5-1 AC #7: the preview returns BOTH numbers — total_items stays
// movies-only (frozen: it is what scope=missing runs), the additive count
// adds the episode twin for the F17 toast.
func TestGenerationBatch_PreviewMissing_IncludesEpisodes(t *testing.T) {
	finder := &fakeCandidateFinder{count: 38}
	episodes := &fakeEpisodeFinder{count: 104}
	p, _ := newTestGenerationProcessorWithEpisodes(t, &fakeGenerationRunner{available: true}, finder, episodes, 5)

	movies, includingEpisodes, err := p.PreviewMissing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 38, movies, "total_items semantics frozen — movies only")
	assert.Equal(t, 142, includingEpisodes)
}

// CR M2: an episode-count failure DEGRADES to the movies-only count instead
// of 500ing the whole preview — total_items is a frozen pre-existing key and
// its availability must not start depending on the episodes table. The
// degraded number equals the pre-sub-5-1 toast behavior (undercount,
// direction-safe).
func TestGenerationBatch_PreviewMissing_EpisodeCountErrorDegradesToMovies(t *testing.T) {
	finder := &fakeCandidateFinder{count: 38}
	episodes := &fakeEpisodeFinder{countErr: errors.New("db locked")}
	p, _ := newTestGenerationProcessorWithEpisodes(t, &fakeGenerationRunner{available: true}, finder, episodes, 5)

	movies, includingEpisodes, err := p.PreviewMissing(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 38, movies)
	assert.Equal(t, 38, includingEpisodes)
}

// The movie half failing still fails the preview — it IS total_items.
func TestGenerationBatch_PreviewMissing_MovieCountErrorStillFails(t *testing.T) {
	finder := &fakeCandidateFinder{findErr: errors.New("db locked")}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, finder, 5)

	_, _, err := p.PreviewMissing(context.Background())
	require.Error(t, err)
}

// Enumeration failure surfaces as a start error (500 at the handler).
func TestGenerationBatch_EnumerationError(t *testing.T) {
	finder := &fakeCandidateFinder{findErr: errors.New("db locked")}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrGenerationBatchRunning)
	assert.False(t, p.IsRunning())
}

// 9R-18: UUID row ids are enumerated as-is (the old ParseInt silently dropped
// them); only rows without a media file are skipped fail-soft in missing scope.
func TestGenerationBatch_MissingScope_UUIDIDsEnumerated(t *testing.T) {
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/w.mkv"),
		{ID: uuidC, Title: "NoFile"},
	}}
	p, _ := newTestGenerationProcessor(t, runner, finder, 5)

	_, items, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	require.Len(t, items, 2, "every UUID-keyed movie with a file enumerates; only the file-less row is skipped")
	assert.Equal(t, uuidA, items[0].MediaID)
	assert.Equal(t, uuidB, items[1].MediaID)
	waitUntilIdle(t, p)
}

// ─── sub-4-2: mixed movie+episode selection (AC #2, D1 ruling) ──────────────

// D1 acceptance: a mixed movie+episode batch enumerates both, preserves the
// caller's order, carries media_type per item, and hands each item's type to
// the runner (which routes the writeback to the correct table).
func TestGenerationBatch_SelectedScope_MixedMovieEpisode(t *testing.T) {
	m7 := genMovie(uuidSeven, "Seven", "/m/7.mkv")
	ep := genEpisode(uuidEight, 4, 7, "The Massacre", "/tv/s04e07.mkv")
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{byID: map[string]*models.Movie{uuidSeven: &m7}}
	episodes := &fakeEpisodeFinder{byID: map[string]*models.Episode{uuidEight: &ep}}
	p, _ := newTestGenerationProcessorWithEpisodes(t, runner, finder, episodes, 5)

	_, items, err := p.Start(context.Background(), "selected", []string{uuidEight, uuidSeven}, 0)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, uuidEight, items[0].MediaID)
	assert.Equal(t, models.SubtitleRunMediaEpisode, items[0].MediaType)
	assert.Equal(t, "S04E07 The Massacre", items[0].Title)
	assert.Equal(t, uuidSeven, items[1].MediaID)
	assert.Equal(t, models.SubtitleRunMediaMovie, items[1].MediaType)

	waitUntilIdle(t, p)
	assert.Equal(t, []string{uuidEight, uuidSeven}, runner.callIDs())
	assert.Equal(t, []string{models.SubtitleRunMediaEpisode, models.SubtitleRunMediaMovie}, runner.callMediaTypes(),
		"the runner must receive each item's media type — a movie default would write episodes onto the movies table")
}

// AC #2: an id unknown in BOTH tables still rejects the whole batch — the
// consented list is the amount confirmed on F16; no silent filtering.
func TestGenerationBatch_SelectedScope_UnknownInBothTablesRejected(t *testing.T) {
	m7 := genMovie(uuidSeven, "Seven", "/m/7.mkv")
	ep := genEpisode(uuidEight, 1, 1, "Pilot", "/tv/s01e01.mkv")
	finder := &fakeCandidateFinder{byID: map[string]*models.Movie{uuidSeven: &m7}}
	episodes := &fakeEpisodeFinder{byID: map[string]*models.Episode{uuidEight: &ep}}
	p, _ := newTestGenerationProcessorWithEpisodes(t, &fakeGenerationRunner{available: true}, finder, episodes, 5)

	_, _, err := p.Start(context.Background(), "selected",
		[]string{uuidSeven, "9ff0c000-dead-4bee-8f00-000000000999", uuidEight}, 0)
	assert.ErrorIs(t, err, ErrGenerationSelectionInvalid)
	assert.False(t, p.IsRunning(), "a rejected selection must not start a batch")
}

// AC #2: a found episode without a media file is a hard selection error,
// same rule as movies.
func TestGenerationBatch_SelectedScope_EpisodeWithoutFileRejected(t *testing.T) {
	noFile := models.Episode{ID: uuidNine, SeriesID: "series-1", SeasonNumber: 2, EpisodeNumber: 3}
	episodes := &fakeEpisodeFinder{byID: map[string]*models.Episode{uuidNine: &noFile}}
	p, _ := newTestGenerationProcessorWithEpisodes(t, &fakeGenerationRunner{available: true}, &fakeCandidateFinder{}, episodes, 5)

	_, _, err := p.Start(context.Background(), "selected", []string{uuidNine}, 0)
	assert.ErrorIs(t, err, ErrGenerationSelectionInvalid)
	assert.False(t, p.IsRunning())
}

// A nil episode finder degrades to the pre-sub-4-2 movies-only behavior:
// an episode id rejects like any unknown id (defensive wiring, movie-only tests).
func TestGenerationBatch_NilEpisodeFinder_MoviesOnly(t *testing.T) {
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, &fakeCandidateFinder{}, 5)

	_, _, err := p.Start(context.Background(), "selected", []string{uuidEight}, 0)
	assert.ErrorIs(t, err, ErrGenerationSelectionInvalid)
}

// Missing scope stays movies-only and stamps media_type=movie on every item
// (AC #3 — the frozen preview count must keep matching the batch size).
func TestGenerationBatch_MissingScope_MovieMediaType(t *testing.T) {
	runner := &fakeGenerationRunner{available: true}
	finder := &fakeCandidateFinder{movies: []models.Movie{genMovie(uuidA, "A", "/m/a.mkv")}}
	episodes := &fakeEpisodeFinder{byID: map[string]*models.Episode{}}
	p, _ := newTestGenerationProcessorWithEpisodes(t, runner, finder, episodes, 5)

	_, items, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, models.SubtitleRunMediaMovie, items[0].MediaType)
	waitUntilIdle(t, p)
	assert.Equal(t, []string{models.SubtitleRunMediaMovie}, runner.callMediaTypes())
}

// ─── sub-4-2: user-approved budget ceiling (AC #1, #6) ──────────────────────

// AC #1: a provided ceiling overrides the configured default for THIS batch —
// observable on GetProgress().BudgetUSD (fed to the F8 cost line) and on the
// SSE budget_usd key (both read budget.Snapshot(), so the override reaching
// them proves ai.NewBudget was built with the request value).
func TestGenerationBatch_RequestedBudgetOverridesDefault(t *testing.T) {
	started := make(chan struct{})
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(ctx context.Context, _ string) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{genMovie(uuidA, "A", "/m/a.mkv")}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 2.5)
	require.NoError(t, err)
	<-started

	prog := p.GetProgress()
	require.NotNil(t, prog)
	assert.Equal(t, 2.5, prog.BudgetUSD, "the user-approved ceiling, not AI_RUN_BUDGET_USD, governs this batch")

	p.Cancel()
	waitUntilIdle(t, p)
	events := eventsUntilTerminal(t, client)
	require.NotEmpty(t, events)
	assert.Equal(t, 2.5, events[len(events)-1]["budget_usd"])
}

// AC #1: the user-approved ceiling is ENFORCED, not just displayed — spend
// beyond it pauses the remainder even though the default would have allowed it.
func TestGenerationBatch_RequestedBudgetEnforced(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		onCall: func(ctx context.Context, mediaID string) error {
			if mediaID == uuidA {
				// Spend ~$3: over the $1 request ceiling, under the $5 default.
				ai.BudgetFromContext(ctx).RecordLLM("claude-sonnet-5", 1_000_000, 0)
			}
			return nil
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 1.0)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []string{uuidA}, runner.callIDs(), "the request ceiling must gate the queue")
	events := eventsUntilTerminal(t, client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusBudgetCeiling, last["status"])
	assert.Equal(t, 1, last["paused_count"])
}

// ─── sub-4-2 CR fixes ───────────────────────────────────────────────────────

// CR M2: a REAL lookup failure (locked DB, cancelled ctx) must NOT be
// classified as "your selection is invalid" — it propagates so the handler
// answers 500, and the FE knows to retry instead of blaming the selection.
func TestGenerationBatch_SelectedScope_DBErrorIsNotSelectionInvalid(t *testing.T) {
	finder := &fakeCandidateFinder{byIDErr: errors.New("database is locked")}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, finder, 5)

	_, _, err := p.Start(context.Background(), "selected", []string{uuidSeven}, 0)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrGenerationSelectionInvalid,
		"a transient DB error must not read as an invalid selection (400)")
	assert.False(t, p.IsRunning())

	// Same classification on the episode leg.
	episodes := &fakeEpisodeFinder{byIDErr: errors.New("database is locked")}
	p2, _ := newTestGenerationProcessorWithEpisodes(t, &fakeGenerationRunner{available: true}, &fakeCandidateFinder{}, episodes, 5)
	_, _, err = p2.Start(context.Background(), "selected", []string{uuidEight}, 0)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrGenerationSelectionInvalid)
}

// CR H1: an item the pipeline routed out (skipped run — e.g. no_text_source
// with no live ASR) counts as a FAILURE, never a success; the loop continues.
func TestGenerationBatch_SkippedItemCountsAsFail(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		errs: map[string]error{
			uuidB: fmt.Errorf("media %s: no text source: %w", uuidB, ErrGenerationItemSkipped),
		},
	}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
		genMovie(uuidC, "C", "/m/c.mkv"),
	}}
	p, client := newTestGenerationProcessor(t, runner, finder, 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0)
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []string{uuidA, uuidB, uuidC}, runner.callIDs(), "a skipped item must not stop the batch")

	events := eventsUntilTerminal(t, client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusComplete, last["status"])
	assert.Equal(t, 2, last["success_count"], "success means a subtitle exists — skips are not successes")
	assert.Equal(t, 1, last["fail_count"])
}
