package subtitle

// Story 9R-10b AC #4 / AC #8 — the scan-complete callback COMPOSES, never
// replaces.
//
// The failure this suite exists to prevent is the obvious one: wiring the
// auto-trigger by calling SetOnScanComplete a second time, which drops post-scan
// enrichment on the floor and would only surface weeks later.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeScanCallback_RunsPrevBeforeNext(t *testing.T) {
	var order []string
	composed := ComposeScanCallback(
		func() { order = append(order, "enrichment") },
		func() { order = append(order, "auto-generation") },
	)
	composed()

	assert.Equal(t, []string{"enrichment", "auto-generation"}, order,
		"enrichment must run first — the auto-trigger's language routing depends on the metadata it writes")
}

func TestComposeScanCallback_EachHalfRunsExactlyOncePerScan(t *testing.T) {
	var prevCalls, nextCalls int
	composed := ComposeScanCallback(
		func() { prevCalls++ },
		func() { nextCalls++ },
	)
	composed()
	composed()

	assert.Equal(t, 2, prevCalls)
	assert.Equal(t, 2, nextCalls)
}

func TestComposeScanCallback_NilNextReturnsPrevUntouched(t *testing.T) {
	called := 0
	prev := func() { called++ }

	composed := ComposeScanCallback(prev, nil)
	composed()

	assert.Equal(t, 1, called,
		"with the feature absent the caller must get back exactly what it had — no wrapper to reason about")
}

func TestComposeScanCallback_NilPrevReturnsNext(t *testing.T) {
	called := 0
	composed := ComposeScanCallback(nil, func() { called++ })
	composed()
	assert.Equal(t, 1, called)
}

func TestComposeScanCallback_BothNilIsASafeNoOp(t *testing.T) {
	composed := ComposeScanCallback(nil, nil)
	require.NotNil(t, composed, "the scanner stores this value and calls it unconditionally")
	assert.NotPanics(t, func() { composed() })
}

// TestComposeScanCallback_PanicInPrevPropagates pins a DELIBERATE choice rather
// than an oversight (AC #8 asks for the semantic to be stated explicitly).
//
// Composition is transparent: today a panic in the enrichment callback reaches
// the scanner's goroutine and takes the process down. Recovering it here would
// change that crash semantic as a side effect of adding a subtitle feature, and
// would bury a real defect. So the panic propagates, and `next` is not reached —
// exactly what happens today, when `prev` is the only callback there is.
func TestComposeScanCallback_PanicInPrevPropagatesAndSkipsNext(t *testing.T) {
	nextCalled := false
	composed := ComposeScanCallback(
		func() { panic("enrichment exploded") },
		func() { nextCalled = true },
	)

	assert.PanicsWithValue(t, "enrichment exploded", func() { composed() })
	assert.False(t, nextCalled,
		"next must not run after prev panicked — the composition does not paper over a crash")
}
