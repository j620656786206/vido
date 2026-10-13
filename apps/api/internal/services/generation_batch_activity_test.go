package services

// disc-2026-09-activity-batch-row-count-in-flight-index — the Activity row
// counts FINISHED items, the way the batch dialog's 已完成 does.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

func TestGenerationBatch_ActivityCountsFinishedItems(t *testing.T) {
	release := make(chan struct{})
	runner := &fakeGenerationRunner{available: true, onCall: func(_ context.Context, mediaID string) error {
		switch mediaID {
		case uuidB:
			return errors.New("boom")
		case uuidC:
			<-release
		}
		return nil
	}}
	finder := &fakeCandidateFinder{movies: []models.Movie{
		genMovie(uuidA, "A", "/m/a.mkv"),
		genMovie(uuidB, "B", "/m/b.mkv"),
		genMovie(uuidC, "C", "/m/c.mkv"),
		genMovie(uuidD, "D", "/m/d.mkv"),
	}}
	p, _ := newTestGenerationProcessor(t, runner, finder, 5)
	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)

	// A done, B failed, C running: 2 of 4 finished — 50%, not "3 / 4" and 75%.
	require.Eventually(t, func() bool {
		active, pct, current, total, item := p.ActivityProgress()
		return active && current == 2 && total == 4 && pct == 50 && item == "C"
	}, time.Second, time.Millisecond)

	close(release)
	waitUntilIdle(t, p)
}

func TestGenerationBatch_ActivityStartsAtZero(t *testing.T) {
	release := make(chan struct{})
	runner := &fakeGenerationRunner{available: true, onCall: func(context.Context, string) error {
		<-release
		return nil
	}}
	p, _ := newTestGenerationProcessor(t, runner, threeMovies(), 5)
	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)

	// The first item has only just started: no bar yet (it used to show 33%).
	require.Eventually(t, func() bool {
		active, pct, current, _, item := p.ActivityProgress()
		return active && current == 0 && pct == 0 && item == "A"
	}, time.Second, time.Millisecond)

	close(release)
	waitUntilIdle(t, p)
}
