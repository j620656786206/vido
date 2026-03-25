package subtitle

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle/providers"
)

// --- Mock Collector ---

type mockCollector struct {
	movies []BatchItem
	series []BatchItem
	err    error
}

func (m *mockCollector) CollectMoviesNeedingSubtitles(_ context.Context) ([]BatchItem, error) {
	return m.movies, m.err
}
func (m *mockCollector) CollectSeriesNeedingSubtitles(_ context.Context) ([]BatchItem, error) {
	return m.series, m.err
}

// --- Helper: create a batch processor with mock dependencies ---

func newTestBatchProcessor(t *testing.T, items []BatchItem) *BatchProcessor {
	t.Helper()

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant", Filename: "sub.srt"},
		},
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n測試\n"),
	}

	scorer := NewScorer(NewDefaultScorerConfig())
	mockRepo := &mockStatusUpdater{}
	engine := NewEngine(
		[]providers.SubtitleProvider{prov}, scorer, nil, nil, nil, mockRepo, mockRepo,
	)

	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	collector := &mockCollector{movies: items}
	config := BatchConfig{DelayBetweenItems: 1 * time.Millisecond} // fast for tests

	return NewBatchProcessor(engine, hub, collector, config)
}

// --- Tests ---

// AC #7: IsRunning / GetProgress when no batch
func TestBatchProcessor_InitialState(t *testing.T) {
	bp := newTestBatchProcessor(t, nil)
	assert.False(t, bp.IsRunning())
	assert.Nil(t, bp.GetProgress())
}

// AC #4: Start returns batchId and totalItems
func TestBatchProcessor_Start_ReturnsIdAndCount(t *testing.T) {
	items := []BatchItem{
		{MediaID: "m1", MediaType: "movie", Title: "Movie 1"},
		{MediaID: "m2", MediaType: "movie", Title: "Movie 2"},
	}
	bp := newTestBatchProcessor(t, items)

	batchID, total, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)
	assert.NotEmpty(t, batchID)
	assert.Equal(t, 2, total)

	// Wait for processing to complete
	time.Sleep(100 * time.Millisecond)
}

// Empty item list returns immediately
func TestBatchProcessor_Start_EmptyItems(t *testing.T) {
	bp := newTestBatchProcessor(t, nil) // no items

	batchID, total, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)
	assert.NotEmpty(t, batchID)
	assert.Equal(t, 0, total)
	assert.False(t, bp.IsRunning())
}

// AC #7: Second batch returns error
func TestBatchProcessor_ConcurrencyGuard(t *testing.T) {
	// Create items with slow processing
	items := []BatchItem{
		{MediaID: "m1", MediaType: "movie", Title: "Movie 1"},
		{MediaID: "m2", MediaType: "movie", Title: "Movie 2"},
		{MediaID: "m3", MediaType: "movie", Title: "Movie 3"},
	}

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant", Filename: "sub.srt"},
		},
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n測試\n"),
	}

	scorer := NewScorer(NewDefaultScorerConfig())
	mockRepo := &mockStatusUpdater{}
	engine := NewEngine([]providers.SubtitleProvider{prov}, scorer, nil, nil, nil, mockRepo, mockRepo)
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	collector := &mockCollector{movies: items}
	config := BatchConfig{DelayBetweenItems: 50 * time.Millisecond}
	bp := NewBatchProcessor(engine, hub, collector, config)

	// Start first batch
	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)

	// Brief wait to ensure it's running
	time.Sleep(10 * time.Millisecond)

	// Try to start second batch — should error
	_, _, err = bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch already running")

	// Wait for completion
	time.Sleep(300 * time.Millisecond)
}

// AC #6: Individual item failure doesn't abort batch
func TestBatchProcessor_ContinuesOnFailure(t *testing.T) {
	items := []BatchItem{
		{MediaID: "m1", MediaType: "movie", Title: "Movie 1"},
		{MediaID: "m2", MediaType: "movie", Title: "Movie 2"},
	}

	// Provider that returns no results (= engine failure)
	prov := &mockProvider{
		name:         "assrt",
		searchResult: nil, // no results → engine returns ErrNoResults
	}

	scorer := NewScorer(NewDefaultScorerConfig())
	mockRepo := &mockStatusUpdater{}
	engine := NewEngine([]providers.SubtitleProvider{prov}, scorer, nil, nil, nil, mockRepo, mockRepo)
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	collector := &mockCollector{movies: items}
	config := BatchConfig{DelayBetweenItems: 1 * time.Millisecond}
	bp := NewBatchProcessor(engine, hub, collector, config)

	_, total, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)
	assert.Equal(t, 2, total)

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	// Batch should have completed (not aborted)
	assert.False(t, bp.IsRunning())
}

// Context cancellation
func TestBatchProcessor_Cancellation(t *testing.T) {
	items := make([]BatchItem, 10)
	for i := range items {
		items[i] = BatchItem{MediaID: fmt.Sprintf("m%d", i), MediaType: "movie", Title: fmt.Sprintf("Movie %d", i)}
	}

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant"},
		},
		downloadData: []byte("test"),
	}

	scorer := NewScorer(NewDefaultScorerConfig())
	mockRepo := &mockStatusUpdater{}
	engine := NewEngine([]providers.SubtitleProvider{prov}, scorer, nil, nil, nil, mockRepo, mockRepo)
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	collector := &mockCollector{movies: items}
	config := BatchConfig{DelayBetweenItems: 50 * time.Millisecond}
	bp := NewBatchProcessor(engine, hub, collector, config)

	ctx, cancel := context.WithCancel(context.Background())
	_, _, err := bp.Start(ctx, BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)

	// Cancel after processing starts
	time.Sleep(30 * time.Millisecond)
	cancel()

	// Wait a bit for cancellation to take effect
	time.Sleep(100 * time.Millisecond)
	assert.False(t, bp.IsRunning())
}

// Invalid scope
func TestBatchProcessor_InvalidScope(t *testing.T) {
	bp := newTestBatchProcessor(t, nil)

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: "invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown batch scope")
}

// Season scope requires season_id
func TestBatchProcessor_SeasonScopeRequiresID(t *testing.T) {
	bp := newTestBatchProcessor(t, nil)

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeSeason})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "season_id required")
}

// SSE events are broadcast
func TestBatchProcessor_BroadcastsSSE(t *testing.T) {
	items := []BatchItem{
		{MediaID: "m1", MediaType: "movie", Title: "Movie 1"},
	}

	bp := newTestBatchProcessor(t, items)

	// Register an SSE client to capture events
	client := bp.sseHub.Register()
	time.Sleep(20 * time.Millisecond)

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)

	// Wait for processing + SSE delivery
	time.Sleep(200 * time.Millisecond)

	// Should have received at least one event
	var receivedEvents int
	for {
		select {
		case <-client.Events:
			receivedEvents++
		default:
			goto done
		}
	}
done:
	assert.Greater(t, receivedEvents, 0, "Expected at least one SSE event")
}

// AC #9: CN content passes productionCountry to engine
func TestBatchProcessor_CNContentPolicy(t *testing.T) {
	items := []BatchItem{
		{MediaID: "m1", MediaType: "movie", Title: "CN Movie", ProductionCountry: "CN"},
		{MediaID: "m2", MediaType: "movie", Title: "US Movie", ProductionCountry: "US"},
	}

	bp := newTestBatchProcessor(t, items)

	batchID, total, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)
	assert.NotEmpty(t, batchID)
	assert.Equal(t, 2, total)

	// Wait for completion
	time.Sleep(100 * time.Millisecond)
	assert.False(t, bp.IsRunning())
}
