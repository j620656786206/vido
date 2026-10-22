package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
)

// disc-2026-09-batch-reparse-never-runs: a batch re-parse sets rows pending
// and must then get them MATCHED — now, or right after the running pass.

// gatedMovieRepo counts pass starts (each pass queries pending rows first)
// and can hold the first query open so a test can act mid-pass.
type gatedMovieRepo struct {
	mockMovieRepoForNFO
	mu      sync.Mutex
	queries int
	gate    chan struct{} // nil = never block; else the FIRST query waits on it
	gated   bool
	entered chan struct{}
}

func (m *gatedMovieRepo) FindByParseStatus(_ context.Context, status models.ParseStatus) ([]models.Movie, error) {
	m.mu.Lock()
	m.queries++
	first := status == models.ParseStatusPending && !m.gated && m.gate != nil
	if first {
		m.gated = true
	}
	m.mu.Unlock()
	if first {
		close(m.entered)
		<-m.gate
	}
	return nil, nil
}

// passes: each pass asks for pending then "" rows — two queries per pass.
func (m *gatedMovieRepo) passes() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.queries / 2
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestEnrichmentService_RequestRun_IdleStartsAPassNow(t *testing.T) {
	repo := &gatedMovieRepo{}
	svc := NewEnrichmentService(repo, nil, nil, nil, nil, nil, nil, nil)

	assert.True(t, svc.RequestRun(), "nothing running → a pass starts right away")

	waitFor(t, "the pass to finish", func() bool { return !svc.IsEnrichmentActive() && repo.passes() == 1 })
	assert.Equal(t, 1, repo.passes())
}

// The bug the story is about, in its worst shape: the user re-parses while a
// post-scan pass is already running. That pass loaded its rows before the
// re-parse, so it never sees them. Before the fix the rows stayed 整理中 until
// the next scan that happened to find a changed file.
func TestEnrichmentService_RequestRun_WhileRunningRunsOneMorePass(t *testing.T) {
	repo := &gatedMovieRepo{gate: make(chan struct{}), entered: make(chan struct{})}
	svc := NewEnrichmentService(repo, nil, nil, nil, nil, nil, nil, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := svc.StartEnrichment(context.Background())
		assert.NoError(t, err)
	}()
	<-repo.entered // pass 1 is mid-flight, rows loaded

	assert.False(t, svc.RequestRun(), "a pass is running → queued, not started")
	assert.False(t, svc.RequestRun(), "asking twice queues ONE extra pass, not two")

	close(repo.gate)
	<-done
	assert.False(t, svc.IsEnrichmentActive())
	assert.Equal(t, 2, repo.passes(), "exactly one extra pass after the running one")
}

// Cancel means stop — a queued pass must not resurrect the work.
func TestEnrichmentService_RequestRun_CancelDropsTheQueuedPass(t *testing.T) {
	repo := &gatedMovieRepo{gate: make(chan struct{}), entered: make(chan struct{})}
	svc := NewEnrichmentService(repo, nil, nil, nil, nil, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = svc.StartEnrichment(ctx)
	}()
	<-repo.entered

	assert.False(t, svc.RequestRun())
	cancel()
	close(repo.gate)
	<-done

	assert.False(t, svc.IsEnrichmentActive())
	assert.Equal(t, 1, repo.passes(), "the cancelled pass is the only pass")
}

func TestEnrichmentService_StartEnrichment_StillRefusesWhileRunning(t *testing.T) {
	repo := &gatedMovieRepo{gate: make(chan struct{}), entered: make(chan struct{})}
	svc := NewEnrichmentService(repo, nil, nil, nil, nil, nil, nil, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = svc.StartEnrichment(context.Background())
	}()
	<-repo.entered

	_, err := svc.StartEnrichment(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ENRICHMENT_ALREADY_RUNNING")

	close(repo.gate)
	<-done
}

// completeCounter counts enrich_complete events through the hub's observer —
// what the re-parse dialog listens to.
func completeCounter(hub *sse.Hub) func() int {
	var mu sync.Mutex
	n := 0
	hub.AddObserver(func(e sse.Event) {
		if e.Type == sse.EventEnrichComplete {
			mu.Lock()
			n++
			mu.Unlock()
		}
	})
	return func() int {
		mu.Lock()
		defer mu.Unlock()
		return n
	}
}

// disc-2026-10-reparse-dialog-empty-followup-pass: a mixed movie+series
// re-parse sends two requests; the second queues a follow-up pass that finds
// nothing (the first pass already took those rows). Its {total:0} completion
// overwrote the real result with 「成功 0・失敗 0」 — an empty follow-up now
// stays silent.
func TestEnrichmentService_EmptyFollowUpPassIsNotAnnounced(t *testing.T) {
	repo := &gatedMovieRepo{gate: make(chan struct{}), entered: make(chan struct{})}
	hub := sse.NewHub()
	completes := completeCounter(hub)
	svc := NewEnrichmentService(repo, nil, nil, nil, nil, nil, hub, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := svc.StartEnrichment(context.Background())
		assert.NoError(t, err)
	}()
	<-repo.entered
	assert.False(t, svc.RequestRun(), "queued behind the running pass")
	close(repo.gate)
	<-done

	assert.Equal(t, 2, repo.passes(), "the follow-up still runs")
	assert.Equal(t, 1, completes(), "only the first pass reports; the empty follow-up does not")
}

// An empty pass that is NOT a follow-up still reports: a re-parse whose rows
// were already handled must still end its dialog.
func TestEnrichmentService_EmptyFirstPassStillAnnounced(t *testing.T) {
	repo := &gatedMovieRepo{}
	hub := sse.NewHub()
	completes := completeCounter(hub)
	svc := NewEnrichmentService(repo, nil, nil, nil, nil, nil, hub, nil)

	require.True(t, svc.RequestRun())
	waitFor(t, "the pass to finish", func() bool { return !svc.IsEnrichmentActive() && repo.passes() == 1 })
	assert.Equal(t, 1, completes())
}
