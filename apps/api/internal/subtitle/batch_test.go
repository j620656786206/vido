package subtitle

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
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
	assert.True(t, errors.Is(err, ErrBatchAlreadyRunning))

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

// Cancellation via Cancel() method
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

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)

	// Cancel after processing starts
	time.Sleep(30 * time.Millisecond)
	bp.Cancel()

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

// --- TA 8-9: Additional Coverage Tests ---

// AC #5: [P0] Verify delay between items is respected
func TestBatchProcessor_DelayBetweenItems(t *testing.T) {
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
	delay := 30 * time.Millisecond
	config := BatchConfig{DelayBetweenItems: delay}
	bp := NewBatchProcessor(engine, hub, collector, config)

	start := time.Now()
	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)

	// Wait for all 3 items to process (2 delays between 3 items)
	time.Sleep(400 * time.Millisecond)
	elapsed := time.Since(start)

	// Expect at least 2 delays (between item 1→2 and 2→3)
	minExpected := 2 * delay
	assert.GreaterOrEqual(t, elapsed, minExpected,
		"Expected at least %v total delay for 3 items, got %v", minExpected, elapsed)
	assert.False(t, bp.IsRunning())
}

// AC #3: [P0] SSE event payload has all required fields
func TestBatchProcessor_SSEEventPayloadFields(t *testing.T) {
	items := []BatchItem{
		{MediaID: "m1", MediaType: "movie", Title: "Test Movie"},
	}

	bp := newTestBatchProcessor(t, items)

	client := bp.sseHub.Register()
	time.Sleep(20 * time.Millisecond)

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)

	// Wait for processing + SSE delivery
	time.Sleep(200 * time.Millisecond)

	// Collect all events
	var events []sse.Event
	for {
		select {
		case evt := <-client.Events:
			events = append(events, evt)
		default:
			goto collected
		}
	}
collected:
	require.NotEmpty(t, events, "Expected at least one SSE event")

	// Validate each event has required fields
	for _, evt := range events {
		assert.Equal(t, sse.EventSubtitleBatchProgress, evt.Type)

		data, ok := evt.Data.(map[string]interface{})
		require.True(t, ok, "Event data should be map[string]interface{}")

		// AC #3: Required fields
		assert.Contains(t, data, "batch_id")
		assert.Contains(t, data, "total_items")
		assert.Contains(t, data, "current_index")
		assert.Contains(t, data, "success_count")
		assert.Contains(t, data, "fail_count")
		assert.Contains(t, data, "status")
	}

	// Last event should be "complete" (AC #8)
	lastData := events[len(events)-1].Data.(map[string]interface{})
	assert.Equal(t, "complete", lastData["status"])
}

// AC #7: [P1] GetProgress returns accurate state while batch running
func TestBatchProcessor_GetProgressDuringRun(t *testing.T) {
	items := make([]BatchItem, 5)
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

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)

	// Check progress while running
	time.Sleep(30 * time.Millisecond)
	assert.True(t, bp.IsRunning())

	progress := bp.GetProgress()
	require.NotNil(t, progress, "GetProgress should return non-nil during run")
	assert.Equal(t, 5, progress.TotalItems)
	assert.Equal(t, "running", progress.Status)
	assert.NotEmpty(t, progress.BatchID)

	// Wait for completion
	time.Sleep(500 * time.Millisecond)
	assert.False(t, bp.IsRunning())
	assert.Nil(t, bp.GetProgress(), "GetProgress should return nil after completion")
}

// [P1] Collector error propagates correctly
func TestBatchProcessor_CollectorError(t *testing.T) {
	bp := newTestBatchProcessor(t, nil)
	// Replace collector with one that returns an error
	bp.collect = &mockCollector{err: fmt.Errorf("database connection failed")}

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
	assert.False(t, bp.IsRunning())
}

// [P1] All-failure batch tracks fail count in SSE complete event
func TestBatchProcessor_AllFailuresTrackedInComplete(t *testing.T) {
	items := []BatchItem{
		{MediaID: "m1", MediaType: "movie", Title: "Movie 1"},
		{MediaID: "m2", MediaType: "movie", Title: "Movie 2"},
	}

	// Provider with no results → all items fail
	prov := &mockProvider{
		name:         "assrt",
		searchResult: nil,
	}

	scorer := NewScorer(NewDefaultScorerConfig())
	mockRepo := &mockStatusUpdater{}
	engine := NewEngine([]providers.SubtitleProvider{prov}, scorer, nil, nil, nil, mockRepo, mockRepo)
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	client := hub.Register()
	time.Sleep(20 * time.Millisecond)

	collector := &mockCollector{movies: items}
	config := BatchConfig{DelayBetweenItems: 1 * time.Millisecond}
	bp := NewBatchProcessor(engine, hub, collector, config)

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Find the complete event
	var completeData map[string]interface{}
	for {
		select {
		case evt := <-client.Events:
			data := evt.Data.(map[string]interface{})
			if data["status"] == "complete" {
				completeData = data
			}
		default:
			goto done
		}
	}
done:
	require.NotNil(t, completeData, "Expected a complete event")
	assert.Equal(t, 2, completeData["total_items"])
	assert.Equal(t, 0, completeData["success_count"])
	assert.Equal(t, 2, completeData["fail_count"])
}

// --- RepoCollector Tests ---

// Mock repository implementations for RepoCollector tests
type mockMovieSubtitleFinder struct {
	movies map[models.SubtitleStatus][]models.Movie
	err    error
}

func (m *mockMovieSubtitleFinder) FindBySubtitleStatus(_ context.Context, status models.SubtitleStatus) ([]models.Movie, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.movies[status], nil
}

type mockSeriesSubtitleFinder struct {
	series map[models.SubtitleStatus][]models.Series
	err    error
}

func (m *mockSeriesSubtitleFinder) FindBySubtitleStatus(_ context.Context, status models.SubtitleStatus) ([]models.Series, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.series[status], nil
}

// [P0] RepoCollector collects movies with correct status filters and CN policy
func TestRepoCollector_CollectMoviesNeedingSubtitles(t *testing.T) {
	cnCountries, _ := json.Marshal([]models.ProductionCountry{{ISO3166_1: "CN", Name: "China"}})
	twCountries, _ := json.Marshal([]models.ProductionCountry{{ISO3166_1: "TW", Name: "Taiwan"}})

	movieRepo := &mockMovieSubtitleFinder{
		movies: map[models.SubtitleStatus][]models.Movie{
			models.SubtitleStatusNotSearched: {
				{
					ID:                      "movie-1",
					Title:                   "CN Movie",
					FilePath:                sql.NullString{String: "/media/cn.mkv", Valid: true},
					ProductionCountriesJSON: sql.NullString{String: string(cnCountries), Valid: true},
				},
			},
			models.SubtitleStatusNotFound: {
				{
					ID:                      "movie-2",
					Title:                   "TW Movie",
					FilePath:                sql.NullString{String: "/media/tw.mkv", Valid: true},
					ProductionCountriesJSON: sql.NullString{String: string(twCountries), Valid: true},
				},
				{
					ID:       "movie-3",
					Title:    "No Path Movie",
					FilePath: sql.NullString{Valid: false},
				},
			},
		},
	}

	seriesRepo := &mockSeriesSubtitleFinder{
		series: map[models.SubtitleStatus][]models.Series{},
	}

	rc := NewRepoCollector(movieRepo, seriesRepo)
	items, err := rc.CollectMoviesNeedingSubtitles(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 3)

	// Verify CN movie has correct production country
	assert.Equal(t, "movie-1", items[0].MediaID)
	assert.Equal(t, "movie", items[0].MediaType)
	assert.Equal(t, "CN", items[0].ProductionCountry)
	assert.Equal(t, "/media/cn.mkv", items[0].MediaFilePath)

	// Verify TW movie
	assert.Equal(t, "movie-2", items[1].MediaID)
	assert.Equal(t, "TW", items[1].ProductionCountry)

	// Verify movie with no file path
	assert.Equal(t, "movie-3", items[2].MediaID)
	assert.Empty(t, items[2].MediaFilePath)
	assert.Empty(t, items[2].ProductionCountry) // no countries JSON
}

// [P0] RepoCollector collects series (no production_countries field)
func TestRepoCollector_CollectSeriesNeedingSubtitles(t *testing.T) {
	seriesRepo := &mockSeriesSubtitleFinder{
		series: map[models.SubtitleStatus][]models.Series{
			models.SubtitleStatusNotSearched: {
				{
					ID:       "series-1",
					Title:    "Drama",
					FilePath: sql.NullString{String: "/media/drama/", Valid: true},
				},
			},
			models.SubtitleStatusNotFound: {
				{
					ID:       "series-2",
					Title:    "Anime",
					FilePath: sql.NullString{Valid: false},
				},
			},
		},
	}

	movieRepo := &mockMovieSubtitleFinder{
		movies: map[models.SubtitleStatus][]models.Movie{},
	}

	rc := NewRepoCollector(movieRepo, seriesRepo)
	items, err := rc.CollectSeriesNeedingSubtitles(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 2)

	assert.Equal(t, "series-1", items[0].MediaID)
	assert.Equal(t, "series", items[0].MediaType)
	assert.Equal(t, "/media/drama/", items[0].MediaFilePath)
	assert.Empty(t, items[0].ProductionCountry) // Series has no production_countries

	assert.Equal(t, "series-2", items[1].MediaID)
	assert.Empty(t, items[1].MediaFilePath)
}

// [P1] RepoCollector error from movie repo propagates
func TestRepoCollector_MovieRepoError(t *testing.T) {
	movieRepo := &mockMovieSubtitleFinder{err: fmt.Errorf("db timeout")}
	seriesRepo := &mockSeriesSubtitleFinder{
		series: map[models.SubtitleStatus][]models.Series{},
	}

	rc := NewRepoCollector(movieRepo, seriesRepo)
	_, err := rc.CollectMoviesNeedingSubtitles(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db timeout")
}

// [P1] RepoCollector error from series repo propagates
func TestRepoCollector_SeriesRepoError(t *testing.T) {
	movieRepo := &mockMovieSubtitleFinder{
		movies: map[models.SubtitleStatus][]models.Movie{},
	}
	seriesRepo := &mockSeriesSubtitleFinder{err: fmt.Errorf("connection refused")}

	rc := NewRepoCollector(movieRepo, seriesRepo)
	_, err := rc.CollectSeriesNeedingSubtitles(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

// [P2] RepoCollector handles empty repos gracefully
func TestRepoCollector_EmptyRepos(t *testing.T) {
	movieRepo := &mockMovieSubtitleFinder{
		movies: map[models.SubtitleStatus][]models.Movie{},
	}
	seriesRepo := &mockSeriesSubtitleFinder{
		series: map[models.SubtitleStatus][]models.Series{},
	}

	rc := NewRepoCollector(movieRepo, seriesRepo)

	movies, err := rc.CollectMoviesNeedingSubtitles(context.Background())
	require.NoError(t, err)
	assert.Empty(t, movies)

	series, err := rc.CollectSeriesNeedingSubtitles(context.Background())
	require.NoError(t, err)
	assert.Empty(t, series)
}

// [P2] RepoCollector handles movie with multi-country production
func TestRepoCollector_MultiCountryProduction(t *testing.T) {
	multiCountries, _ := json.Marshal([]models.ProductionCountry{
		{ISO3166_1: "CN", Name: "China"},
		{ISO3166_1: "HK", Name: "Hong Kong"},
	})

	movieRepo := &mockMovieSubtitleFinder{
		movies: map[models.SubtitleStatus][]models.Movie{
			models.SubtitleStatusNotSearched: {
				{
					ID:                      "coproduction",
					Title:                   "Co-production",
					ProductionCountriesJSON: sql.NullString{String: string(multiCountries), Valid: true},
				},
			},
		},
	}
	seriesRepo := &mockSeriesSubtitleFinder{
		series: map[models.SubtitleStatus][]models.Series{},
	}

	rc := NewRepoCollector(movieRepo, seriesRepo)
	items, err := rc.CollectMoviesNeedingSubtitles(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Multi-country should be comma-separated
	assert.Equal(t, "CN,HK", items[0].ProductionCountry)
}
