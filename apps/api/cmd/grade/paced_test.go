package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingCompleter struct{ calls int }

func (c *countingCompleter) CompleteText(context.Context, string, string, int) (string, error) {
	c.calls++
	return "ok", nil
}

func TestPacedCompleter_SpacesCallsByTheGap(t *testing.T) {
	inner := &countingCompleter{}
	var slept []time.Duration
	p := &pacedCompleter{inner: inner, gap: 10 * time.Second, sleep: func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}}
	for i := 0; i < 3; i++ {
		_, err := p.CompleteText(context.Background(), "", "u", 1)
		require.NoError(t, err)
	}
	assert.Equal(t, 3, inner.calls)
	// The first call is immediate; later ones take the next free slot. The
	// fake sleep does not advance the clock, so the slots pile up: 10 s, 20 s.
	require.Len(t, slept, 2)
	assert.InDelta(t, 10*time.Second, slept[0], float64(time.Second))
	assert.InDelta(t, 20*time.Second, slept[1], float64(time.Second))
}

func TestPacedCompleter_ZeroGapIsTheInnerCompleter(t *testing.T) {
	inner := &countingCompleter{}
	assert.Same(t, inner, newPacedCompleter(inner, 0))
}

func TestPacedCompleter_CancelledWaitReturnsCtxErr(t *testing.T) {
	p := &pacedCompleter{inner: &countingCompleter{}, gap: time.Hour, sleep: sleepCtx}
	_, err := p.CompleteText(context.Background(), "", "u", 1) // first: immediate
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = p.CompleteText(ctx, "", "u", 1)
	assert.ErrorIs(t, err, context.Canceled)
}
