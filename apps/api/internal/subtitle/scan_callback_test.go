package subtitle

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// ─── AC #2: the scan-complete callback composes, it never replaces ─────────
//
// `ScannerService.SetOnScanComplete` holds exactly ONE callback and main.go
// already occupies it with post-scan enrichment (main.go:422). The failure this
// suite exists to prevent is the obvious one: wiring FR13 by CALLING the setter
// a second time, which silently drops enrichment on the floor and would only
// surface weeks later as "my newly-added files never get metadata".

var errEnumerate = errors.New("enumerate failed")

// idleProcessor accepts every item and returns immediately.
func idleProcessor() *blockingProcessor {
	p := newBlockingProcessor(8)
	p.blocking = false
	return p
}

// orderedFinder records the moment enumeration happens, so the test can pin the
// ORDER of the two halves rather than just their occurrence.
type orderedFinder struct {
	onCall func()
	movies []models.Movie
}

func (f *orderedFinder) FindMissingZhHantSubtitle(context.Context) ([]models.Movie, error) {
	f.onCall()
	return f.movies, nil
}

func TestComposeScanCallback_RunsThePreviousCallbackFirst(t *testing.T) {
	// ComposeScanCallback runs its halves inline, so a plain slice is
	// deterministic here — and a slice pins ORDER, which a pair of channel
	// receives would not if both halves simply swapped.
	var order []string

	prev := func() { order = append(order, "enrichment") }
	finder := &orderedFinder{
		onCall: func() { order = append(order, "enumerate") },
		movies: []models.Movie{{ID: "m1"}},
	}

	pool := NewWorkerPool(idleProcessor(), nil, WithCandidateFinders(finder, nil))
	require.NoError(t, pool.Start(context.Background()))
	defer pool.Stop()

	composed := ComposeScanCallback(prev, pool, func(int, error) { order = append(order, "observed") })
	composed()

	assert.Equal(t, []string{"enrichment", "enumerate", "observed"}, order,
		"the pre-existing callback must run FIRST and must always run — the pipeline enqueue is additive")
}

func TestComposeScanCallback_LegacyModeReturnsThePreviousCallbackUntouched(t *testing.T) {
	calls := 0
	prev := func() { calls++ }

	// A nil pool IS legacy mode: main.go only constructs the pool when
	// VIDO_SUBTITLE_PIPELINE_MODE=pipeline, so the composition must degrade to
	// the shipped behaviour rather than panic or drop the enrichment.
	composed := ComposeScanCallback(prev, nil, nil)
	require.NotNil(t, composed)
	composed()

	assert.Equal(t, 1, calls, "legacy mode must still run post-scan enrichment exactly once")
}

func TestComposeScanCallback_NilPreviousCallbackStillEnqueues(t *testing.T) {
	enqueued := make(chan int, 1)

	pool := NewWorkerPool(idleProcessor(), nil,
		WithCandidateFinders(&fakeMovieFinder{movies: []models.Movie{{ID: "m1"}, {ID: "m2"}}}, nil))
	require.NoError(t, pool.Start(context.Background()))
	defer pool.Stop()

	composed := ComposeScanCallback(nil, pool, func(n int, err error) {
		assert.NoError(t, err)
		enqueued <- n
	})
	composed()

	assert.Equal(t, 2, <-enqueued, "both enumerated movies should be offered to the pool")
}

func TestComposeScanCallback_EnqueueErrorIsReportedNotSwallowed(t *testing.T) {
	done := make(chan error, 1)

	pool := NewWorkerPool(idleProcessor(), nil,
		WithCandidateFinders(&fakeMovieFinder{err: errEnumerate}, nil))
	require.NoError(t, pool.Start(context.Background()))
	defer pool.Stop()

	composed := ComposeScanCallback(nil, pool, func(_ int, err error) { done <- err })
	composed()

	assert.ErrorIs(t, <-done, errEnumerate,
		"a failed enumeration must reach the observer — a scan that silently enqueues nothing is the failure mode of record")
}
