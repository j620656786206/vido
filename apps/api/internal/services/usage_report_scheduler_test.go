package services

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingTicker struct{ n atomic.Int32 }

func (c *countingTicker) Tick(context.Context) { c.n.Add(1) }

func TestUsageReportScheduler_ChecksAtStartThenEveryInterval(t *testing.T) {
	ticker := &countingTicker{}
	s := NewUsageReportScheduler(ticker)
	s.interval = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Start(ctx)
	require.Eventually(t, func() bool { return ticker.n.Load() >= 3 }, time.Second, 5*time.Millisecond)
	s.Stop()
}

func TestUsageReportScheduler_StopWaitsForTheLoopToExit(t *testing.T) {
	ticker := &countingTicker{}
	s := NewUsageReportScheduler(ticker)
	s.interval = 10 * time.Millisecond

	go s.Start(context.Background())
	require.Eventually(t, func() bool { return ticker.n.Load() >= 1 }, time.Second, 5*time.Millisecond)
	s.Stop()
	after := ticker.n.Load()
	time.Sleep(40 * time.Millisecond)
	assert.Equal(t, after, ticker.n.Load(), "no tick after Stop returned — main closes the DB next")
	s.Stop() // idempotent
}

func TestUsageReportScheduler_HonorsContextCancellation(t *testing.T) {
	s := NewUsageReportScheduler(&countingTicker{})
	ctx, cancel := context.WithCancel(context.Background())
	exited := make(chan struct{})
	go func() { s.Start(ctx); close(exited) }()

	cancel()
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("Start did not return after ctx cancel")
	}
}

func TestUsageReportScheduler_StopWithoutStartReturnsAtOnce(t *testing.T) {
	s := NewUsageReportScheduler(&countingTicker{})
	begin := time.Now()
	s.Stop()
	assert.Less(t, time.Since(begin), time.Second)
}
