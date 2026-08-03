package subtitle

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle/providers"
)

// ─── D5 flag seam (AC #1) ──────────────────────────────────────────────────

// engineCall records every argument the batch loop hands the search engine, so
// "legacy is byte-identical" is a comparison against captured values rather
// than an eyeball over a diff.
type engineCall struct {
	mediaID    string
	mediaType  string
	filePath   string
	query      providers.SubtitleQuery
	resolution string
	opts       []ProcessOptions
}

// spyEngine is the legacy processor, instrumented. It is the whole reason
// batchEngine exists as a port: without it the D5 conditional could only be
// verified by reading the code.
type spyEngine struct {
	mu     sync.Mutex
	calls  []engineCall
	result EngineResult
}

func (e *spyEngine) Process(_ context.Context, mediaID, mediaType, mediaFilePath string,
	query providers.SubtitleQuery, mediaResolution string, opts ...ProcessOptions) EngineResult {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls = append(e.calls, engineCall{
		mediaID: mediaID, mediaType: mediaType, filePath: mediaFilePath,
		query: query, resolution: mediaResolution, opts: opts,
	})
	return e.result
}

func (e *spyEngine) snapshot() []engineCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]engineCall(nil), e.calls...)
}

// spyItemProcessor stands in for the generation pipeline at the seam.
type spyItemProcessor struct {
	mu    sync.Mutex
	refs  []MediaRef
	opts  []ProcessItemOptions
	err   error
	outFn func(MediaRef) *ProcessOutcome
}

func (p *spyItemProcessor) ProcessItem(_ context.Context, ref MediaRef, opts ProcessItemOptions) (*ProcessOutcome, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refs = append(p.refs, ref)
	p.opts = append(p.opts, opts)
	if p.err != nil {
		return nil, p.err
	}
	if p.outFn != nil {
		return p.outFn(ref), nil
	}
	return &ProcessOutcome{Kind: RouteTranslate, SubtitlePath: "/media/" + ref.ID + ".zh-Hant.srt"}, nil
}

func (p *spyItemProcessor) snapshot() ([]MediaRef, []ProcessItemOptions) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]MediaRef(nil), p.refs...), append([]ProcessItemOptions(nil), p.opts...)
}

func seamItems() []BatchItem {
	return []BatchItem{
		{MediaID: "movie-1", MediaType: "movie", MediaFilePath: "/media/A.mkv",
			Title: "A", Resolution: "1080p", ProductionCountry: "US"},
		{MediaID: "movie-2", MediaType: "movie", MediaFilePath: "/media/B.mkv",
			Title: "B", Resolution: "2160p", ProductionCountry: "CN"},
	}
}

func newSeamProcessor(t *testing.T, engine batchEngine, items []BatchItem, opts ...BatchProcessorOption) *BatchProcessor {
	t.Helper()
	hub := sse.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Close() })
	return NewBatchProcessor(engine, hub, &mockCollector{movies: items},
		BatchConfig{DelayBetweenItems: time.Millisecond}, opts...)
}

// runSeamBatch runs one batch to completion and returns the terminal
// success/fail counts, read from the `complete` broadcast — the batch clears
// its progress struct when it finishes, so the SSE event is the only place the
// final tally survives.
func runSeamBatch(t *testing.T, bp *BatchProcessor) (success, fail int) {
	t.Helper()
	client := bp.sseHub.Register()
	t.Cleanup(func() { bp.sseHub.Unregister(client) })
	time.Sleep(20 * time.Millisecond) // let the hub finish registering

	_, _, err := bp.Start(context.Background(), BatchRequest{Scope: ScopeLibrary})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return !bp.IsRunning() }, 3*time.Second, 5*time.Millisecond,
		"the batch must finish")

	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-client.Events:
			data, ok := ev.Data.(map[string]interface{})
			if !ok || data["status"] != "complete" {
				continue
			}
			return data["success_count"].(int), data["fail_count"].(int)
		case <-deadline:
			t.Fatal("no terminal batch-complete event arrived")
			return 0, 0
		}
	}
}

// TestBatchSeam_LegacyModeIsByteIdentical is the D5 safety proof: with no item
// processor wired the loop must call the search engine with exactly the
// arguments it always did, and must never touch the generation pipeline.
func TestBatchSeam_LegacyModeIsByteIdentical(t *testing.T) {
	engine := &spyEngine{result: EngineResult{Success: true, SubtitlePath: "/media/A.zh-Hant.srt"}}
	pipeline := &spyItemProcessor{}

	bp := newSeamProcessor(t, engine, seamItems())
	success, fail := runSeamBatch(t, bp)
	assert.Equal(t, 2, success)
	assert.Zero(t, fail)

	calls := engine.snapshot()
	require.Len(t, calls, 2, "every item still goes through the search engine in legacy mode")

	assert.Equal(t, "movie-1", calls[0].mediaID)
	assert.Equal(t, "movie", calls[0].mediaType)
	assert.Equal(t, "/media/A.mkv", calls[0].filePath)
	assert.Equal(t, providers.SubtitleQuery{Title: "A"}, calls[0].query,
		"the query is still built from the item title")
	assert.Equal(t, "1080p", calls[0].resolution)
	require.Len(t, calls[0].opts, 1)
	assert.Equal(t, "US", calls[0].opts[0].ProductionCountry,
		"the CN conversion policy still rides ProcessOptions")
	assert.Equal(t, "CN", calls[1].opts[0].ProductionCountry)

	refs, _ := pipeline.snapshot()
	assert.Empty(t, refs, "legacy mode must not reach the generation pipeline at all")
}

// TestBatchSeam_PipelineModeRoutesToProcessItem — the other half of the one
// conditional. The search engine must be bypassed entirely, not called first.
func TestBatchSeam_PipelineModeRoutesToProcessItem(t *testing.T) {
	engine := &spyEngine{result: EngineResult{Success: true}}
	pipeline := &spyItemProcessor{}

	bp := newSeamProcessor(t, engine, seamItems(), WithItemProcessor(pipeline))
	success, fail := runSeamBatch(t, bp)
	assert.Equal(t, 2, success)
	assert.Zero(t, fail)

	refs, opts := pipeline.snapshot()
	require.Len(t, refs, 2)
	assert.Equal(t, MediaRef{ID: "movie-1", MediaType: "movie"}, refs[0])
	assert.Equal(t, MediaRef{ID: "movie-2", MediaType: "movie"}, refs[1])
	assert.Equal(t, ProcessItemOptions{}, opts[0],
		"the batch seam never forces — force is the operator's lever on the endpoint")

	assert.Empty(t, engine.snapshot(),
		"pipeline mode must BYPASS the search engine, not run both")
}

// TestBatchSeam_PipelineFailureCountsAsAFailedItem — the batch's success/fail
// bookkeeping must read the same either side of the seam, or the progress UI
// would report a clean run over a pipeline that failed every item.
func TestBatchSeam_PipelineFailureCountsAsAFailedItem(t *testing.T) {
	pipeline := &spyItemProcessor{err: errors.New("provider 500")}

	bp := newSeamProcessor(t, &spyEngine{}, seamItems(), WithItemProcessor(pipeline))
	success, fail := runSeamBatch(t, bp)

	assert.Zero(t, success)
	assert.Equal(t, 2, fail, "a pipeline error is a failed item, exactly like an engine error")
}

// TestBatchSeam_PreflightSkipIsNotAFailure — ProcessItem returns Run == nil when
// the P5 pre-flight found an acceptable sidecar (sub-1-5b AC #2). That is the
// desired outcome of a re-scan, not a failure.
func TestBatchSeam_PreflightSkipIsNotAFailure(t *testing.T) {
	pipeline := &spyItemProcessor{
		outFn: func(ref MediaRef) *ProcessOutcome {
			return &ProcessOutcome{SubtitlePath: "/media/" + ref.ID + ".zh-Hant.srt"} // Run == nil
		},
	}

	bp := newSeamProcessor(t, &spyEngine{}, seamItems(), WithItemProcessor(pipeline))
	success, fail := runSeamBatch(t, bp)

	assert.Equal(t, 2, success, "an item that already has its sidecar is a success, not a failure")
	assert.Zero(t, fail)
}
